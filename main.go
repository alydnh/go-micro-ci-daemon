package main

import (
	"fmt"
	"github.com/alydnh/go-micro-ci-common/yaml"
	"github.com/alydnh/go-micro-ci-daemon/docker"
	"github.com/alydnh/go-micro-ci-daemon/logs"
	"github.com/alydnh/go-micro-ci-daemon/registry"
	"github.com/micro/go-micro/v2"
	"github.com/micro/go-micro/v2/logger"
	_ "github.com/micro/go-plugins/registry/consul/v2"
	"github.com/sirupsen/logrus"
	"os"
	"time"
)

const defaultCIPath = "micro-ci/ci.yaml"

func main() {
	mainLog := logrus.New()
	mainLog.Out = os.Stdout
	mainLog.SetFormatter(&logrus.JSONFormatter{})
	initializeLogEntry := mainLog.WithField("type", "initialize")
	initializeLogEntry.Info("open:", defaultCIPath)
	ci, err := yaml.OpenCI(defaultCIPath, true)
	if nil != err {
		initializeLogEntry.WithField("file", defaultCIPath).Error(err)
		os.Exit(1)
	}
	initializeLogEntry.WithFields(logrus.Fields{
		"name": ci.CIName,
	}).Info("open successes.")

	initializeLogEntry.Info("check docker version")
	v, err := docker.GetDockerVersion()
	if nil != err {
		initializeLogEntry.Error(err)
		os.Exit(1)
	}
	initializeLogEntry.WithField("version", v).Info("docker version detected")

	initializeLogEntry.Info("starting service...")
	logger.DefaultLogger = logs.NewMicroLogrus(mainLog)
	serviceName := fmt.Sprintf("%s.%s", ci.Namespace, ci.CIName)
	service := micro.NewService(
		micro.Name(serviceName),
		micro.RegisterInterval(time.Second*30),
		micro.RegisterTTL(time.Second*60),
		micro.Version("default"),
		registry.Registry(ci.Registry),
		micro.AfterStart(func() error {
			initializeLogEntry.WithField("serviceName", serviceName).Info("Started")
			return nil
		}),
	)
	service.Init()
	if err = service.Run(); nil != err {
		initializeLogEntry.WithField("serviceName", serviceName).Error("service start failed with error:", err)
		os.Exit(2)
	}
}
