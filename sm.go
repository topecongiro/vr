package vr

import "sync"

// SM is canonical implementation of StateMachine interface
type SM struct {
	mu sync.RWMutex
	x  int64
}

// Simple counter service commands
const (
	Inc Command = iota
	Dec
	Get
)

// Exec executes the given commnad in a state machine
func (sm *SM) Exec(com Command, args []byte) ([]byte, error) {
	result := make([]byte, 1)
	switch com {
	case Inc:
		sm.mu.Lock()
		sm.x++
		sm.mu.Unlock()
	case Dec:
		sm.mu.Lock()
		sm.x--
		sm.mu.Unlock()
	case Get:
		sm.mu.RLock()
		result[0] = byte(sm.x)
		sm.mu.RUnlock()
	}
	return result, nil
}

// Start initiates the StateMachine
func (sm *SM) Start() error {
	return nil
}
