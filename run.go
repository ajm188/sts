package main

import (
	"fmt"
	"strconv"
	"sync/atomic"
	"time"

	"github.com/aws/aws-sdk-go/service/sqs"
	"github.com/urfave/cli/v2"
)

type RunArgs struct {
	queue           string
	handle          string
	twitter         *TwitterCreds
	calibrationRate int
}

func ParseArgs(c *cli.Context) (*RunArgs, error) {
	queue := c.Value("queue").(string)
	handle := c.Value("handle").(string)

	twitterCreds := &TwitterCreds{
		consumerKey:    c.Value("twitter-key").(string),
		consumerSecret: c.Value("twitter-consumer-secret").(string),
		accessToken:    c.Value("twitter-token").(string),
		accessSecret:   c.Value("twitter-access-secret").(string),
	}

	calibrationRate := c.Value("calibration-rate").(int)

	if calibrationRate < 0 {
		return nil, fmt.Errorf("Calibration Rate cannot be negative. Got %d.", calibrationRate)
	}

	return &RunArgs{
		queue:           queue,
		handle:          handle,
		twitter:         twitterCreds,
		calibrationRate: calibrationRate,
	}, nil
}

type Service struct {
	queueName       string
	calibrationRate int
	tweetRate       int64
}

func NewService(args *RunArgs) *Service {
	return &Service{
		queueName:       args.queue,
		calibrationRate: args.calibrationRate,
		tweetRate:       0,
	}
}

func (this *Service) RunForever(twitter TwitterAPI, sqsAPI SQS) error {
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

		time.Sleep(5 * time.Second)
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
			QueueUrl: &this.queueName, // TODO: we currently pass the name, and we need a way to get the URL
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

	atomic.StoreInt64(&this.tweetRate, int64(retention/backlog))
	return nil
}

func (this *Service) Tweet(twitter TwitterAPI, sqs SQS) (string, error) {
	tweetText, err := sqs.Recieve(this.queueName)

	if err != nil {
		return "", err
	}

	if tweetText == "" {
		return "", nil
	}

	_, _, err = twitter.GetStatusService().Update(tweetText, nil)
	if err != nil {
		return tweetText, err
	}

	return tweetText, nil
}
