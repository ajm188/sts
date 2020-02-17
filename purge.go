package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

func Purge(sqsAPI SQS) error {
	reader := bufio.NewReader(os.Stdin)
	stop := false
	for !stop {
		message, err := sqsAPI.Receive()
		if err != nil {
			return err
		}

		if message == nil {
			continue
		}
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
				fmt.Printf("Not purging. Since queue is FIFO, exiting now.\n")
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
