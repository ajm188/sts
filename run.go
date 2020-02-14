package main

import (
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

func (this *Service) RunForever(twitter TwitterAPI, sqsAPI SQS) error {
	log.Println("Performing initial calibration.")
	_, err := this.Calibrate(sqsAPI)
	if err != nil {
		return err
	}

	calibrationErrors := make(chan error)
	tweetErrors := make(chan error)

	tweetWakeupChan := make(chan bool)

	go func(calibrationErrors chan error) {
		for true {
			change, err := this.Calibrate(sqsAPI)
			if err != nil {
				calibrationErrors <- err
			}

			switch change {
			case TWEET_FASTER:
				tweetWakeupChan <- true
			case TWEET_SLOWER:
			default:
			}

			log.Printf("Finished calibration iteration. Sleeping for %d seconds.\n", this.calibrationRate)
			time.Sleep(time.Duration(this.calibrationRate) * time.Second)
		}
	}(calibrationErrors)

	go func(tweetErrors chan error) {
		for true {
			tweet, err := this.Tweet(twitter, sqsAPI)
			if err != nil {
				if tweet != "" {
					// TODO: log the tweet text so we don't lose it forever
				}
				tweetErrors <- err
			}
			tweetSleepTime := atomic.LoadInt64(&this.tweetRate)
			log.Printf("Finished tweet iteration. Sleeping for %d seconds.\n", tweetSleepTime)

			var sleeper func(int64, int64)
			sleeper = func(totalTimeElapsed, remainingTime int64) {
				localStart := time.Now()
				log.Printf("[tweet_sleep_loop]: Sleeping for up to %d seconds before tweeting again.\n", remainingTime)

				select {
				case <-tweetWakeupChan:
					newTotal := atomic.LoadInt64(&this.tweetRate)
					timeElapsed := int64(time.Since(localStart).Seconds()) + totalTimeElapsed
					log.Printf(
						"[tweet_sleep_loop]: Detected changed tweet rate. New rate is: %d. Total time slept this cycle is: %d.\n",
						newTotal,
						timeElapsed,
					)
					if timeElapsed < newTotal {
						sleeper(timeElapsed, newTotal-timeElapsed)
					} else {
						log.Println("[tweet_sleep_loop]: sleep time already exceeds the new rate. Preparing a new tweet immediately.")
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
	TWEET_FASTER = iota
	TWEET_SLOWER
	TWEET_SAME
)

// Compute how long we can afford to sleep between tweets such that tweets
// don't drop off the queue from retention policy.
// Roughly, this is "seconds of retention" / "num messages in queue".
func (this *Service) Calibrate(sqsAPI SQS) (CalibrationChange, error) {
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
	tweetRate := int64(retention / (backlog * 10))
	log.Printf("[calibration]: Found %d messages in the backlog.\n", backlog)
	log.Printf("[calibration]: Message retention period is %d.\n", retention)
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

func (this *Service) Tweet(twitter TwitterAPI, sqs SQS) (string, error) {
	log.Println("Getting a tweet from the queue.")
	message, err := sqs.Receive()

	if err != nil {
		return "", err
	}

	if message == nil {
		log.Println("Got an empty message from the queue. Not tweeting that.")
		return "", nil
	}

	if *message.Body == "" {
		log.Println("Got an empty message from the queue. Not tweeting that. Still going to delete it though.")
		return "", sqs.DeleteMessage(message.ReceiptHandle)
	}

	tweet, err := twitter.Tweet(*message.Body, nil)
	if err != nil {
		return "", err
	}

	return tweet, sqs.DeleteMessage(message.ReceiptHandle)
}
