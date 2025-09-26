package main

import (
	"fmt"
	"log"
	"miraclevpn/internal/config/db"
	"miraclevpn/internal/repo"
	"miraclevpn/pkg/ovpn"
	"os"
	"time"

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

	srvs, err := serverRepo.FindAll()
	if err != nil {
		log.Fatal(err)
	}

	for {
		for _, s := range srvs {
			status, err := vpnSrv.GetStatus(s.Host)
			if err != nil {
				fmt.Println(s.Host, "err: ", err)
			} else {
				fmt.Println(s.Host, len(status.Clients))
				for _, c := range status.Clients {
					us, err := usRepo.FindByConfigFile(c.CommonName, true)

					if err != nil || us.UserID == "" {
						us.UserID = "nil"
					}

					fmt.Printf("CommonName:%s,UserID:%s,RealAddress:%s,BytesReceived:%d,BytesSent:%d,ConnectedSince:%s\n", c.CommonName, us.UserID, c.RealAddress, c.BytesReceived, c.BytesSent, c.ConnectedSince)
				}
			}

			fmt.Println()
		}

		time.Sleep(1 * time.Second)
	}
}
