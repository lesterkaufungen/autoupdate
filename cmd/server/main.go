package main

import (
	"log"
	"net/http"
	"os"

	"github.com/lesterkaufungen/autoupdate"
	"github.com/lesterkaufungen/autoupdate/server"
)

func main() {
	config := server.Config{
		DatabasePath: "./autoupdate.db",
		StoragePath:  "./binaries",
		ListenAddr:   ":8080",
		AdminToken:   os.Getenv("ADMIN_TOKEN"),
		UI:           os.Getenv("ENABLE_UI") == "true",
	}

	if os.Getenv("DB_PATH") != "" {
		config.DatabasePath = os.Getenv("DB_PATH")
	}
	if os.Getenv("STORAGE_PATH") != "" {
		config.StoragePath = os.Getenv("STORAGE_PATH")
	}
	if os.Getenv("PORT") != "" {
		config.ListenAddr = ":" + os.Getenv("PORT")
	}

	// Bootstrap Admin
	if os.Getenv("BOOTSTRAP_USER") != "" && os.Getenv("BOOTSTRAP_PASS") != "" {
		config.Admin = server.NewAdmin(os.Getenv("BOOTSTRAP_USER"), os.Getenv("BOOTSTRAP_PASS"))
	}

	// Bootstrap App
	if os.Getenv("BOOTSTRAP_APP_NAME") != "" {
		config.App = server.NewApp(os.Getenv("BOOTSTRAP_APP_NAME"))
		if os.Getenv("BOOTSTRAP_APP_ICON") != "" {
			config.App.Icon = os.Getenv("BOOTSTRAP_APP_ICON")
		}
	}

	mux := http.NewServeMux()
	s, err := autoupdate.HTTP(mux, config)
	if err != nil {
		log.Fatalf("failed to setup autoupdate: %v", err)
	}

	if os.Getenv("DEMO") == "true" {
		adminUser := os.Getenv("BOOTSTRAP_USER")
		if adminUser == "" {
			adminUser = "admin"
		}
		adminPass := os.Getenv("BOOTSTRAP_PASS")
		if adminPass == "" {
			adminPass = "admin"
		}
		if err := s.SeedDemoData(adminUser, adminPass); err != nil {
			log.Printf("Warning: failed to seed demo data: %v", err)
		}
	}

	srv := &http.Server{
		Addr:    config.ListenAddr,
		Handler: mux,
	}

	log.Printf("Update server listening on %s\n", config.ListenAddr)
	if err := srv.ListenAndServe(); err != nil {
		log.Fatalf("server failed: %v", err)
	}
}
