package ci

import (
	"github.com/alydnh/go-micro-ci-common/logs"
	"io"
)

func CreateLogrusScopeWriter(scope *logs.LogrusScope) io.Writer {
	return &logrusScopeWriter{
		scope:      scope,
		lineBuffer: make([]byte, 0, 1000),
	}
}

type logrusScopeWriter struct {
	scope      *logs.LogrusScope
	lineBuffer []byte
}

func (l *logrusScopeWriter) Write(p []byte) (n int, err error) {
	for _, c := range p {
		if c == '\n' {
			l.scope.Info(string(l.lineBuffer))
			l.lineBuffer = make([]byte, 0, 1000)
		} else {
			l.lineBuffer = append(l.lineBuffer, c)
		}
	}

	return len(p), nil
}
