package middleware

import (
	"fmt"
	"os/exec"
	"strings"
)

func BanIPWithFail2ban(ip string) error {
	if ip == "" || strings.Contains(ip, " ") {
		return fmt.Errorf("invalid IP address: %s", ip)
	}

	cmd := exec.Command("sudo", "fail2ban-client", "set", "nginx-bad-requests", "banip", ip)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("fail2ban command failed: %v, output: %s", err, string(output))
	}

	fmt.Printf("Successfully banned IP %s: %s\n", ip, string(output))
	return nil
}
