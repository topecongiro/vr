package vr

import "log"

// Log implements Logger interface
type Log struct {
	log map[ID]Msg
}

// NewLog creates a new Log
func NewLog() *Log {
	return &Log{
		log: make(map[ID]Msg),
	}
}

// Append appends the given message to the Log
func (l *Log) Append(m Msg) {
	log.Printf("log[%d] = %d\n", m.Op, m.Client)
	l.log[m.Op] = m
}

// Get returns the Msg to which the given operation number is assigned
func (l *Log) Get(index ID) Msg {
	return l.log[index]
}

// Start initializes and starts the Log
func (l *Log) Start() error {
	l.log = make(map[ID]Msg)
	return nil
}
