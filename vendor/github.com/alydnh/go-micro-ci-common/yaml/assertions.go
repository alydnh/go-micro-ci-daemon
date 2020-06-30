package yaml

import (
	"fmt"
	"github.com/alydnh/go-micro-ci-common/utils"
	"strings"
	"time"
)

type Assertions struct {
	Successes []string `yaml:"successes"`
	Fails     []string `yaml:"fails"`
	Lines     int      `yaml:"lines"`
	Timeout   string   `yaml:"timeout"`
}

func (a *Assertions) Initialize(doValidator bool) error {
	if utils.EmptyArray(a.Successes) {
		a.Successes = make([]string, 0)
	}

	if utils.EmptyArray(a.Fails) {
		a.Fails = make([]string, 0)
	}

	if doValidator {
		if !utils.EmptyOrWhiteSpace(a.Timeout) {
			if _, err := time.ParseDuration(a.Timeout); nil != err {
				return fmt.Errorf("timeout 不是一个有效的时间值")
			}
		}

		if a.Lines < 0 {
			return fmt.Errorf("lines 不能小于0")
		}
	}

	return nil
}

func (a Assertions) GetTimeout() time.Duration {
	if utils.EmptyOrWhiteSpace(a.Timeout) {
		return time.Second * 10
	}

	duration, err := time.ParseDuration(a.Timeout)
	if nil != err {
		panic(err)
	}
	return duration
}

func (a Assertions) GetLines() int {
	if a.Lines <= 0 {
		return 100
	}
	return a.Lines
}

func (a Assertions) Clone() *Assertions {
	return &Assertions{
		Successes: append([]string{}, a.Successes...),
		Fails:     append([]string{}, a.Fails...),
		Lines:     a.Lines,
		Timeout:   a.Timeout,
	}
}

func (a Assertions) Keywords() []string {
	return append(a.Successes, a.Fails...)
}

func (a Assertions) IsSuccess(keyword string) bool {
	return utils.Any(a.Successes, func(text string) bool {
		return strings.Compare(text, keyword) == 0
	})
}
