package vr

import (
	"encoding/gob"
	"errors"
	"fmt"
	"log"
	"net"
	"sync"
)

type status int

const (
	normal status = iota
	viewChange
	recovery
)

type client struct {
	conn *net.TCPConn
	enc  *gob.Encoder
	dec  *gob.Decoder
}

// Replica is a canonical implementation of Replicator interface.
// Currently this is just a placeholder.
type Replica struct {
	// The body of the VR consensus protocol
	vr Mediator
	// Replicated state machine
	sm StateMachine
	// System specific transport layer
	transport Transporter
	// Stores the received/commitd requests
	log Logger

	// socket for client's request
	laddr string
	sock  *net.TCPListener

	// clients connection
	mu sync.RWMutex

	stopc chan struct{}
}

// NewReplica starts a new replica with given transport.
func NewReplica(sockAddr string, vr Mediator, sm StateMachine, transport Transporter, log Logger) (*Replica, error) {

	if vr == nil {
		// TODO: Set default VR consensus protocol
		return nil, errors.New("Give consensus protocol")
	}

	if sm == nil {
		// TODO: Set default state machine
		return nil, errors.New("Give state machine")
	}

	if transport == nil {
		// Use default Transport
		return nil, errors.New("Give transport layer")
	}

	if log == nil {
		return nil, errors.New("Give me log")
	}

	return &Replica{
		vr:        vr,
		sm:        sm,
		transport: transport,
		log:       log,
		stopc:     make(chan struct{}),
		laddr:     sockAddr,
	}, nil
}

// Run starts the replica as a VR running server
func (r *Replica) Run() error {
	if err := r.transport.Start(); err != nil {
		return err
	}
	if err := r.sm.Start(); err != nil {
		return err
	}
	if err := r.vr.Start(); err != nil {
		return err
	}
	if err := r.log.Start(); err != nil {
		return err
	}

	addr, err := net.ResolveTCPAddr("tcp", r.laddr)
	if err != nil {
		return err
	}

	r.sock, err = net.ListenTCP("tcp", addr)
	if err != nil {
		return err
	}

	go r.Handle()

	return nil
}

// Transport ...
func (r *Replica) Transport() Transporter {
	return r.transport
}

// Consensus ...
func (r *Replica) Consensus() Mediator {
	return r.vr
}

// StateMachine ...
func (r *Replica) StateMachine() StateMachine {
	return r.sm
}

// Log ...
func (r *Replica) Log() Logger {
	return r.log
}

func (r *Replica) debug() {
	r.mu.RLock()
	addr := r.sock.Addr()
	r.mu.RUnlock()
	fmt.Printf("Node socket addr: %s\n", addr)
	r.transport.Debug()
}

// Handle handles requests from clients
func (r *Replica) Handle() {
	for {
		conn, err := r.sock.AcceptTCP()
		if err != nil {
			log.Printf("%s\n", err)
			continue
		}

		go r.vr.handleClient(conn)
	}
}
