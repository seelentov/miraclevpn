package ovpn_test

import (
	. "miraclevpn/pkg/ovpn"
	"os"
	"testing"
)

var (
	client *Client

	testHost = "5.129.232.61"
)

func TestMain(m *testing.M) {
	setup()
	c := m.Run()
	teardown()
	os.Exit(c)
}

func setup() {
	sshUser := os.Getenv("SSH_USER")
	sshStatusPath := os.Getenv("SSH_STATUS_PATH")
	sshCreateUserFile := os.Getenv("SSH_CREATE_USER_FILE")
	sshRevokeUserFile := os.Getenv("SSH_REVOKE_USER_FILE")
	sshConfigsDir := os.Getenv("SSH_CONFIGS_DIR")

	client = NewClient(sshUser, sshStatusPath, sshCreateUserFile, sshRevokeUserFile, sshConfigsDir)
}

func teardown() {
	client = nil
}
