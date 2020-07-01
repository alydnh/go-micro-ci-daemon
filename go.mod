module github.com/alydnh/go-micro-ci-daemon

go 1.13

require (
	github.com/alydnh/go-micro-ci-common v0.0.0-20200701124547-ded1686aee57
	github.com/docker/docker v17.12.0-ce-rc1.0.20200618181300-9dc6525e6118+incompatible
	github.com/docker/go-connections v0.4.0
	github.com/docker/go-units v0.4.0
	github.com/micro/go-micro/v2 v2.9.0
	github.com/micro/go-plugins/registry/consul/v2 v2.8.0
	github.com/morikuni/aec v0.0.0-20170113033406-39771216ff4c
	github.com/pkg/errors v0.9.1
	github.com/sirupsen/logrus v1.6.0
	github.com/spf13/cobra v1.0.0 // indirect
	golang.org/x/net v0.0.0-20200520182314-0ba52f642ac2
)

replace golang.org/x/sys => golang.org/x/sys v0.0.0-20190813064441-fde4db37ae7a
