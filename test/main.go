package main

import (
	"fmt"
	"log"
	"time"

	"github.com/topecongiro/vr"
)

func main() {
	config := vr.ClientConfig{
		LeaderAddr: ":1231",
		ID:         1,
		View:       1,
		Timeout:    time.Second,
	}
	client, err := vr.NewClient(config)
	if err != nil {
		log.Fatal(err)
	}

	client.Start()

	var sum int
	for i := 0; i < 1000; i++ {
		res, err := client.Do(vr.Inc, []byte{})
		if err != nil {
			fmt.Printf("%s\n", err)
			return
		}
		sum = int(res)
	}
	if sum != 1000 {
		log.Printf("test failed...sum = %d\n", sum)
	} else {
		fmt.Println("Success!")
	}
}
