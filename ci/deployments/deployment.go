package deployments

import (
	"github.com/alydnh/go-micro-ci-common/logs"
	"github.com/alydnh/go-micro-ci-common/yaml"
	"github.com/alydnh/go-micro-ci-daemon/ci"
	"github.com/alydnh/go-micro-ci-daemon/docker"
	"os"
	"path/filepath"
)

func FromService(containerName string) (*Deployment, error) {
	service := ci.CI.GetService(containerName)
	fileName := filepath.Join(ci.MicroCIDeploymentFolderPath, service.Name())
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
	scope      *logs.LogrusScope
}

func (d *Deployment) SetLogrusScope(scope *logs.LogrusScope) {
	d.scope = scope
}

func (d Deployment) logrusScope() *logs.LogrusScope {
	if nil != d.scope {
		return d.scope
	}
	return &logs.LogrusScope{Entry: ci.DefaultLogger.WithField("deployment", d.service.Name())}
}

func (d *Deployment) Deploy() error {
	if d.service.IsThird() {
		imageRef := ci.GetServiceImageRef(d.service)
		if credential := ci.GetCredential(d.service); nil != credential {
			d.container.SetAuthConfig(credential.Host, credential.UserName, credential.Password)
		}
		_ = d.logrusScope().WithField("ensureImage", d.service.Image.Name).Call(func(ls *logs.LogrusScope) (result interface{}, err error) {
			ls.Info("ensure docker image")
			_, err = d.container.EnsureImage(d.service.Image.Name, imageRef, ci.CreateLogrusScopeWriter(ls))
			return
		}).OnError(func(err error, ls *logs.LogrusScope) error {
			ls.Error(err)
			return err
		})
	}

	return nil
}
