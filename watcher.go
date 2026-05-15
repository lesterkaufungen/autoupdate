package autoupdate

import (
	"os"
	"os/exec"
	"sync"
	"time"
)

type watcher struct {
	config WatchConfig
	binary string
	args   []string
	isSelf bool
	source Source
	nodeID string

	mu           sync.RWMutex
	cmd          *exec.Cmd
	subscribers  []chan UpdateEvent
	callbacks    []func(UpdateEvent)
	stopped      chan struct{}
	restartChan  chan struct{}
	lastEvent      *UpdateEvent
	pendingUpdate  string
	pendingUpdates map[string]updateResponse

	pipe Pipe
}

func (w *watcher) Watch() <-chan UpdateEvent {
	w.mu.Lock()
	defer w.mu.Unlock()

	ch := make(chan UpdateEvent, 10)
	
	// If we already have a 'last' event, send it to the new subscriber
	if w.lastEvent != nil {
		ch <- *w.lastEvent
	}
	
	w.subscribers = append(w.subscribers, ch)
	return ch
}

func (w *watcher) OnUpdate(fn func(UpdateEvent)) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.callbacks = append(w.callbacks, fn)
}

func (w *watcher) allowUpdate(version string) {
	w.mu.Lock()
	if w.pendingUpdates == nil {
		w.mu.Unlock()
		return
	}
	up, ok := w.pendingUpdates[version]
	delete(w.pendingUpdates, version)
	w.mu.Unlock()

	if !ok {
		return
	}

	if err := w.downloadAndVerify(up); err != nil {
		w.emit(UpdateEvent{
			Type:      EventError,
			Timestamp: time.Now(),
			Error:     err,
		})
		return
	}

	if w.config.AutoRestart {
		w.Restart()
	}
}

func (w *watcher) Pipe() LauncherPipe {
	return &w.pipe
}

func (w *watcher) Restart() error {
	w.restartChan <- struct{}{}
	return nil
}

func (w *watcher) Shutdown() error {
	close(w.stopped)
	w.mu.RLock()
	defer w.mu.RUnlock()
	if w.cmd != nil && w.cmd.Process != nil {
		return w.cmd.Process.Kill()
	}
	return nil
}

func (w *watcher) emit(event UpdateEvent) {
	w.mu.Lock()
	w.lastEvent = &event
	subscribers := make([]chan UpdateEvent, len(w.subscribers))
	copy(subscribers, w.subscribers)
	callbacks := make([]func(UpdateEvent), len(w.callbacks))
	copy(callbacks, w.callbacks)
	w.mu.Unlock()

	for _, ch := range subscribers {
		select {
		case ch <- event:
		default:
			// Skip if buffer is full to avoid blocking core logic
		}
	}

	for _, fn := range callbacks {
		go fn(event)
	}
}

func (w *watcher) start() {
	go w.pollLoop()
	go w.runLoop()
}

func (w *watcher) pollLoop() {
	ticker := time.NewTicker(w.config.Interval)
	if w.config.Interval == 0 {
		ticker.Stop()
	}
	defer ticker.Stop()

	for {
		select {
		case <-w.stopped:
			return
		case <-ticker.C:
			w.checkForUpdates()
		}
	}
}

func (w *watcher) runLoop() {
	if w.isSelf {
		for {
			select {
			case <-w.stopped:
				return
			case <-w.restartChan:
				w.execSelf()
			}
		}
	}

	for {
		err := w.spawn()
		if err != nil {
			w.emit(UpdateEvent{
				Type:      EventError,
				Timestamp: time.Now(),
				Error:     err,
			})
			time.Sleep(1 * time.Second)
			continue
		}

		select {
		case <-w.stopped:
			return
		case <-w.restartChan:
			w.emit(UpdateEvent{
				Type:      EventRestart,
				Timestamp: time.Now(),
			})
			w.mu.Lock()
			if w.cmd != nil && w.cmd.Process != nil {
				w.cmd.Process.Kill()
				w.cmd.Wait() // Ensure process is gone before renaming
			}
			
			if w.pendingUpdate != "" {
				// Replace binary for external process
				if err := os.Rename(w.pendingUpdate, w.binary); err != nil {
					w.emit(UpdateEvent{
						Type:      EventError,
						Timestamp: time.Now(),
						Error:     err,
					})
				} else {
					os.Chmod(w.binary, 0755)
					w.pendingUpdate = ""
				}
			}
			w.mu.Unlock()
		}
	}
}

func (w *watcher) spawn() error {
	w.mu.Lock()
	w.cmd = exec.Command(w.binary, w.args...)
	
	stdout, err := w.cmd.StdoutPipe()
	if err != nil {
		w.mu.Unlock()
		return err
	}
	w.pipe.setReader(stdout)
	w.cmd.Stderr = w.cmd.Stdout // Combine stdout and stderr

	stdin, err := w.cmd.StdinPipe()
	if err != nil {
		w.mu.Unlock()
		return err
	}
	w.pipe.setWriter(stdin)

	err = w.cmd.Start()
	if err != nil {
		w.mu.Unlock()
		return err
	}
	w.mu.Unlock()

	w.emit(UpdateEvent{
		Type:      EventStarted,
		Timestamp: time.Now(),
	})

	go func() {
		w.cmd.Wait()
	}()

	return nil
}

func (w *watcher) execSelf() {
	w.emit(UpdateEvent{
		Type:      EventRestart,
		Timestamp: time.Now(),
	})

	w.mu.Lock()
	pending := w.pendingUpdate
	w.mu.Unlock()

	if pending != "" {
		err := runUpdateScript(w.binary, os.Args, pending, w.config.ServiceName)
		if err != nil {
			w.emit(UpdateEvent{
				Type:      EventError,
				Timestamp: time.Now(),
				Error:     err,
			})
		}
		return
	}

	err := execSelf(w.binary, os.Args)
	if err != nil {
		w.emit(UpdateEvent{
			Type:      EventError,
			Timestamp: time.Now(),
			Error:     err,
		})
	}
}
