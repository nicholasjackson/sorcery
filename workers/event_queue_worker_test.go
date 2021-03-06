package workers

import (
	"fmt"
	"log"
	"os"
	"testing"

	"github.com/nicholasjackson/sorcery/entities"
	"github.com/nicholasjackson/sorcery/global"
	"github.com/nicholasjackson/sorcery/handlers"
	"github.com/nicholasjackson/sorcery/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

var mockDispatcher *mocks.MockEventDispatcher
var mockDal *mocks.MockDal
var mockQueue *mocks.MockQueue
var mockStatsD *mocks.MockStatsD
var worker *EventQueueWorker
var registrations []*entities.Registration

func getRegistrations() []*entities.Registration {
	return registrations
}

func setupTests(t *testing.T) {
	mockDispatcher = &mocks.MockEventDispatcher{}
	mockDal = &mocks.MockDal{}
	mockQueue = &mocks.MockQueue{}
	mockStatsD = &mocks.MockStatsD{}
	worker = New(mockDispatcher, mockDal, mockQueue, log.New(os.Stdout, "Testing: ", log.Lshortfile), mockStatsD)
	registrations = []*entities.Registration{&entities.Registration{CallbackUrl: "myendpoint"}}

	global.Config.RetryIntervals = []string{"1d"}

	mockDispatcher.Mock.On("DispatchEvent", mock.Anything, mock.Anything).Return(200, nil)
	mockDal.Mock.On("GetRegistrationsByEvent", mock.Anything).Return(getRegistrations, nil)
	mockDal.Mock.On("DeleteRegistration", mock.Anything).Return(nil)
	mockDal.Mock.On("UpsertEventStore", mock.Anything).Return(nil)
	mockDal.Mock.On("UpsertDeadLetterItem", mock.Anything).Return(nil)
	mockQueue.Mock.On("AddEvent", mock.Anything, mock.Anything).Return(nil)
	mockStatsD.Mock.On("Increment", mock.Anything).Return()
}

func TestSetsEventDispatcherAndDal(t *testing.T) {
	setupTests(t)

	assert.Equal(t, mockDispatcher, worker.eventDispatcher)
	assert.Equal(t, mockDal, worker.dal)
}

func TestHandleEventSavesToEventStore(t *testing.T) {
	setupTests(t)
	event := &entities.Event{}

	_ = worker.HandleItem(event)

	mockDal.Mock.AssertCalled(t, "UpsertEventStore", mock.Anything)
	mockStatsD.Mock.AssertCalled(t, "Increment", handlers.EVENT_QUEUE+handlers.WORKER+handlers.HANDLE)
}

func TestHandleEventAttemptsToDispatchEvent(t *testing.T) {
	setupTests(t)
	event := &entities.Event{}
	endpoint := "myendpoint"

	_ = worker.HandleItem(event)

	mockDispatcher.Mock.AssertCalled(t, "DispatchEvent", event, endpoint)
	mockStatsD.Mock.AssertCalled(t, "Increment", handlers.EVENT_QUEUE+handlers.WORKER+handlers.DISPATCH)
}

func TestHandleEventAttemptsToDispatchMultipleEvent(t *testing.T) {
	setupTests(t)
	event := &entities.Event{}
	endpoint := "myendpoint"
	registrations = []*entities.Registration{
		&entities.Registration{CallbackUrl: endpoint},
		&entities.Registration{CallbackUrl: endpoint},
		&entities.Registration{CallbackUrl: endpoint}}

	_ = worker.HandleItem(event)

	mockDispatcher.Mock.AssertNumberOfCalls(t, "DispatchEvent", 3)
}

func TestHandleEventGetsRegisterdEndpointsFromDB(t *testing.T) {
	setupTests(t)
	event := &entities.Event{EventName: "myevent"}

	_ = worker.HandleItem(event)

	mockDal.Mock.AssertCalled(t, "GetRegistrationsByEvent", "myevent")
}

func TestDispatchEventFailureRemovesRegistrationWhenRegistrationFoundAndEndpointDoesNotExist(t *testing.T) {
	setupTests(t)
	event := &entities.Event{}
	endpoint := "myendpoint"
	registrations := []*entities.Registration{&entities.Registration{CallbackUrl: endpoint}}

	mockDispatcher.Mock.ExpectedCalls = []*mock.Call{} // reset calls
	mockDispatcher.Mock.On("DispatchEvent", event, endpoint).Return(404, fmt.Errorf("Unable to complete"))

	_ = worker.HandleItem(event)

	mockDal.Mock.AssertCalled(t, "DeleteRegistration", registrations[0])
	mockStatsD.Mock.AssertCalled(t, "Increment", handlers.EVENT_QUEUE+handlers.WORKER+handlers.DELETE_REGISTRATION)
}

func TestDispatchEventFailureAddsEventToDeadLetterQueueWhenEndpointInErrorState(t *testing.T) {
	setupTests(t)
	event := &entities.Event{}
	endpoint := "myendpoint"

	mockDispatcher.Mock.ExpectedCalls = []*mock.Call{} // reset calls
	mockDispatcher.Mock.On("DispatchEvent", event, endpoint).Return(500, fmt.Errorf("Unable to complete"))

	_ = worker.HandleItem(event)

	mockQueue.Mock.AssertNumberOfCalls(t, "AddEvent", 1)
	mockStatsD.Mock.AssertCalled(t, "Increment", handlers.EVENT_QUEUE+handlers.WORKER+handlers.PROCESS_REDELIVERY)
}

func TestDispatchEventOKDoesNothing(t *testing.T) {
	setupTests(t)
	event := &entities.Event{}

	_ = worker.HandleItem(event)

	mockDal.Mock.AssertNumberOfCalls(t, "DeleteRegistration", 0)
	mockQueue.Mock.AssertNumberOfCalls(t, "AddEvent", 0)
	mockStatsD.Mock.AssertCalled(t, "Increment", handlers.EVENT_QUEUE+handlers.WORKER+handlers.DISPATCH)
}
