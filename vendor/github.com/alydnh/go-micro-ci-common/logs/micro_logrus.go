package logs

import (
	"github.com/micro/go-micro/v2/logger"
	"runtime"
	"strings"

	"github.com/sirupsen/logrus"
)

func NewMicroLogrus(rootLogger *logrus.Logger) logger.Logger {
	return logger.NewHelper(&microLogrus{
		l:    rootLogger.WithField("source", "micro"),
		opts: &logger.Options{},
	})
}

type microLogrus struct {
	l    *logrus.Entry
	opts *logger.Options
}

func (ml *microLogrus) Init(options ...logger.Option) error {
	for _, o := range options {
		o(ml.opts)
	}
	if nil != ml.opts.Fields {
		ml.l = ml.l.WithFields(ml.opts.Fields)
	}
	if nil != ml.opts.Context {
		ml.l = ml.l.WithContext(ml.opts.Context)
	}

	return nil
}

func (ml microLogrus) Options() logger.Options {
	return *ml.opts
}

func (ml microLogrus) Fields(fields map[string]interface{}) logger.Logger {
	return &microLogrus{
		l:    ml.l.WithFields(fields),
		opts: ml.opts,
	}
}

func (ml microLogrus) String() string {
	return "logrus"
}

func (ml microLogrus) parentCaller() string {
	pc, _, _, ok := runtime.Caller(ml.opts.CallerSkipCount)
	fn := runtime.FuncForPC(pc)
	if ok && fn != nil {
		return fn.Name()
	}

	return ""
}

func (ml microLogrus) Log(level logger.Level, v ...interface{}) {
	pc := ml.parentCaller()
	if strings.HasSuffix(pc, "Fatal") {
		ml.l.Fatal(v...)
	} else {
		ml.l.Info(v...)
	}
}

func (ml microLogrus) Logf(level logger.Level, format string, v ...interface{}) {
	pc := ml.parentCaller()
	if strings.HasSuffix(pc, "Fatalf") {
		ml.l.Fatalf(format, v...)
	} else {
		ml.l.Infof(format, v...)
	}
}
