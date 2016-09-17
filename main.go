package main

import (
	"errors"
	"log"
	"os"

	"strings"

	"code.cloudfoundry.org/lager"
	"fmt"
	"github.com/chendrix/slacker/config"
	"github.com/nlopes/slack"
	"github.com/xoebus/go-tracker"
	"strconv"
	"time"
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

						u, err := api.GetUserInfo(ev.Msg.User)
						if err != nil {
							logger.Error("error getting user info", err, lager.Data{"userID": ev.Msg.User})
							continue
						}

						timestamp, err := strconv.ParseFloat(ev.Msg.Timestamp, 64)
						if err != nil {
							logger.Error("error converting timestamp", err, lager.Data{"timestamp": ev.Msg.Timestamp})
							continue
						}

						s := tracker.Story{
							Name: fmt.Sprintf("**Interrupt** from %v on %v", u.RealName, time.Unix(int64(timestamp), 0).String()),
							Labels: []tracker.Label{
								tracker.Label{Name: "interrupt"},
							},
							Type:        tracker.StoryTypeBug,
							State:       tracker.StoryStatePlanned,
							Description: fmt.Sprintf(`From %v (%v) on %v:\n\n\n>%v`, u.RealName, u.Name, channel.SlackChannelName, ev.Msg.Text),
						}

						logger.Debug("creating story", lager.Data{
							"story": s,
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
