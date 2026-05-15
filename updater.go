package autoupdate

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"
)

type updateResponse struct {
	UpdateAvailable bool   `json:"update_available"`
	Version         string `json:"version"`
	URL             string `json:"url"`
	SHA256          string `json:"sha256"`
}

func (w *watcher) checkForUpdates() {
	if w.source == nil {
		return
	}

	currentVersion := w.config.Version
	if currentVersion == "" {
		currentVersion = "0.0.0"
	}

	upResp, err := w.source.Check(currentVersion)
	if err != nil {
		w.emit(UpdateEvent{Type: EventError, Timestamp: time.Now(), Error: err})
		return
	}

	if upResp == nil || !upResp.UpdateAvailable {
		return
	}

	event := UpdateEvent{
		Type:      EventUpdate,
		Version:   upResp.Version,
		Timestamp: time.Now(),
	}

	w.mu.RLock()
	hasListeners := len(w.subscribers) > 0 || len(w.callbacks) > 0
	w.mu.RUnlock()

	if hasListeners {
		w.mu.Lock()
		if w.pendingUpdates == nil {
			w.pendingUpdates = make(map[string]updateResponse)
		}
		w.pendingUpdates[upResp.Version] = *upResp
		w.mu.Unlock()

		v := upResp.Version
		event.allow = func() {
			w.allowUpdate(v)
		}
		w.emit(event)
		return
	}

	w.emit(event)

	if err := w.downloadAndVerify(*upResp); err != nil {
		w.emit(UpdateEvent{Type: EventError, Timestamp: time.Now(), Error: err})
		return
	}

	if w.config.AutoRestart {
		w.Restart()
	}
}

func (w *watcher) downloadAndVerify(up updateResponse) error {
	resp, err := http.Get(up.URL)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	tmpFile, err := os.CreateTemp("", "autoupdate-")
	if err != nil {
		return err
	}
	tmpPath := tmpFile.Name()
	success := false
	defer func() {
		if !success {
			os.Remove(tmpPath)
		}
	}()
	defer tmpFile.Close()

	hasher := sha256.New()
	multiWriter := io.MultiWriter(tmpFile, hasher)

	if _, err := io.Copy(multiWriter, resp.Body); err != nil {
		return err
	}

	computedHash := hex.EncodeToString(hasher.Sum(nil))
	if computedHash != up.SHA256 {
		return fmt.Errorf("hash mismatch: expected %s, got %s", up.SHA256, computedHash)
	}

	if err := tmpFile.Close(); err != nil {
		return err
	}

	w.mu.Lock()
	// Store the new binary path but don't delete it
	// We need to keep this file until restart
	w.pendingUpdate = tmpFile.Name()
	w.mu.Unlock()

	success = true
	// Successfully downloaded and verified.
	return nil
}
