package vr

import (
	"sync"
	"testing"
)

func newVRwithThreeNodes() error {
	r1, err := newReplicaT(1, ":8080", map[ID]string{2: ":8081", 3: ":8082"})
	if err != nil {
		return err
	}
	r2, err := newReplicaT(2, ":8081", map[ID]string{1: ":8080", 3: ":8082"})
	if err != nil {
		return err
	}
	r3, err := newReplicaT(3, ":8082", map[ID]string{1: ":8080", 2: ":8081"})
	if err != nil {
		return err
	}

	var wg sync.WaitGroup
	wg.Add(3)
	go func() {
		err := r1.Run()
		if err != nil {
			return
		}
		wg.Done()
	}()
	go func() {
		err := r2.Run()
		if err != nil {
			return
		}
		wg.Done()
	}()
	go func() {
		err := r3.Run()
		if err != nil {
			return
		}
		wg.Done()
	}()

	wg.Wait()

	r1.debug()
	r2.debug()
	r3.debug()

	return nil
}

func TestClient(t *testing.T) {
	if err := newVRwithThreeNodes(); err != nil {
		t.Fatal(err)
	}
	var wg sync.WaitGroup
	wg.Add(1)
	wg.Wait()
	// log.Printf("vr setup!\n")
	// client, err := NewClient(":8080", 1, ":5555")
	// if err != nil {
	// 	t.Fatal(err)
	// }

	// client.Start()

	// go func() {
	// 	loop := 1
	// 	for {
	// 		fmt.Printf("Loop %d\n", loop)
	// 		client.Send(Inc, []byte{})
	// 		client.Send(Get, []byte{})
	// 		result := <-client.Get()
	// 		log.Printf("Received %d\n", result)
	// 		loop++
	// 	}
	// }()

	// time.Sleep(10 * time.Second)
}
