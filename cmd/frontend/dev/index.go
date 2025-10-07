package main

import (
	"log"
	"miraclevpn/internal/controller/http/controller"
	"os"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Fatal(err)
	}

	viewCtrl := controller.NewViewIndexController()

	r := gin.Default()
	r.LoadHTMLGlob("./templates/*.html")
	r.SetTrustedProxies(nil)

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

	var files []string
	var dirs []string

	err := filepath.Walk(publicDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		relativePath, err := filepath.Rel(publicDir, path)
		if err != nil {
			return err
		}

		if relativePath == "." {
			return nil
		}

		webPath := "/" + filepath.ToSlash(relativePath)

		if info.IsDir() {
			dirs = append(dirs, webPath)
		} else {
			files = append(files, webPath)
		}
		return nil
	})
	if err != nil {
		return err
	}

	for _, dir := range dirs {
		hasFilesInDir := false
		for _, file := range files {
			if strings.HasPrefix(file, dir+"/") {
				hasFilesInDir = true
				break
			}
		}

		if hasFilesInDir {
			log.Printf("Registering directory: %s -> %s%s", dir, publicDir, dir)
			r.Static(dir, publicDir+dir)
		}
	}

	for _, file := range files {
		inRegisteredDir := false
		for _, dir := range dirs {
			if strings.HasPrefix(file, dir+"/") {
				inRegisteredDir = true
				break
			}
		}

		if !inRegisteredDir {
			log.Printf("Registering file: %s -> %s%s", file, publicDir, file)
			r.StaticFile(file, publicDir+file)
		}
	}

	return nil
}
