package handlers

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"

	"github.com/nicholasjackson/sorcery/logging"
	"github.com/nicholasjackson/sorcery/queue"
)

type EventRequest struct {
	EventName string          `json:"event_name"`
	Payload   json.RawMessage `json:"payload"`
}

type EventDependencies struct {
	// statsD interface must use a name type as injection cannot infer ducktypes
	Stats logging.StatsD `inject:"statsd"`
	Queue queue.Queue    `inject:"eventqueue"`
	Log   *log.Logger    `inject:""`
}

var EventHandlerDependencies *EventDependencies = &EventDependencies{}

const EHTAGNAME = "EventHandler: "

func EventHandler(rw http.ResponseWriter, r *http.Request) {
	EventHandlerDependencies.Stats.Increment(EVENT_HANDLER + POST + CALLED)
	EventHandlerDependencies.Log.Printf("%vHandler Called POST\n", EHTAGNAME)

	defer r.Body.Close()
	data, _ := ioutil.ReadAll(r.Body)
	request := EventRequest{}

	err := json.Unmarshal(data, &request)
	if err != nil || request.EventName == "" || len(request.Payload) < 1 {
		EventHandlerDependencies.Stats.Increment(EVENT_HANDLER + POST + BAD_REQUEST)
		http.Error(rw, "Invalid request object", http.StatusBadRequest)
		return
	}

	if err = EventHandlerDependencies.Queue.Add(request.EventName, string(request.Payload)); err != nil {
		EventHandlerDependencies.Stats.Increment(EVENT_HANDLER + POST + ERROR)
		http.Error(rw, "Error adding item to queue", http.StatusInternalServerError)
		return
	} else {
		EventHandlerDependencies.Stats.Increment(EVENT_HANDLER + POST + SUCCESS)
		var response BaseResponse
		response.StatusEvent = "OK"

		encoder := json.NewEncoder(rw)
		encoder.Encode(&response)
	}
}
