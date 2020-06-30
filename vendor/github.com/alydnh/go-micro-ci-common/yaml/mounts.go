package yaml

type MountContainerPath string
type MountHostPath string
type Mounts map[MountContainerPath]MountHostPath
