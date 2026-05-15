package main

import (
	"log"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/lesterkaufungen/autoupdate"
	"github.com/lesterkaufungen/autoupdate/server"
)

func main() {
	e := echo.New()
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())

	// Initialize with constructors
	admin := server.NewAdmin("admin", "password123")
	app := server.NewApp("My App")
	
	log.Printf("Bootstrapping App with ID: %s", app.ID)

	_, err := autoupdate.Echo(e.Group(""), server.Config{
		AdminToken: "super-secret-admin-token",
		Admin:      admin,
		App:        app,
	})
	
	if err != nil {
		log.Fatalf("failed to setup autoupdate: %v", err)
	}

	log.Println("Echo server with autoupdate listening on :8082")
	e.Start(":8082")
}
