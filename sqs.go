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
	Receive() (*sqs.Message, error)
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

func (this *SQSImpl) Receive() (*sqs.Message, error) {
	/* Why is this number 20, even though we only want the first message? Well, ...
	* From Amazon's docs:
	* > Short poll is the default behavior where a weighted random set of
	* > machines is sampled on a ReceiveMessage call. Thus, only the messages
	* > on the sampled machines are returned. If the number of messages in the
	* > queue is small (fewer than 1,000), you most likely get fewer messages
	* > than you requested per ReceiveMessage call. If the number of messages
	* > in the queue is extremely small, you might not receive any messages in
	* > a particular ReceiveMessage response. If this happens, repeat the request.
	*
	* I initially implemented some retry logic, but it looked really gross. So,
	* I figured, if you receive _less_ than the amount you asked, then asking
	* for 10 when you only want 1 seems like a reasonable workaround. We're
	* only going to use (and later delete) the first message anyway.
	 */
	var maxMessages int64 = 10
	sentTimestampAttribute := "SentTimestamp"
	log.Printf("[sqs_receive]: Retrieving one message from %s.\n", this.queueURL)
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
		log.Println("[sqs_receive]: Message received.")
		message := messages[0]
		return message, nil
	}
	log.Println("[sqs_receive]: No message received from queue.")
	return nil, nil
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
