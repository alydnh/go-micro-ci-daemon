package yaml

type DataSourceFolder string
type DataSourceName string
type DataSource map[DataSourceName]map[MountHostPath]DataSourceFolder

func (ds DataSource) Clone() DataSource {
	dst := make(DataSource)
	for sourceName, hostSourcePair := range ds {
		dst[sourceName] = make(map[MountHostPath]DataSourceFolder)
		for hostPath, sourceFolder := range hostSourcePair {
			dst[sourceName][hostPath] = sourceFolder
		}
	}
	return dst
}
