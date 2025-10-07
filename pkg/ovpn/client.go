// Package ovpn provides OpenVPN client utilities for the application.
package ovpn

import (
	"crypto/rand"
	"encoding/csv"
	"fmt"
	"io"
	"math/big"
	"miraclevpn/internal/services/vpn"
	"os/exec"
	"regexp"
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
	cmd := doCmd(
		c.username, host,
		"cat "+c.statusPath,
	)

	output, err := cmd.Output()
	if err != nil {
		return status, err
	}

	clients, err := ParseOpenVPNStatus(string(output))
	if err != nil {
		return status, err
	}

	status.Online = true
	status.Clients = clients
	return status, nil
}

func (c *Client) CreateUserU(host string, username string) (config string, err error) {
	cmd := doCmd(
		c.username, host,
		"sudo", c.createUserFile, username,
	)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("create user failed: %v, output: %s", err, string(output))
	}

	cmd = doCmd(
		c.username, host,
		"cat", fmt.Sprintf("%s/%s.ovpn", c.userFilesDir, username),
	)

	output, err = cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("failed to get ovpn file after creation: %v, output: %s", err, string(output))
	}

	return string(output), nil
}

func (c *Client) CreateUser(host string) (config string, filename string, err error) {
	username, err := c.generateUsername(host)
	if err != nil {
		return "", "", err
	}

	config, err = c.CreateUserU(host, username)
	if err != nil {
		return "", "", err
	}

	return config, username, nil
}

func (c *Client) DeleteUser(host string, username string) error {
	cmd := doCmd(
		c.username, host,
		"sudo", c.revokeUserFile, username,
	)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("delete user failed: %v, output: %s", err, string(output))
	}

	cmd = doCmd(
		c.username, host,
		"sudo", "rm", c.userFilesDir+username+".ovpn",
	)
	output, err = cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("delete user failed: %v, output: %s", err, string(output))
	}

	return nil
}

func (c *Client) GetRate(host string, address string, sec int) (int64, int64, error) {
	cmd := doCmd(
		c.username, host,
		"sudo", "iftop", "-i", "tun0", "-t", "-s", strconv.Itoa(sec), "-n", "-N", "-P", "-f", "'host "+address+"'",
	)

	output, err := cmd.Output()
	if err != nil {
		return 0, 0, fmt.Errorf("get user (%s) rate failed: %v, output: %s",
			host+":"+address, err, string(output))
	}

	peakSent, peakReceived := parseIfTopOneClient(string(output))

	return peakSent, peakReceived, nil
}

func (c *Client) GetAllRate(host string, sec int) ([]*vpn.TraficStatus, error) {
	cmd := doCmd(
		c.username, host,
		"sudo", "iftop", "-i", "tun0", "-t", "-s", strconv.Itoa(sec), "-n", "-N", "-P",
	)

	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("get rate (%s) failed: %v, output: %s",
			host, err, string(output))
	}

	return parseIfTop(string(output))
}

func (c *Client) KickUser(host string, username string) error {
	cmd := doCmd(
		c.username, host,
		"echo", "kill "+username, "|", "nc", "-q", "0", "localhost", "7505",
	)

	output, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("kick user (%s) rate failed: %v, output: %s",
			username+":"+host, err, string(output))
	}

	return nil
}

func parseIfTopOneClient(input string) (int64, int64) {
	re := regexp.MustCompile(`Peak rate \(sent/received/total\):\s+(\S+)\s+(\S+)\s+(\S+)`)

	var peakSent int64 = 0
	var peakReceived int64 = 0
	lines := strings.Split(input, "\n")
	for _, line := range lines {
		if strings.Contains(line, "Peak rate (sent/received/total)") {
			matches := re.FindStringSubmatch(line)
			if matches != nil {
				peakSentStr := matches[1]
				peakReceivedStr := matches[2]

				peakSent = parseToBytes(peakSentStr)
				peakReceived = parseToBytes(peakReceivedStr)

				break
			}
		}
	}

	return peakSent, peakReceived
}

func parseIfTop(input string) ([]*vpn.TraficStatus, error) {
	panic("not implemented")
}

func parseToBytes(valueStr string) int64 {
	if valueStr == "0b" || valueStr == "0B" {
		return 0
	}

	// Регулярное выражение для разделения числа и единицы измерения
	re := regexp.MustCompile(`^([0-9.]+)([KMGT]?[bB])$`)
	matches := re.FindStringSubmatch(valueStr)
	if matches == nil {
		return 0
	}

	numberStr := matches[1]
	unit := matches[2]

	// Парсим число
	value, err := strconv.ParseFloat(numberStr, 64)
	if err != nil {
		return 0
	}

	// Преобразуем в байты в зависимости от единицы измерения
	switch strings.ToUpper(unit) {
	case "B":
		return int64(value)
	case "KB":
		return int64(value * 1024)
	case "MB":
		return int64(value * 1024 * 1024)
	case "GB":
		return int64(value * 1024 * 1024 * 1024)
	case "TB":
		return int64(value * 1024 * 1024 * 1024 * 1024)
	case "Kb":
		return int64(value * 1024 / 8)
	case "Mb":
		return int64(value * 1024 * 1024 / 8)
	case "Gb":
		return int64(value * 1024 * 1024 * 1024 / 8)
	case "Tb":
		return int64(value * 1024 * 1024 * 1024 * 1024 / 8)
	default:
		return 0
	}
}

func (c *Client) generateUsername(host string) (string, error) {
	cmd := doCmd(
		c.username, host,
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

func ParseOpenVPNStatus(statusText string) ([]*vpn.VpnClient, error) {
	var clients []*vpn.VpnClient
	var inClientSection bool

	reader := csv.NewReader(strings.NewReader(statusText))
	reader.Comma = ','
	reader.FieldsPerRecord = -1 // Allow variable number of fields

	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("error reading CSV: %v", err)
		}

		if len(record) == 0 {
			continue
		}

		// Check for section headers
		if record[0] == "HEADER" && len(record) > 1 {
			if record[1] == "CLIENT_LIST" {
				inClientSection = true
			} else {
				inClientSection = false
			}
			continue
		}

		// Process CLIENT_LIST records
		if inClientSection && record[0] == "CLIENT_LIST" {
			client, err := parseClientRecord(record)
			if err != nil {
				return nil, err
			}
			clients = append(clients, client)
		}
	}

	return clients, nil
}

func parseClientRecord(record []string) (*vpn.VpnClient, error) {
	if len(record) < 9 {
		return nil, fmt.Errorf("invalid CLIENT_LIST record: expected at least 9 fields, got %d", len(record))
	}

	// Parse bytes received and sent
	bytesReceived, err := strconv.ParseInt(record[5], 10, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid bytes received: %v", err)
	}

	bytesSent, err := strconv.ParseInt(record[6], 10, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid bytes sent: %v", err)
	}

	// Parse connected since timestamp
	connectedSince, err := time.Parse("2006-01-02 15:04:05", record[7])
	if err != nil {
		return nil, fmt.Errorf("invalid timestamp format: %v", err)
	}

	return &vpn.VpnClient{
		CommonName:     record[1],
		RealAddress:    record[2],
		VirtualAddress: record[3],
		BytesReceived:  bytesReceived,
		BytesSent:      bytesSent,
		ConnectedSince: connectedSince,
	}, nil
}

func doCmd(username, host string, command ...string) *exec.Cmd {
	args := []string{
		"-o StrictHostKeyChecking=no",
		"-o ConnectTimeout=10",
		"-o ServerAliveInterval=5",
		"-o ServerAliveCountMax=2",
		fmt.Sprintf("%s@%s", username, host),
	}
	args = append(args, command...)

	return exec.Command("ssh", args...)
}
