package yaml

import (
	"fmt"
	"github.com/alydnh/go-micro-ci-common/utils"
)

type Image struct {
	Name       string `yaml:"name"`
	Ref        string `yaml:"ref"`
	DockerFile string `yaml:"dockerFile"`
}

func (y *Image) String() string {
	if !utils.EmptyOrWhiteSpace(y.Ref) {
		return fmt.Sprintf("%s(%s)", y.Name, y.Ref)
	}
	return y.Name
}

func (y *Image) Validate() error {
	if utils.EmptyOrWhiteSpace(y.Name) {
		return fmt.Errorf("name不能为空")
	}
	return nil
}

func (y Image) Clone() *Image {
	return &Image{
		Name:       y.Name,
		Ref:        y.Ref,
		DockerFile: y.DockerFile,
	}
}
