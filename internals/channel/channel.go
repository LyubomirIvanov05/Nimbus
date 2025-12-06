package channel

import (
	"errors"
	"fmt"
	"net"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/LyubomirIvanov05/nimbus/internals/client"
	"github.com/LyubomirIvanov05/nimbus/internals/message"
)

type Channel struct {
	Name        string
	Subscribers map[*client.Client]bool
	mu          sync.RWMutex
	Messages    []*message.Message
}

func NewChannel(name string) *Channel {
	return &Channel{
		Name:        name,
		Subscribers: make(map[*client.Client]bool),
	}
}

func (c *Channel) AddSubscriber(cl *client.Client) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.Subscribers[cl] = true
	fmt.Println("[CHANNEL/ADD_SUBSCRIBER] Added subscriber to channel", c.Name)
	fmt.Println("[CHANNEL/ADD_SUBSCRIBER] Subscribers:", c.Subscribers)
}

func (c *Channel) RemoveSubscriber(conn net.Conn) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	// Find the client by its connection and remove it
	var target *client.Client
	for cl := range c.Subscribers {
		if cl.Conn == conn {
			target = cl
			break
		}
	}
	if target == nil {
		fmt.Println("Subscriber doesn't exist")
		return false
	}
	delete(c.Subscribers, target)
	//NOTE: We are printing the same message for unsubscribing and an user disconnecting
	//These two operations have to have different logging
	fmt.Println("[CHANNEL/REMOVE_SUBSCRIBER] Removed subscriber from channel", c.Name)
	fmt.Println("[CHANNEL/REMOVE_SUBSCRIBER] Subscribers:", c.Subscribers)
	return true
}

func (c *Channel) ListSubscribers() []net.Conn {
	c.mu.RLock()
	defer c.mu.RUnlock()
	subscribers := make([]net.Conn, 0, len(c.Subscribers))
	for sub := range c.Subscribers {
		subscribers = append(subscribers, sub.Conn)
	}
	return subscribers
}

func (c *Channel) Broadcast(channelName string, message *message.Message, backlogLimit int) []*client.Client {
	c.mu.Lock()
	defer c.mu.Unlock()

	offenders := []*client.Client{}
	for sub := range c.Subscribers {
		_, err := fmt.Fprintln(sub.Conn, "MSG", c.Name, message)
		if err != nil {
			sub.Backlog = append(sub.Backlog, message)
			if len(sub.Backlog) > backlogLimit {
				offenders = append(offenders, sub)
			}
		} else {
			if sub.LastSeen != nil {
				sub.LastSeen[c.Name] = message.ID
			}
		}
	}

	filePath := "logs/" + channelName + ".log"
	isFileExist := CheckFileExists(filePath)

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
	return offenders
}

func CheckFileExists(filePath string) bool {
	_, error := os.Stat(filePath)
	//return !os.IsNotExist(err)
	return !errors.Is(error, os.ErrNotExist)
}

func (c *Channel) AddMessageToChannel(content string) *message.Message {
	c.mu.Lock()
	defer c.mu.Unlock()
	msg := message.NewMessage(content, c.Name)
	msg.ID = len(c.Messages) + 1
	c.Messages = append(c.Messages, msg)
	return msg
}

func (c *Channel) GetMessages() []*message.Message {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return c.Messages
}

func (c *Channel) AddMessageStructToChannel(msg *message.Message) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.Messages = append(c.Messages, msg)
}

func (c *Channel) RemoveFile(channelName string) (bool, error) {
	filePath := "logs/" + channelName + ".log"
	e := os.Remove(filePath)
	if e != nil {
		return false, errors.New("couldn't delete file")
	}
	return true, nil
}

func (c *Channel) RewriteFile(channelName string, slicedMessages []*message.Message) (bool, error) {
	filePath := "logs/" + channelName + ".log"
	var b strings.Builder
	for _, m := range slicedMessages {
		b.WriteString(fmt.Sprintf("%d|%s|%s|%s\n",
			m.ID,
			m.Timestamp.UTC().Format(time.RFC3339Nano),
			channelName,
			m.Content,
		))
	}

	if err := os.WriteFile(filePath, []byte(b.String()), 0644); err != nil {
		return false, err
	}
	return true, nil
}
