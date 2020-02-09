package main

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/sqs"
)

type SQSConfig struct {
	queueName, region string
}

type SQS interface {
	GetQueueAttributes(*sqs.GetQueueAttributesInput) (*sqs.GetQueueAttributesOutput, error)
	Receive() (string, error)
}

type SQSImpl struct {
	sqsClient *sqs.SQS
	queueURL  string
}

func NewSQS(conf *SQSConfig) (SQS, error) {
	sess := session.Must(session.NewSession())
	client := sqs.New(sess, &aws.Config{Region: aws.String(conf.region)})

	output, err := client.GetQueueUrl(
		&sqs.GetQueueUrlInput{
			QueueName: &conf.queueName,
		},
	)
	if err != nil {
		return nil, err
	}
	sqsImpl := SQSImpl{
		sqsClient: client,
		queueURL:  *output.QueueUrl,
	}
	return &sqsImpl, nil
}

func (this *SQSImpl) GetQueueAttributes(input *sqs.GetQueueAttributesInput) (*sqs.GetQueueAttributesOutput, error) {
	input.QueueUrl = &this.queueURL
	return this.sqsClient.GetQueueAttributes(input)
}

func (this *SQSImpl) Receive() (string, error) {
	var maxMessages int64 = 1
	resp, err := this.sqsClient.ReceiveMessage(
		&sqs.ReceiveMessageInput{
			QueueUrl:            &this.queueURL,
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
