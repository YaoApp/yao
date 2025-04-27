package message

import (
	"sync"
	"time"

	"github.com/fatih/color"
	"github.com/gin-gonic/gin"
	jsoniter "github.com/json-iterator/go"
	"github.com/yaoapp/kun/log"
)

// AsyncMessageQueue represents a queue for handling message writes
type AsyncMessageQueue struct {
	queue    chan *AsyncTask
	workers  int
	wg       sync.WaitGroup
	shutdown chan struct{}
}

// AsyncTask represents a task to write a message
type AsyncTask struct {
	message *Message
	writer  gin.ResponseWriter
	done    chan bool
}

var (
	defaultQueue *AsyncMessageQueue
	queueOnce    sync.Once
)

// GetQueue returns the default message queue instance
func GetQueue() *AsyncMessageQueue {
	queueOnce.Do(func() {
		defaultQueue = NewAsyncQueue(10) // Initialize with 10 workers
		defaultQueue.Start()
	})
	return defaultQueue
}

// NewAsyncQueue creates a new message queue with the specified number of workers
func NewAsyncQueue(workers int) *AsyncMessageQueue {
	return &AsyncMessageQueue{
		queue:    make(chan *AsyncTask, 1000), // Buffer size of 1000
		workers:  workers,
		shutdown: make(chan struct{}),
	}
}

// Start starts the message queue workers
func (mq *AsyncMessageQueue) Start() {
	for i := 0; i < mq.workers; i++ {
		mq.wg.Add(1)
		go mq.worker()
	}
}

// Stop stops the message queue workers
func (mq *AsyncMessageQueue) Stop() {
	close(mq.shutdown)
	mq.wg.Wait()
}

// worker processes messages from the queue
func (mq *AsyncMessageQueue) worker() {
	defer mq.wg.Done()

	for {
		select {
		case task := <-mq.queue:
			if task == nil {
				continue
			}
			success := writeMessageToResponse(task.message, task.writer)
			if task.done != nil {
				task.done <- success
			}
		case <-mq.shutdown:
			return
		}
	}
}

// WriteMessageAsync writes the message to response writer using the message queue
func WriteMessageAsync(m *Message, w gin.ResponseWriter) bool {
	done := make(chan bool, 1)
	task := &AsyncTask{
		message: m,
		writer:  w,
		done:    done,
	}

	// Try to send the task to the queue with a timeout
	select {
	case GetQueue().queue <- task:
		// Wait for the message to be processed with a longer timeout
		select {
		case success := <-done:
			return success
		case <-time.After(5 * time.Second): // Increased timeout to 5 seconds
			log.Error("Message processing timeout")
			return false
		}
	case <-time.After(1 * time.Second): // Increased queue timeout to 1 second
		log.Error("Queue is full, message dropped")
		return false
	}
}

// writeMessageToResponse writes the message directly to the response writer
func writeMessageToResponse(m *Message, w gin.ResponseWriter) bool {
	// Sync write to response writer
	locker.Lock()
	defer locker.Unlock()

	defer func() {
		if r := recover(); r != nil {
			// Ignore if done is true
			if m.IsDone {
				return
			}

			message := "Write Response Exception: (if client close the connection, it's normal) \n  %s\n\n"
			color.Red(message, r)

			// Print the message
			raw, _ := jsoniter.MarshalToString(m)
			color.White("Message:\n %s", raw)
		}
	}()

	// Ignore silent messages
	if m.Silent {
		return true
	}

	data, err := jsoniter.Marshal(m)
	if err != nil {
		log.Error("%s", err.Error())
		return false
	}

	data = append([]byte("data: "), data...)
	data = append(data, []byte("\n\n")...)

	if _, err := w.Write(data); err != nil {
		color.Red("Write JSON Message Error: %s", err.Error())
		return false
	}
	w.Flush()
	return true
}
