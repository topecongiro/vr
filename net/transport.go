package net

import (
	"errors"
	"log"
	"net"
	"sync"
	"time"

	"github.com/seuchi/vr"
)

// Transporter gives an interface to communicate with remote replicas.
type Transporter interface {
	Start() error
	Handle()
	Send(m []vr.Msg)
	AddReplica(id vr.ID, addr string, conn *net.TCPConn) error
	Stop()
}

// Transport implements Transporter interface.
type Transport struct {
	DialTimeout time.Duration

	config []string // Addresses of remote replicas.

	ID   vr.ID        // Local ID
	Addr *net.TCPAddr // Local address
	VR   vr.VR        // Viewstamped Replication state machine

	sock *net.TCPListener

	mu    sync.RWMutex // Protet the peer map
	peers map[vr.ID]Peer

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

	t.peers = make(map[vr.ID]Peer)

	return nil
}

// Handle handles the requests from other replicas.
func (t *Transport) Handle() {

}

// Send ...
func (t *Transport) Send(msgs []vr.Msg) {
	for _, m := range msgs {
		t.mu.RLock()
		p, ok := t.peers[m.To]
		t.mu.RUnlock()

		if ok {
			p.send(m)
		}
	}
}

// AddReplica add new replica with the given remote address.
// If the connecion is not yet established, dial to the address.
func (t *Transport) AddReplica(id vr.ID, addr string, conn *net.TCPConn) error {
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
			errc <- err
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

	t.peers[id] = startPeer(t, id, conn)
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
