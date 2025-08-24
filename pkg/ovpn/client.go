// Package ovpn provides OpenVPN client utilities for the application.
package ovpn

import (
	"bufio"
	"crypto/rand"
	"fmt"
	"math/big"
	"miraclevpn/internal/services/vpn"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

type Client struct {
	username string

	statusPath     string
	createUserFile string
	revokeUserFile string
	userFilesDir   string
}

func NewClient(username, statusPath, createUserFile, revokeUserFile string, userFilesDir string) *Client {
	return &Client{
		username:       username,
		statusPath:     statusPath,
		createUserFile: createUserFile,
		revokeUserFile: revokeUserFile,
		userFilesDir:   userFilesDir,
	}
}

func (c *Client) GetStatus(host string) (*vpn.Status, error) {
	status := &vpn.Status{}
	if err := c.checkServerOnline(host); err != nil {
		status.Online = false
		return status, err
	}
	status.Online = true

	cmd := exec.Command(
		"ssh",
		"-o StrictHostKeyChecking=no",
		fmt.Sprintf("%s@%s", c.username, host),
		"cat "+c.statusPath,
	)

	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	clients, err := parseOpenVPNStatus(string(output))
	if err != nil {
		return nil, err
	}

	status.Clients = clients
	return status, nil
}

func (c *Client) CreateUser(host string) (config string, filename string, err error) {
	username, err := c.generateUsername(host)
	if err != nil {
		return "", "", err
	}

	cmd := exec.Command(
		"ssh",
		"-o", "StrictHostKeyChecking=no",
		fmt.Sprintf("%s@%s", c.username, host),
		"sudo", c.createUserFile, username,
	)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", "", fmt.Errorf("create user failed: %v, output: %s", err, string(output))
	}

	cmd = exec.Command(
		"ssh",
		"-o", "StrictHostKeyChecking=no",
		fmt.Sprintf("%s@%s", c.username, host),
		"cat", fmt.Sprintf("%s/%s.ovpn", c.userFilesDir, username),
	)
	output, err = cmd.CombinedOutput()
	if err != nil {
		return "", "", fmt.Errorf("failed to get ovpn file after creation: %v, output: %s", err, string(output))
	}

	return string(output), username, nil
}

func (c *Client) DeleteUser(host string, username string) error {
	cmd := exec.Command(
		"ssh",
		"-o", "StrictHostKeyChecking=no",
		fmt.Sprintf("%s@%s", c.username, host),
		"sudo", c.revokeUserFile, username,
	)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("delete user failed: %v, output: %s", err, string(output))
	}

	cmd = exec.Command(
		"ssh",
		"-o", "StrictHostKeyChecking=no",
		fmt.Sprintf("%s@%s", c.username, host),
		"sudo", "rm", c.userFilesDir+username+".ovpn",
	)
	output, err = cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("delete user failed: %v, output: %s", err, string(output))
	}

	return nil
}

func (c *Client) checkServerOnline(host string) error {
	cmd := exec.Command("ping", "-c", "1", "-W", "1", host)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("server %s is unreachable: %v\nOutput: %s", host, err, string(output))
	}

	return nil
}

func (c *Client) generateUsername(host string) (string, error) {
	cmd := exec.Command(
		"ssh",
		"-o", "StrictHostKeyChecking=no",
		fmt.Sprintf("%s@%s", c.username, host),
		"find "+c.userFilesDir+" -maxdepth 1 -type f -name \"*.ovpn\" -printf \"%f\\n\"",
	)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("server %s is unreachable: %v\nOutput: %s", host, err, string(output))
	}

	existingFiles := strings.Split(strings.TrimSpace(string(output)), ".ovpn\n")
	existingFilesMap := make(map[string]bool)
	for _, file := range existingFiles {
		if file != "" {
			nameWithoutExt := strings.TrimSuffix(file, ".ovpn")
			existingFilesMap[nameWithoutExt] = true
		}
	}

	maxAttempts := 100
	for attempt := 0; attempt < maxAttempts; attempt++ {
		name, err := generateRandomDigits(20)
		if err != nil {
			return "", fmt.Errorf("failed to generate random name: %v", err)
		}

		if !existingFilesMap[name] {
			return name, nil
		}
	}

	return "", fmt.Errorf("failed to generate unique username after %d attempts", maxAttempts)
}

func generateRandomDigits(length int) (string, error) {
	result := make([]byte, length)

	for i := 0; i < length; i++ {
		num, err := rand.Int(rand.Reader, big.NewInt(10))
		if err != nil {
			return "", err
		}
		result[i] = byte(num.Int64()) + '0'
	}

	return string(result), nil
}

func generateRandomDigitsSimple(length int) (string, error) {
	const digits = "0123456789"
	result := make([]byte, length)

	for i := 0; i < length; i++ {
		num, err := rand.Int(rand.Reader, big.NewInt(int64(len(digits))))
		if err != nil {
			return "", err
		}
		result[i] = digits[num.Int64()]
	}

	return string(result), nil
}

func parseOpenVPNStatus(output string) ([]*vpn.VpnClient, error) {
	var clients []*vpn.VpnClient
	var inClientList bool

	scanner := bufio.NewScanner(strings.NewReader(output))
	for scanner.Scan() {
		line := scanner.Text()

		if strings.HasPrefix(line, "OpenVPN CLIENT LIST") {
			inClientList = true
			continue
		}

		if strings.HasPrefix(line, "ROUTING TABLE") {
			break
		}

		if !inClientList || line == "" || strings.HasPrefix(line, "Common Name,") || strings.HasPrefix(line, "Updated,") {
			continue
		}

		parts := strings.Split(line, ",")
		if len(parts) < 5 {
			continue
		}

		bytesRecv, _ := strconv.ParseInt(parts[2], 10, 64)
		bytesSent, _ := strconv.ParseInt(parts[3], 10, 64)
		connectedSince, err := time.Parse("Mon Jan 2 15:04:05 2006", parts[4])
		if err != nil {
			continue
		}

		clients = append(clients, &vpn.VpnClient{
			CommonName:     parts[0],
			RealAddress:    parts[1],
			BytesReceived:  bytesRecv,
			BytesSent:      bytesSent,
			ConnectedSince: connectedSince,
		})
	}

	return clients, nil
}
