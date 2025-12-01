package broker

import (
	"net"
	"sync"
	"fmt"
	"github.com/LyubomirIvanov05/nimbus/internals/channel"
)

type Broker struct {
	mu sync.RWMutex
    channels map[string]*channel.Channel
}

func NewBroker() *Broker {
    return &Broker{
        channels: make(map[string]*channel.Channel),
    }
}

// Placeholder method
func (b *Broker) Start() {
	fmt.Println("This is supposed to start the TCP server later")
	b.StartServer()
	fmt.Println("Server started")

    // This will start the TCP server later
}

func (b *Broker) getOrCreateChannel(name string) *channel.Channel {
	fmt.Println("[BROKER/GET_OR_CREATE_CHANNEL] Getting or creating channel:", name)
	b.mu.Lock()
	defer b.mu.Unlock()

	ch, ok := b.channels[name]
	if !ok {
		ch = channel.NewChannel(name)
		b.channels[name] = ch
	}

	return ch
}

func (b *Broker) removeConnection(conn net.Conn) {
	b.mu.Lock()
	defer b.mu.Unlock()

	for _, ch := range b.channels {
		ch.RemoveSubscriber(conn)
	}
}

