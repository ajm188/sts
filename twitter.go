package main

import (
	"net/http"

	"github.com/dghubble/go-twitter/twitter"
	"github.com/dghubble/oauth1"
)

type TwitterCreds struct {
	consumerKey string
	consumerSecret string
	accessToken string
	accessSecret string
}

type StatusService interface {
	Update(string, *twitter.StatusUpdateParams) (*twitter.Tweet, *http.Response, error)
}

type TwitterAPI interface {
	GetStatusService() StatusService
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
