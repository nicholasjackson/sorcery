package main

import (
	"log"
	"sync"
	"time"

	"github.com/nicholasjackson/event-sauce/global"
	"github.com/nicholasjackson/event-sauce/logging"
	"github.com/nicholasjackson/event-sauce/queue"
	"github.com/nicholasjackson/event-sauce/workers"
)

type ClientDependencies struct {
	// statsD interface must use a name type as injection cannot infer ducktypes
	Stats                   logging.StatsD        `inject:"statsd"`
	EventQueue              queue.Queue           `inject:"eventqueue"`
	DeadLetterQueue         queue.Queue           `inject:"deadletterqueue"`
	EventWorkerFactory      workers.WorkerFactory `inject:"eventqueueworkerfactory"`
	DeadLetterWorkerFactory workers.WorkerFactory `inject:"deadletterqueueworkerfactory"`
	Log                     *log.Logger           `inject:""`
}

var ClientDeps *ClientDependencies = &ClientDependencies{}

const EVENT_QUEUE_CLIENT_STARTED = "eventsauce.eventqueue.client.started"
const DEADLETTER_QUEUE_CLIENT_STARTED = "eventsauce.deadletterqueue.client.started"

func startClient(wg *sync.WaitGroup) {
	defer wg.Done()

	go processEventQueue()
	go processDeadLetterQueue()
}

func processEventQueue() {
	ClientDeps.Stats.Increment(EVENT_QUEUE_CLIENT_STARTED)
	ClientDeps.Log.Println("Starting Event Queue")

	ClientDeps.EventQueue.StartConsuming(10, time.Second, func(callbackItem interface{}) {
		worker := ClientDeps.EventWorkerFactory.Create()
		worker.HandleItem(callbackItem)
	})
}

func processDeadLetterQueue() {
	ClientDeps.Log.Println("Starting Dead Letter Queue")
	ClientDeps.Stats.Increment(DEADLETTER_QUEUE_CLIENT_STARTED)

	// set polling to minimum retry duration
	duration, _ := time.ParseDuration(global.Config.RetryIntervals[0])

	ClientDeps.DeadLetterQueue.StartConsuming(10, duration, func(callbackItem interface{}) {
		worker := ClientDeps.DeadLetterWorkerFactory.Create()
		worker.HandleItem(callbackItem)
	})
}
