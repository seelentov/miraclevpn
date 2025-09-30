package controller

import (
	"miraclevpn/internal/services/cookie"
	"miraclevpn/internal/services/user"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

type ViewAuthController struct {
	cookieSrv *cookie.CookieService
	userSrv   *user.UserService
}

func NewViewAuthController(cookieSrv *cookie.CookieService, userSrv *user.UserService) *ViewAuthController {
	return &ViewAuthController{cookieSrv, userSrv}
}

type GetLoginViewModel struct {
	RedirectTo string
}

func (c *ViewAuthController) GetLogin(ctx *gin.Context) {
	redirectTo := ctx.Param("redirect_to")

	if redirectTo == "" {
		redirectTo = "/lk"
	}

	ctx.HTML(http.StatusOK, "login.html", GetLoginViewModel{redirectTo})
}

type PostLoginFReq struct {
	Token      string `form:"token" binding:"required"`
	RedirectTo string `form:"redirect_to" binding:"required"`
}

func (c *ViewAuthController) PostLogin(ctx *gin.Context) {
	var req PostLoginFReq
	if err := ctx.ShouldBindJSON(&req); err != nil {
		panic(err)
	}

	c.cookieSrv.SetAuth(ctx, req.Token)

	ctx.Redirect(http.StatusOK, req.RedirectTo)
}

type GetLKViewModel struct {
	UserID  string
	Days    int
	SubDate *time.Time
}

func (c *ViewAuthController) GetLK(ctx *gin.Context) {
	userID, _ := ctx.Get("user_id")

	user, err := c.userSrv.GetUserByID(userID.(string))
	if err != nil {
		panic("user not found")
	}

	days := int(time.Until(user.ExpiredAt).Hours() / 24)

	var subDate *time.Time
	if user.PaymentID == nil {
		date := user.ExpiredAt.Add(time.Hour * 24 * -1)
		subDate = &date
	}

	ctx.HTML(http.StatusOK, "lk.html", GetLKViewModel{
		UserID:  user.ID,
		Days:    days,
		SubDate: subDate,
	})
}
