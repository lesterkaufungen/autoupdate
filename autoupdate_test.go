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
	"strings"
	"testing"
	"time"
)

func TestWatch(t *testing.T) {
	// Create a dummy binary
	tmpDir := t.TempDir()
	binaryPath := filepath.Join(tmpDir, "dummy")
	
	// Create a simple go program that prints "STARTED" and echo GOT: msg
	src := `package main
import "fmt"
import "bufio"
import "os"
func main() {
	fmt.Println("STARTED")
	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		fmt.Printf("GOT: %s\n", scanner.Text())
	}
}`
	srcPath := filepath.Join(tmpDir, "main.go")
	os.WriteFile(srcPath, []byte(src), 0644)
	
	cmd := exec.Command("go", "build", "-o", binaryPath, srcPath)
	if err := cmd.Run(); err != nil {
		t.Fatalf("failed to build dummy binary: %v", err)
	}

	// Mock server
	var serverVersion = "1.0.0"
	var updateAvailable = false
	
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/update" {
			resp := updateResponse{
				UpdateAvailable: updateAvailable,
				Version:         serverVersion,
				URL:             "", // Will be filled below
				SHA256:          "", // Will be filled below
			}
			
			if updateAvailable {
				// Prepare new binary
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
				
				resp.URL = "http://" + r.Host + "/download"
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
		Interval:     500 * time.Millisecond,
		AutoRestart:  true,
		Target:       binaryPath,
	}

	watcher, err := Watch(config)
	if err != nil {
		t.Fatalf("Watch failed: %v", err)
	}
	defer watcher.Shutdown()

	// Wait for started event
	events := watcher.Watch()
	select {
	case event := <-events:
		if event.Type != EventStarted {
			t.Errorf("expected EventStarted, got %v", event.Type)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for EventStarted")
	}

	// Test pipe
	pipe := watcher.Pipe()
	_, err = pipe.Write([]byte("ping\n"))
	if err != nil {
		t.Errorf("pipe write failed: %v", err)
	}

	// Read from pipe (stdout of the process)
	buf := make([]byte, 1024)
	n, err := pipe.Read(buf)
	if err != nil {
		t.Errorf("pipe read failed: %v", err)
	}
	output := string(buf[:n])
	if n == 0 {
		t.Errorf("expected some output from pipe")
	}
	if !strings.Contains(output, "STARTED") {
		t.Errorf("expected output to contain 'STARTED', got %q", output)
	}

	// Trigger update
	updateAvailable = true
	
	// Wait for update event
	select {
	case event := <-events:
		if event.Type != EventUpdate {
			t.Errorf("expected EventUpdate, got %v", event.Type)
		}
		event.Allow()
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for EventUpdate")
	}

	// Wait for restart event
	select {
	case event := <-events:
		if event.Type != EventRestart {
			t.Errorf("expected EventRestart, got %v", event.Type)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for EventRestart")
	}

	// Wait for started event again (after restart)
	select {
	case event := <-events:
		if event.Type != EventStarted {
			t.Errorf("expected EventStarted after restart, got %v", event.Type)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for EventStarted after restart")
	}
}

func TestWatchSelf(t *testing.T) {
	// This is harder to test because it involves os.Executable() and replacing it.
	// But we can at least test that it starts.
	config := WatchConfig{
		SelfUpdate: true,
		Interval:   time.Hour, // Don't actually poll
	}
	
	watcher, err := Watch(config)
	if err != nil {
		t.Fatalf("Watch (SelfUpdate) failed: %v", err)
	}
	defer watcher.Shutdown()
	
	if watcher == nil {
		t.Fatal("expected watcher to be non-nil")
	}
}
