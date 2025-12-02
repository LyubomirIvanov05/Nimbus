package broker

import (
	"bufio"
	"fmt"
	"net"
	"strings"
	"time"
	"bytes"
	"os"
	"github.com/LyubomirIvanov05/nimbus/internals/message"
	"strconv"
)

func (b *Broker) StartServer() {
	listener, err := net.Listen("tcp", ":7070")
	if err != nil {
		fmt.Println("Error starting server:", err)
		return
	}

	fmt.Println("Server started on port 7070")
	b.loadAllLogs()
	for {
		conn, err := listener.Accept()
		if err != nil {
			fmt.Println("Error accepting connection:", err)
			continue
		}
		fmt.Println("Connection accepted:", conn.RemoteAddr())

		go b.handleConnection(conn)
	}
}

func (b *Broker) handleConnection(conn net.Conn) {
	defer func() {
		fmt.Println("Client disconnected:", conn.RemoteAddr())
		b.removeConnection(conn)
		conn.Close()
	}()
	fmt.Println("This is supposed to handle the connection")

	scanner := bufio.NewScanner(conn)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.TrimSpace(line) == "" {
			continue
		}
		b.handleCommand(conn, line)
	}

	if err := scanner.Err(); err != nil {
		fmt.Println("Error reading from connection:", err)
	}
}

func (b *Broker) handleCommand(conn net.Conn, line string) {
	trimmed := strings.TrimSpace(line)
	if trimmed == "" {
		return
	}

	fields := strings.Fields(trimmed)
	if len(fields) == 0 {
		fmt.Println("Invalid command:", line)
		return
	}

	command := strings.ToUpper(fields[0])

	switch command {
	case "SUBSCRIBE":
		if len(fields) < 2 {
			fmt.Fprintf(conn, "ERR SUBSCRIBE needs channel\n")
			return
		}
		channelName := fields[1]
		b.handleSubscribe(conn, channelName)
	case "PUBLISH":
		if len(fields) < 3 {
			fmt.Fprintf(conn, "ERR PUBLISH needs channel and message\n")
			return
		}
		channelName := fields[1]
		message := strings.Join(fields[2:], " ")
		b.handlePublish(channelName, message)
	case "LIST":
		fmt.Println("[BROKER/HANDLE_COMMAND] Listing channels")
		b.handleList(conn)
	case "FETCH":
		if len(fields) < 2 {
			fmt.Fprintf(conn, "ERR FETCH needs only channel")
			return
		}
		channelName := fields[1]
		b.handleFetch(conn, channelName)
	case "UNSUBSCRIBE":
		if len(fields) < 2 {
			fmt.Fprintf(conn, "ERR UNSUBSCRIBE needs channel")
		}
		channelName := fields[1]
		b.handleUnsubscribe(conn, channelName)
	case "PING":
		b.handlePing(conn)
	default:
		fmt.Fprintf(conn, "ERR Invalid command\n")
	}
}
func (b *Broker) handleList(conn net.Conn) {
	fmt.Println("[BROKER/HANDLE_LIST] Listing channels")
	b.mu.RLock()
	for channelName := range b.channels {
		fmt.Println("[BROKER/HANDLE_LIST] Channel is", channelName)
		fmt.Fprintf(conn, "CH %s\n", channelName)
	}
	fmt.Fprintf(conn, "OK LISTED %d channels\n", len(b.channels))
	b.mu.RUnlock()
	fmt.Println("[BROKER/HANDLE_LIST] Listed", len(b.channels), "channels")
}

func (b *Broker) handleSubscribe(conn net.Conn, channelName string) {
	ch := b.getOrCreateChannel(channelName)
	ch.AddSubscriber(conn)
	fmt.Fprintf(conn, "OK SUBSCRIBED to %s\n", channelName)
}

func (b *Broker) handleUnsubscribe(conn net.Conn, channelName string){
	ch := b.getOrCreateChannel(channelName)
	res := ch.RemoveSubscriber(conn)
	if res {
		fmt.Fprintf(conn, "OK UNSUBSCRIBED TO %s\n", channelName)
	} else {
		fmt.Fprintf(conn, "ERR not subscribed to %s\n", channelName)
	}
}

func (b *Broker) handlePublish(channelName, message string) {
	ch := b.getOrCreateChannel(channelName)
	msg := ch.AddMessageToChannel(message)
	ch.Broadcast(channelName, msg)
}

func (b *Broker) handleFetch(conn net.Conn, channelName string){
	ch := b.getOrCreateChannel(channelName)
	messages := ch.GetMessages()

	for _, m := range messages {
		fmt.Fprintf(conn, "MSG #%d %s %s %s\n",
		m.ID, m.ChannelName, m.Timestamp.Format(time.RFC3339Nano), m.Content)
	}
	fmt.Fprintf(conn, "OK FETCHED %d messages\n", len(messages))
}

func (b *Broker) loadAllLogs(){
	files, err := os.ReadDir("logs")
	if err != nil {
		fmt.Println("ERROR while reading /logs folder")
	}

	for _, file := range files {
		fmt.Println(file.Name())
		//NOTE: Format is 1|2025-02-01T12:05:05Z|chat|hello
		currFile := "logs/" + file.Name()
		fmt.Println("currFile:", currFile)
		currLogs, err := os.ReadFile(currFile)
		if err != nil {
			fmt.Println("ERROR while reading logs from file")
		}
		scanner := bufio.NewScanner(bytes.NewReader(currLogs))
		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			if line == ""{
				continue
			}
			parts := strings.SplitN(line, "|", 4)
			currId, err := strconv.Atoi(parts[0])
			if err != nil {
				fmt.Println("Couldn't convert ID to int", err)
        		return
			}
			currTimestamp, err := time.Parse(time.RFC3339Nano, parts[1])

			if err != nil {
				fmt.Println("Couldn't convert Timestamp to time", err)
        		return
			}
			currChanneleName := parts[2]
			currContent := parts[3]


			msg := &message.Message{
				ID: currId,
				ChannelName: currChanneleName,
				Content: currContent,
				Timestamp: currTimestamp,
			}
			ch := b.getOrCreateChannel(currChanneleName)
			ch.AddMessageStructToChannel(msg)

		}

	}
}

func (b *Broker) handlePing(conn net.Conn){
	fmt.Fprintf(conn, "PONG\n")
}