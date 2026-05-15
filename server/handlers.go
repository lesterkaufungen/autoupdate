package server

import (
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/lesterkaufungen/autoupdate/server/models"
)

func (s *Server) handleUpdateCheck(w http.ResponseWriter, r *http.Request) {
	appID := r.Header.Get("X-App-ID")
	os := r.Header.Get("X-OS")
	arch := r.Header.Get("X-Arch")
	currentVersion := r.Header.Get("X-Version")
	nodeID := r.Header.Get("X-Node-ID")

	if appID == "" || os == "" || arch == "" || currentVersion == "" {
		http.Error(w, "missing required headers (X-App-ID, X-OS, X-Arch, X-Version)", http.StatusBadRequest)
		return
	}

	// In public mode, we just check if the app exists
	var app models.App
	if err := s.db.First(&app, "id = ?", appID).Error; err != nil {
		s.logAnalytics(appID, 0, r, nodeID, currentVersion, "check", "fail", "app not found")
		http.Error(w, "app not found", http.StatusNotFound)
		return
	}

	// Check for latest active release
	var release models.Release
	now := time.Now()
	query := s.db.Where("app_id = ? AND os = ? AND arch = ? AND is_active = ?", appID, os, arch, true).
		Where("scheduled_at IS NULL OR scheduled_at <= ?", now)

	if err := query.Order("created_at DESC").First(&release).Error; err != nil {
		s.logAnalytics(appID, 0, r, nodeID, currentVersion, "check", "success", "no update available")
		json.NewEncoder(w).Encode(map[string]interface{}{"update_available": false})
		return
	}

	// Semantic version comparison could be more sophisticated
	// For now, we assume simple string comparison
	updateAvailable := release.Version != currentVersion
	forceUpdate := false

	// Check minimum acceptable version
	if app.MinVersion != "" && currentVersion < app.MinVersion {
		updateAvailable = true
		forceUpdate = true
	}

	if !updateAvailable {
		s.logAnalytics(appID, release.ID, r, nodeID, currentVersion, "check", "success", "already on latest")
		json.NewEncoder(w).Encode(map[string]interface{}{"update_available": false})
		return
	}

	s.logAnalytics(appID, release.ID, r, nodeID, currentVersion, "check", "success", "update available")

	// Construct absolute URL
	scheme := "http"
	if r.TLS != nil {
		scheme = "https"
	}
	downloadURL := fmt.Sprintf("%s://%s/download/%s/%d", scheme, r.Host, appID, release.ID)

	resp := map[string]interface{}{
		"update_available": true,
		"version":          release.Version,
		"url":              downloadURL,
		"sha256":           release.SHA256,
		"required":         forceUpdate,
	}
	json.NewEncoder(w).Encode(resp)
}

func (s *Server) handleDownload(w http.ResponseWriter, r *http.Request) {
	// Support extremely flexible path and query combinations:
	// 1. /download/:app_id/:release_id
	// 2. /download/:app_id/:os/:arch/:version
	// 3. /download?app_id=...&os=...&arch=...&version=...

	var appID string
	var releaseID uint
	var osName, arch, version string
	
	nodeID := r.URL.Query().Get("node_id")

	// Parse path segments
	path := strings.Trim(strings.TrimPrefix(r.URL.Path, "/download"), "/")
	parts := []string{}
	if path != "" {
		parts = strings.Split(path, "/")
	}

	if len(parts) > 0 {
		appID = parts[0]
	}

	if len(parts) >= 2 {
		// Could be releaseID or OS
		if rid, err := strconv.Atoi(parts[1]); err == nil {
			releaseID = uint(rid)
		} else {
			osName = parts[1]
		}
	}

	if len(parts) >= 3 {
		if osName != "" {
			arch = parts[2]
		}
	}

	if len(parts) >= 4 {
		version = parts[3]
	}

	// Override with query params if provided
	if qAppID := r.URL.Query().Get("app_id"); qAppID != "" {
		appID = qAppID
	}
	if qOS := r.URL.Query().Get("os"); qOS != "" {
		osName = qOS
	}
	if qArch := r.URL.Query().Get("arch"); qArch != "" {
		arch = qArch
	}
	if qVersion := r.URL.Query().Get("version"); qVersion != "" {
		version = qVersion
	}

	if appID == "" {
		http.Error(w, "app_id is required", http.StatusBadRequest)
		return
	}

	var release models.Release
	query := s.db.Where("app_id = ?", appID)

	if releaseID != 0 {
		if err := query.First(&release, releaseID).Error; err != nil {
			http.Error(w, "release not found", http.StatusNotFound)
			return
		}
	} else {
		query = query.Where("is_active = ?", true)
		if osName != "" {
			query = query.Where("os = ?", osName)
		}
		if arch != "" {
			query = query.Where("arch = ?", arch)
		}
		if version != "" && version != "latest" {
			query = query.Where("version = ?", version)
		}
		
		now := time.Now()
		query = query.Where("scheduled_at IS NULL OR scheduled_at <= ?", now)

		if err := query.Order("created_at DESC").First(&release).Error; err != nil {
			msg := "no matching release found"
			if osName != "" || arch != "" {
				msg = fmt.Sprintf("no matching release found for %s/%s", osName, arch)
			}
			http.Error(w, msg, http.StatusNotFound)
			return
		}
		releaseID = release.ID
	}

	// Binary fetch
	data, err := s.blobs.Get(appID, releaseID)
	if err != nil {
		s.logAnalytics(appID, releaseID, r, nodeID, release.Version, "download", "fail", err.Error())
		http.Error(w, "binary data not found", http.StatusNotFound)
		return
	}

	// For analytics tracking when called via simple browser link
	// We ensure logAnalytics gets the right OS/Arch even if not in headers
	if r.Header.Get("X-OS") == "" && osName != "" {
		r.Header.Set("X-OS", osName)
	}
	if r.Header.Get("X-Arch") == "" && arch != "" {
		r.Header.Set("X-Arch", arch)
	}

	s.logAnalytics(appID, releaseID, r, nodeID, release.Version, "download", "success", "")

	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Length", strconv.Itoa(len(data)))
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s-%s-%s-%s", appID, release.Version, release.OS, release.Arch))
	w.Write(data)
}

func (s *Server) handlePublicReleases(w http.ResponseWriter, r *http.Request) {
	appID := r.URL.Query().Get("app_id")
	osName := r.URL.Query().Get("os")
	arch := r.URL.Query().Get("arch")
	version := r.URL.Query().Get("version")

	if appID == "" {
		http.Error(w, "app_id is required", http.StatusBadRequest)
		return
	}

	query := s.db.Model(&models.Release{}).Where("app_id = ? AND is_active = ?", appID, true)
	if osName != "" {
		query = query.Where("os = ?", osName)
	}
	if arch != "" {
		query = query.Where("arch = ?", arch)
	}
	if version != "" {
		query = query.Where("version = ?", version)
	}

	now := time.Now()
	query = query.Where("scheduled_at IS NULL OR scheduled_at <= ?", now)

	var releases []models.Release
	if err := query.Order("created_at DESC").Find(&releases).Error; err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(releases)
}

func (s *Server) logAnalytics(appID string, releaseID uint, r *http.Request, nodeID, version, action, status, errMsg string) {
	osName := r.Header.Get("X-OS")
	if osName == "" {
		osName = r.URL.Query().Get("os")
	}
	arch := r.Header.Get("X-Arch")
	if arch == "" {
		arch = r.URL.Query().Get("arch")
	}

	if action == "download" && releaseID != 0 {
		// Lookup os/arch from release for downloads if not in headers
		if osName == "" || arch == "" {
			var release models.Release
			if err := s.db.First(&release, releaseID).Error; err == nil {
				osName = release.OS
				arch = release.Arch
			}
		}
	}

	ip, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		ip = r.RemoteAddr
	}

	country := ""
	if s.geoIP != nil {
		if city, err := s.geoIP.Country(net.ParseIP(ip)); err == nil {
			country = city.Country.IsoCode
		}
	}

	analytics := models.Analytics{
		AppID:        appID,
		ReleaseID:    releaseID,
		NodeID:       nodeID,
		IP:           ip,
		Country:      country,
		OS:           osName,
		Arch:         arch,
		Version:      version,
		Action:       action,
		Status:       status,
		ErrorMessage: errMsg,
		CreatedAt:    time.Now(),
	}
	s.db.Create(&analytics)

	// Update or create Node record
	var node models.Node
	if err := s.db.Where("id = ? AND app_id = ?", nodeID, appID).First(&node).Error; err != nil {
		// New node
		node = models.Node{
			ID:        nodeID,
			AppID:     appID,
			FirstSeen: time.Now(),
			LastSeen:  time.Now(),
			OS:        osName,
			Arch:      arch,
			Version:   version,
		}
		s.db.Create(&node)
	} else {
		// Existing node
		node.LastSeen = time.Now()
		node.OS = osName
		node.Arch = arch
		node.Version = version
		s.db.Save(&node)
	}

	analytics.Node = &node

	// Broadcast the new analytics entry
	s.Broadcast("analytics", analytics)
}
