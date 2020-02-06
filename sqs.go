package main

import (
	"github.com/aws/aws-sdk-go/service/sqs"
)

type SQS interface {
	GetQueueAttributes(*sqs.GetQueueAttributesInput) (*sqs.GetQueueAttributesOutput, error)
	Recieve(string) (string, error)
} // TODO: implement this

type SQSImpl struct {
	sqsClient *sqs.SQS
}

func (this *SQSImpl) Recieve(queueURL string) (string, error) {
	maxMessages := 1
	resp, err := this.RecieveMessage(
		&sqs.RecieveMessageInput{
			QueueURL:            &queueURL,
			MaxNumberOfMessages: &maxMessages,
		},
	)

	if err != nil {
		return "", nil
	}

	messages := resp.Messages
	if len(messages) > 0 {
		return *messages[0].Body, nil
	}
	return "", nil
}
