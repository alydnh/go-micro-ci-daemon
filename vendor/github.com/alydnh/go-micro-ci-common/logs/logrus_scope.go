package logs

import "github.com/sirupsen/logrus"

type LogrusScopeCallHandler func(ls *LogrusScope) (result interface{}, err error)
type LogrusScopeThenHandler func(last interface{}, ls *LogrusScope) (result interface{}, err error)
type LogrusScopeErrorHandler func(err error, ls *LogrusScope) error

type LogrusScope struct {
	*logrus.Entry
}

func (ls LogrusScope) WithFields(fields logrus.Fields) *LogrusScope {
	return &LogrusScope{ls.Entry.WithFields(fields)}
}

func (ls LogrusScope) WithField(key string, value interface{}) *LogrusScope {
	return &LogrusScope{ls.Entry.WithField(key, value)}
}

func (ls *LogrusScope) Call(h LogrusScopeCallHandler) *LogrusScopeResult {
	result, err := h(ls)
	return &LogrusScopeResult{
		err:    err,
		Entry:  ls.Entry,
		result: result,
	}
}

type LogrusScopeResult struct {
	err    error
	result interface{}
	*logrus.Entry
}

func (r *LogrusScopeResult) Then(h LogrusScopeThenHandler) *LogrusScopeResult {
	if r.HasError() {
		return r
	}
	result, err := h(r.result, &LogrusScope{r.Entry})
	return &LogrusScopeResult{
		err:    err,
		Entry:  r.Entry,
		result: result,
	}
}

func (r LogrusScopeResult) WithFields(fields logrus.Fields) *LogrusScopeResult {
	return &LogrusScopeResult{
		err:    r.err,
		result: r.result,
		Entry:  r.Entry.WithFields(fields),
	}
}

func (r LogrusScopeResult) WithField(key string, value interface{}) *LogrusScopeResult {
	return &LogrusScopeResult{
		err:    r.err,
		result: r.result,
		Entry:  r.Entry.WithField(key, value),
	}
}

func (r *LogrusScopeResult) HasError() bool {
	return nil != r.err
}

func (r *LogrusScopeResult) GetError() error {
	return r.err
}

func (r *LogrusScopeResult) OnError(h LogrusScopeErrorHandler) error {
	if r.HasError() {
		return h(r.err, &LogrusScope{r.Entry})
	}

	return nil
}
