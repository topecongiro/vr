package vr

import (
	"encoding/gob"
	"errors"
	"fmt"
	"log"
	"net"
	"sync"
	"time"
)

// Transport implements Transporter interface.
type Transport struct {
	DialTimeout time.Duration

	config map[ID]string // Addresses of remote replicas.

	ID   ID           // Local ID
	Addr *net.TCPAddr // Local address
	VR   Mediator     // Viewstamped Replication state machine

	sock *net.TCPListener

	mu    sync.RWMutex // Protet the peer map
	peers map[ID]Peer

	done   chan struct{} // Signal the end of bootstrap
	ErrorC chan error
}

// Start starts the VR transport.
func (t *Transport) Start() error {
	sock, err := net.ListenTCP("tcp", t.Addr)
	if err != nil {
		return err
	}
	t.sock = sock
	go t.Handle()

	t.peers = make(map[ID]Peer)
	t.done = make(chan struct{}, 5)
	t.ErrorC = make(chan error)

	for id, addr := range t.config {
		if id < t.ID {
			go t.AddReplica(id, addr, nil)
		}
	}

	select {
	case <-t.done:
		return nil
	case <-time.After(t.DialTimeout):
		return errors.New("Dial timeout")
	}
}

// Handle starts listening to other replicas
func (t *Transport) Handle() {
	for {
		conn, err := t.sock.AcceptTCP()
		if err != nil {
			return
		}

		go t.handleConn(conn)
	}
}

func (t *Transport) handleConn(conn *net.TCPConn) {
	dec := gob.NewDecoder(conn)
	var m Msg
	if err := dec.Decode(&m); err != nil {
		return
	}
	t.AddReplica(m.From, "", conn)
}

// Send ...
func (t *Transport) Send(msgs []Msg) {
	for _, m := range msgs {
		t.mu.RLock()
		p, ok := t.peers[m.To]
		t.mu.RUnlock()

		if ok {
			p.send(m)
		}
	}
}

// Broadcast sends the given message to all other replicas
func (t *Transport) Broadcast(msg Msg) {
	msgs := make([]Msg, len(t.peers))
	var i int
	t.mu.RLock()
	for id := range t.peers {
		newMsg := msg
		newMsg.To = id
		msgs[i] = newMsg
		i++
	}
	t.mu.RUnlock()
	t.Send(msgs)
}

// AddReplica add new replica with the given remote address.
// If the connecion is not yet established, dial to the address.
func (t *Transport) AddReplica(id ID, addr string, conn *net.TCPConn) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	if _, ok := t.peers[id]; ok {
		return nil
	}

	// If the connection is not yet established, dial to the remote address.
	if conn == nil {
		tcpAddr, err := net.ResolveTCPAddr("tcp", addr)
		if err != nil {
			return err
		}

		errc := make(chan error)
		go func() {
			conn, err = net.DialTCP("tcp", nil, tcpAddr)
			if err != nil {
				errc <- err
			}

			// Send your ID to remote replica
			enc := gob.NewEncoder(conn)
			msg := &Msg{To: id, From: t.ID}
			if err := enc.Encode(msg); err != nil {
				errc <- err
			}

			// Connection established
			errc <- nil
		}()
		select {
		case err := <-errc:
			if err != nil {
				return err
			}
			break
		case <-time.After(t.DialTimeout):
			return errors.New("Dial timeout")
		}
	}

	// Start handling messages from replicas.
	t.peers[id] = startPeer(t, id, conn)

	// Signal when connections are establishd with all other replicas
	if len(t.peers) == len(t.config) {
		close(t.done)
	}

	return nil
}

// Stop ...
func (t *Transport) Stop() {
	t.mu.Lock()
	defer t.mu.Unlock()

	for _, p := range t.peers {
		p.stop()
	}

	err := t.sock.Close()
	if err != nil {
		log.Fatal(err)
	}
}

// Debug prints transport specific information
func (t *Transport) Debug() {
	t.printPeers()
}

func (t *Transport) printPeers() {
	t.mu.RLock()
	defer t.mu.RUnlock()

	fmt.Printf("I am Replica(%d)\n", t.ID)
	for _, peer := range t.peers {
		peer.print()
	}
	fmt.Println("")
}

func newTransport(id ID, laddr string, timeout time.Duration, config map[ID]string) (*Transport, error) {
	addr, err := net.ResolveTCPAddr("tcp", laddr)
	if err != nil {
		return nil, err
	}
	transport := &Transport{
		DialTimeout: timeout,
		config:      config,
		ID:          id,
		Addr:        addr,
	}
	if err = transport.Start(); err != nil {
		return nil, err
	}
	return transport, nil
}

func (t *Transport) quorum() int {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return (len(t.peers)+1)/2 + 1
}
