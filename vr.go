package vr

import (
	"context"
	"net"
)

// Replicator fulfills the role of replica in viewstamped replication protocol
type Replicator interface {
	Run() error
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
	Process(ctx context.Context, m Msg) error
	StartViewChange() error
	Start() error
}

// StateMachine implements genereal methods for replicated state machine.
type StateMachine interface {
	Exec(Command, []byte) ([]byte, error)
	Start() error
}

// Logger records the received and commited requests.
type Logger interface {
	Append(Msg)
	Get(ID) Msg
	Start() error
}
