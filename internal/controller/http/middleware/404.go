// Package middleware provides HTTP middleware for the application.
package middleware

import (
	"github.com/gin-gonic/gin"
)

func NotFound() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		panic("not found")
	}
}
