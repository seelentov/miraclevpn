package middleware

import (
	"fmt"
	"os/exec"
	"strings"

	"github.com/gin-gonic/gin"
)

func ProofMiddleware(proofKey string) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		proofHeader := ctx.GetHeader("MII_VPN_PROOF")
		if proofKey != proofHeader {
			ip := ctx.ClientIP()
			if err := banIPWithFail2ban(ip); err != nil {
				panic(err)
			}
			panic("dont have proof")
		}
		ctx.Next()
	}
}

func banIPWithFail2ban(ip string) error {
	if ip == "" || strings.Contains(ip, " ") {
		return fmt.Errorf("invalid IP address: %s", ip)
	}

	cmd := exec.Command("sudo", "fail2ban-client", "set", "nginx-badrequests", "banip", ip)

	// Выполняем команду
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("fail2ban command failed: %v, output: %s", err, string(output))
	}

	fmt.Printf("Successfully banned IP %s: %s\n", ip, string(output))
	return nil
}
