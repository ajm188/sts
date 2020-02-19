package main

import (
	"context"
	"io/ioutil"
	"log"
	"strconv"
	"sync/atomic"
	"time"

	"github.com/aws/aws-sdk-go/service/sqs"
)

type Service struct {
	calibrationRate int
	tweetRate       int64
}

func NewService(args *RunArgs) *Service {
	return &Service{
		calibrationRate: args.calibrationRate,
		tweetRate:       0,
	}
}

func (this *Service) RunForever(ctx context.Context, twitter TwitterAPI, sqsAPI SQS) error {
	logger, ok := ctx.Value(STSContextKey("logger")).(*log.Logger)
	if !ok {
		return NoLoggerInContext()
	}
	logger.Println("Performing initial calibration.")
	_, err := this.Calibrate(ctx, sqsAPI)
	if err != nil {
		return err
	}

	calibrationErrors := make(chan error)
	tweetErrors := make(chan error)

	tweetWakeupChan := make(chan bool)

	go func(calibrationErrors chan error) {
		for true {
			change, err := this.Calibrate(ctx, sqsAPI)
			if err != nil {
				calibrationErrors <- err
			}

			switch change {
			case TWEET_FASTER:
				tweetWakeupChan <- true
			case TWEET_SLOWER:
			default:
			}

			logger.Printf("Finished calibration iteration. Sleeping for %d seconds.\n", this.calibrationRate)
			time.Sleep(time.Duration(this.calibrationRate) * time.Second)
		}
	}(calibrationErrors)

	go func(tweetErrors chan error) {
		for true {
			tweet, err := this.Tweet(ctx, twitter, sqsAPI)
			if err != nil {
				if tweet != "" {
					// TODO: log the tweet text so we don't lose it forever
				}
				tweetErrors <- err
			}
			tweetSleepTime := atomic.LoadInt64(&this.tweetRate)
			logger.Printf("Finished tweet iteration. Sleeping for %d seconds.\n", tweetSleepTime)

			var sleeper func(int64, int64)
			sleeper = func(totalTimeElapsed, remainingTime int64) {
				localStart := time.Now()
				logger.Printf("[tweet_sleep_loop]: Sleeping for up to %d seconds before tweeting again.\n", remainingTime)

				select {
				case <-tweetWakeupChan:
					newTotal := atomic.LoadInt64(&this.tweetRate)
					timeElapsed := int64(time.Since(localStart).Seconds()) + totalTimeElapsed
					logger.Printf(
						"[tweet_sleep_loop]: Detected changed tweet rate. New rate is: %d. Total time slept this cycle is: %d.\n",
						newTotal,
						timeElapsed,
					)
					if timeElapsed < newTotal {
						sleeper(timeElapsed, newTotal-timeElapsed)
					} else {
						logger.Println("[tweet_sleep_loop]: sleep time already exceeds the new rate. Preparing a new tweet immediately.")
					}
				case <-time.After(time.Duration(remainingTime) * time.Second):
				}

				return
			}
			sleeper(0, tweetSleepTime)
		}
	}(tweetErrors)

	for true {
		select {
		case calibrationErr := <-calibrationErrors:
			return calibrationErr
		case tweetErr := <-tweetErrors:
			return tweetErr
		default:
		}

		time.Sleep(time.Second)
	}

	return nil
}

type CalibrationChange int

const (
	TWEET_FASTER CalibrationChange = iota
	TWEET_SLOWER
	TWEET_SAME
)

// Compute how long we can afford to sleep between tweets such that tweets
// don't drop off the queue from retention policy.
// Roughly, this is "seconds of retention" / "num messages in queue".
func (this *Service) Calibrate(ctx context.Context, sqsAPI SQS) (CalibrationChange, error) {
	numMessagesAttribute := "ApproximateNumberOfMessages"
	retentionAttribute := "MessageRetentionPeriod"
	resp, err := sqsAPI.GetQueueAttributes(
		&sqs.GetQueueAttributesInput{
			AttributeNames: []*string{
				&numMessagesAttribute,
				&retentionAttribute,
			},
		},
	)

	if err != nil {
		return TWEET_SAME, err
	}

	backlogStr := *resp.Attributes["ApproximateNumberOfMessages"]
	retentionStr := *resp.Attributes["MessageRetentionPeriod"]

	backlog, err := strconv.Atoi(backlogStr)
	if err != nil {
		return TWEET_SAME, err
	}
	retention, err := strconv.Atoi(retentionStr)
	if err != nil {
		return TWEET_SAME, err
	}

	message, err := sqsAPI.Receive(ctx)
	remainingRetention := int64(retention)
	lastTweetEnqueueTime := int64(-1)

	if err != nil {
		// This error is either:
		// (1) intermittent, in which case the next round of calibration will
		// run fine
		// OR
		// (2) permanent, in which case it will be caught in the "tweet"
		// goroutine, and we'll consider it crash-worthy there.
		return TWEET_SAME, nil
		log.Println("[calibration]: No messages in the queue. Using the full retention period.")
	} else {
		timestampMillis, err := strconv.Atoi(*message.Attributes["SentTimestamp"])
		if err == nil {
			// just use the full retention window; it's probably fine, and
			// better than crashing
			// This assignment is technically redundant, but it makes this less
			// confusing to read.
			remainingRetention = int64(retention)
		} else {
			lastTweetEnqueueTime = int64(timestampMillis / 1000)
			elapsedSinceLastEnqueue := time.Now().Unix() - lastTweetEnqueueTime
			log.Printf("[calibration]: last tweet was enqueued at %s. %d seconds have passed since then.\n",
				time.Unix(lastTweetEnqueueTime, 0).String(),
				elapsedSinceLastEnqueue,
			)
			remainingRetention = int64(retention) - (time.Now().Unix() - lastTweetEnqueueTime)
		}
	}

	tweetRate := remainingRetention / int64(backlog)

	log.Printf("[calibration]: Found %d messages in the backlog.\n", backlog)
	log.Printf("[calibration]: Message retention period is %d.\n", retention)
	if lastTweetEnqueueTime > 0 {
		log.Printf("[calibration]: Given last enqueue time of %s, %d seconds of retention remain.\n", time.Unix(lastTweetEnqueueTime, 0).String(), remainingRetention)
	}
	log.Printf("[calibration]: Setting tweet rate to %d.\n", tweetRate)

	var change CalibrationChange
	switch {
	case tweetRate < this.tweetRate:
		change = TWEET_FASTER
	case tweetRate > this.tweetRate:
		change = TWEET_SLOWER
	default:
		change = TWEET_SAME
	}
	atomic.StoreInt64(&this.tweetRate, tweetRate)
	return change, nil
}

func (this *Service) Tweet(ctx context.Context, twitter TwitterAPI, sqsAPI SQS) (string, error) {
	logger, ok := ctx.Value(STSContextKey("logger")).(*log.Logger)
	if !ok {
		return "", NoLoggerInContext()
	}
	logger.Println("Getting a tweet from the queue.")
	childLogger := getLogger()
	childLogger.SetOutput(ioutil.Discard)
	childCtx := context.WithValue(ctx, STSContextKey("logger"), childLogger)
	msg, err := Retry(
		func() (interface{}, error) {
			return sqsAPI.Receive(childCtx)
		},
		&BasicRetrier{delayMillis: 50, maxAttempts: 500, description: "SQS ReceiveMessage()"},
	)

	if err != nil {
		return "", err
	}
	if msg == nil {
		return "", nil
	}

	message := msg.(*sqs.Message)
	if *message.Body == "" {
		log.Println("[tweet]: Got an empty message from the queue. Not tweeting that. Still going to delete it though.")
		return "", sqsAPI.DeleteMessage(message.ReceiptHandle)
	}

	tweet, err := twitter.Tweet(*message.Body, nil)
	if err != nil {
		return "", err
	}

	return tweet, sqsAPI.DeleteMessage(message.ReceiptHandle)
}
