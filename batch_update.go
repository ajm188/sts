package main

import (
	"context"
	"errors"
	"log"
)

var errTweetTooLong = errors.New("tweet is too long, twitter API is going to complain")

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

	err = nil
	for i, tweet := range tweets {
		// I'm finding that sending tweets that get _close_ to the 280 character limit get rejected from the API, even
		// though they're totally absolutely unequivocally less than 280 characters.
		if len(tweet) > 220 {
			log.Printf("tweet %d is too long (length: %d; text: %s). please edit and rerun batch-update", i, len(tweet), tweet)
			err = errTweetTooLong
		}
	}
	if err != nil {
		return err
	}

	return sqs.SendAll(tweets, username)
}
