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
	err := this.Calibrate(sqsAPI)
	if err != nil {
		return err
	}

	calibrationErrors := make(chan error)
	tweetErrors := make(chan error)

	go func(calibrationErrors chan error) {
		for true {
			err := this.Calibrate(sqsAPI)
			if err != nil {
				calibrationErrors <- err
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
			time.Sleep(time.Duration(tweetSleepTime) * time.Second)
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

// Compute how long we can afford to sleep between tweets such that tweets
// don't drop off the queue from retention policy.
// Roughly, this is "seconds of retention" / "num messages in queue".
func (this *Service) Calibrate(sqsAPI SQS) error {
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
		return err
	}

	backlogStr := *resp.Attributes["ApproximateNumberOfMessages"]
	retentionStr := *resp.Attributes["MessageRetentionPeriod"]

	backlog, err := strconv.Atoi(backlogStr)
	if err != nil {
		return err
	}
	retention, err := strconv.Atoi(retentionStr)
	if err != nil {
		return err
	}
	tweetRate := int64(retention / (backlog * 10))
	log.Printf("[calibration]: Found %d messages in the backlog.\n", backlog)
	log.Printf("[calibration]: Message retention period is %d.\n", retention)
	log.Printf("[calibration]: Setting tweet rate to %d.\n", tweetRate)

	atomic.StoreInt64(&this.tweetRate, tweetRate)
	return nil
}

func (this *Service) Tweet(twitter TwitterAPI, sqs SQS) (string, error) {
	log.Println("Getting a tweet from the queue.")
	tweetText, err := sqs.Receive()

	if err != nil {
		return "", err
	}

	if tweetText == "" {
		log.Println("Got an empty message from the queue. Not tweeting that.")
		return "", nil
	}

	return twitter.Tweet(tweetText, nil)
}
