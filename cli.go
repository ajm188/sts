package main

import (
	"fmt"

	"github.com/urfave/cli/v2"
)

type RunArgs struct {
	sqs             *SQSConfig
	twitter         *TwitterCreds
	calibrationRate int
}

func ParseRunArgs(c *cli.Context) (*RunArgs, error) {
	sqsConfig := &SQSConfig{
		queueName: c.Value("queue").(string),
		region:    c.Value("region").(string),
	}

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
		sqs:             sqsConfig,
		twitter:         twitterCreds,
		calibrationRate: calibrationRate,
	}, nil
}
