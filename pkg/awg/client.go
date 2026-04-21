// Package awg provides AmneziaWG client utilities for the application.
package awg

import (
	"crypto/rand"
	"fmt"
	"math/big"
	"miraclevpn/internal/services/vpn"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

// Client manages AmneziaWG VPN users via SSH.
// It uses wg-manage.sh on the remote server for user lifecycle operations.
type Client struct {
	username     string // SSH user
	manageScript string // path to wg-manage.sh on remote, e.g. /usr/local/bin/wg-manage.sh
	clientsDir   string // directory with per-client .conf files, e.g. /etc/wireguard/clients
	wg0ConfPath  string // path to server wg0.conf, e.g. /etc/amnezia/amneziawg/wg0.conf
}

func NewClient(username, manageScript, clientsDir string) *Client {
	return &Client{
		username:     username,
		manageScript: manageScript,
		clientsDir:   clientsDir,
		wg0ConfPath:  "/etc/amnezia/amneziawg/wg0.conf",
	}
}

// GetStatus returns current interface status and connected peers.
// A peer is considered "connected" if its latest handshake was within the last 3 minutes.
func (c *Client) GetStatus(host string) (*vpn.Status, error) {
	status := &vpn.Status{}

	// awg show wg0 dump: first line = interface, subsequent lines = peers
	// peer line: pubkey preshared endpoint allowed-ips handshake-unix rx tx keepalive
	cmd := doCmd(c.username, host, "sudo", "awg", "show", "wg0", "dump")
	output, err := cmd.Output()
	if err != nil {
		return status, fmt.Errorf("awg show dump on %s failed: %v", host, err)
	}

	pubkeyToName, _ := c.getPubkeyNameMap(host) // non-fatal if fails

	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	now := time.Now().Unix()
	const connectedThreshold = int64(180) // 3 minutes

	status.Online = true
	for i, line := range lines {
		if i == 0 || line == "" {
			continue // skip interface line
		}
		fields := strings.Fields(line)
		if len(fields) < 8 {
			continue
		}
		pubkey := fields[0]
		allowedIPs := fields[3]
		handshake, _ := strconv.ParseInt(fields[4], 10, 64)
		rx, _ := strconv.ParseInt(fields[5], 10, 64)
		tx, _ := strconv.ParseInt(fields[6], 10, 64)

		if handshake == 0 || (now-handshake) > connectedThreshold {
			continue
		}

		name := pubkeyToName[pubkey]
		if name == "" {
			name = pubkey
		}

		ip := strings.Split(allowedIPs, "/")[0]
		status.Clients = append(status.Clients, &vpn.VpnClient{
			CommonName:     name,
			VirtualAddress: ip,
			BytesReceived:  rx,
			BytesSent:      tx,
			ConnectedSince: time.Unix(handshake, 0),
		})
	}

	return status, nil
}

// CreateUser generates a new VPN user and returns its config and username.
func (c *Client) CreateUser(host string) (config string, filename string, err error) {
	username, err := c.generateUsername(host)
	if err != nil {
		return "", "", err
	}

	cmd := doCmd(c.username, host, "sudo", c.manageScript, "add", username)
	output, err := cmd.Output()
	if err != nil {
		return "", "", fmt.Errorf("awg add user %s on %s failed: %v\noutput: %s", username, host, err, output)
	}

	// wg-manage.sh add outputs header lines then the config starting from [Interface]
	config = extractConfig(string(output))
	if config == "" {
		return "", "", fmt.Errorf("awg add: could not parse config from output: %s", string(output))
	}

	return config, username, nil
}

// DeleteUser removes a VPN user from the server.
func (c *Client) DeleteUser(host string, username string) error {
	cmd := doCmd(c.username, host, "sudo", c.manageScript, "remove", username)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("awg remove user %s on %s failed: %v\noutput: %s", username, host, err, output)
	}
	return nil
}

// KickUser drops an active WireGuard session by removing and re-adding the peer.
// This forces a reconnect without permanently removing the user.
func (c *Client) KickUser(host string, username string) error {
	pubkey, ip, err := c.getClientPubkeyAndIP(host, username)
	if err != nil {
		return fmt.Errorf("kick user %s on %s: %v", username, host, err)
	}

	cmd := doCmd(c.username, host, "sudo", "awg", "set", "wg0", "peer", pubkey, "remove")
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("awg kick remove peer %s on %s: %v\noutput: %s", username, host, err, out)
	}

	// Re-add so the user can reconnect (drops current session)
	cmd = doCmd(c.username, host, "sudo", "awg", "set", "wg0", "peer", pubkey, "allowed-ips", ip+"/32")
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("awg kick re-add peer %s on %s: %v\noutput: %s", username, host, err, out)
	}

	return nil
}

// CheckAvailable tests SSH reachability and outbound internet access via curl.
func (c *Client) CheckAvailable(host string) (bool, error) {
	cmd := doCmd(c.username, host,
		"curl", "-s", "-o", "/dev/null", "-w", "%{http_code}",
		"--connect-timeout", "5", "-m", "10", "https://google.com",
	)
	output, err := cmd.Output()
	if err != nil {
		return false, nil
	}
	code := strings.TrimSpace(string(output))
	return code != "" && code != "000", nil
}

// GetAllRate returns per-user transfer rates by sampling awg dump twice over sec seconds.
func (c *Client) GetAllRate(host string, sec int) ([]*vpn.TraficStatus, error) {
	snap1, err := c.dumpTransfer(host)
	if err != nil {
		return nil, err
	}

	sleep := time.Duration(sec) * time.Second
	if sleep < time.Second {
		sleep = time.Second
	}
	time.Sleep(sleep)

	snap2, err := c.dumpTransfer(host)
	if err != nil {
		return nil, err
	}

	pubkeyToName, _ := c.getPubkeyNameMap(host)

	var result []*vpn.TraficStatus
	for pubkey, t2 := range snap2 {
		t1, ok := snap1[pubkey]
		if !ok {
			continue
		}
		rxDelta := t2[0] - t1[0]
		txDelta := t2[1] - t1[1]
		if rxDelta <= 0 && txDelta <= 0 {
			continue
		}

		name := pubkeyToName[pubkey]
		if name == "" {
			name = pubkey
		}

		result = append(result, &vpn.TraficStatus{
			ClientName:    name,
			BytesReceived: rxDelta,
			BytesSend:     txDelta,
		})
	}

	return result, nil
}

// GetRate returns transfer rate for a specific VPN IP address.
func (c *Client) GetRate(host string, address string, sec int) (int64, int64, error) {
	// Build IP → pubkey map from wg0.conf, then measure rate for that pubkey.
	peers, err := c.parseWg0Conf(host)
	if err != nil {
		return 0, 0, err
	}

	var targetPubkey string
	for _, p := range peers {
		if p.ip == address {
			targetPubkey = p.pubkey
			break
		}
	}
	if targetPubkey == "" {
		return 0, 0, nil
	}

	snap1, err := c.dumpTransfer(host)
	if err != nil {
		return 0, 0, err
	}

	sleep := time.Duration(sec) * time.Second
	if sleep < time.Second {
		sleep = time.Second
	}
	time.Sleep(sleep)

	snap2, err := c.dumpTransfer(host)
	if err != nil {
		return 0, 0, err
	}

	if t2, ok := snap2[targetPubkey]; ok {
		if t1, ok := snap1[targetPubkey]; ok {
			return t2[1] - t1[1], t2[0] - t1[0], nil // tx=sent, rx=received
		}
	}

	return 0, 0, nil
}

// peerInfo holds parsed peer data from wg0.conf.
type peerInfo struct {
	name   string
	pubkey string
	ip     string
}

// parseWg0Conf reads the server wg0.conf and returns peer metadata.
// wg0.conf format (added by wg-manage.sh):
//
//	[Peer]
//	# <name>
//	PublicKey = <pubkey>
//	AllowedIPs = <ip>/32
func (c *Client) parseWg0Conf(host string) ([]peerInfo, error) {
	cmd := doCmd(c.username, host, "sudo", "cat", c.wg0ConfPath)
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("cat wg0.conf on %s: %v", host, err)
	}

	var peers []peerInfo
	var cur peerInfo
	inPeer := false

	for _, line := range strings.Split(string(output), "\n") {
		line = strings.TrimSpace(line)
		if line == "[Peer]" {
			if inPeer && cur.pubkey != "" {
				peers = append(peers, cur)
			}
			cur = peerInfo{}
			inPeer = true
			continue
		}
		if strings.HasPrefix(line, "[") && line != "[Peer]" {
			if inPeer && cur.pubkey != "" {
				peers = append(peers, cur)
			}
			inPeer = false
			continue
		}
		if !inPeer {
			continue
		}
		if strings.HasPrefix(line, "# ") {
			cur.name = strings.TrimPrefix(line, "# ")
		} else if strings.HasPrefix(line, "PublicKey") {
			parts := strings.SplitN(line, "=", 2)
			if len(parts) == 2 {
				cur.pubkey = strings.TrimSpace(parts[1])
			}
		} else if strings.HasPrefix(line, "AllowedIPs") {
			parts := strings.SplitN(line, "=", 2)
			if len(parts) == 2 {
				cur.ip = strings.TrimSpace(strings.Split(parts[1], "/")[0])
			}
		}
	}
	if inPeer && cur.pubkey != "" {
		peers = append(peers, cur)
	}

	return peers, nil
}

// getPubkeyNameMap returns pubkey → client name by parsing wg0.conf.
func (c *Client) getPubkeyNameMap(host string) (map[string]string, error) {
	peers, err := c.parseWg0Conf(host)
	if err != nil {
		return nil, err
	}
	m := make(map[string]string, len(peers))
	for _, p := range peers {
		if p.pubkey != "" && p.name != "" {
			m[p.pubkey] = p.name
		}
	}
	return m, nil
}

// getClientPubkeyAndIP returns the pubkey and VPN IP for a named client by parsing wg0.conf.
func (c *Client) getClientPubkeyAndIP(host, username string) (pubkey, ip string, err error) {
	peers, err := c.parseWg0Conf(host)
	if err != nil {
		return "", "", err
	}
	for _, p := range peers {
		if p.name == username {
			return p.pubkey, p.ip, nil
		}
	}
	return "", "", fmt.Errorf("peer %q not found in wg0.conf", username)
}

// dumpTransfer runs "awg show wg0 dump" and returns pubkey → [rx, tx] cumulative bytes.
func (c *Client) dumpTransfer(host string) (map[string][2]int64, error) {
	cmd := doCmd(c.username, host, "sudo", "awg", "show", "wg0", "dump")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("awg dump on %s: %v", host, err)
	}

	result := map[string][2]int64{}
	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	for i, line := range lines {
		if i == 0 || line == "" {
			continue // skip interface line
		}
		fields := strings.Fields(line)
		if len(fields) < 8 {
			continue
		}
		pubkey := fields[0]
		rx, _ := strconv.ParseInt(fields[5], 10, 64)
		tx, _ := strconv.ParseInt(fields[6], 10, 64)
		result[pubkey] = [2]int64{rx, tx}
	}
	return result, nil
}

// generateUsername picks a random 20-digit name not already in use on the server.
func (c *Client) generateUsername(host string) (string, error) {
	cmd := doCmd(c.username, host, "sudo", "ls", c.clientsDir)
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("server %s is unreachable: %v", host, err)
	}

	existing := map[string]bool{}
	for _, line := range strings.Split(strings.TrimSpace(string(output)), "\n") {
		name := strings.TrimSuffix(strings.TrimSpace(line), ".conf")
		if name != "" {
			existing[name] = true
		}
	}

	for range 100 {
		name, err := generateRandomDigits(20)
		if err != nil {
			return "", fmt.Errorf("random name gen: %v", err)
		}
		if !existing[name] {
			return name, nil
		}
	}

	return "", fmt.Errorf("failed to generate unique username after 100 attempts")
}

func generateRandomDigits(length int) (string, error) {
	result := make([]byte, length)
	for i := range length {
		num, err := rand.Int(rand.Reader, big.NewInt(10))
		if err != nil {
			return "", err
		}
		result[i] = byte(num.Int64()) + '0'
	}
	return string(result), nil
}

// extractConfig pulls the WireGuard client config block from wg-manage.sh add output.
func extractConfig(output string) string {
	idx := strings.Index(output, "[Interface]")
	if idx < 0 {
		return ""
	}
	return strings.TrimSpace(output[idx:])
}

func doCmd(username, host string, command ...string) *exec.Cmd {
	args := []string{
		"-o", "StrictHostKeyChecking=no",
		"-o", "ConnectTimeout=10",
		"-o", "ServerAliveInterval=5",
		"-o", "ServerAliveCountMax=2",
		fmt.Sprintf("%s@%s", username, host),
	}
	args = append(args, command...)
	return exec.Command("ssh", args...)
}
