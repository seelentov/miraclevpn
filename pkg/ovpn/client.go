package ovpn

import (
	"bufio"
	"fmt"
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
}

func NewClient(username, statusPath, createUserFile, revokeUserFile string) *Client {
	return &Client{
		username:       username,
		statusPath:     statusPath,
		createUserFile: createUserFile,
		revokeUserFile: revokeUserFile,
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

func (c *Client) CreateUser(host string, username string) (string, error) {
	username = strings.ReplaceAll(username, "+", "_")

	cmd := exec.Command(
		"ssh",
		"-o", "StrictHostKeyChecking=no",
		fmt.Sprintf("%s@%s", c.username, host),
		"cat", fmt.Sprintf("/etc/openvpn/server/easy-rsa/%s.ovpn", username),
	)
	output, err := cmd.CombinedOutput()
	if err != nil {
		cmd := exec.Command(
			"ssh",
			"-o", "StrictHostKeyChecking=no",
			fmt.Sprintf("%s@%s", c.username, host),
			"sudo", c.createUserFile, username,
		)
		output, err := cmd.CombinedOutput()
		if err != nil {
			return "", fmt.Errorf("create user failed: %v, output: %s", err, string(output))
		}

		cmd = exec.Command(
			"ssh",
			"-o", "StrictHostKeyChecking=no",
			fmt.Sprintf("%s@%s", c.username, host),
			"cat", fmt.Sprintf("/etc/openvpn/server/easy-rsa/%s.ovpn", username),
		)
		output, err = cmd.CombinedOutput()
		if err != nil {
			return "", fmt.Errorf("failed to get ovpn file after creation: %v, output: %s", err, string(output))
		}

		return string(output), nil
	}

	return string(output), nil
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
