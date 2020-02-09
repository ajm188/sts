package main

import (
	"log"
)

func BatchUpdate(sqs SQS, tweetSource TweetProvider, username string) error {
	tweets, err := tweetSource.All()
	if err != nil {
		return err
	}

	log.Printf("Found %d tweets in %s.\n", len(tweets), tweetSource.Name())
	return sqs.SendAll(tweets, username)
}
