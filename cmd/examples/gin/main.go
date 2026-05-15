package main

import (
	"log"

	"github.com/gin-gonic/gin"
	"github.com/lesterkaufungen/autoupdate"
	"github.com/lesterkaufungen/autoupdate/server"
)

func main() {
	r := gin.Default()

	// Initialize with constructors
	admin := server.NewAdmin("admin", "password123")
	app := server.NewApp("My App")
	
	log.Printf("Bootstrapping App with ID: %s", app.ID)

	_, err := autoupdate.Gin(r, server.Config{
		AdminToken: "super-secret-admin-token",
		Admin:      admin,
		App:        app,
	})
	
	if err != nil {
		log.Fatalf("failed to setup autoupdate: %v", err)
	}

	log.Println("Gin server with autoupdate listening on :8081")
	r.Run(":8081")
}
