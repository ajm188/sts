package main

import (
	"log"
	"os"
	"path"

	"github.com/urfave/cli/v2"
)

func main() {
	workDir, err := os.Getwd()
	if err != nil {
		panic(err)
	}

	app := &cli.App{
		Commands: []*cli.Command{
			{
				Name:  "run",
				Usage: "run the sts daemon",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:     "region",
						Aliases:  []string{"r"},
						Usage:    "",
						Required: true,
					},
					&cli.StringFlag{
						Name:     "queue",
						Aliases:  []string{"q"},
						Usage:    "",
						Required: true,
					},
					&cli.StringFlag{
						Name:     "twitter-key",
						Usage:    "",
						FilePath: path.Join(workDir, ".twitter", "key"),
					},
					&cli.StringFlag{
						Name:     "twitter-consumer-secret",
						Usage:    "",
						FilePath: path.Join(workDir, ".twitter", "consumer-secret"),
					},
					&cli.StringFlag{
						Name:     "twitter-token",
						Usage:    "",
						FilePath: path.Join(workDir, ".twitter", "token"),
					},
					&cli.StringFlag{
						Name:     "twitter-access-secret",
						Usage:    "",
						FilePath: path.Join(workDir, ".twitter", "access-secret"),
					},
					&cli.IntFlag{
						Name:  "calibration-rate",
						Usage: "How often (in seconds), to update tweeting rate.",
						Value: 600,
					},
				},
				Action: func(c *cli.Context) error {
					args, err := ParseArgs(c)

					if err != nil {
						return err
					}

					log.Println("Initialing API components.")

					twitter := NewTwitter(args.twitter)
					sqs, err := NewSQS(args.sqs)
					if err != nil {
						return err
					}

					log.Println("Running forever ....")

					return NewService(args).RunForever(twitter, sqs)
				},
			},
			{
				Name:  "batch-update",
				Usage: "Add a batch of tweets to the queue.",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:     "user",
						Aliases:  []string{"u"},
						Required: true,
					},
					&cli.StringFlag{
						Name:     "file",
						Aliases:  []string{"f"},
						Required: true,
					},
					&cli.StringFlag{
						Name:    "delimiter",
						Aliases: []string{"d"},
						Value:   "====================",
					},
					&cli.StringFlag{
						Name:     "region",
						Aliases:  []string{"r"},
						Usage:    "",
						Required: true,
					},
					&cli.StringFlag{
						Name:     "queue",
						Aliases:  []string{"q"},
						Usage:    "",
						Required: true,
					},
				},
				Action: func(c *cli.Context) error {
					return nil
				},
			},
		},
	}

	log.SetFlags(log.Ldate | log.LUTC | log.Lmicroseconds | log.Lshortfile)

	err = app.Run(os.Args)
	if err != nil {
		log.Fatal(err)
	}
}
