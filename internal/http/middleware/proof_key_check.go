package middleware

import (
	"fmt"
	"os/exec"
	"strings"

	"github.com/gin-gonic/gin"
)

func ProofMiddleware(proofKeys map[string]string) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		proofHeader := ctx.GetHeader("MII_VPN_PROOF")
		version := ctx.GetHeader("APP_VERSION")
		proofKey := proofKeys[version]

		if proofKey != proofHeader {
			ip := ctx.ClientIP()
			if err := banIPWithFail2ban(ip); err != nil {
				panic(err)
			}
			panic("dont have proof: expected " + proofHeader[:5] + "***" + proofHeader[len(proofHeader):] + " but got " + proofHeader[:5] + "***" + proofHeader[len(proofHeader):])
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
