package yaml

import (
	"fmt"
	"github.com/alydnh/go-micro-ci-common/utils"
	"gopkg.in/yaml.v3"
	"io/ioutil"
	"os"
)

func ReadDeployment(path string) (*Deployment, error) {
	d := &Deployment{}
	if bytes, err := ioutil.ReadFile(path); nil != err {
		return nil, err
	} else if err := yaml.Unmarshal(bytes, d); nil != err {
		return nil, err
	}
	for name, port := range d.ExposedPorts {
		if err := port.Validate(name); nil != err {
			return nil, fmt.Errorf("port[%s]:%s", name, err.Error())
		}
	}
	return d, nil
}

func CreateDeployment(serviceName, containerID, containerName, dockerImageID string, args []string, env map[string]string, exposedPorts ExposedPorts, mounts Mounts) *Deployment {
	return &Deployment{
		Args:          args,
		Env:           env,
		ContainerID:   containerID,
		ContainerName: containerName,
		DockerImageID: dockerImageID,
		ServiceName:   serviceName,
		ExposedPorts:  exposedPorts,
		Mounts:        mounts,
	}
}

type Deployment struct {
	Args          []string          `yaml:"args"`
	Env           map[string]string `yaml:"env"`
	ServiceName   string            `yaml:"serviceName"`
	ContainerName string            `yaml:"containerName"`
	ContainerID   string            `yaml:"containerID"`
	DockerImageID string            `yaml:"dockerImageID"`
	ExposedPorts  ExposedPorts      `yaml:"exposedPorts"`
	Mounts        Mounts            `yaml:"mounts"`
}

func (d Deployment) Equals(args []string, env map[string]string, exposedPorts ExposedPorts, mounts Mounts) bool {
	if equal, _ := utils.EqualValues(args, d.Args); !equal {
		return false
	}
	if equal, _ := utils.EqualValues(env, d.Env); !equal {
		return false
	}
	if equal, _ := utils.EqualValues(exposedPorts, d.ExposedPorts); !equal {
		return false
	}
	if equal, _ := utils.EqualValues(mounts, d.Mounts); !equal {
		return false
	}

	return true
}

func (d Deployment) SaveToFile(fileName string) error {
	if bytes, err := yaml.Marshal(d); nil != err {
		return err
	} else {
		return ioutil.WriteFile(fileName, bytes, os.FileMode(0775))
	}
}
