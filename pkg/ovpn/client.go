// Package ovpn provides OpenVPN client utilities for the application.
package ovpn

import (
	"bufio"
	"crypto/rand"
	"encoding/csv"
	"fmt"
	"io"
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

func (c *Client) CreateUser(host string) (config string, filename string, err error) {
	username, err := c.generateUsername(host)
	if err != nil {
		return "", "", err
	}

	cmd := doCmd(
		c.username, host,
		"sudo", c.createUserFile, username,
	)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", "", fmt.Errorf("create user failed: %v, output: %s", err, string(output))
	}

	cmd = doCmd(
		c.username, host,
		"cat", fmt.Sprintf("%s/%s.ovpn", c.userFilesDir, username),
	)

	output, err = cmd.CombinedOutput()
	if err != nil {
		return "", "", fmt.Errorf("failed to get ovpn file after creation: %v, output: %s", err, string(output))
	}

	return string(output), username, nil
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

func (c *Client) GetTraficRates(host string, seconds int) (*vpn.TraficStatus, error) {
	cmd := doCmd(
		c.username, host,
		"sudo", "iftop", "-i", "tun0", "-t", "-s", strconv.Itoa(seconds), "-n", "-N", "-P",
	)

	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	return parseTrafficOutput(string(output))
}

func parseTrafficOutput(output string) (*vpn.TraficStatus, error) {
	scanner := bufio.NewScanner(strings.NewReader(output))
	status := &vpn.TraficStatus{
		Rates: make([]*vpn.TraficRate, 0),
	}

	var inTable bool
	var currentHost string

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// Пропускаем пустые строки и разделители
		if line == "" || strings.HasPrefix(line, "===") || strings.HasPrefix(line, "---") {
			continue
		}

		// Проверяем начало таблицы
		if strings.Contains(line, "Host name") && strings.Contains(line, "last 10s") {
			inTable = true
			continue
		}

		// Парсим строки таблицы
		if inTable {
			// Если строка содержит "=>" - это исходящий трафик
			if strings.Contains(line, "=>") {
				parts := strings.Fields(line)
				if len(parts) >= 5 {
					// Ищем виртуальный адрес вида 10.8.0.x
					if strings.HasPrefix(parts[0], "10.8.0.") {
						currentHost = strings.Split(parts[0], ":")[0]

						// Парсим трафик за последние 10 секунд
						rateStr := parts[3]
						rate, err := parseRate(rateStr)
						if err == nil {
							status.Rates = append(status.Rates, &vpn.TraficRate{
								VirtualAddress: currentHost,
								Rate:           rate,
							})
						}
					}
				}
			}
			continue
		}

		// Парсим итоговые показатели
		if strings.HasPrefix(line, "Total send rate:") {
			parts := strings.Fields(line)
			if len(parts) >= 7 {
				status.TotalSendRate, _ = parseRate(parts[6]) // last 10s
			}
			continue
		}

		if strings.HasPrefix(line, "Total receive rate:") {
			parts := strings.Fields(line)
			if len(parts) >= 7 {
				status.TotalReceiveRate, _ = parseRate(parts[6]) // last 10s
			}
			continue
		}

		if strings.HasPrefix(line, "Peak rate (sent/received/total):") {
			parts := strings.Fields(line)
			if len(parts) >= 7 {
				status.PeakSendRate, _ = parseRate(parts[5])
				status.PeakReceiveRate, _ = parseRate(parts[6])
				status.PeakRate, _ = parseRate(parts[7])
			}
			continue
		}

		if strings.HasPrefix(line, "Cumulative (sent/received/total):") {
			parts := strings.Fields(line)
			if len(parts) >= 7 {
				status.CumulativeSendRate, _ = parseBytes(parts[5])
				status.CumulativeReceiveRate, _ = parseBytes(parts[6])
				status.CumulativeRate, _ = parseBytes(parts[7])
			}
			continue
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return status, nil
}

// parseRate парсит строку с rate (например, "1.14Mb", "33.5Kb")
func parseRate(rateStr string) (int64, error) {
	rateStr = strings.TrimSpace(rateStr)

	// Если 0b, возвращаем 0
	if rateStr == "0b" {
		return 0, nil
	}

	multiplier := int64(1)
	if strings.HasSuffix(rateStr, "Kb") {
		multiplier = 1024
		rateStr = strings.TrimSuffix(rateStr, "Kb")
	} else if strings.HasSuffix(rateStr, "Mb") {
		multiplier = 1024 * 1024
		rateStr = strings.TrimSuffix(rateStr, "Mb")
	} else if strings.HasSuffix(rateStr, "Gb") {
		multiplier = 1024 * 1024 * 1024
		rateStr = strings.TrimSuffix(rateStr, "Gb")
	} else if strings.HasSuffix(rateStr, "b") {
		rateStr = strings.TrimSuffix(rateStr, "b")
	}

	rate, err := strconv.ParseFloat(rateStr, 64)
	if err != nil {
		return 0, err
	}

	return int64(rate * float64(multiplier)), nil
}

// parseBytes парсит строку с байтами (например, "1.42MB", "41.9KB")
func parseBytes(bytesStr string) (int64, error) {
	bytesStr = strings.TrimSpace(bytesStr)

	// Если 0B, возвращаем 0
	if bytesStr == "0B" {
		return 0, nil
	}

	multiplier := int64(1)
	if strings.HasSuffix(bytesStr, "KB") {
		multiplier = 1024
		bytesStr = strings.TrimSuffix(bytesStr, "KB")
	} else if strings.HasSuffix(bytesStr, "MB") {
		multiplier = 1024 * 1024
		bytesStr = strings.TrimSuffix(bytesStr, "MB")
	} else if strings.HasSuffix(bytesStr, "GB") {
		multiplier = 1024 * 1024 * 1024
		bytesStr = strings.TrimSuffix(bytesStr, "GB")
	} else if strings.HasSuffix(bytesStr, "B") {
		bytesStr = strings.TrimSuffix(bytesStr, "B")
	}

	bytes, err := strconv.ParseFloat(bytesStr, 64)
	if err != nil {
		return 0, err
	}

	return int64(bytes * float64(multiplier)), nil
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
