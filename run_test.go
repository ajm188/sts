package main

import (
	"errors"
	"testing"

	"github.com/aws/aws-sdk-go/service/sqs"
)

type FakeSQS struct {
	shouldErrorOnGetQueueAttributes bool
	shouldErrorOnReceive            bool
	numMessagesInQueue              string
	messageRetention                string
	sentTimestampOnMessage          string
}

func (this *FakeSQS) GetQueueAttributes(in *sqs.GetQueueAttributesInput) (*sqs.GetQueueAttributesOutput, error) {
	if this.shouldErrorOnGetQueueAttributes {
		return nil, errors.New("")
	}

	out := &sqs.GetQueueAttributesOutput{
		Attributes: map[string]*string{
			"ApproximateNumberOfMessages": &this.numMessagesInQueue,
			"MessageRetentionPeriod":      &this.messageRetention,
		},
	}
	return out, nil
}

func (this *FakeSQS) Receive(opts map[string]bool) (*sqs.Message, error) {
	if this.shouldErrorOnReceive {
		return nil, errors.New("")
	}
	return nil, nil // TODO
}

func (this *FakeSQS) DeleteMessage(handle *string) error {
	return nil
}

func (this *FakeSQS) SendAll(messages []string, user string) error {
	return nil
}

func TestCalibrate(t *testing.T) {
	service := &Service{0, 0}
	testTables := []struct {
		shouldError    bool
		expectedChange CalibrationChange
		sqs            *FakeSQS
	}{
		{
			shouldError:    true,
			expectedChange: TWEET_SAME,
			sqs: &FakeSQS{
				shouldErrorOnGetQueueAttributes: true,
				shouldErrorOnReceive:            false,
				numMessagesInQueue:              "",
				messageRetention:                "",
				sentTimestampOnMessage:          "",
			},
		},
		{
			// unparseable backlog
			shouldError:    true,
			expectedChange: TWEET_SAME,
			sqs: &FakeSQS{
				shouldErrorOnGetQueueAttributes: false,
				shouldErrorOnReceive:            false,
				numMessagesInQueue:              "abc",
				messageRetention:                "",
				sentTimestampOnMessage:          "",
			},
		},
		{
			// unparseable retention
			shouldError:    true,
			expectedChange: TWEET_SAME,
			sqs: &FakeSQS{
				shouldErrorOnGetQueueAttributes: false,
				shouldErrorOnReceive:            false,
				numMessagesInQueue:              "10",
				messageRetention:                "abc",
				sentTimestampOnMessage:          "",
			},
		},
	}

	for _, test := range testTables {
		change, err := service.Calibrate(test.sqs)
		if test.shouldError {
			if err == nil {
				t.Errorf("Expected an error but got none.")
			}
		} else {
			if err != nil {
				t.Errorf("Expected no error but got %s.", err)
			}
		}

		if change != test.expectedChange {
			t.Errorf("Expected a calibration of %d, but got %d.", test.expectedChange, change)
		}
	}
}
