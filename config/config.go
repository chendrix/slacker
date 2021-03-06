package config

import (
	"flag"

	"gopkg.in/validator.v2"

	"errors"
	"fmt"

	"code.cloudfoundry.org/cflager"
	"code.cloudfoundry.org/lager"
	"github.com/nlopes/slack"
	"github.com/pivotal-cf-experimental/service-config"
)

type Config struct {
	SlackAPIToken string           `yaml:"SlackAPIToken" validate:"nonzero"`
	Channels      []*ChannelConfig `yaml:"Channels" validate:"nonzero,min=1"`
	Logger        lager.Logger
}

type ChannelConfig struct {
	SlackChannelName string `yaml:"SlackChannelName" validate:"nonzero"`
	SlackChannelID   string
	SlackTrigger     string `yaml:"SlackTrigger" validate:"nonzero"`
	TrackerAPIToken  string `yaml:"TrackerAPIToken" validate:"nonzero"`
	TrackerProjectID int    `yaml:"TrackerProjectID" validate:"nonzero"`
}

func NewConfig(osArgs []string) (*Config, error) {
	var rootConfig Config

	binaryName := osArgs[0]
	configurationOptions := osArgs[1:]

	serviceConfig := service_config.New()
	flags := flag.NewFlagSet(binaryName, flag.ExitOnError)

	cflager.AddFlags(flags)

	serviceConfig.AddFlags(flags)
	flags.Parse(configurationOptions)

	err := serviceConfig.Read(&rootConfig)

	rootConfig.Logger, _ = cflager.New(binaryName)

	return &rootConfig, err
}

func (c *Config) Validate() error {
	rootConfigErr := validator.Validate(c)
	var errString string
	if rootConfigErr != nil {
		errString = formatErrorString(rootConfigErr, "")
	}

	// validator.Validate does not work on nested arrays
	for i, channel := range c.Channels {
		e := validator.Validate(channel)
		if e != nil {
			errString += formatErrorString(
				e,
				fmt.Sprintf("Proxy.Backends[%d].", i),
			)
		}
	}

	if len(errString) > 0 {
		return errors.New(fmt.Sprintf("Validation errors: %s\n", errString))
	}
	return nil
}

func formatErrorString(err error, keyPrefix string) string {
	errs := err.(validator.ErrorMap)
	var errsString string
	for fieldName, validationMessage := range errs {
		errsString += fmt.Sprintf("%s%s : %s\n", keyPrefix, fieldName, validationMessage)
	}
	return errsString
}

func (c *Config) HydrateFromSlack(cs []slack.Channel) (err error) {
	m := make(map[string]slack.Channel)

	for _, c := range cs {
		m[c.Name] = c
	}

	for _, channel := range c.Channels {
		c, ok := m[channel.SlackChannelName]
		if !ok {
			return fmt.Errorf(`could not find information about channel "%v"`, channel.SlackChannelName)
		}

		channel.SlackChannelID = c.ID
	}

	return nil
}
