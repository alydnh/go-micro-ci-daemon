package deployments

import (
	"github.com/alydnh/go-micro-ci-common/utils"
	"github.com/alydnh/go-micro-ci-common/yaml"
	"github.com/alydnh/go-micro-ci-daemon/docker"
	"github.com/alydnh/go-micro-ci-daemon/types"
	"os"
	"path/filepath"
)

func FromService(containerName string, service *yaml.Service) (*Deployment, error) {
	fileName := filepath.Join(types.MicroCIDeploymentFolderPath, service.Name())
	deployment, err := yaml.ReadDeployment(fileName)
	if nil != err && !os.IsNotExist(err) {
		return nil, err
	}
	return &Deployment{
		deployment: deployment,
		service:    service,
		container:  docker.NewContainer(containerName, nil),
	}, nil
}

type Deployment struct {
	deployment *yaml.Deployment
	service    *yaml.Service
	container  *docker.Container
}

func (d *Deployment) Deploy(auth *yaml.Credential) error {
	if nil != auth {
		d.container.SetAuthConfig(auth.Host, auth.UserName, auth.Password)
	}
	if d.service.IsThird() {
		var ref *string = nil
		if !utils.EmptyOrWhiteSpace(d.service.Image.Ref) {
			ref = &d.service.Image.Ref
		}

		if _, err := d.container.EnsureImage(d.service.Image.Name, ref, os.Stdout); nil != err {
			return err
		}
	}
}
