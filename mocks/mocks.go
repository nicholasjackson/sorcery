package mocks

import (
	"time"

	"github.com/nicholasjackson/event-sauce/entities"
	"github.com/stretchr/testify/mock"
)

type MockStatsD struct {
	mock.Mock
}

func (m *MockStatsD) Increment(label string) {
	_ = m.Mock.Called(label)
}

type MockDal struct {
	mock.Mock
	UpsertObject *entities.Registration
	DeleteObject *entities.Registration
}

func (m *MockDal) GetRegistrationsByEvent(event string) ([]*entities.Registration, error) {
	args := m.Mock.Called(event)
	if args.Get(0) != nil {
		return args.Get(0).([]*entities.Registration), args.Error(1)
	} else {
		return nil, args.Error(1)
	}
}

func (m *MockDal) GetRegistrationByEventAndCallback(event string, callback_url string) (*entities.Registration, error) {
	args := m.Mock.Called(event, callback_url)
	if args.Get(0) != nil {
		return args.Get(0).(*entities.Registration), args.Error(1)
	} else {
		return nil, args.Error(1)
	}
}

func (m *MockDal) UpsertRegistration(registration *entities.Registration) error {
	args := m.Mock.Called(registration)
	m.UpsertObject = registration
	return args.Error(0)
}

func (m *MockDal) DeleteRegistration(registration *entities.Registration) error {
	args := m.Mock.Called(registration)
	m.DeleteObject = registration
	return args.Error(0)
}

func (m *MockDal) UpsertEvent(event *entities.Event) error {
	args := m.Mock.Called(event)

	return args.Error(0)
}

type MockQueue struct {
	mock.Mock
	ConsumerCallback func(event *entities.Event)
}

func (m *MockQueue) Add(event_name string, payload string) error {
	args := m.Mock.Called(event_name, payload)
	return args.Error(0)
}

func (m *MockQueue) AddEvent(event *entities.Event) error {
	args := m.Mock.Called(event)
	return args.Error(0)
}

func (m *MockQueue) StartConsuming(size int, poll_interval time.Duration, callback func(event *entities.Event)) {
	m.ConsumerCallback = callback
	_ = m.Mock.Called(size, poll_interval, callback)
}

type MockWorker struct {
	mock.Mock
}

func (m *MockWorker) HandleEvent(event *entities.Event) error {
	args := m.Mock.Called(event)
	return args.Error(0)
}

type MockEventDispatcher struct {
	mock.Mock
}

func (m *MockEventDispatcher) DispatchEvent(event *entities.Event, endpoint string) (int, error) {
	args := m.Mock.Called(event, endpoint)
	return args.Int(0), args.Error(1)
}
