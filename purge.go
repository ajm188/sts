package main

import (
	"bufio"
	"context"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"strings"

	"github.com/aws/aws-sdk-go/service/sqs"
)

func Purge(ctx context.Context, sqsAPI SQS) error {
	logger, ok := ctx.Value(STSContextKey("logger")).(*log.Logger)
	if !ok {
		return NoLoggerInContext()
	}

	reader := bufio.NewReader(os.Stdin)
	stop := false
	for !stop {
		logger.Println("Getting a message from the queue.")
		childLogger := getLogger()
		childLogger.SetOutput(ioutil.Discard)
		childCtx := context.WithValue(ctx, STSContextKey("logger"), childLogger)
		msg, err := Retry(
			func() (interface{}, error) {
				return sqsAPI.Receive(childCtx)
			},
			&BasicRetrier{delayMillis: 50, maxAttempts: 500, description: "SQS ReceiveMessage()"},
		)
		if err != nil {
			return err
		}

		if msg == nil {
			continue
		}
		message := msg.(*sqs.Message)
		for true {
			fmt.Printf("Next message in queue:\n%s\n=> Purge? [yes/no]: ", *message.Body)
			text, _ := reader.ReadString('\n')
			text = strings.ToLower(strings.Replace(text, "\n", "", -1))

			switch text {
			case "yes":
				err := sqsAPI.DeleteMessage(message.ReceiptHandle)
				if err != nil {
					return err
				}
				break
			case "no":
				logger.Printf("Not purging. Since queue is FIFO, exiting now.\n")
				return nil
			default:
				fmt.Printf("Was expecting 'yes' or 'no'. Got '%s'.\n", text)
				continue
			}
		}

		for true {
			fmt.Printf("Continue? [yes/no]: ")
			text, _ := reader.ReadString('\n')
			text = strings.ToLower(strings.Replace(text, "\n", "", -1))
			switch text {
			case "yes":
				break
			case "no":
				stop = true
				break
			default:
				fmt.Printf("Was expecting 'yes' or 'no'. Got '%s'.\n", text)
				continue
			}
		}
	}
	return nil
}
