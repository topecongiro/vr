package vr

import (
	"net"
	"sync"
	"testing"
	"time"
)

func newTransportT(id ID, laddr string, config map[ID]string) (*Transport, error) {
	addr, err := net.ResolveTCPAddr("tcp", laddr)
	if err != nil {
		return nil, err
	}

	transport := &Transport{
		DialTimeout: time.Second,
		config:      config,
		ID:          id,
		Addr:        addr,
	}

	return transport, nil
}

func noTestTransport(t *testing.T) {
	t1, err := newTransportT(1, ":8080", map[ID]string{2: ":8081", 3: ":8082"})
	if err != nil {
		t.Fatal(err)
	}
	t2, err := newTransportT(2, ":8081", map[ID]string{1: ":8080", 3: ":8082"})
	if err != nil {
		t.Fatal(err)
	}
	t3, err := newTransportT(3, ":8082", map[ID]string{1: ":8080", 2: ":8081"})
	if err != nil {
		t.Fatal(err)
	}

	var wg sync.WaitGroup
	wg.Add(3)
	go func() {
		err := t1.Start()
		if err != nil {
			t.Error()
		}
		wg.Done()
	}()
	go func() {
		err := t2.Start()
		if err != nil {
			t.Error()
		}
		wg.Done()
	}()
	go func() {
		err := t3.Start()
		if err != nil {
			t.Error()
		}
		wg.Done()
	}()
	// m1 := Msg{
	// 	To:   2,
	// 	From: 1,
	// }
	// m2 := Msg{
	// 	To:   3,
	// 	From: 1,
	// }
	// t1.Send([]Msg{m1, m2})

	wg.Wait()

	t1.printPeers()
	t2.printPeers()
	t3.printPeers()
}

func newReplicaT(id ID, laddr string, config map[ID]string) (*Replica, error) {
	transport, err := newTransportT(id, laddr, config)
	if err != nil {
		return nil, err
	}
	log := NewLog()
	sm := &SM{}
	vr := newMediator(id, time.Second, 2*time.Second, transport, log, sm, config)
	transport.VR = vr
	vr.transport.VR = vr
	replica := &Replica{
		log:       NewLog(),
		sm:        &SM{},
		transport: transport,
		vr:        vr,
	}

	return replica, nil
}

func TestReplica(t *testing.T) {
	r1, err := newReplicaT(1, ":8080", map[ID]string{2: ":8081", 3: ":8082"})
	if err != nil {
		t.Fatal(err)
	}
	r2, err := newReplicaT(2, ":8081", map[ID]string{1: ":8080", 3: ":8082"})
	if err != nil {
		t.Fatal(err)
	}
	r3, err := newReplicaT(3, ":8082", map[ID]string{1: ":8080", 2: ":8081"})
	if err != nil {
		t.Fatal(err)
	}

	var wg sync.WaitGroup
	wg.Add(3)
	go func() {
		err := r1.Run()
		if err != nil {
			t.Error()
		}
		wg.Done()
	}()
	go func() {
		err := r2.Run()
		if err != nil {
			t.Error()
		}
		wg.Done()
	}()
	go func() {
		err := r3.Run()
		if err != nil {
			t.Error()
		}
		wg.Done()
	}()

	wg.Wait()

	r1.debug()
	r2.debug()
	r3.debug()

	time.Sleep(time.Second * 5)
}
