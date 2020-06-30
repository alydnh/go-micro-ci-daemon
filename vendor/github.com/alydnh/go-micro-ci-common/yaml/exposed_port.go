package yaml

import (
	"fmt"
	"github.com/alydnh/go-micro-ci-common/utils"
)

type PortName string
type ExposedPorts map[PortName]*ExposedPort

type ExposedPort struct {
	portName PortName
	HostIP   string `yaml:"hostIP"`
	HostPort int    `yaml:"hostPort"`
}

func (y *ExposedPort) Validate(portName PortName) error {
	y.portName = portName
	if utils.EmptyOrWhiteSpace(y.HostIP) {
		return fmt.Errorf("hostIP不能为空")
	}

	if y.HostPort < 0 || y.HostPort > 65535 {
		return fmt.Errorf("hostPort范围只能在1-65535之间")
	}

	return nil
}

func (y ExposedPort) Clone() *ExposedPort {
	return &ExposedPort{
		portName: y.portName,
		HostIP:   y.HostIP,
		HostPort: y.HostPort,
	}
}
