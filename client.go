package vr

import (
	"encoding/gob"
	"log"
	"net"
	"sync"
	"time"
)

// ClientConfig is given to the client at the very beginning.
// this must not be modified.
type ClientConfig struct {
	LeaderAddr string
	ID         ID
	View       ID
	Timeout    time.Duration
}

// Client holds the client-side information
type Client struct {
	mu sync.RWMutex

	Config  ClientConfig
	ID      ID
	Request ID // Monotonical increasing reqeust number
	View    ID

	resultc chan Result
	Results map[ID]Result

	conn *net.TCPConn // TCP connection to the current leader
	enc  *gob.Encoder
	dec  *gob.Decoder
}

// NewClient creates new client instance with given information
func NewClient(config ClientConfig) (*Client, error) {
	addr, err := net.ResolveTCPAddr("tcp", config.LeaderAddr)
	if err != nil {
		return nil, err
	}

	conn, err := net.DialTCP("tcp", nil, addr)
	if err != nil {
		return nil, err
	}

	client := &Client{
		ID:      config.ID,
		Config:  config,
		Request: 1,
		View:    1,
		conn:    conn,
		enc:     gob.NewEncoder(conn),
		dec:     gob.NewDecoder(conn),
		resultc: make(chan Result, 10),
		Results: make(map[ID]Result),
	}

	return client, nil
}

// Do sends the request with given command and args, then waits for the response
func (c *Client) Do(com Command, args []byte) (Result, error) {
	c.mu.RLock()
	conn := c.conn
	enc := c.enc
	c.mu.RUnlock()

	// Check the connection
	if _, err := conn.Write(nil); err != nil {
		return 0, err
	}
	// TODO: Try to get the address of new leader

	req := &Msg{
		Type:    RequestT,
		To:      c.View,
		View:    c.View,
		Request: c.Request,
		Client:  c.ID,
		Command: com,
		Args:    args,
	}

	if err := enc.Encode(req); err != nil {
		return 0, err
	}

	c.mu.Lock()
	c.Request++
	c.mu.Unlock()

	go func() {
		msg := Msg{}
		if err := c.dec.Decode(&msg); err != nil {
			return
		}
		c.addResult(&msg)
	}()

	select {
	case res := <-c.resultc:
		return res, nil
		// case <-time.After(c.Config.Timeout):
		// 	return 0, errors.New("Timeout")
	}
}

// Start creates the client with connection to the current leader
func (c *Client) Start() {
	log.Printf("Starting with laddr: %s\n", c.conn.LocalAddr().String())
}

func (c *Client) addResult(msg *Msg) {
	c.mu.Lock()

	result, ok := c.Results[msg.Request]
	if ok {
		log.Printf("Same result received!\n")
		c.mu.Unlock()
		if result != msg.Result {
			log.Fatal("Wrong results sent with same request number!")
		}
		return
	}

	c.Results[msg.Request] = msg.Result
	c.mu.Unlock()

	// This might block, so don't use defer Unlock()
	c.resultc <- msg.Result
}
