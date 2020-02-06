package main

import (
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
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:     "queue, q",
				Usage:    "",
				Required: true,
			},
			&cli.StringFlag{
				Name:     "handle",
				Usage:    "",
				Required: true,
			},
			&cli.StringFlag{
				Name:     "twitter-key",
				Usage:    "",
				FilePath: path.Join(workDir, ".twitter", "key"),
			},
			&cli.StringFlag{
				Name:     "twitter-token",
				Usage:    "",
				FilePath: path.Join(workDir, ".twitter", "token"),
			},
			&cli.IntFlag{
				Name:  "calibration-rate",
				Usage: "How often (in seconds), to update tweeting rate.",
				Value: 600,
			},
		},

		Commands: []*cli.Commands{
			{
				Name:    "run",
				Aliases: []string{"r"},
				Usage:   "Run the STS service",
				Action: func(c *cli.Context) error {
					args, err := ParseArgs(c)

					if err != nil {
						return err
					}

					return nil
				},
			},
		},
	}
}
