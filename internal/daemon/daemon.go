package daemon

type Daemon interface {
	Start()
	Stop()
}
