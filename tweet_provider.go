package main

import (
	"log"
	"io/ioutil"
	"strings"
)

type TweetProvider interface {
	All() ([]string, error)
	Name() string
}

type FileTweetProvider struct {
	filename string
	delimiter string
}

func (this *FileTweetProvider) All() ([]string, error) {
	log.Printf("Scanning %s for tweets.\n", this.filename)
	bytes, err := ioutil.ReadFile(this.filename)
	if err != nil {
		return []string{}, nil
	}


	splitTweets := strings.Split(string(bytes), this.delimiter)
	tweets := make([]string, len(splitTweets))
	for i, rawTweet := range splitTweets {
		tweets[i] = rawTweet[:len(rawTweet) - 1]
	}
	return tweets, nil
}

func (this *FileTweetProvider) Name() string {
	return this.filename
}
