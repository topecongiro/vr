package vr

import "fmt"

// MsgType specifies the type of the message
type MsgType int

// VR message types
const (
	RequestT MsgType = iota
	PrepareT
	PrepareOKT
	ReplyT
	CommitT
)

// Msg ...
type Msg struct {
	To   ID // Receiver replica's ID
	From ID // Sender replica's ID

	Client  ID // Client's ID
	Request ID // Request number assigned to the request by the client

	Type MsgType

	View    ID // Current view-number
	OldView ID // Previous view-number
	Op      ID // Operation number assigned to the request by the primary
	Commit  ID // Operation number of the most recently commited operation

	Msg *Msg // Prepare

	Command Command
	Args    []byte
	Result  Result
}

type Prepare struct {
	View   uint64
	Op     uint64
	Commit uint64
	Msg    *Msg
}

type PrepareOK struct {
	View uint64
	Op   uint64
	From ID
}

type Commit struct {
	View   uint64
	Commit uint64
}

// for debug
func (m *Msg) print() {
	if m.Type == RequestT {
		fmt.Printf("\t***\n\tClinet: %d, Request: %d\n", m.Client, m.Request)
	} else {
		fmt.Printf("\t***\n\tTo: %d, From: %d\n", m.To, m.From)
	}

	fmt.Printf("\tView: %d\n\t***\n", m.View)
}
