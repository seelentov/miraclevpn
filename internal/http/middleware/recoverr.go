package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func Recovery(debug bool) gin.HandlerFunc {
	return gin.RecoveryWithWriter(gin.DefaultErrorWriter, func(ctx *gin.Context, err interface{}) {
		if debug {
			ctx.JSON(http.StatusInternalServerError, gin.H{
				"error": err,
			})
		} else {
			ctx.JSON(http.StatusInternalServerError, gin.H{
				"error": "Внутренняя ошибка сервера",
			})
		}

		ctx.Abort()
	})
}
