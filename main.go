package main

import (
	"errors"
	"log"
	"os"

	"strings"

	"code.cloudfoundry.org/lager"
	"github.com/chendrix/slacker/config"
	"github.com/nlopes/slack"
)

func main() {
	cfg, err := config.NewConfig(os.Args)
	if err != nil {
		panic(err)
	}

	err = cfg.Validate()
	if err != nil {
		panic(err)
	}
	logger := cfg.Logger
	logger.Info("Starting bot")

	api := slack.New(cfg.SlackAPIToken)

	l := log.New(os.Stdout, "slack-bot: ", log.Lshortfile|log.LstdFlags)
	slack.SetLogger(l)

	logger.Debug("Starting getting channel info from Slack")
	cs, err := api.GetChannels(true)
	if err != nil {
		logger.Error("Error retrieving channels", err)
		return
	}
	logger.Debug("Finished getting channel info from Slack")

	err = cfg.HydrateFromSlack(cs)
	if err != nil {
		logger.Error("Error hydrating channels", err)
		return
	}

	logger.Debug("Starting listening for Slack Real Time Messaging")
	rtm := api.NewRTM()
	go rtm.ManageConnection()

	for {
		select {
		case msg := <-rtm.IncomingEvents:
			switch ev := msg.Data.(type) {
			case *slack.MessageEvent:
				for _, channel := range cfg.Channels {
					logger.Debug("Message Received", lager.Data{
						"event": ev,
					})

					if ev.Msg.Channel == channel.SlackChannelID &&
						strings.Contains(ev.Msg.Text, channel.SlackTrigger) {
						logger.Debug("Message Response Triggered", lager.Data{
							"message": ev.Msg.Text,
						})
					}
				}
			case *slack.RTMError:
				logger.Error("RTMError", ev)
			case *slack.InvalidAuthEvent:
				logger.Fatal("Invalid Auth", errors.New("invalid auth"), lager.Data{
					"event": ev,
				})
				return
			default:
			}
		}
	}
}

