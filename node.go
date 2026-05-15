package autoupdate

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/google/uuid"
)

func getOrCreateNodeID() string {
	configDir, err := os.UserConfigDir()
	if err != nil {
		configDir = "."
	}

	appDir := filepath.Join(configDir, "autoupdate")
	os.MkdirAll(appDir, 0755)

	idFile := filepath.Join(appDir, "node_id")
	data, err := os.ReadFile(idFile)
	if err == nil {
		id := strings.TrimSpace(string(data))
		if _, err := uuid.Parse(id); err == nil {
			return id
		}
	}

	id := uuid.New().String()
	os.WriteFile(idFile, []byte(id), 0644)
	return id
}
