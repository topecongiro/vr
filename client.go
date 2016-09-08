package vr

import (
	"encoding/gob"
	"log"
	"net"
	"sync"
)

// Client holds the client-side information
type Client struct {
	mu sync.RWMutex

	ID      ID
	Leader  string // The address of the current leader this Client believes
	Request ID     // Monotonical increasing reqeust number
	View    ID

	resultc chan Result
	Results map[ID]Result

	conn *net.TCPConn // TCP connection to the current leader
	enc  *gob.Encoder
	dec  *gob.Decoder

	stopc chan struct{}
}

// NewClient creates new client instance with given information
func NewClient(leader string, id ID, laddr string) (*Client, error) {
	addr, err := net.ResolveTCPAddr("tcp", leader)
	if err != nil {
		return nil, err
	}

	conn, err := net.DialTCP("tcp", nil, addr)
	if err != nil {
		return nil, err
	}

	client := &Client{
		ID:      id,
		Leader:  leader,
		Request: 1,
		View:    1,
		conn:    conn,
		enc:     gob.NewEncoder(conn),
		dec:     gob.NewDecoder(conn),
		stopc:   make(chan struct{}),
		resultc: make(chan Result, 10),
		Results: make(map[ID]Result),
	}

	return client, nil
}

// Send sends the request with given command and args
func (c *Client) Send(com Command, args []byte) error {
	c.mu.RLock()
	conn := c.conn
	enc := c.enc
	c.mu.RUnlock()

	// Check the connection
	if _, err := conn.Write(nil); err != nil {
		return err
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
		return err
	}

	c.mu.Lock()
	c.Request++
	c.mu.Unlock()

	return nil
}

// Get receives the result
func (c *Client) Get() <-chan Result {
	return c.resultc
}

// Start creates the client with connection to the current leader
func (c *Client) Start() {
	log.Printf("Starting with laddr: %s\n", c.conn.LocalAddr().String())
	go c.handleResult()
}

func (c *Client) handleResult() {
	for {

		msg := Msg{}
		if err := c.dec.Decode(&msg); err != nil {
			return
		}

		go c.addResult(&msg)
	}
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
	go func() {
		c.resultc <- msg.Result
	}()
}
