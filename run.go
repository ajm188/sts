package main

import (
	"github.com/urfave/cli/v2"
)

type RunArgs struct {
	queue           string
	handle          string
	twitterKey      string
	twitterToken    string
	calibrationRate int
}

func ParseArgs(c *cli.Context) (*RunArgs, error) {
	queue := c.Value("queue").(string)
	handle := c.Value("handle").(string)

	twitterKey := c.Value("twitter-key").(string)
	twitterToken := c.Value("twitter-token").(string)

	calibrationRate := c.Value("calibration-rate").(int)

	if calibrationRate < 0 {
		return nil, fmt.Errorf("Calibration Rate cannot be negative. Got %d.", calibrationRate)
	}

	return &RunArgs{
		queue:           queue,
		handle:          handle,
		twitterKey:      twitterKey,
		twitterToken:    twitterToken,
		calibrationRate: calibrationRate,
	}, nil
}
