package ci

import (
	"github.com/alydnh/go-micro-ci-common/utils"
	"github.com/alydnh/go-micro-ci-common/yaml"
	"github.com/sirupsen/logrus"
	"path/filepath"
)

var (
	DefaultLogger                   *logrus.Logger
	CI                              *yaml.CI
	MicroCIFolderPath               = "micro-ci"
	MicroCICacheFolderName          = ".ci"
	MicroCISourceFolderName         = "source"
	MicroCIArtifactManifestFileName = "ci.manifest"
	MicroCIMountFolderPath          = "mounts"
	MicroCIDataSourceFolderPath     = "data-source"
	MicroCIConfigFilePath           = filepath.Join(MicroCIFolderPath, "ci.yaml")
	MicroCIArtifactFolderPath       = filepath.Join(MicroCICacheFolderName, "artifacts")
	MicroCIDeploymentFolderPath     = filepath.Join(MicroCICacheFolderName, "deployments")
)

func GetServiceImageRef(service *yaml.Service) *string {
	if !utils.EmptyOrWhiteSpace(service.Image.Ref) {
		return &service.Image.Ref
	}
	return nil
}

func GetCredential(service *yaml.Service) *yaml.Credential {
	return CI.GetCredential(service.Image.Name, GetServiceImageRef(service))
}
