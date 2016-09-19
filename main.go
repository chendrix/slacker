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

	slacker := &Slacker{
		ChannelConfigs: cfg.Channels,
		SlackClient:    api,
		SlackRTM:       rtm,
		TrackerClientFactory: func(apiToken string, projectID int) tracker.ProjectClient {
			return tracker.NewClient(apiToken).InProject(projectID)
		},
		Logger: logger,
	}

	for {
		select {
		case msg := <-rtm.IncomingEvents:
			switch ev := msg.Data.(type) {
			case *slack.MessageEvent:
				slacker.HandleMessage(ev)
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

type Slacker struct {
	ChannelConfigs       []*config.ChannelConfig
	SlackClient          *slack.Client
	SlackRTM             *slack.RTM
	TrackerClientFactory func(apiToken string, projectID int) tracker.ProjectClient
	Logger               lager.Logger
}

func (s *Slacker) HandleMessage(ev *slack.MessageEvent) {
	for _, channel := range s.ChannelConfigs {
		s.Logger.Debug("Message Received", lager.Data{
			"event": ev,
		})

		if ev.Msg.Channel == channel.SlackChannelID &&
			strings.Contains(ev.Msg.Text, channel.SlackTrigger) {
			s.Logger.Debug("Message Response Triggered", lager.Data{
				"message": ev.Msg.Text,
			})

			u, err := s.SlackClient.GetUserInfo(ev.Msg.User)
			if err != nil {
				s.Logger.Error("error getting user info", err, lager.Data{"userID": ev.Msg.User})
				continue
			}

			timestamp, err := strconv.ParseFloat(ev.Msg.Timestamp, 64)
			if err != nil {
				s.Logger.Error("error converting timestamp", err, lager.Data{"timestamp": ev.Msg.Timestamp})
				continue
			}

			st := tracker.Story{
				Name: fmt.Sprintf("**Interrupt** from %s on %v", u.RealName, time.Unix(int64(timestamp), 0).String()),
				Labels: []tracker.Label{
					{Name: "interrupt"},
				},
				Type:        tracker.StoryTypeBug,
				Description: fmt.Sprintf(`From %v (%v) on %v:

				> %v`, u.RealName, u.Name, channel.SlackChannelName, ev.Msg.Text),
			}

			s.Logger.Debug("creating story", lager.Data{
				"story": st,
			})

			trackerClient := s.TrackerClientFactory(channel.TrackerAPIToken, channel.TrackerProjectID)

			story, err := trackerClient.CreateStory(st)
			if err != nil {
				s.Logger.Error("error creating story", err, lager.Data{"story": st})
				continue
			}

			slackResponse := fmt.Sprintf("Hi @%s , we've made a story %v for your interrupt. You should be contacted by the pair who picks up your story. Have a great day!", u.Name, story.URL)
			s.SlackRTM.SendMessage(s.SlackRTM.NewOutgoingMessage(slackResponse, channel.SlackChannelID))
		}
	}
}
