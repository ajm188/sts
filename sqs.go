package main

import (
	"log"

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

	log.Printf("Finding URL for queue %s in %s.\n", conf.queueName, conf.region)
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
	log.Printf("Fetching attributes for queue %s\n", this.queueURL)
	input.QueueUrl = &this.queueURL
	return this.sqsClient.GetQueueAttributes(input)
}

func (this *SQSImpl) Receive() (string, error) {
	var maxMessages int64 = 1
	log.Printf("[sqs_receive]: Retrieving one message from %s.\n", this.queueURL)
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
		log.Println("[sqs_receive]: Message received.")
		return *messages[0].Body, nil
	}
	log.Println("[sqs_receive]: No message received from queue.")
	return "", nil
}
