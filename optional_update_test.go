package autoupdate

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"
)

func TestOptionalUpdate(t *testing.T) {
	// Create a dummy binary
	tmpDir := t.TempDir()
	binaryPath := filepath.Join(tmpDir, "dummy")
	
	src := `package main
import "fmt"
func main() {
	fmt.Println("STARTED")
}`
	srcPath := filepath.Join(tmpDir, "main.go")
	os.WriteFile(srcPath, []byte(src), 0644)
	
	cmd := exec.Command("go", "build", "-o", binaryPath, srcPath)
	if err := cmd.Run(); err != nil {
		t.Fatalf("failed to build dummy binary: %v", err)
	}

	// Mock server
	var serverVersion = "1.1.0"
	var updateAvailable = false
	
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/update" {
			resp := updateResponse{
				UpdateAvailable: updateAvailable,
				Version:         serverVersion,
				URL:             "http://" + r.Host + "/download",
				SHA256:          "", 
			}
			
			if updateAvailable {
				newBinaryPath := binaryPath + ".new"
				newSrc := `package main
import "fmt"
func main() {
	fmt.Println("STARTED V2")
}`
				newSrcPath := filepath.Join(tmpDir, "main_v2.go")
				os.WriteFile(newSrcPath, []byte(newSrc), 0644)
				exec.Command("go", "build", "-o", newBinaryPath, newSrcPath).Run()
				
				f, _ := os.Open(newBinaryPath)
				defer f.Close()
				h := sha256.New()
				io.Copy(h, f)
				resp.SHA256 = hex.EncodeToString(h.Sum(nil))
			}
			
			json.NewEncoder(w).Encode(resp)
			return
		}
		
		if r.URL.Path == "/download" {
			newBinaryPath := binaryPath + ".new"
			f, _ := os.Open(newBinaryPath)
			defer f.Close()
			io.Copy(w, f)
			return
		}
	}))
	defer ts.Close()

	config := WatchConfig{
		UpdateServer: ts.URL,
		Interval:     100 * time.Millisecond,
		AutoRestart:  true,
		Target:       binaryPath,
		Version:      "0.0.0",
	}

	watcher, err := Watch(config)
	if err != nil {
		t.Fatalf("Watch failed: %v", err)
	}
	defer watcher.Shutdown()

	events := watcher.Watch()
	
	// Skip started event
	<-events

	// Trigger update
	updateAvailable = true
	
	var updateEvent UpdateEvent
	// Wait for update event
	select {
	case event := <-events:
		if event.Type != EventUpdate {
			t.Errorf("expected EventUpdate, got %v (Error: %v)", event.Type, event.Error)
		}
		updateEvent = event
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for EventUpdate")
	}

	// Wait a bit to ensure it DOES NOT restart automatically
	select {
	case event := <-events:
		if event.Type == EventRestart {
			t.Fatal("Update started automatically despite having a listener!")
		}
	case <-time.After(500 * time.Millisecond):
		// Good, no restart yet
	}

	// Now allow it
	updateEvent.Allow()

	// Now it should restart
	select {
	case event := <-events:
		if event.Type != EventRestart {
			t.Errorf("expected EventRestart after Allow(), got %v", event.Type)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for EventRestart after Allow()")
	}
}
