package server

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/lesterkaufungen/autoupdate/server/models"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

func (s *Server) handleLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}

	var user models.User
	if err := s.db.Where("username = ?", req.Username).First(&user).Error; err != nil {
		http.Error(w, "invalid credentials", http.StatusUnauthorized)
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		http.Error(w, "invalid credentials", http.StatusUnauthorized)
		return
	}

	// Return the admin token as the "session" token
	json.NewEncoder(w).Encode(map[string]string{
		"token": s.adminToken,
	})
}

func (s *Server) handleApps(w http.ResponseWriter, r *http.Request) {
	userIDStr := r.Header.Get("User-ID")
	var userID uint
	if userIDStr != "" {
		id, _ := strconv.Atoi(userIDStr)
		userID = uint(id)
	}

	switch r.Method {
	case http.MethodGet:
		apps := []models.App{}
		query := s.db.Preload("Releases", func(db *gorm.DB) *gorm.DB {
			return db.Order("created_at DESC")
		})
		if userID != 0 {
			query = query.Where("user_id = ?", userID)
		}
		query.Find(&apps)

		type AppResponse struct {
			ID             string     `json:"id"`
			Name           string     `json:"name"`
			Description    string     `json:"description"`
			Icon           string     `json:"icon"`
			MinVersion     string     `json:"min_version"`
			LatestVersion  string     `json:"latest_version"`
			LastReleaseAt  *time.Time `json:"last_release_at"`
			ReleaseCount   int        `json:"release_count"`
			TodayInstalled int64      `json:"today_installed"`
			TotalInstalled int64      `json:"total_installed"`
			CreatedAt      time.Time  `json:"created_at"`
		}

		// Get total installs per app
		type countResult struct {
			AppID string
			Count int64
		}
		var totals []countResult
		s.db.Model(&models.Analytics{}).Select("app_id, COUNT(DISTINCT node_id) as count").Group("app_id").Scan(&totals)
		totalMap := make(map[string]int64)
		for _, r := range totals {
			totalMap[r.AppID] = r.Count
		}

		// Get today's installs per app
		var todayInstalls []countResult
		today := time.Now().Truncate(24 * time.Hour)
		s.db.Raw("SELECT app_id, COUNT(*) as count FROM (SELECT app_id, node_id, MIN(created_at) as first_seen FROM analytics GROUP BY app_id, node_id) WHERE first_seen >= ? GROUP BY app_id", today).Scan(&todayInstalls)
		todayMap := make(map[string]int64)
		for _, r := range todayInstalls {
			todayMap[r.AppID] = r.Count
		}

		result := []AppResponse{}
		for _, app := range apps {
			latest := "N/A"
			var lastReleaseAt *time.Time
			if len(app.Releases) > 0 {
				latest = app.Releases[0].Version
				lastReleaseAt = &app.Releases[0].CreatedAt
			}
			result = append(result, AppResponse{
				ID:             app.ID,
				Name:           app.Name,
				Description:    app.Description,
				Icon:           app.Icon,
				MinVersion:     app.MinVersion,
				LatestVersion:  latest,
				LastReleaseAt:  lastReleaseAt,
				ReleaseCount:   len(app.Releases),
				TodayInstalled: todayMap[app.ID],
				TotalInstalled: totalMap[app.ID],
				CreatedAt:      app.CreatedAt,
			})
		}
		json.NewEncoder(w).Encode(result)

	case http.MethodPost:
		var app models.App
		if err := json.NewDecoder(r.Body).Decode(&app); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		app.ID = uuid.New().String()
		if userID != 0 {
			app.UserID = userID
		}
		if err := s.db.Create(&app).Error; err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		json.NewEncoder(w).Encode(app)

	case http.MethodPut:
		var req struct {
			ID          string `json:"id"`
			Name        string `json:"name"`
			Description string `json:"description"`
			Icon        string `json:"icon"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		var app models.App
		if err := s.db.First(&app, "id = ?", req.ID).Error; err != nil {
			http.Error(w, "app not found", http.StatusNotFound)
			return
		}

		app.Name = req.Name
		app.Description = req.Description
		app.Icon = req.Icon
		if err := s.db.Save(&app).Error; err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		json.NewEncoder(w).Encode(app)
	}
}

func (s *Server) handleStats(w http.ResponseWriter, r *http.Request) {
	appID := r.URL.Query().Get("app_id")
	if appID == "" {
		http.Error(w, "app_id is required", http.StatusBadRequest)
		return
	}

	fromStr := r.URL.Query().Get("from")
	toStr := r.URL.Query().Get("to")

	type NodeStats struct {
		New    int64 `json:"new"`
		Active int64 `json:"active"`
	}

	type PlatformStats map[string]*NodeStats
	type CountryStats map[string]PlatformStats

	type VersionStats struct {
		Version   string        `json:"version"`
		New       int64         `json:"new"`
		Active    int64         `json:"active"`
		Platforms PlatformStats `json:"platforms"`
		Countries CountryStats `json:"countries"`
	}

	query := s.db.Model(&models.Analytics{}).Where("app_id = ? AND action = ? AND node_id != ''", appID, "check")

	if fromStr != "" {
		if t, err := time.Parse(time.RFC3339, fromStr); err == nil {
			query = query.Where("created_at >= ?", t)
		}
	}
	if toStr != "" {
		if t, err := time.Parse(time.RFC3339, toStr); err == nil {
			query = query.Where("created_at <= ?", t)
		}
	}

	// Logic: Get the latest 'check' for each NodeID within the range.
	var latestRecords []models.Analytics
	subQuery := query.Select("MAX(id)").Group("node_id")

	if err := s.db.Where("id IN (?)", subQuery).Find(&latestRecords).Error; err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Group results by version
	versionMap := make(map[string]*VersionStats)

	for _, rec := range latestRecords {
		v := rec.Version
		if v == "" {
			v = "unknown"
		}

		if _, ok := versionMap[v]; !ok {
			versionMap[v] = &VersionStats{
				Version:   v,
				Platforms: make(PlatformStats),
				Countries: make(CountryStats),
			}
		}

		stat := versionMap[v]

		// Determine if "new" or "active"
		var firstRecord models.Analytics
		s.db.Where("app_id = ? AND node_id = ?", appID, rec.NodeID).Order("created_at ASC").First(&firstRecord)

		isNew := false
		if fromStr != "" {
			if t, err := time.Parse(time.RFC3339, fromStr); err == nil {
				if !firstRecord.CreatedAt.Before(t) {
					isNew = true
				}
			}
		} else {
			today := time.Now().Truncate(24 * time.Hour)
			if !firstRecord.CreatedAt.Before(today) {
				isNew = true
			}
		}

		if isNew {
			stat.New++
		} else {
			stat.Active++
		}

		platform := fmt.Sprintf("%s-%s", rec.OS, rec.Arch)
		if _, ok := stat.Platforms[platform]; !ok {
			stat.Platforms[platform] = &NodeStats{}
		}
		if isNew {
			stat.Platforms[platform].New++
		} else {
			stat.Platforms[platform].Active++
		}

		country := strings.ToUpper(rec.Country)
		if country == "" {
			country = "Unknown"
		}

		if _, ok := stat.Countries[country]; !ok {
			stat.Countries[country] = make(PlatformStats)
		}
		if _, ok := stat.Countries[country][platform]; !ok {
			stat.Countries[country][platform] = &NodeStats{}
		}
		if isNew {
			stat.Countries[country][platform].New++
		} else {
			stat.Countries[country][platform].Active++
		}
	}

	result := []*VersionStats{}
	for _, v := range versionMap {
		result = append(result, v)
	}

	json.NewEncoder(w).Encode(result)
}

func (s *Server) handleDashboardStats(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var stats struct {
		TodayInstalled int64 `json:"today_installed"`
		TotalInstalled int64 `json:"total_installed"`
		TotalDownloads int64 `json:"total_downloads"`
		TotalNodes     int64 `json:"total_nodes"`
	}

	// Total unique nodes globally
	s.db.Model(&models.Analytics{}).Select("COUNT(DISTINCT node_id)").Scan(&stats.TotalNodes)
	stats.TotalInstalled = stats.TotalNodes // These are effectively the same in this context

	// Total downloads
	s.db.Model(&models.Analytics{}).Where("action = ?", "download").Count(&stats.TotalDownloads)

	// Today's installs (first time node seen globally)
	today := time.Now().Truncate(24 * time.Hour)
	s.db.Raw("SELECT COUNT(*) FROM (SELECT node_id, MIN(created_at) as first_seen FROM analytics GROUP BY node_id) WHERE first_seen >= ?", today).Scan(&stats.TodayInstalled)

	json.NewEncoder(w).Encode(stats)
}

func (s *Server) handleTrends(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	appID := r.URL.Query().Get("app_id")
	daysStr := r.URL.Query().Get("days")
	days := 30
	if daysStr != "" {
		if d, err := strconv.Atoi(daysStr); err == nil {
			days = d
		}
	}

	type TrendPoint struct {
		Date      string `json:"date"`
		Checks    int64  `json:"checks"`
		Downloads int64  `json:"downloads"`
	}

	var results []TrendPoint
	
	// We want to get the last N days
	now := time.Now().Truncate(24 * time.Hour)
	startDate := now.AddDate(0, 0, -(days - 1))

	// SQLite specific query to group by date
	// We'll query for both 'check' and 'download' actions
	var rawStats []struct {
		Date   string `gorm:"column:date"`
		Action string `gorm:"column:action"`
		Count  int64  `gorm:"column:count"`
	}

	query := s.db.Model(&models.Analytics{}).
		Select("strftime('%Y-%m-%d', created_at) as date, action, count(*) as count").
		Where("created_at >= ?", startDate).
		Group("date, action")

	if appID != "" {
		query = query.Where("app_id = ?", appID)
	}

	query.Scan(&rawStats)

	// Organize into a map for easy lookup
	statsMap := make(map[string]*TrendPoint)
	for i := 0; i < days; i++ {
		d := startDate.AddDate(0, 0, i).Format("2006-01-02")
		statsMap[d] = &TrendPoint{Date: d}
	}

	for _, rs := range rawStats {
		if tp, ok := statsMap[rs.Date]; ok {
			if rs.Action == "check" {
				tp.Checks = rs.Count
			} else if rs.Action == "download" {
				tp.Downloads = rs.Count
			}
		}
	}

	// Convert map back to sorted slice
	for i := 0; i < days; i++ {
		d := startDate.AddDate(0, 0, i).Format("2006-01-02")
		results = append(results, *statsMap[d])
	}

	json.NewEncoder(w).Encode(results)
}

func (s *Server) handleReleases(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		appID := r.URL.Query().Get("app_id")
		version := r.URL.Query().Get("version")
		pageStr := r.URL.Query().Get("page")
		limitStr := r.URL.Query().Get("limit")

		page, _ := strconv.Atoi(pageStr)
		if page <= 0 {
			page = 1
		}
		limit, _ := strconv.Atoi(limitStr)
		if limit <= 0 {
			limit = 10
		}
		offset := (page - 1) * limit

		releases := []models.Release{}
		query := s.db.Model(&models.Release{})
		if appID != "" {
			query = query.Where("app_id = ?", appID)
		}
		if version != "" {
			query = query.Where("version LIKE ?", "%"+version+"%")
		}

		var total int64
		query.Count(&total)

		query.Order("created_at DESC").Limit(limit).Offset(offset).Find(&releases)

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"items": releases,
			"total": total,
			"page":  page,
			"limit": limit,
		})
	case http.MethodPost:
		// Expect multipart form for binary upload
		err := r.ParseMultipartForm(500 << 20) // 500MB max for multiple binaries
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		appID := r.FormValue("app_id")
		version := r.FormValue("version")
		scheduledAtStr := r.FormValue("scheduled_at")

		var scheduledAt *time.Time
		if scheduledAtStr != "" {
			t, err := time.Parse(time.RFC3339, scheduledAtStr)
			if err == nil {
				scheduledAt = &t
			}
		}

		createdReleases := []models.Release{}

		// Iterate through all files in the form
		for key, headers := range r.MultipartForm.File {
			// Expecting key format: binary_os_arch (e.g., binary_linux_amd64)
			// Or just 'binary' for single upload (backwards compatibility)
			var osName, arch string
			if key == "binary" {
				osName = r.FormValue("os")
				arch = r.FormValue("arch")
			} else if strings.HasPrefix(key, "binary_") {
				parts := strings.Split(key, "_")
				if len(parts) == 3 {
					osName = parts[1]
					arch = parts[2]
				}
			}

			if osName == "" || arch == "" {
				continue
			}

			fileHeader := headers[0]
			file, err := fileHeader.Open()
			if err != nil {
				continue
			}
			defer file.Close()

			// Calculate hash and size
			hasher := sha256.New()
			tmpFile, _ := os.CreateTemp("", "release-")
			defer os.Remove(tmpFile.Name())
			defer tmpFile.Close()

			size, _ := io.Copy(io.MultiWriter(tmpFile, hasher), file)
			sha256Sum := hex.EncodeToString(hasher.Sum(nil))

			release := models.Release{
				AppID:       appID,
				Version:     version,
				OS:          osName,
				Arch:        arch,
				SHA256:      sha256Sum,
				Size:        size,
				ScheduledAt: scheduledAt,
			}

			if err := s.db.Create(&release).Error; err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

			// Store in blob storage
			tmpFile.Seek(0, 0)
			if err := s.blobs.Store(appID, release.ID, tmpFile); err != nil {
				s.db.Delete(&release)
				http.Error(w, "failed to store binary", http.StatusInternalServerError)
				return
			}

			createdReleases = append(createdReleases, release)
		}

		if len(createdReleases) == 0 {
			http.Error(w, "at least one binary is required", http.StatusBadRequest)
			return
		}

		json.NewEncoder(w).Encode(createdReleases)
	}
}

func (s *Server) handleLogs(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		appID := r.URL.Query().Get("app_id")
		fromStr := r.URL.Query().Get("from")
		toStr := r.URL.Query().Get("to")
		pageStr := r.URL.Query().Get("page")
		limitStr := r.URL.Query().Get("limit")
		action := r.URL.Query().Get("action")
		status := r.URL.Query().Get("status")
		search := r.URL.Query().Get("search")
		sortBy := r.URL.Query().Get("sort_by")
		order := r.URL.Query().Get("order")

		page, _ := strconv.Atoi(pageStr)
		if page <= 0 {
			page = 1
		}
		limit, _ := strconv.Atoi(limitStr)
		if limit <= 0 {
			limit = 50
		}
		offset := (page - 1) * limit

		logs := []models.Analytics{}
		query := s.db.Model(&models.Analytics{})
		if appID != "" {
			query = query.Where("app_id = ?", appID)
		}
		if action != "" {
			query = query.Where("action = ?", action)
		}
		if status != "" {
			query = query.Where("status = ?", status)
		}
		if search != "" {
			searchTerm := "%" + search + "%"
			query = query.Where("(node_id LIKE ? OR ip LIKE ? OR version LIKE ?)", searchTerm, searchTerm, searchTerm)
		}
		if fromStr != "" {
			if t, err := time.Parse(time.RFC3339, fromStr); err == nil {
				query = query.Where("created_at >= ?", t)
			}
		}
		if toStr != "" {
			if t, err := time.Parse(time.RFC3339, toStr); err == nil {
				query = query.Where("created_at <= ?", t)
			}
		}

		var total int64
		query.Count(&total)

		sortCol := "created_at"
		if sortBy == "version" {
			sortCol = "version"
		}
		if order != "asc" {
			order = "desc"
		}

		query.Preload("Node").Order(fmt.Sprintf("%s %s", sortCol, order)).Limit(limit).Offset(offset).Find(&logs)

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"items": logs,
			"total": total,
			"page":  page,
			"limit": limit,
		})
	}
}
