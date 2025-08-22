// Package daemon provides system daemon management for the application.
package daemon

type Daemon interface {
	Start()
	Stop()
}
