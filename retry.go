package main

import (
	"log"
	"time"
)

type Retrier interface {
	NextDelayMillis(int) int
	MaxAttempts() int
	Description() string
}

func Retry(f func() (interface{}, error), retrier Retrier) (interface{}, error) {
	log.Printf("[retry]: Attempting to %s for up to %d attempts.\n", retrier.Description(), retrier.MaxAttempts())
	var err error
	var res interface{}
	for attempt := 0; attempt < retrier.MaxAttempts(); attempt++ {
		//log.Printf("[retry]: Attempt %d.\n", attempt+1)
		res, err = f()
		if err == nil {
			log.Printf("[retry]: Got a non-error result on attempt %d.\n", attempt+1)
			return res, err
		}
		time.Sleep(time.Duration(retrier.NextDelayMillis(attempt)) * time.Millisecond)
	}

	return nil, nil
}

type BasicRetrier struct {
	delayMillis int
	maxAttempts int
	description string
}

func (retrier *BasicRetrier) NextDelayMillis(attempt int) int {
	return retrier.delayMillis
}

func (retrier *BasicRetrier) MaxAttempts() int {
	return retrier.maxAttempts
}

func (retrier *BasicRetrier) Description() string {
	return retrier.description
}
