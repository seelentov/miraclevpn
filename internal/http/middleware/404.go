// Package middleware provides HTTP middleware for the application.
package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func NotFound() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		ctx.JSON(http.StatusNotFound, gin.H{
			"error": "Ресурс не найден",
		})
	}
}
