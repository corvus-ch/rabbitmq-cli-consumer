package consumer

import (
	"errors"
	"fmt"

	"github.com/bketelsen/logr"
)

// ChannelMultiplexer describes an object that holds multiple Channels
type ChannelMultiplexer interface {
	AddChannel()
	Channels() []Channel
	FirstChannel() Channel
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
	var ch Channel
	chs := cl.Channels()
	if len(chs) < 1 {
		return ch, errors.New("tried to get the first channel from an uninitialized ChannelList")
	}

	ch = cl.channels[0]
	return ch, nil
}
