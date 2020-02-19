package main

import (
	"context"
	"log"
)

func BatchUpdate(ctx context.Context, sqs SQS, tweetSource TweetProvider, username string) error {
	logger, ok := ctx.Value(STSContextKey("logger")).(*log.Logger)
	if !ok {
		return NoLoggerInContext()
	}
	tweets, err := tweetSource.All()
	if err != nil {
		return err
	}

	logger.Printf("Found %d tweets in %s.\n", len(tweets), tweetSource.Name())
	return sqs.SendAll(tweets, username)
}
