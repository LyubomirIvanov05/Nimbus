package channel

import (
    "os"
    "time"
    "errors"
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

func (c *Channel) RemoveSubscriber(conn net.Conn) bool{
	c.mu.Lock()
	defer c.mu.Unlock()
	_, ok := c.Subscribers[conn]
	if !ok {
        fmt.Println("Subscriber doesn't exist")
		return false
	}
	delete(c.Subscribers, conn)
    fmt.Println("[CHANNEL/REMOVE_SUBSCRIBER] Removed subscriber from channel", c.Name)
    fmt.Println("[CHANNEL/REMOVE_SUBSCRIBER] Subscribers:", c.Subscribers)
    return true
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

func (c *Channel) Broadcast(channelName string, message *message.Message) {
	c.mu.Lock()
	defer c.mu.Unlock()

	for sub := range c.Subscribers {
		fmt.Fprintln(sub, "MSG", c.Name, message)
	}

    filePath := "logs/" + channelName + ".log"
	isFileExist := checkFileExists(filePath)

    if isFileExist {
		fmt.Println("file exist, about to add to the file")
        file, err := os.OpenFile(filePath, os.O_APPEND|os.O_WRONLY, 0644)
        if err != nil {
            fmt.Println("There was an error opening the file")
        }
        defer file.Close()

        line := fmt.Sprintf("%d|%s|%s|%s\n", message.ID, message.Timestamp.Format(time.RFC3339Nano), message.ChannelName, message.Content)
        _, err = file.WriteString(line)
        if err != nil {
            fmt.Println("There was an error adding to the file")
        }
	} else {

		fmt.Println("file not exists, about to create a new file and add to it")
        err := os.WriteFile(filePath, []byte(fmt.Sprintf("%d|%s|%s|%s\n", message.ID, message.Timestamp.Format(time.RFC3339Nano), message.ChannelName, message.Content)), 0644)
        if err != nil {
            fmt.Println("There was an error creating the file")
        }

	}
}

func checkFileExists(filePath string) bool {
	_, error := os.Stat(filePath)
	//return !os.IsNotExist(err)
	return !errors.Is(error, os.ErrNotExist)
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

func (c *Channel) AddMessageStructToChannel(msg *message.Message){
    c.mu.Lock()
    defer c.mu.Unlock()
    c.Messages = append(c.Messages, msg)

}