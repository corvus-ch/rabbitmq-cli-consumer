package consumer_test

import (
	"fmt"
	"testing"

	"github.com/corvus-ch/rabbitmq-cli-consumer/config"
	"github.com/corvus-ch/rabbitmq-cli-consumer/consumer"
	"github.com/stretchr/testify/assert"
)

const qosConfig = "qos"

var channelListTests = []struct {
	name   string
	config string
	setup  func(*consumer.ChannelList)
	err    error
}{
	// Set QoS.
	{
		"setQos",
		qosConfig,
		func(cl *consumer.ChannelList) {
			for _, ch := range cl.Channels() {
				tc := ch.(*TestChannel)
				tc.On("Qos", 42, 0, true).Return(nil).Once()
			}
		},
		nil,
	},
	// Set QoS fails.
	{
		"setQosFail",
		qosConfig,
		func(cl *consumer.ChannelList) {
			for _, ch := range cl.Channels() {
				tc := ch.(*TestChannel)
				tc.On("Qos", 42, 0, true).Return(fmt.Errorf("QoS error")).Once()
			}
		},
		fmt.Errorf("failed to set QoS on channel 0: QoS error"),
	},
}

func TestChannelList(t *testing.T) {
	for _, test := range channelListTests {
		t.Run(test.name, func(t *testing.T) {
			cfg, _ := config.LoadAndParse(fmt.Sprintf("fixtures/%s.conf", test.config))
			cl := &consumer.ChannelList{}
			cl.AddChannel(new(TestChannel))
			test.setup(cl)
			assert.Equal(t, test.err, cl.Qos(cfg.PrefetchCount(), 0, cfg.PrefetchIsGlobal()))
			for _, ch := range cl.Channels() {
				tc := ch.(*TestChannel)
				tc.AssertExpectations(t)
			}
		})
	}
}
