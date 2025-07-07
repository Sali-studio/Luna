package servers

import "os/exec"

// Server defines the interface for a manageable server process.
type Server interface {
	Start() error
	Stop() error
	Name() string
	Cmd() *exec.Cmd
}
