package middleware

import (
	"fmt"
	"os/exec"
	"strings"

	"github.com/gin-gonic/gin"
)

func ProofMiddleware(proofKeys map[string]string, banIfFail bool, debug bool) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		proofHeader := ctx.GetHeader("Mii-Vpn-Proof")
		version := ctx.GetHeader("App-Version")
		proofKey := proofKeys[version]

		if proofHeader == "" || version == "" || proofKey != proofHeader {
			ip := ctx.ClientIP()

			if banIfFail {
				if err := banIPWithFail2ban(ip); err != nil {
					panic(err)
				}
			}
			if debug {
				panic("dont have proof: " + ip + " expected " + proofKey + " but got " + proofHeader)
			} else {
				panic("dont have proof: " + proofHeader + "-" + version)
			}
		}

		ctx.Next()
	}
}

func banIPWithFail2ban(ip string) error {
	if ip == "" || strings.Contains(ip, " ") {
		return fmt.Errorf("invalid IP address: %s", ip)
	}

	cmd := exec.Command("sudo", "fail2ban-client", "set", "nginx-badrequests", "banip", ip)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("fail2ban command failed: %v, output: %s", err, string(output))
	}

	fmt.Printf("Successfully banned IP %s: %s\n", ip, string(output))
	return nil
}
