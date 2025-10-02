package viewutils

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func RenderHtml(ctx *gin.Context, template string, obj any) {
	ctx.HTML(http.StatusOK, template, obj)
}
