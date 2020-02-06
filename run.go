package main

import (
	"fmt"

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
		consumerKey: c.Value("twitter-key").(string),
		consumerSecret: c.Value("twitter-consumer-secret").(string),
		accessToken: c.Value("twitter-token").(string),
		accessSecret: c.Value("twitter-access-secret").(string),
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
