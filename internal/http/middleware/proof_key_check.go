package middleware

import (
	"github.com/gin-gonic/gin"
)

func ProofMiddleware(proofKeys map[string]string, banIfFail bool, debug bool) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		proofHeader := ctx.GetHeader("Mii-Vpn-Proof")
		version := ctx.GetHeader("App-Version")
		proofKey := proofKeys[version]

		if proofHeader == "" || version == "" || proofKey != proofHeader {
			ip := ctx.GetHeader("X-Real-Ip")

			if banIfFail {
				if err := BanIPWithFail2ban(ip); err != nil {
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
