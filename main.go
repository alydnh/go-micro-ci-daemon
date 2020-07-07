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
	"strings"
	"time"
)

const defaultCIPath = "micro-ci/ci.yaml"

var ErrNetworkCreated = fmt.Errorf("ERROR_NETWORK_CREATED")

func main() {
	ci.DefaultLogger = logrus.New()
	ci.DefaultLogger.Out = os.Stdout
	ci.DefaultLogger.SetFormatter(&logrus.TextFormatter{})

	if v, ok := os.LookupEnv("ENABLE_LOGGER_CALLER"); ok && strings.Compare(v, "true") == 0 {
		ci.DefaultLogger.ReportCaller = true
	}

	mainLogScope := &logs.LogrusScope{Entry: ci.DefaultLogger.WithField("name", "main")}
	_ = mainLogScope.WithField("type", "initialize").
		Call(initializeCI).
		Then(checkDockerVersion).
		Then(ensureDockerNetwork).
		Then(deploy).
		Then(run).
		OnError(func(err error, ls *logs.LogrusScope) error {
			if err == ErrNetworkCreated {
				ls.Info("docker network:", ci.GetNetworkMode(), "created, please delete this container and run container again with --network=", ci.GetNetworkMode())
				os.Exit(1)
			}

			ls.Error(err)
			os.Exit(2)
			return err
		})
}

func initializeCI(ls *logs.LogrusScope) (err error) {
	ls.Info("open:", defaultCIPath)
	ci.CI, err = yaml.OpenCI(defaultCIPath, true)
	if nil != err {
		ls.WithField("file", defaultCIPath).Error(err)
		return err
	}
	ls.WithField("ciName", ci.CI.CIName).Info("open successes.")
	return
}

func checkDockerVersion(ls *logs.LogrusScope) (err error) {
	ls.Info("check docker version")
	v, err := docker.GetDockerVersion()
	if nil != err {
		return
	}
	ls.WithField("dockerVersion", v).Info("docker version detected")
	return
}

func ensureDockerNetwork(ls *logs.LogrusScope) error {
	networkMode := ci.GetNetworkMode()
	ls.WithField("dockerNetworkMode", networkMode).Info("prepare docker network...")
	created, err := docker.EnsureNetworkMode(networkMode, "bridge")
	if nil != err {
		return err
	}
	if created {
		return ErrNetworkCreated
	}

	return nil
}

func deploy(ls *logs.LogrusScope) (err error) {
	ls.Info("starting third services...")
	for _, containerName := range ci.CI.GetSequencedContainerNames() {
		service := ci.CI.GetService(containerName)
		if service.IsThird() {
			result := ls.WithField("containerName", containerName).Handle(func(ls *logs.LogrusScope) (result interface{}, err error) {
				deployment, err := deployments.FromService(containerName)
				if nil != err {
					return nil, err
				}
				return nil, deployment.Deploy()
			})
			if result.HasError() {
				return result.GetError()
			}
		}
	}
	return
}

func run(ls *logs.LogrusScope) (err error) {
	ls.Info("deploying service...")
	logger.DefaultLogger = logs.NewMicroLogrus(ci.DefaultLogger)
	serviceName := fmt.Sprintf("%s.%s", ci.CI.Namespace, ci.CI.CIName)
	service := micro.NewService(
		micro.Name(serviceName),
		micro.RegisterInterval(time.Second*30),
		micro.RegisterTTL(time.Second*60),
		registry.Registry(ci.CI.Registry),
		micro.AfterStart(func() error {
			ls.WithField("serviceName", serviceName).Info("Started")
			return nil
		}),
	)
	service.Init()
	return service.Run()
}
