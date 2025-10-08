package controller

import (
	"miraclevpn/internal/services/cookie"
	"miraclevpn/internal/services/user"
	"net/http"
	"strings"
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
	ViewBase
	RedirectTo string
}

func (c *ViewAuthController) GetLogin(ctx *gin.Context) {
	_, ok := ctx.Get("user_id")

	if ok {
		ctx.Redirect(http.StatusMovedPermanently, "/user")
	}

	redirectTo := ctx.Param("redirect_to")

	if redirectTo == "" {
		redirectTo = "/user"
	} else {
		redirectTo = strings.ReplaceAll(redirectTo, "%2F", "/")
	}

	ctx.HTML(http.StatusOK, "login.html", GetLoginViewModel{
		RedirectTo: redirectTo,
	})
}

type PostLoginFReq struct {
	Token      string `form:"token" binding:"required"`
	RedirectTo string `form:"redirect_to"`
}

func (c *ViewAuthController) PostLogin(ctx *gin.Context) {
	var req PostLoginFReq
	if err := ctx.ShouldBind(&req); err != nil {
		panic(err)
	}

	c.cookieSrv.SetAuth(ctx, req.Token)

	ctx.Redirect(http.StatusMovedPermanently, req.RedirectTo)
}

type GetLKViewModel struct {
	ViewBase
	UserID        string
	Days          int
	SubDate       time.Time
	HavePaymentID bool
}

func (c *ViewAuthController) GetLK(ctx *gin.Context) {
	userID, _ := ctx.Get("user_id")

	user, err := c.userSrv.GetUserByID(userID.(string))
	if err != nil {
		panic("user not found")
	}

	days := int(time.Until(user.ExpiredAt).Hours() / 24)

	subDate := user.ExpiredAt.Add(time.Hour * 24 * -1)

	ctx.HTML(http.StatusOK, "lk.html", GetLKViewModel{
		UserID:        user.ID,
		Days:          days,
		SubDate:       subDate,
		HavePaymentID: user.PaymentID != nil,
	})
}

func (c *ViewAuthController) PostLogout(ctx *gin.Context) {
	c.cookieSrv.RemoveAuth(ctx)
	ctx.Redirect(http.StatusMovedPermanently, "/")
}
