package server

import (
	"fmt"
	"math/rand"
	"time"

	"github.com/google/uuid"
	"github.com/lesterkaufungen/autoupdate/server/models"
)

func (s *Server) SeedDemoData(adminUsername, adminPassword string) error {
	fmt.Println("🌱 Seeding demo data...")

	// 1. Ensure Admin User
	var user models.User
	if err := s.db.Where("username = ?", adminUsername).First(&user).Error; err != nil {
		u, err := s.CreateUser(adminUsername, adminPassword)
		if err != nil {
			return err
		}
		user = *u
	}

	// 2. Create Demo Apps
	demoApps := []struct {
		Name        string
		Description string
	}{
		{"SkyNet Core", "Neural network management and satellite uplink control."},
		{"GlobalConnect", "Enterprise-grade VPN and secure tunneling service."},
		{"DataStream", "Real-time data ingestion and processing engine."},
		{"OmegaTerminal", "Advanced command-line interface for multi-cloud operations."},
	}

	apps := make([]models.App, 0)
	for _, da := range demoApps {
		var app models.App
		if err := s.db.Where("name = ? AND user_id = ?", da.Name, user.ID).First(&app).Error; err != nil {
			app = models.App{
				ID:          uuid.New().String(),
				Name:        da.Name,
				Description: da.Description,
				Icon:        fmt.Sprintf("https://api.dicebear.com/7.x/shapes/svg?seed=%s", da.Name),
				UserID:      user.ID,
				MinVersion:  "1.0.0",
			}
			if err := s.db.Create(&app).Error; err != nil {
				return err
			}
		}
		apps = append(apps, app)
	}

	// 3. Create Releases for each app
	oss := []string{"linux", "windows", "darwin"}
	archs := []string{"amd64", "arm64"}
	versions := []string{"1.0.0", "1.1.0", "1.2.0", "1.2.1", "1.3.0", "2.0.0"}

	for _, app := range apps {
		for i, v := range versions {
			releaseTime := time.Now().AddDate(0, 0, - (len(versions)-i)*5)
			
			// Make the last version of SkyNet Core scheduled for the future
			var scheduledAt *time.Time
			if app.Name == "SkyNet Core" && i == len(versions)-1 {
				future := time.Now().Add(48 * time.Hour)
				scheduledAt = &future
			}

			for _, osName := range oss {
				for _, arch := range archs {
					var release models.Release
					if err := s.db.Where("app_id = ? AND version = ? AND os = ? AND arch = ?", app.ID, v, osName, arch).First(&release).Error; err != nil {
						release = models.Release{
							AppID:       app.ID,
							Version:     v,
							OS:          osName,
							Arch:        arch,
							SHA256:      "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
							Size:        1024 * 1024 * rand.Int63n(50),
							ScheduledAt: scheduledAt,
							CreatedAt:   releaseTime,
						}
						if err := s.db.Create(&release).Error; err != nil {
							return err
						}
					}
				}
			}
		}
	}

	// 4. Generate Analytics (Logs/Stats)
	fmt.Println("📊 Generating simulated analytics (this may take a few seconds)...")
	
	countries := []string{"US", "GB", "DE", "FR", "JP", "CN", "BR", "IN", "CA", "AU"}
	r := rand.New(rand.NewSource(time.Now().UnixNano()))

	// Create ~500 unique nodes across all apps
	type nodeInfo struct {
		id      string
		appID   string
		os      string
		arch    string
		country string
		firstSeen time.Time
	}
	nodes := make([]nodeInfo, 0)
	for i := 0; i < 500; i++ {
		app := apps[r.Intn(len(apps))]
		osName := oss[r.Intn(len(oss))]
		arch := archs[r.Intn(len(archs))]
		country := countries[r.Intn(len(countries))]
		
		daysAgo := r.Intn(30)
		if r.Float32() < 0.1 {
			daysAgo = 0 // 10% new today
		}
		firstSeen := time.Now().AddDate(0, 0, -daysAgo).Truncate(24 * time.Hour).Add(time.Duration(r.Intn(24)) * time.Hour)
		
		nodes = append(nodes, nodeInfo{
			id:        uuid.New().String(),
			appID:     app.ID,
			os:        osName,
			arch:      arch,
			country:   country,
			firstSeen: firstSeen,
		})
	}

	// Generate ~5000 analytics records
	records := make([]models.Analytics, 0)
	nodesMap := make(map[string]*models.Node)
	for i := 0; i < 5000; i++ {
		node := nodes[r.Intn(len(nodes))]
		
		// Interaction time must be after firstSeen
		daysSinceFirstSeen := int(time.Since(node.firstSeen).Hours() / 24)
		var interactionTime time.Time
		if daysSinceFirstSeen > 0 {
			interactionTime = node.firstSeen.Add(time.Duration(r.Intn(daysSinceFirstSeen*24)) * time.Hour)
		} else {
			interactionTime = node.firstSeen
		}

		// Determine version based on interaction time
		version := versions[0]
		for j, v := range versions {
			releaseTime := time.Now().AddDate(0, 0, - (len(versions)-j)*5)
			if interactionTime.After(releaseTime) {
				version = v
			}
		}

		action := "check"
		if r.Float32() < 0.1 {
			action = "download"
		}

		status := "success"
		if r.Float32() < 0.05 {
			status = "fail"
		}

		records = append(records, models.Analytics{
			AppID:     node.appID,
			NodeID:    node.id,
			IP:        fmt.Sprintf("%d.%d.%d.%d", r.Intn(256), r.Intn(256), r.Intn(256), r.Intn(256)),
			Country:   node.country,
			OS:        node.os,
			Arch:      node.arch,
			Version:   version,
			Action:    action,
			Status:    status,
			CreatedAt: interactionTime,
		})

		// Track nodes for demo
		if _, ok := nodesMap[node.id]; !ok {
			nodesMap[node.id] = &models.Node{
				ID:        node.id,
				AppID:     node.appID,
				FirstSeen: node.firstSeen,
				LastSeen:  interactionTime,
				OS:        node.os,
				Arch:      node.arch,
				Version:   version,
			}
		} else {
			if interactionTime.After(nodesMap[node.id].LastSeen) {
				nodesMap[node.id].LastSeen = interactionTime
				nodesMap[node.id].Version = version
			}
			if interactionTime.Before(nodesMap[node.id].FirstSeen) {
				nodesMap[node.id].FirstSeen = interactionTime
			}
		}

		if len(records) >= 1000 {
			if err := s.db.Create(&records).Error; err != nil {
				return err
			}
			records = records[:0]
		}
	}

	if len(records) > 0 {
		if err := s.db.Create(&records).Error; err != nil {
			return err
		}
	}

	// Save all generated nodes
	nodeList := make([]models.Node, 0, len(nodesMap))
	for _, n := range nodesMap {
		nodeList = append(nodeList, *n)
	}
	if len(nodeList) > 0 {
		if err := s.db.Create(&nodeList).Error; err != nil {
			return err
		}
	}

	fmt.Println("✅ Demo data seeded successfully.")
	return nil
}
