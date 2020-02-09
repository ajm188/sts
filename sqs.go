package main

import (
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/sqs"
)

type SQS interface {
	GetQueueAttributes(*sqs.GetQueueAttributesInput) (*sqs.GetQueueAttributesOutput, error)
	Receive(string) (string, error)
}

type SQSImpl struct {
	sqsClient *sqs.SQS
}

func NewSQS() SQS {
	sess := session.Must(session.NewSession())
	return &SQSImpl{sqs.New(sess)}
}

func (this *SQSImpl) GetQueueAttributes(input *sqs.GetQueueAttributesInput) (*sqs.GetQueueAttributesOutput, error) {
	return this.sqsClient.GetQueueAttributes(input)
}

func (this *SQSImpl) Receive(queueURL string) (string, error) {
	var maxMessages int64 = 1
	resp, err := this.sqsClient.ReceiveMessage(
		&sqs.ReceiveMessageInput{
			QueueUrl:            &queueURL,
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
