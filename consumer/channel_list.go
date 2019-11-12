package consumer

import (
	"errors"
	"fmt"

	"github.com/bketelsen/logr"
)

// ChannelMultiplexer describes an object that holds multiple Channels
type ChannelMultiplexer interface {
	AddChannel(Channel)
	Channels() []Channel
	FirstChannel() (Channel, error)
	Qos(prefetchCount, prefetchSize int, global bool) error
}

// ChannelList is an implementation of the ChannelMultiplexer interface.
type ChannelList struct {
	channels []Channel
}

// NewChannelList creates a new ChannelList with a single Channel
func NewChannelList(conn Connection, len int, l logr.Logger) (*ChannelList, error) {
	if len < 1 {
		return nil, fmt.Errorf("cannot create a ChannelList with less than one channel (got %d)", len)
	}

	chs := make([]Channel, 0)
	l.Infof("Opening %d channel(s)...", len)
	for i := 0; i < len; i++ {
		ch, err := conn.Channel()
		if nil != err {
			return nil, fmt.Errorf("failed to open a channel: %v", err)
		}
		l.Infof("Opened channel %d...", i)
		chs = append(chs, ch)
	}
	l.Info("Done.")

	return &ChannelList{channels: chs}, nil
}

// AddChannel adds the given channel to the end of the ChannelList
func (cl *ChannelList) AddChannel(c Channel) {
	cl.channels = append(cl.channels, c)
}

// Channels returns a slice of Channels represented by this ChannelList.
func (cl *ChannelList) Channels() []Channel {
	return cl.channels
}

// FirstChannel is a convenience function to get the first Channel in this ChannelList.
func (cl *ChannelList) FirstChannel() (Channel, error) {
	if len(cl.channels) < 1 {
		var ch Channel
		return ch, errors.New("tried to get the first channel from an uninitialized ChannelList")
	}

	return cl.channels[0], nil
}

// Qos sets Qos settings for all Channels in this ChannelList.
func (cl *ChannelList) Qos(prefetchCount, prefetchSize int, global bool) error {
	for i, ch := range cl.channels {
		if err := ch.Qos(prefetchCount, prefetchSize, global); err != nil {
			return fmt.Errorf("failed to set QoS on channel %d: %v", i, err)
		}
	}
	return nil
}
