package autoupdate

import (
	"io"
	"net/http"
	"os"
	"time"
)

// WatchConfig defines how the module interacts with remote servers and manages the update lifecycle.
type WatchConfig struct {
	// UpdateServer is the base endpoint of the update server.
	// Deprecated: Use Source instead.
	UpdateServer string
	// Source defines where to fetch updates from.
	Source Source
	// Version is the current version of the application.
	Version string
	// ServiceName is the name of the system service (systemd or Windows Service).
	// If set, the update script will restart the service instead of the binary.
	ServiceName string
	// Headers contains HTTP headers included with update requests.
	Headers map[string]string
	// Interval defines how frequently updates are checked.
	Interval time.Duration
	// Timeout defines the maximum duration for update requests.
	Timeout time.Duration
	// Target is the external executable path to launch and monitor.
	// Leave empty when using SelfUpdate.
	Target string
	// Args are arguments passed to the target executable.
	Args []string
	// SelfUpdate enables self-watching/self-updating mode.
	SelfUpdate bool
	// AutoRestart automatically restarts after a successful update.
	AutoRestart bool
	// Callbacks registers lifecycle/update hooks during initialization.
	Callbacks Callbacks
}

// Callbacks registers lifecycle/update hooks.
type Callbacks struct {
	OnUpdate func(UpdateEvent)
}

// UpdateEventType defines the type of event emitted by the watcher.
type UpdateEventType string

const (
	EventStarted UpdateEventType = "started"
	EventUpdate  UpdateEventType = "update"
	EventRestart UpdateEventType = "restart"
	EventStopped UpdateEventType = "stopped"
	EventError   UpdateEventType = "error"
)

// UpdateEvent represents a lifecycle or update event.
type UpdateEvent struct {
	Type      UpdateEventType
	Version   string
	Timestamp time.Time
	Error     error

	allow func()
}

// Allow authorizes the update to proceed.
// This is only relevant for EventUpdate events when listeners are present.
func (e UpdateEvent) Allow() {
	if e.allow != nil {
		e.allow()
	}
}

// UpdateWatcher is the central runtime controller.
type UpdateWatcher interface {
	// Watch returns a channel that receives update events.
	Watch() <-chan UpdateEvent
	// OnUpdate registers a callback function to be called when an update event occurs.
	OnUpdate(func(UpdateEvent))
	// Pipe returns a LauncherPipe for communication with the managed process.
	Pipe() LauncherPipe
	// Restart triggers a manual restart of the managed process.
	Restart() error
	// Shutdown stops the watcher and the managed process.
	Shutdown() error
}

// LauncherPipe provides streaming communication into the managed process stdin/stdout.
type LauncherPipe interface {
	io.ReadWriter
}

// Watch initializes the runtime watcher, starts update monitoring, and optionally launches or manages a target process.
func Watch(config WatchConfig) (UpdateWatcher, error) {
	binary := config.Target
	isSelf := config.SelfUpdate

	if isSelf {
		executable, err := os.Executable()
		if err != nil {
			return nil, err
		}
		binary = executable
	}

	nodeID := getOrCreateNodeID()

	source := config.Source
	if source == nil && config.UpdateServer != "" {
		source = &GenericSource{
			URL:     config.UpdateServer,
			AppID:   "",
			Headers: config.Headers,
			Client:  &http.Client{Timeout: config.Timeout},
		}
	}

	if gs, ok := source.(*GenericSource); ok {
		if gs.NodeID == "" {
			gs.NodeID = nodeID
		}
	}

	w := &watcher{
		config:      config,
		binary:      binary,
		args:        config.Args,
		stopped:     make(chan struct{}),
		restartChan: make(chan struct{}, 1),
		isSelf:      isSelf,
		source:      source,
		nodeID:      nodeID,
	}

	if config.Callbacks.OnUpdate != nil {
		w.OnUpdate(config.Callbacks.OnUpdate)
	}

	if isSelf {
		go w.pollLoop()
	} else {
		w.start()
	}

	return w, nil
}
