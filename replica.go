package vr

import "errors"

type status int

const (
	normal status = iota
	viewChange
	recovery
)

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
}

// NewReplica starts a new replica with given transport.
func NewReplica(vr Mediator, sm StateMachine, transport Transporter, log Logger) (*Replica, error) {

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

	return &Replica{vr: vr, sm: sm, transport: transport, log: log}, nil
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
	r.transport.Debug()
}
