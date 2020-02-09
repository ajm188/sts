package main

import (
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
	Receive() (string, error)
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
		log.Println("[sqs_receive]: Message received. Deleting message from queue.")
		message := messages[0]
		err := this.DeleteMessage(message.ReceiptHandle)
		if err != nil {
			return "", err
		}
		return *message.Body, nil
	}
	log.Println("[sqs_receive]: No message received from queue.")
	return "", nil
}

func (this *SQSImpl) DeleteMessage(receiptHandle *string) error {
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
		for j := 0; j < 10 && i + j < len(messages); j++ {
			id := fmt.Sprintf("%d", i + j)
			entry := &sqs.SendMessageBatchRequestEntry{
				Id:             &id,
				MessageGroupId: &group,
				MessageBody:    &messages[i+j],
			}
			entries = append(entries, entry)
		}
		log.Println("[sqs_sendall]: Sending 10 messages to SQS.")
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
