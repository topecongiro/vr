package main

import (
	"fmt"
	"log"

	"github.com/seuchi/vr"
)

func main() {
	client, err := vr.NewClient(":1231", 1, ":5555")
	if err != nil {
		log.Fatal(err)
	}

	client.Start()

	loop := 1
	for {
		fmt.Printf("Loop %d\n", loop)
		client.Send(vr.Inc, []byte{})
		client.Send(vr.Get, []byte{})
		fmt.Printf("Messages sent\n")
		// result := <-client.Get()
		// log.Printf("Received %d\n", result)
		loop++
	}
}
