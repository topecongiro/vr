package vr

import "net"

// Replicator fulfills the role of replica in viewstamped replication protocol
type Replicator interface {
	Run() error
	Handle() // Handle client's request
	Transport() Transporter
	Consensus() Mediator
	StateMachine() StateMachine
	Log() Logger
}

// Transporter gives an interface to communicate with remote replicas.
type Transporter interface {
	Start() error
	Handle()
	Send(m []Msg)
	AddReplica(id ID, addr string, conn *net.TCPConn) error
	Stop()
	// for debug
	Debug()
}

// Mediator implements Viewstamped Replication consesnsus protocol
type Mediator interface {
	Process(m Msg) error
	StartViewChange() error
	AddClient(ID, *net.TCPConn) error
	Start() error
	handleClient(*net.TCPConn)
}

// StateMachine implements genereal methods for replicated state machine.
type StateMachine interface {
	Exec(Command, []byte) (int, error)
	Start() error
}

// Logger records the received and commited requests.
type Logger interface {
	Append(Msg)
	Get(ID) Msg
	Start() error
}

// NewVR starts the local VR server
func NewVR() Replicator {
	return nil
}
