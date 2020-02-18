package main

import (
	"errors"
	"io/ioutil"
	"log"
	"testing"
)

func init() {
	log.SetOutput(ioutil.Discard)
}

func TestRetry(t *testing.T) {
	count := 0
	retrier := &BasicRetrier{
		delayMillis: 0,
		maxAttempts: 10,
		description: "",
	}

	testTables := []struct {
		f           func() (interface{}, error)
		startCount  int
		endCount    int
		shouldError bool
	}{
		{
			func() (interface{}, error) {
				count++
				return nil, errors.New("")
			},
			0,
			10,
			true,
		},
		{
			func() (interface{}, error) {
				count++
				if count < 5 {
					return nil, errors.New("")
				}
				return nil, nil
			},
			0,
			5,
			false,
		},
	}

	for _, test := range testTables {
		count = test.startCount
		_, err := Retry(test.f, retrier)
		if test.shouldError {
			if err == nil {
				t.Errorf("Expected an error but no error returned.")
			}
		} else {
			if err != nil {
				t.Errorf("Expected no errors but got %s.", err)
			}
		}

		if count != test.endCount {
			t.Errorf("Expected retrying to result in a count of %d, but got a count of %d.", test.endCount, count)
		}
	}
}
