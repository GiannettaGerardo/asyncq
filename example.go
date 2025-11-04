package main

import (
	"fmt"
	"log"
	"os"
	"time"
)

func example() {
	var queue AsyncQ[int]
	queue = NewAsyncQueue[int](10, log.New(os.Stdout, "", 0))

	go queue.RunEventLoop()

	word := "world"
	queue.Enqueue(func() {
		fmt.Printf("Hello, %s!\n", word)
	})

	queue.Close()
	time.Sleep(1 * time.Second)
}
