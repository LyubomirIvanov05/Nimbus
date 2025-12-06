package client

import(
	"net"
	"time"
	"github.com/LyubomirIvanov05/nimbus/internals/message"
)
type Client struct {
	Conn net.Conn
	Backlog []*message.Message
	LastHeartbeat time.Time
	LastSeen map[string]int
}

func NewClient(conn net.Conn, backlog []*message.Message, lastHeartbeat time.Time, lastSeen map[string]int) *Client {
    return &Client{
        Conn: conn,
		Backlog: backlog,
		LastHeartbeat: lastHeartbeat,
		LastSeen: lastSeen,
    }
}