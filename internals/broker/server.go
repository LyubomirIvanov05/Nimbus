package broker

import (
	"bufio"
	"bytes"
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"
	"time"
	"github.com/LyubomirIvanov05/nimbus/internals/channel"
	"github.com/LyubomirIvanov05/nimbus/internals/client"
	"github.com/LyubomirIvanov05/nimbus/internals/message"
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
	c := client.NewClient(conn, []*message.Message{}, time.Now(), make(map[string]int))
	b.mu.Lock()
	b.clients[conn] = c
	b.heartbeats[conn] = time.Now()
	b.mu.Unlock()
	go b.monitorHeartbeat(conn)

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

	b.mu.Lock()
    b.heartbeats[conn] = time.Now()
    b.mu.Unlock()
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
		client := b.clients[conn]
		b.handleSubscribe(client, channelName)
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
			fmt.Fprintf(conn, "ERR FETCH needs only channel\n")
			return
		}
		channelName := fields[1]
		b.handleFetch(conn, channelName)
	case "UNSUBSCRIBE":
		if len(fields) < 2 {
			fmt.Fprintf(conn, "ERR UNSUBSCRIBE needs channel\n")
			return
		}
		channelName := fields[1]
		b.handleUnsubscribe(conn, channelName)
	case "PING":
		b.handlePing(conn)
	case "WHOAMI":
		b.handleIdentity(conn)
	case "INFO":
		if len(fields) < 2 {
			fmt.Fprintf(conn, "ERR INFO needs channel\n")
			return
		}
		channelName := fields[1]
		b.handleInfo(conn, channelName)
	case "DELETE":
		if len(fields) < 2 {
			fmt.Fprintf(conn, "ERR DELETE needs channel\n")
			return
		}
		channelName := fields[1]
		b.deleteChannel(conn, channelName)
	case "GET":
		if len(fields) < 3 {
			fmt.Fprintf(conn, "ERR GET needs channel and id\n")
			return
		}
		channelName := fields[1]
		stringId := fields[2]
		id, err := strconv.Atoi(stringId)
		if err != nil {
			fmt.Fprintf(conn, "ERR GET invalid last id\n")
			return
		}
		b.handleGet(conn, channelName, id)
	case "RETENTION":
		if len(fields) < 3 {
			fmt.Fprintf(conn, "ERR RETENTION needs channel and number of messages\n")
			return
		}
		channelName := fields[1]
		msgsCount, err := strconv.Atoi(fields[2])
		if err != nil {
			fmt.Fprintf(conn, "ERR RETENTION invalid messages count\n")
			return
		}
		if msgsCount <= 0 {
			fmt.Fprintf(conn, "ERR RETENTION messages count has to be positive number\n")
			return
		}
		b.handleRetention(conn, channelName, msgsCount)

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

func (b *Broker) handleSubscribe(client *client.Client, channelName string) {
	ch := b.getOrCreateChannel(channelName)
	ch.AddSubscriber(client)
	messages := ch.GetMessages()
	last := client.LastSeen[channelName]
	maxId := last
	for _, m := range messages {
		if m.ID > last {
			ts := m.Timestamp.UTC().Format("2006-01-02 15:04:05.000000")
			fmt.Fprintf(client.Conn, "MSG #%d %s %s %s\n", m.ID, m.ChannelName, ts, m.Content)
			if m.ID > maxId {
				maxId = m.ID
			}
		}
	}
	client.LastSeen[channelName] = maxId
	fmt.Fprintf(client.Conn, "OK SUBSCRIBED to %s\n", channelName)
}

func (b *Broker) handleUnsubscribe(conn net.Conn, channelName string) {
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

	offenders := ch.Broadcast(channelName, msg, b.maxBacklog)

	if len(offenders) > 0 {
		for _, cl := range offenders {
			fmt.Println("[BACKLOG] disconnecting overloaded cliend:", cl.Conn.RemoteAddr())
			cl.Conn.Close()
			b.removeConnection(cl.Conn)
			b.mu.Lock()
			delete(b.heartbeats, cl.Conn)
			delete(b.clients, cl.Conn)
			b.mu.Unlock()
		}
	}
}

func (b *Broker) handleFetch(conn net.Conn, channelName string) {
	ch := b.getOrCreateChannel(channelName)
	messages := ch.GetMessages()

	for _, m := range messages {
		fmt.Fprintf(conn, "MSG #%d %s %s %s\n",
			m.ID, m.ChannelName, m.Timestamp.Format(time.RFC3339Nano), m.Content)
	}
	fmt.Fprintf(conn, "OK FETCHED %d messages\n", len(messages))
}

func (b *Broker) loadAllLogs() {
	files, err := os.ReadDir("logs")
	if err != nil {
		fmt.Println("ERROR while reading /logs folder")
	}

	for _, file := range files {
		//NOTE: Commented out because not needed on daily basis, only for debugging
		// fmt.Println(file.Name())
		//NOTE: Format is 1|2025-02-01T12:05:05Z|chat|hello
		currFile := "logs/" + file.Name()
		// fmt.Println("currFile:", currFile)
		currLogs, err := os.ReadFile(currFile)
		if err != nil {
			fmt.Println("ERROR while reading logs from file")
		}
		scanner := bufio.NewScanner(bytes.NewReader(currLogs))
		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			if line == "" {
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
				ID:          currId,
				ChannelName: currChanneleName,
				Content:     currContent,
				Timestamp:   currTimestamp,
			}
			ch := b.getOrCreateChannel(currChanneleName)
			ch.AddMessageStructToChannel(msg)

		}

	}
}

func (b *Broker) handlePing(conn net.Conn) {
	b.mu.Lock()
	b.heartbeats[conn] = time.Now()
	b.mu.Unlock()
	fmt.Fprintf(conn, "PONG\n")
}

func (b *Broker) handleIdentity(conn net.Conn) {
	fmt.Fprintf(conn, "YOU ARE %s\n", conn.RemoteAddr().String())
}

func (b *Broker) handleInfo(conn net.Conn, channelName string) {
	ch := b.getOrCreateChannel(channelName)
	messages := ch.GetMessages()
	subscribers := ch.ListSubscribers()
	fmt.Fprintf(conn, "SUBSCRIBERS %d\n", len(subscribers))
	fmt.Fprintf(conn, "MESSAGES %d\n", len(messages))
}

func (b *Broker) deleteChannel(conn net.Conn, channelName string) {
	ok, err := b.removeChannel(channelName)
	if err != nil {
		fmt.Fprintf(conn, "ERR %s\n", err.Error())
		return
	}
	if ok {
		fileDir := "logs/" + channelName + ".log"
		isFileExist := channel.CheckFileExists(fileDir)
		if isFileExist {
			e := os.Remove(fileDir)
			if e != nil {
				fmt.Println("ERROR couldn't delete file")
			}
		}
		fmt.Fprintf(conn, "OK DELETED %s\n", channelName)
	}
}

func (b *Broker) handleGet(conn net.Conn, channelName string, id int) {
	ok := b.checkChannelExist(channelName)
	if !ok {
		fmt.Fprintf(conn, "ERR channel doesn't exist\n")
		return
	}
	ch := b.getOrCreateChannel(channelName)
	res := []string{}
	for _, message := range ch.Messages {
		if message.ID > id {
			ts := message.Timestamp.UTC().Format("2006-01-02 15:04:05.000000")
			formatted := fmt.Sprintf("MSG #%d %s %s %s", message.ID, message.ChannelName, ts, message.Content)
			res = append(res, formatted)
		}
	}
	finalMsg := fmt.Sprintf("OK GOT %d messages\n", len(res))
	if len(res) > 0 {
		fmt.Fprint(conn, strings.Join(res, "\n"))
		fmt.Fprint(conn, "\n")
		fmt.Fprint(conn, finalMsg)
	} else {
		fmt.Fprint(conn, finalMsg)
	}
}

func (b *Broker) handleRetention(conn net.Conn, channelName string, msgCount int) {
	ok := b.checkChannelExist(channelName)
	if !ok {
		fmt.Fprintf(conn, "ERR channel doesn't exist\n")
		return
	}
	ch := b.getOrCreateChannel(channelName)
	// NOTE: not rlly sure abt this
	// if msgCount == 0 {
	// 	ch.RemoveFile(channelName)
	// }
	start := len(ch.Messages) - msgCount
	if start < 0 {
		start = 0
	}
	slicedMessages := ch.Messages[start:]
	ok, err := ch.RewriteFile(channelName, slicedMessages)
	if ok {
		ch.Messages = slicedMessages
		fmt.Fprintf(conn, "OK RETENTION %s %d", channelName, len(slicedMessages))
	} else {
		fmt.Fprintf(conn, "ERR RETENTION server error")
		fmt.Println(err)
	}
}

func (b *Broker) monitorHeartbeat(conn net.Conn) {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	// timeout := 15 * time.Second // tweak later
	//THIS IS ONLY FOR TESTING
	timeout := 5 * time.Minute


	for range ticker.C {
		b.mu.Lock()
		last, ok := b.heartbeats[conn]
		b.mu.Unlock()

		if !ok {
			return // connection removed
		}

		if time.Since(last) > timeout {
			fmt.Println("[HEARTBEAT] Client timed out:", conn.RemoteAddr())
			conn.Close()
			b.removeConnection(conn)
			b.mu.Lock()
			delete(b.heartbeats, conn)
			b.mu.Unlock()
			return
		}
	}
}
