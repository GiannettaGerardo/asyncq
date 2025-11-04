package main

import (
	"errors"
	"log"
	"os"
	"sync"
	"time"
)

// Constants and global variables

const defaultCap = 10

var emptyQueueErr = errors.New("empty queue")

// Structs and interfaces

type AsyncQ[T any] interface {
	Enqueue(task func())
	TryEnqueue(task func()) bool
	Close()
	RunEventLoop()
}

type AsyncQueue[T any] struct {
	queue  []func()
	syn    chan bool
	mutex  sync.Mutex
	logger *log.Logger
}

// Functions and methods

func NewAsyncQueue[T any](initCap int, logger *log.Logger) *AsyncQueue[T] {
	if logger == nil {
		logger = log.New(os.Stdout, "", 0)
	}
	if initCap < defaultCap {
		initCap = defaultCap
	}

	return &AsyncQueue[T]{
		queue:  make([]func(), 0, initCap),
		syn:    make(chan bool),
		logger: logger,
	}
}

func (aq *AsyncQueue[T]) Enqueue(task func()) {
	if task == nil {
		return
	}

	aq.mutex.Lock()
	defer aq.mutex.Unlock()

	aq.queue = append(aq.queue, task)

	select {
	case aq.syn <- true:
		// wake up the event loop
	default:
		// event loop is already awake
	}
}

func (aq *AsyncQueue[T]) TryEnqueue(task func()) bool {
	if task == nil {
		return true
	}

	if !aq.mutex.TryLock() {
		return false
	}
	defer aq.mutex.Unlock()

	aq.queue = append(aq.queue, task)

	select {
	case aq.syn <- true:
		// wake up the event loop
	default:
		// event loop is already awake
	}

	return true
}

func (aq *AsyncQueue[T]) dequeue() (func(), error) {
	aq.mutex.Lock()
	defer aq.mutex.Unlock()

	if len(aq.queue) == 0 {
		return nil, emptyQueueErr
	}

	task := aq.queue[0]
	aq.queue = aq.queue[1:]

	return task, nil
}

func (aq *AsyncQueue[T]) Close() {
	aq.mutex.Lock()
	defer aq.mutex.Unlock()

	aq.queue = append(aq.queue, nil)

	select {
	case aq.syn <- true:
		// wake up the event loop
	default:
		// event loop is already awake
	}

	close(aq.syn)
}

func (aq *AsyncQueue[T]) RunEventLoop() {
	aq.logger.Println("Starting the Async Event Loop")

	for {
		task, err := aq.dequeue()
		if err != nil {
			start := time.Now()
			// wait and sleep
			<-aq.syn
			elapsed := time.Since(start)
			aq.logger.Printf("Event Loop waited %s.\n", elapsed)
			continue
		}

		if task == nil {
			aq.logger.Println("Closing the Async Event Loop")
			break
		}

		task()
	}
}
