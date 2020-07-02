package main

import (
	"fmt"
	"github.com/alydnh/go-micro-ci-common/logs"
	"github.com/alydnh/go-micro-ci-common/yaml"
	"github.com/alydnh/go-micro-ci-daemon/ci"
	"github.com/alydnh/go-micro-ci-daemon/ci/deployments"
	"github.com/alydnh/go-micro-ci-daemon/docker"
	"github.com/alydnh/go-micro-ci-daemon/registry"
	"github.com/micro/go-micro/v2"
	"github.com/micro/go-micro/v2/logger"
	"github.com/sirupsen/logrus"
	"os"
	"time"
)

const defaultCIPath = "micro-ci/ci.yaml"

func main() {
	ci.DefaultLogger = logrus.New()
	ci.DefaultLogger.Out = os.Stdout
	ci.DefaultLogger.SetFormatter(&logrus.TextFormatter{})
	mainLogScope := &logs.LogrusScope{Entry: ci.DefaultLogger.WithField("name", "main")}
	_ = mainLogScope.WithField("type", "initialize").Call(func(ls *logs.LogrusScope) (result interface{}, err error) {
		ls.Info("open:", defaultCIPath)
		ci.CI, err = yaml.OpenCI(defaultCIPath, true)
		if nil != err {
			ls.WithField("file", defaultCIPath).Error(err)
			return nil, err
		}
		ls.WithField("ciName", ci.CI.CIName).Info("open successes.")
		return
	}).Then(func(last interface{}, ls *logs.LogrusScope) (result interface{}, err error) {
		ls.Info("check docker version")
		v, err := docker.GetDockerVersion()
		if nil != err {
			return
		}
		ls.WithField("dockerVersion", v).Info("docker version detected")
		return
	}).Then(func(last interface{}, ls *logs.LogrusScope) (result interface{}, err error) {
		networkMode := fmt.Sprintf("%s-network", ci.CI.CIName)
		ls.WithField("dockerNetworkMode", networkMode).Info("prepare docker network...")
		return nil, docker.EnsureNetworkMode(networkMode, "bridge")
	}).Then(func(last interface{}, ls *logs.LogrusScope) (result interface{}, err error) {
		ls.Info("starting third services...")
		for _, containerName := range ci.CI.GetSequencedContainerNames() {
			service := ci.CI.GetService(containerName)
			if service.IsThird() {
				result := ls.WithField("containerName", containerName).Call(func(ls *logs.LogrusScope) (result interface{}, err error) {
					deployment, err := deployments.FromService(containerName)
					return nil, deployment.Deploy()
				})
				if result.HasError() {
					return nil, result.GetError()
				}
			}
		}
		return
	}).Then(func(last interface{}, ls *logs.LogrusScope) (result interface{}, err error) {
		ls.Info("starting service...")
		logger.DefaultLogger = logs.NewMicroLogrus(ci.DefaultLogger)
		serviceName := fmt.Sprintf("%s.%s", ci.CI.Namespace, ci.CI.CIName)
		service := micro.NewService(
			micro.Name(serviceName),
			micro.RegisterInterval(time.Second*30),
			micro.RegisterTTL(time.Second*60),
			micro.Version("default"),
			registry.Registry(ci.CI.Registry),
			micro.AfterStart(func() error {
				ls.WithField("serviceName", serviceName).Info("Started")
				return nil
			}),
		)
		service.Init()
		return nil, service.Run()
	}).OnError(func(err error, ls *logs.LogrusScope) error {
		ls.Error(err)
		os.Exit(2)
		return err
	})
}
