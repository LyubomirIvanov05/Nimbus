package channel

import (
	"fmt"
	"net"
	"sync"
    "github.com/LyubomirIvanov05/nimbus/internals/message"
)

type Channel struct {
    Name        string
    Subscribers map[net.Conn]bool
    mu sync.RWMutex
    Messages []*message.Message
}

func NewChannel(name string) *Channel {
    return &Channel{
        Name:        name,
        Subscribers: make(map[net.Conn]bool),
    }
}

func (c *Channel) AddSubscriber(conn net.Conn) {
    c.mu.Lock()
    defer c.mu.Unlock()
    c.Subscribers[conn] = true
    fmt.Println("[CHANNEL/ADD_SUBSCRIBER] Added subscriber to channel", c.Name)
    fmt.Println("[CHANNEL/ADD_SUBSCRIBER] Subscribers:", c.Subscribers)
}

func (c *Channel) RemoveSubscriber(conn net.Conn) {
	c.mu.Lock()
	defer c.mu.Unlock()
	_, ok := c.Subscribers[conn]
	if !ok {
		return
	}
	delete(c.Subscribers, conn)
}

func (c *Channel) ListSubscribers() []net.Conn {
    c.mu.RLock()
    defer c.mu.RUnlock()
    subscribers := make([]net.Conn, 0, len(c.Subscribers))
    for sub := range c.Subscribers {
        subscribers = append(subscribers, sub)
    }
    return subscribers
}

func (c *Channel) Broadcast(message string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	for sub := range c.Subscribers {
		fmt.Fprintln(sub, "MSG", c.Name, message)
	}
}

func (c *Channel) AddMessageToChannel(content string) *message.Message{
    c.mu.Lock()
    defer c.mu.Unlock()
    msg := message.NewMessage(content, c.Name)
    msg.ID = len(c.Messages) + 1
    c.Messages = append(c.Messages, msg)
    return msg
}

func (c *Channel) GetMessages() []*message.Message{
    c.mu.RLock()
    defer c.mu.RUnlock()
    
    return c.Messages
}