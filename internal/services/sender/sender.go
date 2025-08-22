// Package sender provides messaging services for the application.
package sender

type Sender interface {
	SendMessage(to string, message string) error
	GetName() string
	GetStatus() (bool, error)
}
