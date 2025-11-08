# AsyncQ
#### Asynchronous queue and event loop in GO.

### Implementations

Package *gg/asyncq* exposes an interface **AsyncQ** and two implementations: **AsyncQueue** and **AsyncDoubleQueue**.

- **AsyncQueue**: it's a simple queue with a single mutex for enqueue and dequeue operations. 
  - Use a channel to put the event loop to sleep or to wake-up it;
  - The event loop is **panic safe**;
  - A **nil task** closes the event loop (but not the channel, so it's safe to only use **Close**).


- **AsyncDoubleQueue**: uses two queues, one in for enqueue operations (*inputQueue*) and another one for dequeues operations (*outputQueue*).
  - A single mutex is used to lock only the *inputQueue*. The event loop acquire the lock only in one occasion: when *outputQueue* is empty and exchanges the two queues;
  - Use a channel to put the event loop to sleep or to wake-up it;
  - The event loop is **panic safe**;
  - A **nil task** closes the event loop (but not the channel, so it's safe to only use **Close**).

### Example

```go
package main

import (
	"fmt"
	"log"
	"os"
	
	"gg/asyncq"
)

func main() {
    var queue asyncq.AsyncQ[int]
    queue = asyncq.NewAsyncDoubleQueue[int](10, log.New(os.Stdout, "", 0))
    
    go queue.RunEventLoop()
    
    word := "world"
    queue.Enqueue(func() {
        fmt.Printf("Hello, %s!\n", word)
    })
    
    queue.Close()
}
```