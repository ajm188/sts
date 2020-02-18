package main

import (
	"log"
	"net/http"

	"github.com/dghubble/go-twitter/twitter"
	"github.com/dghubble/oauth1"
)

type TwitterCreds struct {
	consumerKey    string
	consumerSecret string
	accessToken    string
	accessSecret   string
}

type StatusService interface {
	Update(string, *twitter.StatusUpdateParams) (*twitter.Tweet, *http.Response, error)
}

type TwitterAPI interface {
	GetStatusService() StatusService
	Tweet(string, *twitter.StatusUpdateParams) (string, error)
}

type Twitter struct {
	*twitter.Client
}

func (t *Twitter) GetStatusService() StatusService {
	return t.Statuses
}

func NewTwitter(creds *TwitterCreds) TwitterAPI {
	config := oauth1.NewConfig(
		creds.consumerKey,
		creds.consumerSecret,
	)
	token := oauth1.NewToken(
		creds.accessToken,
		creds.accessSecret,
	)

	httpClient := config.Client(oauth1.NoContext, token)
	return &Twitter{twitter.NewClient(httpClient)}
}

func (t *Twitter) Tweet(text string, params *twitter.StatusUpdateParams) (string, error) {
	log.Printf("[tweet]: Sending tweet: %s\n", text)
	tweet, resp, err := t.GetStatusService().Update(text, params)
	if err != nil {
		log.Printf("%T\n", err)
	}
	switch resp.StatusCode {
	case 186: // tweet too long
		log.Printf("[tweet]: TWEET_TOO_LONG: %s\n", text)
		return "", err
	case 187:
		// Twitter thinks this was a dupe.
		// Log it and continue working through the queue.
		log.Printf("[tweet]: DUPE FOUND -- %s\n", text)
		return text, err
	default:
		return tweet.FullText, err
	}
}
