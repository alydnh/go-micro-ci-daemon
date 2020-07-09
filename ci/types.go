package ci

import (
	"fmt"
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
	GitCommit                       string
	GitTag                          string
	BuildDate                       string
	Version                         = "latest"
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

func GetServiceEnvironments(service *yaml.Service) map[string]string {
	env := utils.CopyMap(CI.CommonEnvs).(map[string]string)
	if !service.IsThird() && nil != service.Env {
		for key, value := range service.Env {
			env[key] = value
		}
	}
	return env
}

func GetNetworkMode() string {
	return fmt.Sprintf("%s-network", CI.CIName)
}

func GetVersion() string {
	version := Version
	if !utils.EmptyOrWhiteSpace(GitTag) {
		version = GitTag
	}

	if !utils.EmptyOrWhiteSpace(GitCommit) {
		version = fmt.Sprintf("%s-%s", version, GitCommit)
	}

	if !utils.EmptyOrWhiteSpace(BuildDate) {
		version = fmt.Sprintf("%s-%s", version, BuildDate)
	}
	return version
}
