package config_test

import (
	. "github.com/chendrix/slacker/config"

	"fmt"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Config", func() {
	Describe("Validate", func() {

		var (
			config     *Config
			configFile string
		)

		JustBeforeEach(func() {
			osArgs := []string{
				"slacker",
				fmt.Sprintf("-configPath=%s", configFile),
			}

			var err error
			config, err = NewConfig(osArgs)
			Expect(err).ToNot(HaveOccurred())
		})

		BeforeEach(func() {
			configFile = "fixtures/sampleConfig.yml"
		})

		It("does not return error on valid config", func() {
			Expect(config.Validate()).To(Succeed())

			Expect(config.SlackAPIToken).To(Equal("REPLACE_WITH_SLACK_API_TOKEN"))

			sampleChannel := config.Channels[0]
			Expect(sampleChannel.SlackChannelName).To(Equal("#default"))
			Expect(sampleChannel.TrackerProject).To(Equal("REPLACE_WITH_TRACKER_PROJECT_NAME"))
			Expect(sampleChannel.TrackerAPIToken).To(Equal("REPLACE_WITH_TRACKER_API_TOKEN"))
			Expect(sampleChannel.SlackTrigger).To(Equal("@interrupt"))
		})
	})
})
