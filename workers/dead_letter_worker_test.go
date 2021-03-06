package workers

import (
	"fmt"
	"log"
	"os"
	"testing"
	"time"

	"github.com/nicholasjackson/sorcery/entities"
	"github.com/nicholasjackson/sorcery/global"
	"github.com/nicholasjackson/sorcery/handlers"
	"github.com/nicholasjackson/sorcery/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

var mockDeadDispatcher *mocks.MockEventDispatcher
var mockDeadDal *mocks.MockDal
var mockDeadStatsD *mocks.MockStatsD
var deadWorker *DeadLetterWorker
var deadReg []*entities.Registration
var deadError error

func getDeadRegistrations() []*entities.Registration {
	return deadReg
}

func getDeadRegistration() (*entities.Registration, error) {
	if len(deadReg) > 0 {
		return deadReg[0], deadError
	} else {
		return nil, deadError
	}
}

func setupDeadTests(t *testing.T) {
	mockDeadDispatcher = &mocks.MockEventDispatcher{}
	mockDeadDal = &mocks.MockDal{}
	mockDeadStatsD = &mocks.MockStatsD{}
	deadWorker = NewDeadLetterWorker(mockDeadDispatcher, mockDeadDal, log.New(os.Stdout, "testing: ", log.Lshortfile), mockDeadStatsD)
	deadReg = []*entities.Registration{&entities.Registration{CallbackUrl: "myendpoint"}}
	deadError = nil

	global.Config.RetryIntervals = []string{"1d", "2d", "5d"}

	mockDeadDispatcher.Mock.On("DispatchEvent", mock.Anything, mock.Anything).Return(200, nil)
	mockDeadDal.Mock.On("GetRegistrationsByEvent", mock.Anything).Return(getDeadRegistrations, nil)
	mockDeadDal.Mock.On("GetRegistrationByEventAndCallback", mock.Anything, mock.Anything).Return(getDeadRegistration)
	mockDeadDal.Mock.On("DeleteRegistration", mock.Anything).Return(nil)
	mockDeadDal.Mock.On("UpsertEventStore", mock.Anything).Return(nil)
	mockDeadDal.Mock.On("UpsertDeadLetterItem", mock.Anything).Return(nil)
	mockDeadStatsD.Mock.On("Increment", mock.Anything).Return()
}

func TestHandleItemDoesNothingIfNoRegisteredEndpoint(t *testing.T) {
	setupDeadTests(t)

	event := entities.Event{EventName: "mytestevent"}
	deadLetter := &entities.DeadLetterItem{Event: event, CallbackUrl: "myurl"}
	deadError = fmt.Errorf("Not found")
	deadReg = []*entities.Registration{}

	deadWorker.HandleItem(deadLetter)

	mockDeadDispatcher.Mock.AssertNotCalled(t, "DispatchEvent", mock.Anything, mock.Anything)
	mockDeadStatsD.Mock.AssertCalled(t, "Increment", handlers.DEAD_LETTER_QUEUE+handlers.WORKER+handlers.HANDLE)
	mockDeadStatsD.Mock.AssertCalled(t, "Increment", handlers.DEAD_LETTER_QUEUE+handlers.WORKER+handlers.NO_ENDPOINT)
}

func TestHandleItemDispatchesEvent(t *testing.T) {
	setupDeadTests(t)

	event := entities.Event{EventName: "mytestevent"}
	deadLetter := &entities.DeadLetterItem{Event: event, CallbackUrl: "myurl"}

	deadWorker.HandleItem(deadLetter)

	mockDeadDispatcher.Mock.AssertCalled(t, "DispatchEvent", mock.Anything, deadLetter.CallbackUrl)
	mockDeadStatsD.Mock.AssertCalled(t, "Increment", handlers.DEAD_LETTER_QUEUE+handlers.WORKER+handlers.DISPATCH)
}

func TestHandleItemDispatchesEventDoesNotRetry(t *testing.T) {
	setupDeadTests(t)

	event := entities.Event{EventName: "mytestevent"}
	deadLetter := &entities.DeadLetterItem{Event: event, CallbackUrl: "myurl"}

	deadWorker.HandleItem(deadLetter)

	mockDeadDal.Mock.AssertNotCalled(t, "UpsertDeadLetterItem", mock.Anything)
}

func TestHandleItemDispatchesEventDoesNotDeleteRegistration(t *testing.T) {
	setupDeadTests(t)

	event := entities.Event{EventName: "mytestevent"}
	deadLetter := &entities.DeadLetterItem{Event: event, CallbackUrl: "myurl"}

	deadWorker.HandleItem(deadLetter)

	mockDeadDal.Mock.AssertNotCalled(t, "DeleteRegistration", mock.Anything)
}

func TestHandleItemWithUndeliverableSetsRedeliveryCriteria(t *testing.T) {
	setupDeadTests(t)

	mockDeadDispatcher.Mock.ExpectedCalls = []*mock.Call{} // reset calls
	mockDeadDispatcher.Mock.On("DispatchEvent", mock.Anything, mock.Anything).Return(500, fmt.Errorf("Unable to complete"))

	event := entities.Event{}
	deadLetter := &entities.DeadLetterItem{Event: event, CallbackUrl: "myurl", FailureCount: 1, NextRetryDate: time.Now()}

	deadWorker.HandleItem(deadLetter)

	duration, _ := time.ParseDuration(global.Config.RetryIntervals[1])
	retryDate := deadLetter.NextRetryDate.Add(duration)

	assert.Equal(t, 2, deadLetter.FailureCount)
	assert.Equal(t, retryDate, deadLetter.NextRetryDate)
}

func TestHandleItemWithErrorStateAddsToDeadLetterQueue(t *testing.T) {
	setupDeadTests(t)

	mockDeadDispatcher.Mock.ExpectedCalls = []*mock.Call{} // reset calls
	mockDeadDispatcher.Mock.On("DispatchEvent", mock.Anything, mock.Anything).Return(500, fmt.Errorf("Unable to complete"))

	event := entities.Event{}
	deadLetter := &entities.DeadLetterItem{Event: event, CallbackUrl: "myurl", FailureCount: 1, NextRetryDate: time.Now()}

	deadWorker.HandleItem(deadLetter)

	mockDeadDal.Mock.AssertCalled(t, "UpsertDeadLetterItem", deadLetter)
	mockDeadStatsD.Mock.AssertCalled(t, "Increment", handlers.DEAD_LETTER_QUEUE+handlers.WORKER+handlers.PROCESS_REDELIVERY)
}

func TestHandleItemWithErrorStateWithExceededRetryCountDoesNotReAdd(t *testing.T) {
	setupDeadTests(t)

	mockDeadDispatcher.Mock.ExpectedCalls = []*mock.Call{} // reset calls
	mockDeadDispatcher.Mock.On("DispatchEvent", mock.Anything, mock.Anything).Return(500, fmt.Errorf("Unable to complete"))

	event := entities.Event{}
	deadLetter := &entities.DeadLetterItem{Event: event, CallbackUrl: "myurl", FailureCount: 3, NextRetryDate: time.Now()}

	deadWorker.HandleItem(deadLetter)

	mockDeadDal.Mock.AssertNumberOfCalls(t, "UpsertDeadLetterItem", 0)
}

func TestHandleItemWithErrorStateWithExceededRetryCountDeletesRegisteredEndpoint(t *testing.T) {
	setupDeadTests(t)

	mockDeadDispatcher.Mock.ExpectedCalls = []*mock.Call{} // reset calls
	mockDeadDispatcher.Mock.On("DispatchEvent", mock.Anything, mock.Anything).Return(500, fmt.Errorf("Unable to complete"))

	event := entities.Event{}
	deadLetter := &entities.DeadLetterItem{Event: event, CallbackUrl: "myurl", FailureCount: 3, NextRetryDate: time.Now()}

	deadWorker.HandleItem(deadLetter)

	mockDeadDal.Mock.AssertNumberOfCalls(t, "DeleteRegistration", 1)
	mockDeadStatsD.Mock.AssertCalled(t, "Increment", handlers.DEAD_LETTER_QUEUE+handlers.WORKER+handlers.DELETE_REGISTRATION)
}

func TestHandleItemWithUndeliverableDeletesRegisteredEndpoint(t *testing.T) {
	setupDeadTests(t)

	mockDeadDispatcher.Mock.ExpectedCalls = []*mock.Call{} // reset calls
	mockDeadDispatcher.Mock.On("DispatchEvent", mock.Anything, mock.Anything).Return(404, fmt.Errorf("Unable to complete"))

	event := entities.Event{}
	deadLetter := &entities.DeadLetterItem{Event: event, CallbackUrl: "myurl", FailureCount: 1, NextRetryDate: time.Now()}

	deadWorker.HandleItem(deadLetter)

	mockDeadDal.Mock.AssertNumberOfCalls(t, "DeleteRegistration", 1)
	mockDeadStatsD.Mock.AssertCalled(t, "Increment", handlers.DEAD_LETTER_QUEUE+handlers.WORKER+handlers.DELETE_REGISTRATION)
}
