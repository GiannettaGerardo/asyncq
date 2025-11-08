package main

import (
	"log"
	"os"
	"sync"
	"time"
)

// Constants and global variables

const defaultCap = 10

// Structs and interfaces

// AsyncQ is an interface for any AsyncQ implementation. Allows
// queuing, closing the queue, and starting the Event Loop.
// Tasks are simply parameterless procedures, because you can
// use closures that capture external context variables.
type AsyncQ[T any] interface {

	// Enqueue a new task in queue.
	Enqueue(task func())

	// TryEnqueue enqueue a new task in queue if it's possible. The 'possibility'
	// must be decided by the specific implementation. For example, can be
	// useful when a lock mechanism is used, like a mutex.
	TryEnqueue(task func()) bool

	// Close the queue. No more tasks can be enqueued.
	Close()

	// RunEventLoop start the event loop. It must be started in a new goroutine.
	RunEventLoop()
}

// AsyncQueue simple queue with a single mutex for enqueue and dequeue operations.
//
//   - Use a channel to put the event loop to sleep or to wake-up it.
//   - The event loop is 'panic' safe.
//   - A nil task closes the event loop (but not the channel, so it's safe to only use Close).
type AsyncQueue[T any] struct {
	queue  []func()
	syn    chan bool
	mutex  sync.Mutex
	logger *log.Logger
}

// AsyncDoubleQueue uses two queues, one in for enqueue operations (inputQueue)
// and another one for dequeues operations (outputQueue).
//
//   - A single mutex is used to lock only the inputQueue. The event loop acquire the
//     lock only in one occasion: when outputQueue is empty and exchanges the two queues.
//   - Use a channel to put the event loop to sleep or to wake-up it.
//   - The event loop is 'panic' safe.
//   - A nil task closes the event loop (but not the channel, so it's safe to only use Close).
type AsyncDoubleQueue[T any] struct {
	inputQueue  []func()
	outputQueue []func()
	outputIdx   int
	syn         chan bool
	mutex       sync.Mutex
	logger      *log.Logger
}

// Functions and methods for AsyncQueue

// NewAsyncQueue returns a new heap allocated AsyncQueue.
//
//   - If 'initCap' is less the 'defaultCap', 'defaultCap' is used as initial capacity.
//   - If 'logger' is nil, plain STDOUT is used.
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

// TryEnqueue returns 'true' when the enqueue is done, 'false' otherwise.
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

// dequeue returns the task dequeued plus 'false', or a nil task plus 'true'
// when the queue is empty.
func (aq *AsyncQueue[T]) dequeue() (func(), bool) {
	aq.mutex.Lock()
	defer aq.mutex.Unlock()

	if len(aq.queue) == 0 {
		return nil, true
	}

	task := aq.queue[0]
	aq.queue = aq.queue[1:]

	return task, false
}

// Close the event loop (add a nil task to the queue) and the channel.
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

	for aq.safeRunEventLoop() {
		// empty body
		// panic can't stop the event loop
	}
}

func (aq *AsyncQueue[T]) safeRunEventLoop() (result bool) {
	result = true
	defer func() {
		if r := recover(); r != nil {
			aq.logger.Println("recover from panic.")
		}
	}()

	task, isEmpty := aq.dequeue()
	if isEmpty {
		start := time.Now()
		// wait and sleep
		<-aq.syn
		elapsed := time.Since(start)
		aq.logger.Printf("Event Loop waited %s.\n", elapsed)
		return
	}

	if task == nil {
		aq.logger.Println("Closing the Async Event Loop")
		result = false
		return
	}

	task()
	return
}

// Functions and methods for AsyncDoubleQueue

// NewAsyncDoubleQueue returns a new heap allocated AsyncDoubleQueue.
//
//   - If 'initCap' is less the 'defaultCap', 'defaultCap' is used as initial capacity.
//   - If 'logger' is nil, plain STDOUT is used.
func NewAsyncDoubleQueue[T any](initCap int, logger *log.Logger) *AsyncDoubleQueue[T] {
	if logger == nil {
		logger = log.New(os.Stdout, "", 0)
	}
	if initCap < defaultCap {
		initCap = defaultCap
	}

	return &AsyncDoubleQueue[T]{
		inputQueue:  make([]func(), 0, initCap),
		outputQueue: make([]func(), 0, initCap),
		outputIdx:   0,
		syn:         make(chan bool),
		logger:      logger,
	}
}

func (aq *AsyncDoubleQueue[T]) Enqueue(task func()) {
	if task == nil {
		return
	}

	aq.mutex.Lock()
	defer aq.mutex.Unlock()

	aq.inputQueue = append(aq.inputQueue, task)

	select {
	case aq.syn <- true:
		// wake up the event loop
	default:
		// event loop is already awake
	}
}

// TryEnqueue returns 'true' when the enqueue is done, 'false' otherwise.
func (aq *AsyncDoubleQueue[T]) TryEnqueue(task func()) bool {
	if task == nil {
		return true
	}

	if !aq.mutex.TryLock() {
		return false
	}
	defer aq.mutex.Unlock()

	aq.inputQueue = append(aq.inputQueue, task)

	select {
	case aq.syn <- true:
		// wake up the event loop
	default:
		// event loop is already awake
	}

	return true
}

// Close the event loop (add a nil task to the inputQueue) and the channel.
func (aq *AsyncDoubleQueue[T]) Close() {
	aq.mutex.Lock()
	defer aq.mutex.Unlock()

	aq.inputQueue = append(aq.inputQueue, nil)

	select {
	case aq.syn <- true:
		// wake up the event loop
	default:
		// event loop is already awake
	}

	close(aq.syn)
}

// exchangeQueues exchanges the input and output queues. This method is
// the only one the acquires the lock for the inputQueue in the event loop.
// Returns 'false' when the input queue is empty, 'true' otherwise.
func (aq *AsyncDoubleQueue[T]) exchangeQueues() bool {
	aq.mutex.Lock()
	defer aq.mutex.Unlock()

	if len(aq.inputQueue) == 0 {
		return false
	}

	tmp := aq.inputQueue
	aq.inputQueue = aq.outputQueue[:0]
	aq.outputQueue = tmp

	return true
}

// dequeue does not acquire the lock unless the outputQueue is empty,
// but acquires the lock only for the time needed to exchange the queues.
// Returns the task dequeued plus 'false', or a nil task plus 'true'
// when the queue is empty.
func (aq *AsyncDoubleQueue[T]) dequeue() (func(), bool) {

	if aq.outputIdx >= len(aq.outputQueue) {
		if !aq.exchangeQueues() {
			return nil, true
		}
		aq.outputIdx = 0
	}

	task := aq.outputQueue[aq.outputIdx]
	aq.outputIdx++

	return task, false
}

func (aq *AsyncDoubleQueue[T]) RunEventLoop() {
	aq.logger.Println("Starting the Async Event Loop")

	for aq.safeRunEventLoop() {
		// empty body
		// panic can't stop the event loop
	}
}

func (aq *AsyncDoubleQueue[T]) safeRunEventLoop() (result bool) {
	result = true
	defer func() {
		if r := recover(); r != nil {
			aq.logger.Println("recover from panic.")
		}
	}()

	task, isEmpty := aq.dequeue()
	if isEmpty {
		start := time.Now()
		// wait and sleep
		<-aq.syn
		elapsed := time.Since(start)
		aq.logger.Printf("Event Loop waited %s.\n", elapsed)
		return
	}

	if task == nil {
		aq.logger.Println("Closing the Async Event Loop")
		result = false
		return
	}

	task()
	return
}
