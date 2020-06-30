package yaml

import (
	"fmt"
	"github.com/alydnh/go-micro-ci-common/utils"
)

type Build struct {
	Git             string            `yaml:"git"`
	Branch          string            `yaml:"branch"`
	Tag             string            `yaml:"tag"`
	TargetFolder    string            `yaml:"target"`
	Scripts         []string          `yaml:"scripts"`
	AdditionalFiles []string          `yaml:"additionalFiles"`
	Env             map[string]string `yaml:"env"`
}

func (y *Build) Initialize() error {
	if nil == y.AdditionalFiles {
		y.AdditionalFiles = make([]string, 0)
	}

	if nil == y.Env {
		y.Env = make(map[string]string)
	}

	if utils.EmptyOrWhiteSpace(y.Git) {
		return fmt.Errorf("git不能为空")
	}

	if !utils.EmptyOrWhiteSpace(y.Branch) &&
		!utils.EmptyOrWhiteSpace(y.Tag) {
		return fmt.Errorf("branch不能和tag共存")
	}

	if utils.EmptyOrWhiteSpace(y.TargetFolder) {
		return fmt.Errorf("target不能为空")
	}

	if utils.EmptyArray(y.Scripts) {
		return fmt.Errorf("scripts不能为空")
	}

	return nil
}

func (y Build) Clone() *Build {
	return &Build{
		Git:             y.Git,
		Branch:          y.Branch,
		Tag:             y.Tag,
		TargetFolder:    y.TargetFolder,
		Scripts:         append([]string{}, y.Scripts...),
		AdditionalFiles: append([]string{}, y.AdditionalFiles...),
		Env:             utils.CopyMap(y.Env).(map[string]string),
	}
}
