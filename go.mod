module github.com/alydnh/go-micro-ci-daemon

go 1.13

require (
	github.com/alydnh/go-micro-ci-common v0.0.0-20200706062435-f798f520ecf6
	github.com/docker/docker v17.12.0-ce-rc1.0.20200618181300-9dc6525e6118+incompatible
	github.com/docker/go-connections v0.4.0
	github.com/docker/go-metrics v0.0.1 // indirect
	github.com/docker/go-units v0.4.0
	github.com/docker/swarmkit v1.12.0 // indirect
	github.com/golang/protobuf v1.4.2
	github.com/hashicorp/go-memdb v1.2.1 // indirect
	github.com/micro/go-micro/v2 v2.9.0
	github.com/micro/go-plugins/registry/consul/v2 v2.8.0
	github.com/morikuni/aec v0.0.0-20170113033406-39771216ff4c
	github.com/opencontainers/selinux v1.5.2 // indirect
	github.com/pkg/errors v0.9.1
	github.com/sirupsen/logrus v1.6.0
	github.com/spf13/cobra v1.0.0 // indirect
	github.com/vbatts/tar-split v0.11.1 // indirect
	go.uber.org/atomic v1.5.0
	golang.org/x/net v0.0.0-20200520182314-0ba52f642ac2
)

replace golang.org/x/sys => golang.org/x/sys v0.0.0-20190813064441-fde4db37ae7a

replace github.com/Sirupsen/logrus v1.6.0 => github.com/sirupsen/logrus v1.6.0
