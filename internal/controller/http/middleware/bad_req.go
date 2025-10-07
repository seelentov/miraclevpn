package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

func BadRequestsMiddleware(banIfFail bool, logger *zap.Logger, badPaths []string) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		path := ctx.Request.URL.Path

		for _, badPath := range badPaths {
			if strings.Contains(path, badPath) {
				ip := ctx.GetHeader("X-Real-Ip")

				if banIfFail {
					if err := BanIPWithFail2ban(ip); err != nil {
						panic(err)
					}
				}

				logger.Warn("bad path", zap.String("path", path), zap.String("ip", ip))
				ctx.AbortWithStatus(http.StatusInternalServerError)
			}
		}

		ctx.Next()
	}
}
