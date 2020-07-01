package registry

import (
	"fmt"
	"github.com/alydnh/go-micro-ci-common/yaml"
	"github.com/micro/go-micro/v2"
	"github.com/micro/go-micro/v2/registry"
	"github.com/micro/go-plugins/registry/consul/v2"
)

var registries = map[string]func(r *yaml.Registry) micro.Option{
	"consul": func(r *yaml.Registry) micro.Option {
		return micro.Registry(consul.NewRegistry(
			registry.Addrs(fmt.Sprintf("%s:%d", r.Address, r.Port)),
			registry.Secure(r.UseSSL),
		))
	},
	"default": func(registry *yaml.Registry) micro.Option {
		return func(options *micro.Options) {}
	},
}

func Registry(registry *yaml.Registry) micro.Option {
	key := "default"
	if nil != registry {
		if _, ok := registries[registry.Type]; ok {
			key = registry.Type
		}
	}
	return registries[key](registry)
}
