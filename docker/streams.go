package docker

import (
	"context"
	"github.com/alydnh/go-micro-ci-common/utils"
	"go.uber.org/atomic"
	"io"
	"strings"
)

func CreateConditionWriter(innerWriter io.Writer, maxLines int, lineCounter *atomic.Int32, exitChan ExitChan, stopKeywords ...string) *ConditionWriter {
	w := &ConditionWriter{
		innerWriter:   innerWriter,
		maxLines:      int32(maxLines),
		exitChan:      exitChan,
		lineCounter:   lineCounter,
		lastLineBytes: make([]byte, 0),
		stopKeywords:  make(map[string]bool),
	}

	for _, keyword := range stopKeywords {
		w.stopKeywords[keyword] = true
	}

	return w
}

type ConditionWriter struct {
	innerWriter   io.Writer
	maxLines      int32
	lineCounter   *atomic.Int32
	exitChan      ExitChan
	lastLineBytes []byte
	stopKeywords  map[string]bool
	cancelWait    context.CancelFunc
}

func (w *ConditionWriter) Write(p []byte) (n int, err error) {
	if n, err = w.innerWriter.Write(p); nil == err {
		for _, c := range p {
			if c == '\n' {
				for keyword := range w.stopKeywords {
					if strings.Contains(string(w.lastLineBytes), keyword) {
						w.exitChan <- struct {
							ExitCode int
							Message  string
						}{
							ExitStopKeyword,
							keyword,
						}
						_, _ = w.innerWriter.Write([]byte{'\n'})
						return
					}
				}
				w.lastLineBytes = make([]byte, 0)
				if w.maxLines > 0 && w.lineCounter.Inc() > w.maxLines {
					w.exitChan <- struct {
						ExitCode int
						Message  string
					}{
						ExitMaxWriteLines,
						utils.EmptyString,
					}
					_, _ = w.innerWriter.Write([]byte{'\n'})
					return
				}

			} else {
				w.lastLineBytes = append(w.lastLineBytes, c)
			}
		}
	}

	return
}

func NewStreams(in io.ReadCloser, out io.Writer, err io.Writer) *streams {
	s := &streams{err: err}
	if nil != in {
		s.in = NewInStream(in)
	}
	if nil != out {
		s.out = NewOutStream(out)
	}
	return s
}

type streams struct {
	in  *InStream
	out *OutStream
	err io.Writer
}

func (s *streams) In() *InStream {
	return s.in
}

func (s *streams) Out() *OutStream {
	return s.out
}

func (s *streams) Err() io.Writer {
	return s.out
}
