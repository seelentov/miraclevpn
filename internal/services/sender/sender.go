package sender

type Sender interface {
	SendMessage(to string, message string) error
	GetName() string
	GetStatus() (bool, error)
}
