package docker

import "io"

type Credential struct {
	UserName string
	Password string
}

const (
	ExitCodeTimeout   = 90408
	ExitMaxWriteLines = 90509
	ExitStopKeyword   = 90510
)

type ExitChan chan struct {
	ExitCode int
	Message  string
}

type Streams interface {
	In() *InStream
	Out() *OutStream
	Err() io.Writer
}
