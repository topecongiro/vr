package vr

import (
	"encoding/gob"
	"errors"
	"log"
	"net"
	"sync"
	"time"
)

// Command specifies the type of command given to the state machine.
type Command int

// Result is the return value of the state machine.
type Result int

type clientInfo struct {
	recent ID
	result Msg
	conn   *net.TCPConn
	enc    *gob.Encoder
	dec    *gob.Decoder
}

// canonical implementaiton of Mediator interface
type vrMediator struct {
	transport *Transport
	log       *Log
	sm        *SM

	Heartbeat  time.Duration
	LeaderIdle time.Duration

	mu sync.RWMutex

	Config      map[ID]string
	ID          ID
	status      status
	Commit      ID // Most recently commited request
	Op          ID // Most recently received request
	View        ID
	ClientTable map[ID]*clientInfo

	prepareOK map[ID]bool
	received  chan struct{}
}

// Process runs the protocol considering the given message
func (vr *vrMediator) Process(m Msg) error {

	vr.mu.Lock()
	defer vr.mu.Unlock()

	switch m.Type {
	case RequestT:
		if vr.isLeader() {
			m.print()
			ci := vr.ClientTable[m.Client]
			if m.Request < ci.recent {
				return nil // Drop the outdated request
			} else if m.Request == ci.recent {
				vr.reply(ci.result) // Resend to the most recent request
			} else {
				vr.Op++
				m.Op = vr.Op
				m.Commit = vr.Commit
				vr.log.Append(m)
				ci.recent = m.Request
				// Send Prepare message
				vr.broadcast(Msg{Client: m.Client, Type: PrepareT, View: vr.View, Op: vr.Op, Commit: vr.Commit, Msg: &m})
			}
		}
	case PrepareT:
		// TODO: Request state transfer if necessary
		if !vr.isLeader() && m.Op == vr.Op+1 {
			// Send PrepareOK
			vr.Op++
			vr.log.Append(m)
			vr.AddClient(m.Client, nil)
			vr.ClientTable[m.Client].recent = m.Request
			vr.send(Msg{Type: PrepareOKT, View: vr.View, Op: vr.Op, To: m.View})
			log.Printf("Replica received prepare\n")

			// Commit up to received commit-number
			vr.commit(m.Commit)
		}
	case PrepareOKT:
		if vr.isLeader() {
			log.Printf("receiving prepareok\n")
			if m.View == vr.View && m.Op == vr.Op {
				vr.prepareOK[m.From] = true
			} else {
				return nil
			}

			// commited
			log.Printf("commit: %d, quorum: %d\n", len(vr.prepareOK), vr.quorum())
			if len(vr.prepareOK) >= vr.quorum() {
				log.Printf("Commited\n")
				vr.commit(vr.Op)
				vr.prepareOK = make(map[ID]bool)
			}
		}
	case ReplyT:
		return errors.New("Replicas shouldn't receive Reply")
	case CommitT:
		if !vr.isLeader() {
			vr.commit(m.Commit)
		}
	default:
		log.Println("Unkown!")
		return errors.New("Unknown message received")
	}
	vr.received <- struct{}{}
	return nil
}

func (vr *vrMediator) Start() error {
	// Start heartbeat goroutines
	go func() {
		var debug int
		id := vr.ID
		for {
			if vr.isLeader() {
				select {
				case <-vr.received:
				case <-time.After(vr.Heartbeat):
					log.Printf("Primary(%d) sending heartbeat\n", vr.ID)
					vr.broadcast(Msg{Type: CommitT, View: vr.View, Commit: vr.Commit})
				}
			} else {
				select {
				case <-vr.received:
				case <-time.After(vr.LeaderIdle):
					log.Printf("Backup(%d) wants to start view change\n", id)
					vr.StartViewChange()
				}
			}
			debug++
		}
	}()
	return nil
}

// StartViewChange starts the view change.
// Backup starts view change whenever:
// 1. couldn't here from the current primary for Leader
// 2. received StartViewChange from other replicas
func (vr *vrMediator) StartViewChange() error {
	// TODO: implement view change protocol
	return nil
}

// AddClient update the client table with new client
func (vr *vrMediator) AddClient(id ID, conn *net.TCPConn) error {
	if _, ok := vr.ClientTable[id]; ok {
		return nil
	}

	vr.ClientTable[id] = &clientInfo{
		conn: conn,
		enc:  gob.NewEncoder(conn),
		dec:  gob.NewDecoder(conn),
	}

	return nil
}

// newMediator creates new VR state machine.
func newMediator(id ID, heartbeat time.Duration, leaderIdle time.Duration, transport *Transport, log *Log, sm *SM, config map[ID]string) *vrMediator {
	return &vrMediator{
		transport:   transport,
		log:         log,
		sm:          sm,
		Heartbeat:   heartbeat,
		LeaderIdle:  leaderIdle,
		Config:      config,
		ID:          id,
		View:        1,
		status:      normal,
		ClientTable: make(map[ID]*clientInfo),
		prepareOK:   make(map[ID]bool),
		received:    make(chan struct{}, 10),
	}
}

func (vr *vrMediator) isLeader() bool {
	return vr.View == vr.ID
}

func (vr *vrMediator) broadcast(m Msg) {
	m.From = vr.ID
	vr.transport.Broadcast(m)
}

func (vr *vrMediator) send(m Msg) {
	m.From = vr.ID
	vr.transport.Send([]Msg{m})
}

func (vr *vrMediator) quorum() int {
	return vr.transport.quorum()
}

// commit executes commands up to given argument
// if the vr is the current primary, it sends the results to the clients.
func (vr *vrMediator) commit(upTo ID) {
	// must make the log up-to-date before commiting
	if vr.Op < upTo {
		return
	}

	for vr.Commit < upTo {
		vr.Commit++
		m := vr.log.Get(vr.Commit)
		result, err := vr.sm.Exec(m.Command, m.Args)
		if err != nil {
			log.Fatal(err)
		}
		ci, ok := vr.ClientTable[m.Client]
		if !ok {
			log.Fatal("no client with id")
		}
		ci.result = Msg{Result: Result(result)}
		if vr.isLeader() {
			rep := m
			rep.Result = Result(result)
			vr.reply(rep)
		}
	}
}

func (vr *vrMediator) handleClient(conn *net.TCPConn) {
	// Get client ID from the first message
	msg := Msg{}
	dec := gob.NewDecoder(conn)
	if err := dec.Decode(&msg); err != nil {
		log.Printf("Replica(%d): %s\n", vr.ID, err)
		return
	}
	vr.AddClient(msg.Client, conn)
	if err := vr.Process(msg); err != nil {
		log.Fatal("State machine failed")
	}
	for {
		msg := Msg{}
		if err := dec.Decode(&msg); err != nil {
			log.Printf("Replica(%d): %s\n", vr.ID, err)
			return
		}
		if err := vr.Process(msg); err != nil {
			log.Fatal("State machine failed")
		}
	}
}

// send reply to the client
func (vr *vrMediator) reply(m Msg) {
	ci, ok := vr.ClientTable[m.Client]
	if !ok {
		return
	}

	log.Printf("primary %d replying to %d(%s), reply = %d\n", vr.ID, m.Client, ci.conn.RemoteAddr().String(), m.Result)
	ci.enc.Encode(&m)
}
