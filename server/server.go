package server

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"

	"github.com/google/uuid"
	"github.com/gin-gonic/gin"
	"github.com/labstack/echo/v4"
	"github.com/lesterkaufungen/autoupdate/server/middleware"
	"github.com/lesterkaufungen/autoupdate/server/models"
	"github.com/lesterkaufungen/autoupdate/server/storage"
	"github.com/oschwald/geoip2-golang"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type Config struct {
	DatabasePath string
	StoragePath  string
	ListenAddr   string
	AdminToken   string
	GeoIPDBPath  string
	UI           bool
	Admin        *BootstrapAdmin
	App          *BootstrapApp
}

type BootstrapAdmin struct {
	Username string
	Password string
}

type BootstrapApp struct {
	ID   string
	Name string
	Icon string
}

func NewAdmin(username, password string) *BootstrapAdmin {
	return &BootstrapAdmin{
		Username: username,
		Password: password,
	}
}

func NewApp(name string) *BootstrapApp {
	return &BootstrapApp{
		ID:   uuid.New().String(),
		Name: name,
	}
}

const DefaultGeoIPURL = "https://github.com/P3TERX/GeoLite.mmdb/releases/latest/download/GeoLite2-City.mmdb"

func New(configs ...Config) (*Server, error) {
	var config Config
	if len(configs) > 0 {
		config = configs[0]
	}

	// Always ensure AdminToken is a UUID if not set
	if config.AdminToken == "" {
		config.AdminToken = uuid.New().String()
		fmt.Printf("⚠️ No Admin Token provided, generated a random one: %s\n", config.AdminToken)
	}
	if config.DatabasePath == "" {
		config.DatabasePath = "./autoupdate.db"
	}
	if config.StoragePath == "" {
		config.StoragePath = "./binaries"
	}
	if config.GeoIPDBPath == "" {
		config.GeoIPDBPath = "./geoip.mmdb"
	}

	dsn := fmt.Sprintf("%s?_journal_mode=WAL&_busy_timeout=5000&_synchronous=NORMAL", config.DatabasePath)
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		return nil, err
	}

	// Set connection limits for SQLite
	sqlDB, err := db.DB()
	if err == nil {
		sqlDB.SetMaxOpenConns(1)
	}

	// Auto-migrate models
	if err := db.AutoMigrate(
		&models.User{},
		&models.App{},
		&models.Release{},
		&models.Analytics{},
		&models.Node{},
	); err != nil {
		return nil, err
	}

	blobs, err := storage.NewBlobStorage(config.StoragePath)
	if err != nil {
		return nil, err
	}

	s := &Server{
		db:         db,
		blobs:      blobs,
		config:     config,
		adminToken: config.AdminToken,
		hub:        NewHub(),
	}

	go s.hub.Run()

	// Ensure GeoIP database exists
	if _, err := os.Stat(config.GeoIPDBPath); os.IsNotExist(err) {
		fmt.Printf("GeoIP database not found at %s, attempting to download...\n", config.GeoIPDBPath)
		if err := s.downloadGeoIP(config.GeoIPDBPath); err != nil {
			fmt.Printf("Warning: failed to download GeoIP database: %v\n", err)
		}
	}

	reader, err := geoip2.Open(config.GeoIPDBPath)
	if err == nil {
		s.geoIP = reader
	} else {
		fmt.Printf("Warning: failed to open GeoIP database: %v\n", err)
	}

	if err := s.bootstrap(); err != nil {
		return nil, fmt.Errorf("bootstrap failed: %w", err)
	}

	return s, nil
}

func (s *Server) downloadGeoIP(path string) error {
	// Ensure directory exists
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	resp, err := http.Get(DefaultGeoIPURL)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	out, err := os.Create(path)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, resp.Body)
	return err
}

func (s *Server) Close() error {
	if s.geoIP != nil {
		s.geoIP.Close()
	}
	sqlDB, err := s.db.DB()
	if err == nil {
		sqlDB.Close()
	}
	if s.blobs != nil {
		s.blobs.Close()
	}
	return nil
}

func (s *Server) bootstrap() error {
	if s.config.Admin == nil {
		return nil
	}

	var user models.User
	if err := s.db.Where("username = ?", s.config.Admin.Username).First(&user).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			u, err := s.CreateUser(s.config.Admin.Username, s.config.Admin.Password)
			if err != nil {
				return err
			}
			user = *u
		} else {
			return err
		}
	}

	// App requires Admin
	if s.config.App == nil {
		return nil
	}

	var app models.App
	if err := s.db.Where("id = ?", s.config.App.ID).First(&app).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			app = models.App{
				ID:          s.config.App.ID,
				Name:        s.config.App.Name,
				Icon:        s.config.App.Icon,
				UserID:      user.ID,
				Description: "Demo application bootstrapped on first run.",
			}
			if err := s.db.Create(&app).Error; err != nil {
				return err
			}
			fmt.Printf("✅ Bootstrapped initial App: %s (%s)\n", app.Name, app.ID)
		} else {
			return err
		}
	}

	return nil
}

type Server struct {
	db         *gorm.DB
	blobs      *storage.BlobStorage
	config     Config
	adminToken string
	geoIP      *geoip2.Reader
	hub        *Hub
}

func (s *Server) HTTP(mux *http.ServeMux) {
	adminAuth := middleware.UserAuth(s.db, s.adminToken)
	appAuth := middleware.AppAuth(s.db)

	// Client API
	mux.Handle("/update/", appAuth(http.HandlerFunc(s.HandleUpdateCheck)))
	mux.Handle("/download/", http.HandlerFunc(s.HandleDownload))
	mux.Handle("/releases", http.HandlerFunc(s.HandlePublicReleases))

	// Admin API
	mux.Handle("/admin/login", http.HandlerFunc(s.HandleLogin))
	mux.Handle("/admin/apps", adminAuth(http.HandlerFunc(s.HandleApps)))
	mux.Handle("/admin/dashboard/stats", adminAuth(http.HandlerFunc(s.HandleDashboardStats)))
	mux.Handle("/admin/dashboard/trends", adminAuth(http.HandlerFunc(s.HandleTrends)))
	mux.Handle("/admin/releases", adminAuth(http.HandlerFunc(s.HandleReleases)))
	mux.Handle("/admin/logs", adminAuth(http.HandlerFunc(s.HandleLogs)))
	mux.Handle("/admin/stats", adminAuth(http.HandlerFunc(s.HandleStats)))
	mux.Handle("/admin/ws", adminAuth(http.HandlerFunc(s.HandleWS)))

	// UI
	s.serveUI(mux)
}

func (s *Server) Gin(r gin.IRouter) {
	adminAuth := middleware.GinUserAuth(s.adminToken)
	appAuth := middleware.GinAppAuth(s.db)

	r.GET("/update/*path", appAuth, gin.WrapH(http.HandlerFunc(s.HandleUpdateCheck)))
	r.GET("/download/*path", gin.WrapH(http.HandlerFunc(s.HandleDownload)))
	r.GET("/releases", gin.WrapH(http.HandlerFunc(s.HandlePublicReleases)))

	admin := r.Group("/admin", adminAuth)
	{
		admin.POST("/login", gin.WrapH(http.HandlerFunc(s.HandleLogin)))
		admin.GET("/apps", gin.WrapH(http.HandlerFunc(s.HandleApps)))
		admin.PUT("/apps", gin.WrapH(http.HandlerFunc(s.HandleApps)))
		admin.POST("/apps", gin.WrapH(http.HandlerFunc(s.HandleApps)))
		admin.GET("/dashboard/stats", gin.WrapH(http.HandlerFunc(s.HandleDashboardStats)))
		admin.GET("/dashboard/trends", gin.WrapH(http.HandlerFunc(s.HandleTrends)))
		admin.GET("/releases", gin.WrapH(http.HandlerFunc(s.HandleReleases)))
		admin.POST("/releases", gin.WrapH(http.HandlerFunc(s.HandleReleases)))
		admin.GET("/logs", gin.WrapH(http.HandlerFunc(s.HandleLogs)))
		admin.GET("/stats", gin.WrapH(http.HandlerFunc(s.HandleStats)))
		admin.GET("/ws", gin.WrapH(http.HandlerFunc(s.HandleWS)))
	}

	if s.config.UI {
		r.Any("/ui/*path", gin.WrapH(s.UIHandler()))
		r.GET("/ui", func(c *gin.Context) {
			c.Redirect(http.StatusMovedPermanently, "/ui/")
		})
	}
}

func (s *Server) Echo(e *echo.Group) {
	adminAuth := middleware.EchoUserAuth(s.adminToken)
	appAuth := middleware.EchoAppAuth(s.db)

	e.GET("/update/*", echo.WrapHandler(http.HandlerFunc(s.HandleUpdateCheck)), appAuth)
	e.GET("/download/*", echo.WrapHandler(http.HandlerFunc(s.HandleDownload)))
	e.GET("/releases", echo.WrapHandler(http.HandlerFunc(s.HandlePublicReleases)))

	admin := e.Group("/admin", adminAuth)
	{
		admin.POST("/login", echo.WrapHandler(http.HandlerFunc(s.HandleLogin)))
		admin.GET("/apps", echo.WrapHandler(http.HandlerFunc(s.HandleApps)))
		admin.PUT("/apps", echo.WrapHandler(http.HandlerFunc(s.HandleApps)))
		admin.POST("/apps", echo.WrapHandler(http.HandlerFunc(s.HandleApps)))
		admin.GET("/dashboard/stats", echo.WrapHandler(http.HandlerFunc(s.HandleDashboardStats)))
		admin.GET("/dashboard/trends", echo.WrapHandler(http.HandlerFunc(s.HandleTrends)))
		admin.GET("/releases", echo.WrapHandler(http.HandlerFunc(s.HandleReleases)))
		admin.POST("/releases", echo.WrapHandler(http.HandlerFunc(s.HandleReleases)))
		admin.GET("/logs", echo.WrapHandler(http.HandlerFunc(s.HandleLogs)))
		admin.GET("/stats", echo.WrapHandler(http.HandlerFunc(s.HandleStats)))
		admin.GET("/ws", echo.WrapHandler(http.HandlerFunc(s.HandleWS)))
	}

	if s.config.UI {
		e.Any("/ui/*", echo.WrapHandler(s.UIHandler()))
		e.GET("/ui", func(c echo.Context) error {
			return c.Redirect(http.StatusMovedPermanently, "/ui/")
		})
	}
}

func Gin(r interface{}, configs ...Config) (*Server, error) {
	s, err := New(configs...)
	if err != nil {
		return nil, err
	}
	if ir, ok := r.(gin.IRouter); ok {
		s.Gin(ir)
	}
	return s, nil
}

func Echo(e interface{}, configs ...Config) (*Server, error) {
	s, err := New(configs...)
	if err != nil {
		return nil, err
	}
	if eg, ok := e.(*echo.Group); ok {
		s.Echo(eg)
	}
	return s, nil
}

func HTTP(mux *http.ServeMux, configs ...Config) (*Server, error) {
	s, err := New(configs...)
	if err != nil {
		return nil, err
	}
	s.HTTP(mux)
	return s, nil
}

func (s *Server) HandleLogin(w http.ResponseWriter, r *http.Request) {
	s.handleLogin(w, r)
}

func (s *Server) HandleUpdateCheck(w http.ResponseWriter, r *http.Request) {
	s.handleUpdateCheck(w, r)
}

func (s *Server) HandleDownload(w http.ResponseWriter, r *http.Request) {
	s.handleDownload(w, r)
}

func (s *Server) HandlePublicReleases(w http.ResponseWriter, r *http.Request) {
	s.handlePublicReleases(w, r)
}

func (s *Server) HandleApps(w http.ResponseWriter, r *http.Request) {
	s.handleApps(w, r)
}

func (s *Server) HandleDashboardStats(w http.ResponseWriter, r *http.Request) {
	s.handleDashboardStats(w, r)
}

func (s *Server) HandleTrends(w http.ResponseWriter, r *http.Request) {
	s.handleTrends(w, r)
}

func (s *Server) HandleReleases(w http.ResponseWriter, r *http.Request) {
	s.handleReleases(w, r)
}

func (s *Server) HandleLogs(w http.ResponseWriter, r *http.Request) {
	s.handleLogs(w, r)
}

func (s *Server) HandleStats(w http.ResponseWriter, r *http.Request) {
	s.handleStats(w, r)
}

func (s *Server) CreateUser(username, password string) (*models.User, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	user := &models.User{
		Username:     username,
		PasswordHash: string(hash),
	}

	if err := s.db.Create(user).Error; err != nil {
		return nil, err
	}

	return user, nil
}

func (s *Server) CreateApp(userID uint, name, description string) (*models.App, error) {
	app := &models.App{
		ID:          uuid.New().String(),
		Name:        name,
		Description: description,
		UserID:      userID,
	}

	if err := s.db.Create(app).Error; err != nil {
		return nil, err
	}

	return app, nil
}
