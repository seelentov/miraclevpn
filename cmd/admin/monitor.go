package main

import (
	"html/template"
	"log"
	"miraclevpn/internal/config/db"
	"miraclevpn/internal/http/controller"
	"miraclevpn/internal/repo"
	"miraclevpn/internal/services/admin"
	viewutils "miraclevpn/internal/utils/view_utils"
	"miraclevpn/pkg/ovpn"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Fatal(err)
	}

	dbUser := os.Getenv("DB_USER")
	dbHost := os.Getenv("DB_HOST")
	dbPass := os.Getenv("DB_PASSWORD")
	dbName := os.Getenv("DB_NAME")
	dbPort := os.Getenv("DB_PORT")
	dbSsl := os.Getenv("DB_SSLMODE")
	dbTZ := os.Getenv("DB_TIMEZONE")
	gormDB, err := db.NewPostgreConn(dbHost, dbUser, dbPass, dbName, dbPort, dbSsl, dbTZ, "MIIVPN_MONITOR")
	if err != nil {
		log.Fatal(err)
	}

	sshUser := os.Getenv("SSH_USER")
	sshStatusPath := os.Getenv("SSH_STATUS_PATH")
	sshCreateUserFile := os.Getenv("SSH_CREATE_USER_FILE")
	sshRevokeUserFile := os.Getenv("SSH_REVOKE_USER_FILE")
	sshConfigsDir := os.Getenv("SSH_CONFIGS_DIR")
	vpnSrv := ovpn.NewClient(sshUser, sshStatusPath, sshCreateUserFile, sshRevokeUserFile, sshConfigsDir)

	serverRepo := repo.NewServerRepository(gormDB)
	usRepo := repo.NewUserServerRepository(gormDB)

	monitorSrv := admin.NewMonitorService(
		vpnSrv,
		usRepo,
		serverRepo,
	)

	monitorCtrl := controller.NewAdminMonitorController(monitorSrv)

	r := gin.Default()
	r.SetFuncMap(template.FuncMap{
		"formatBytes": viewutils.FormatBytes,
	})

	r.LoadHTMLGlob("templates/admin/monitor/*.html")
	r.SetTrustedProxies(nil)

	r.GET("/", monitorCtrl.GetIndex)
	r.GET("/:host", monitorCtrl.GetHost)
	r.GET("/rate/:host/:ip", monitorCtrl.GetRate)

	r.Run(":" + os.Getenv("PORT_MONITOR"))
}
