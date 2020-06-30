package yaml

import (
	"fmt"
	"github.com/alydnh/go-micro-ci-common/utils"
	"strings"
)

type Credential struct {
	Host     string `yaml:"host"`
	UserName string `yaml:"userName"`
	Password string `yaml:"password"`
}

func (c Credential) Validate() error {
	if utils.EmptyOrWhiteSpace(c.Host) {
		return fmt.Errorf("host不能为空")
	}

	if utils.EmptyOrWhiteSpace(c.UserName) {
		return fmt.Errorf("userName不能为空")
	}

	if utils.EmptyOrWhiteSpace(c.Password) {
		return fmt.Errorf("password不能为空")
	}

	return nil
}

func (c Credential) Match(name string, ref *string) bool {
	text := strings.TrimPrefix(name, "http://")
	text = strings.TrimPrefix(name, "https://")
	texts := strings.Split(text, "/")
	if len(texts) > 1 {
		if strings.Compare(c.Host, texts[0]) == 0 {
			return true
		}
	}
	if nil != ref {
		return c.Match(*ref, nil)
	}

	return false
}

func (c Credential) Clone() *Credential {
	return &Credential{
		Host:     c.Host,
		UserName: c.UserName,
		Password: c.Password,
	}
}
