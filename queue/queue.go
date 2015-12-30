package queue

import (
	"time"

	"github.com/nicholasjackson/event-sauce/entities"
)

type Queue interface {
	Add(message_name string, payload string) error
	AddEvent(event *entities.Event) error
	StartConsuming(size int, poll_interval time.Duration, callback func(event *entities.Event))
}
