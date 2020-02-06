package main

import (
	"fmt"
	"strconv"
	"sync/atomic"

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
