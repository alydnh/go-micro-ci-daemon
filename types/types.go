package types

import (
	"path/filepath"
)

var (
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
