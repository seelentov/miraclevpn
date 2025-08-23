package middleware

import (
	"encoding/json"
	"fmt"
	"miraclevpn/internal/services/sender"
	"miraclevpn/internal/utils"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

func Recovery(debug bool, sender sender.Sender, adminTo string, logger *zap.Logger) gin.HandlerFunc {
	return gin.RecoveryWithWriter(gin.DefaultErrorWriter, func(ctx *gin.Context, err interface{}) {
		requestInfo := collectRequestInfo(ctx)

		errField := err
		er, ok := err.(error)
		if ok {
			errField = utils.GetStackTrace(er)
		}

		errorMessage := fmt.Sprintf("Panic recovered: %v", errField)
		fullErrorMessage := fmt.Sprintf("%s\n\nRequest details:\n%s", errorMessage, requestInfo)

		logger.Error("Panic recovered",
			zap.Any("error", errField),
			zap.String("path", ctx.Request.URL.Path),
			zap.String("method", ctx.Request.Method),
			zap.Any("query_params", ctx.Request.URL.Query()),
			zap.Any("headers", getHeaders(ctx.Request.Header)),
		)

		if debug {
			ctx.JSON(http.StatusInternalServerError, gin.H{
				"error":   er,
				"details": requestInfo,
			})
		} else {
			ctx.JSON(http.StatusInternalServerError, gin.H{
				"error": "Внутренняя ошибка сервера",
			})
			sender.SendMessage(adminTo, fullErrorMessage)
		}

		ctx.Abort()
	})
}

func collectRequestInfo(ctx *gin.Context) string {
	var info strings.Builder

	info.WriteString(fmt.Sprintf("Method: %s\n", ctx.Request.Method))
	info.WriteString(fmt.Sprintf("Path: %s\n", ctx.Request.URL.Path))
	info.WriteString(fmt.Sprintf("Full URL: %s\n", ctx.Request.URL.String()))

	if len(ctx.Request.URL.Query()) > 0 {
		queryJSON, _ := json.Marshal(ctx.Request.URL.Query())
		info.WriteString(fmt.Sprintf("Query Params: %s\n", string(queryJSON)))
	} else {
		info.WriteString("Query Params: none\n")
	}

	info.WriteString("Headers:\n")
	for name, values := range ctx.Request.Header {
		if isSensitiveHeader(name) {
			info.WriteString(fmt.Sprintf("  %s: [REDACTED]\n", name))
		} else {
			info.WriteString(fmt.Sprintf("  %s: %s\n", name, strings.Join(values, ", ")))
		}
	}

	if ctx.Request.Body != nil && ctx.Request.ContentLength > 0 && ctx.Request.ContentLength < 1024 {
		info.WriteString(fmt.Sprintf("Body: present (%d bytes)\n", ctx.Request.ContentLength))
	}

	return info.String()
}

func getHeaders(header http.Header) map[string]string {
	headers := make(map[string]string)
	for name, values := range header {
		if isSensitiveHeader(name) {
			headers[name] = "[REDACTED]"
		} else {
			headers[name] = strings.Join(values, ", ")
		}
	}
	return headers
}

func isSensitiveHeader(name string) bool {
	sensitiveHeaders := map[string]bool{
		"Authorization":  true,
		"Cookie":         true,
		"Set-Cookie":     true,
		"X-Api-Key":      true,
		"X-Token":        true,
		"X-Access-Token": true,
		"X-Secret":       true,
	}
	return sensitiveHeaders[name] ||
		strings.Contains(strings.ToLower(name), "token") ||
		strings.Contains(strings.ToLower(name), "secret") ||
		strings.Contains(strings.ToLower(name), "key") ||
		strings.Contains(strings.ToLower(name), "password")
}
