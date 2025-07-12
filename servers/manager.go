package servers

import (
	"os"
	"os/exec"

	"luna/logger"
)

// Manager holds and manages all the servers.
type Manager struct {
	servers []Server
	log     logger.Logger
}

// NewManager creates a new server manager.
func NewManager(log logger.Logger) *Manager {
	return &Manager{log: log}
}

// AddServer adds a new server to the manager.
func (m *Manager) AddServer(server Server) {
	m.servers = append(m.servers, server)
}

// StartAll starts all registered servers.
func (m *Manager) StartAll() {
	for _, s := range m.servers {
		m.log.Info("Starting server", "name", s.Name())
		if err := s.Start(); err != nil {
			m.log.Fatal("Failed to start server", "name", s.Name(), "error", err)
		}
		m.log.Info("Server started successfully", "name", s.Name())
	}
}

// StopAll stops all registered servers.
func (m *Manager) StopAll() {
	for _, s := range m.servers {
		m.log.Info("Stopping server", "name", s.Name())
		if err := s.Stop(); err != nil {
			m.log.Error("Failed to stop server", "name", s.Name(), "error", err)
		}
		m.log.Info("Server stopped successfully", "name", s.Name())
	}
}

// --- Concrete Server Implementations ---

// GenericServer is a generic implementation of the Server interface.
type GenericServer struct {
	name    string
	command string
	args    []string
	dir     string
	cmd     *exec.Cmd
}

// NewGenericServer creates a new generic server.
func NewGenericServer(name, command string, args []string, dir string) *GenericServer {
	return &GenericServer{
		name:    name,
		command: command,
		args:    args,
		dir:     dir,
	}
}

// Name returns the server's name.
func (s *GenericServer) Name() string {
	return s.name
}

// Cmd returns the server's command.
func (s *GenericServer) Cmd() *exec.Cmd {
	return s.cmd
}

// Start starts the server.
func (s *GenericServer) Start() error {
	s.cmd = exec.Command(s.command, s.args...)
	if s.dir != "" {
		s.cmd.Dir = s.dir
	}
	s.cmd.Stdout = os.Stdout
	s.cmd.Stderr = os.Stderr
	return s.cmd.Start()
}

// Stop stops the server.
func (s *GenericServer) Stop() error {
	if s.cmd != nil && s.cmd.Process != nil {
		return s.cmd.Process.Kill()
	}
	return nil
}