package main

import (
	"log"
	"miraclevpn/internal/http/controller"
	"os"
	"path/filepath"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Fatal(err)
	}

	viewCtrl := controller.NewViewController()

	r := gin.Default()
	r.LoadHTMLGlob("templates/*.html")
	r.SetTrustedProxies([]string{"127.0.0.1", "::1"})
	r.NoRoute(viewCtrl.NotFound)
	r.Use(gin.RecoveryWithWriter(gin.DefaultErrorWriter, viewCtrl.Panic))

	r.GET("/", viewCtrl.GetIndex)

	if err := setupStatic(r); err != nil {
		log.Fatalf("Error registering static files: %v", err)
	}

	r.Run(":" + os.Getenv("PORT_FRONTEND"))
}

func setupStatic(r *gin.Engine) error {
	publicDir := "./public"

	var walkDir func(string) error
	walkDir = func(path string) error {
		entries, err := os.ReadDir(path)
		if err != nil {
			return err
		}

		for _, entry := range entries {
			fullPath := filepath.Join(path, entry.Name())
			relativePath, err := filepath.Rel(publicDir, fullPath)
			if err != nil {
				return err
			}

			webPath := "/" + relativePath

			if entry.IsDir() {
				log.Printf("Registering directory: %s -> %s", webPath, fullPath)
				r.Static(webPath, fullPath)

				if err := walkDir(fullPath); err != nil {
					return err
				}
			} else {
				log.Printf("Registering file: %s -> %s", webPath, fullPath)
				r.StaticFile(webPath, fullPath)
			}
		}
		return nil
	}

	return walkDir(publicDir)
}
