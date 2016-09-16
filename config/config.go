package config

import (
	"flag"

	"code.cloudfoundry.org/cflager"
	"code.cloudfoundry.org/lager"
	"github.com/pivotal-cf-experimental/service-config"
)

type Config struct {
	SlackAPIToken string `yaml:"SlackAPIToken"`
	Channels      []ChannelConfig `yaml:"Channels"`
	Logger        lager.Logger
}

type ChannelConfig struct {
	SlackChannelName string `yaml:"SlackChannelName"`
	SlackTrigger     string `yaml:"SlackTrigger"`
	TrackerAPIToken  string `yaml:"TrackerAPIToken"`
	TrackerProject   string `yaml:"TrackerProject"`
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
