package main

import (
	"fmt"
	"log"
	"os"
)

func example() {
	var queue AsyncQ[int]
	queue = NewAsyncDoubleQueue[int](10, log.New(os.Stdout, "", 0))

	go queue.RunEventLoop()

	word := "world"
	queue.Enqueue(func() {
		fmt.Printf("Hello, %s!\n", word)
	})

	queue.Close()
}
