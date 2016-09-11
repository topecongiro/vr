package vr

import (
	"log"
	"sync"
)

// SM is canonical implementation of StateMachine interface
type SM struct {
	mu sync.RWMutex
	x  int
}

// Simple counter service commands
const (
	Inc Command = iota
	Dec
	Get
)

// Exec executes the given commnad in a state machine
func (sm *SM) Exec(com Command, args []byte) (int, error) {
	var result int
	switch com {
	case Inc:
		sm.mu.Lock()
		sm.x++
		result = sm.x
		sm.mu.Unlock()
	case Dec:
		sm.mu.Lock()
		sm.x--
		result = sm.x
		sm.mu.Unlock()
	case Get:
		sm.mu.RLock()
		result = sm.x
		sm.mu.RUnlock()
	}
	log.Printf("Executing command: result = %d\n", result)
	return result, nil
}

// Start initiates the StateMachine
func (sm *SM) Start() error {
	return nil
}
