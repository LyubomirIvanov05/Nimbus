package message

import (
	"time"
)

type Message struct {
    ID      int
    ChannelName string
    Content string
    Timestamp time.Time
}

func NewMessage(content string, channelName string) *Message {
    return &Message{
        ID: 1,
        ChannelName: channelName,
        Content: content,
        Timestamp: time.Now(),
    }
}