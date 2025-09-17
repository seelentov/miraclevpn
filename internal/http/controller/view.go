package controller

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

type ViewController struct{}

func NewViewController() *ViewController {
	return &ViewController{}
}

func (c *ViewController) GetIndex(ctx *gin.Context) {
	ctx.HTML(http.StatusOK, "index.html", nil)
}

func (c *ViewController) NotFound(ctx *gin.Context) {
	ctx.HTML(http.StatusNotFound, "404.html", nil)
}

func (c *ViewController) Panic(ctx *gin.Context, err interface{}) {
	ctx.HTML(http.StatusInternalServerError, "500.html", nil)
}
