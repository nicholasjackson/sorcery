package data

import "github.com/nicholasjackson/event-sauce/entities"

type Dal interface {
	GetRegistrationsByMessage(message string) ([]*entities.Registration, error)
	GetRegistrationByMessageAndCallback(message string, callback_url string) (*entities.Registration, error)
	UpsertRegistration(registration *entities.Registration) error
	DeleteRegistration(registration *entities.Registration) error

	UpsertEvent(event *entities.Event) error
}
