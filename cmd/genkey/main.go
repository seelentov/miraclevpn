package main

import (
	"flag"
	"fmt"
	"log"
	"math"
	"os"
	"strconv"
	"time"

	"miraclevpn/internal/services/crypt"

	"github.com/joho/godotenv"
)

func main() {
	userID := flag.String("user-id", "", "user ID to generate token for")
	flag.Parse()

	if *userID == "" {
		fmt.Fprintln(os.Stderr, "Usage: genkey --user-id <id>")
		os.Exit(1)
	}

	if err := godotenv.Load(); err != nil {
		log.Fatal(err)
	}

	secret := os.Getenv("JWT_SECRET_AUTH")
	if secret == "" {
		log.Fatal("JWT_SECRET_AUTH not set")
	}

	jwtDuration := math.MaxInt32
	if v := os.Getenv("JWT_DURATION_MIN"); v != "" && v != "0" {
		jwtDuration, _ = strconv.Atoi(v)
	}

	jwtSrv := crypt.NewJwtService(secret, nil)
	token, err := jwtSrv.GenerateToken(map[string]string{"user_id": *userID}, time.Duration(jwtDuration)*time.Minute)
	if err != nil {
		log.Fatalf("generate token: %v", err)
	}

	fmt.Println(token)
}
