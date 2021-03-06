package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strconv"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/sqs"
)

type SQSConfig struct {
	queueName, region string
}

type SQS interface {
	GetQueueAttributes(*sqs.GetQueueAttributesInput) (*sqs.GetQueueAttributesOutput, error)
	Receive(context.Context) (*sqs.Message, error)
	DeleteMessage(*string) error
	SendAll([]string, string) error
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

func (this *SQSImpl) Receive(ctx context.Context) (*sqs.Message, error) {
	logger, ok := ctx.Value(STSContextKey("logger")).(*log.Logger)
	if !ok {
		return nil, NoLoggerInContext()
	}
	var maxMessages int64 = 1
	sentTimestampAttribute := "SentTimestamp"
	logger.Printf("[sqs_receive]: Retrieving one message from %s.\n", this.queueURL)
	resp, err := this.sqsClient.ReceiveMessage(
		&sqs.ReceiveMessageInput{
			QueueUrl:            &this.queueURL,
			MaxNumberOfMessages: &maxMessages,
			AttributeNames:      []*string{&sentTimestampAttribute},
		},
	)

	if err != nil {
		return nil, err
	}

	messages := resp.Messages
	if len(messages) > 0 {
		logger.Println("[sqs_receive]: Message received.")
		message := messages[0]
		return message, nil
	}
	logger.Println("[sqs_receive]: No message received from queue.")
	return nil, errors.New("No message received from queue.")
}

func (this *SQSImpl) DeleteMessage(receiptHandle *string) error {
	log.Println("Deleting message from queue.")
	_, err := this.sqsClient.DeleteMessage(
		&sqs.DeleteMessageInput{
			QueueUrl:      &this.queueURL,
			ReceiptHandle: receiptHandle,
		},
	)
	return err
}

func (this *SQSImpl) SendMessageBatch(input *sqs.SendMessageBatchInput) (*sqs.SendMessageBatchOutput, error) {
	input.QueueUrl = &this.queueURL
	return this.sqsClient.SendMessageBatch(input)
}

func (this *SQSImpl) SendAll(messages []string, group string) error {
	log.Printf("[sqs_sendall]: Sending %d messages in batches of 10.\n", len(messages))
	for i := 0; i < len(messages); i += 10 {
		entries := make([]*sqs.SendMessageBatchRequestEntry, 0)
		for j := 0; j < 10 && i+j < len(messages); j++ {
			id := fmt.Sprintf("%d", i+j)
			entry := &sqs.SendMessageBatchRequestEntry{
				Id:             &id,
				MessageGroupId: &group,
				MessageBody:    &messages[i+j],
			}
			entries = append(entries, entry)
		}
		log.Printf("[sqs_sendall]: Sending %d messages to SQS.\n", len(entries))
		output, err := this.SendMessageBatch(&sqs.SendMessageBatchInput{
			Entries: entries,
		})
		for j := 0; j < len(output.Failed); j++ {
			batchErrorEntry := output.Failed[j]
			id, err := strconv.Atoi(*batchErrorEntry.Id)
			if err != nil {
				log.Printf("[sqs_sendall_failure]: Cannot parse message id %s as int.\n", *batchErrorEntry.Id)
				continue
			}
			log.Printf("[sqs_sendall]: Failed to enqueue %s\n", messages[id])
		}
		if err != nil {
			return err
		}
	}
	return nil
}
