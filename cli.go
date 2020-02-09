package main

import (
	"fmt"

	"github.com/urfave/cli/v2"
	"golang.org/x/sys/unix"
)

type RunArgs struct {
	sqs             *SQSConfig
	twitter         *TwitterCreds
	calibrationRate int
}

func getSQSConfig(c *cli.Context) *SQSConfig {
	return &SQSConfig{
		queueName: c.Value("queue").(string),
		region:    c.Value("region").(string),
	}
}

func ParseRunArgs(c *cli.Context) (*RunArgs, error) {
	sqsConfig := getSQSConfig(c)
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

type BatchUpdateArgs struct {
	sqs       *SQSConfig
	user      string
	filename  string
	delimiter string
}

func ParseBatchUpdateArgs(c *cli.Context) (*BatchUpdateArgs, error) {
	sqsConfig := getSQSConfig(c)

	user := c.Value("user").(string)
	filename := c.Value("file").(string)
	delimiter := c.Value("delimiter").(string)

	if err := unix.Access(filename, unix.R_OK); err != nil {
		return nil, err
	}
	return &BatchUpdateArgs{
		sqs:       sqsConfig,
		user:      user,
		filename:  filename,
		delimiter: delimiter,
	}, nil
}
