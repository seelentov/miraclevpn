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

func (c *Client) GetStatus(host string, port int) (*vpn.Status, error) {
	status := &vpn.Status{}
	if err := c.checkServerOnline(host, port); err != nil {
		status.Online = false
		return status, nil
	}
	status.Online = true

	// Get OpenVPN status
	cmd := exec.Command("ssh", fmt.Sprintf("%s@%s", c.username, host), "cat "+c.statusPath)

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
	cmd := exec.Command(
		"ssh",
		fmt.Sprintf("%s@%s", c.username, host),
		fmt.Sprintf("sudo %s %s", c.createUserFile, username),
	)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("create user failed: %v, output: %s", err, string(output))
	}

	i := strings.Index(string(output), `client
dev tun
proto udp`)

	output = output[i:]

	return string(output), nil
}

func (c *Client) DeleteUser(host string, username string) error {
	cmd := exec.Command(
		"ssh",
		fmt.Sprintf("%s@%s", c.username, host),
		fmt.Sprintf("sudo %s %s", c.revokeUserFile, username),
	)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("delete user failed: %v, output: %s", err, string(output))
	}
	return nil
}

func (c *Client) checkServerOnline(ip string, port int) error {
	cmd := exec.Command("nmap", "-p", strconv.Itoa(port), ip)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("nmap failed: %v", err)
	}

	if !strings.Contains(string(output), "openvpn") {
		return fmt.Errorf("port %d is not open", port)
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
