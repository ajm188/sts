package main

import (
	"github.com/aws/aws-sdk-go/service/sqs"
)

type SQS interface {
	GetQueueAttributes(*sqs.GetQueueAttributesInput) (*sqs.GetQueueAttributesOutput, error)
	Recieve() (string, error)
} // TODO: implement this
