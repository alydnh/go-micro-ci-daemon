package main

import (
	"github.com/alydnh/go-micro-ci-common/yaml"
	"github.com/alydnh/go-micro-ci-daemon/docker"
	"github.com/sirupsen/logrus"
	"os"
)

func main() {
	mainLog := logrus.New()
	mainLog.Out = os.Stdout
	mainLog.SetFormatter(&logrus.JSONFormatter{})
	initializeLogEntry := mainLog.WithField("type", "initialize")
	initializeLogEntry.Info("open micro-ci.yaml")
	_, err := yaml.OpenCI("micro-ci.yaml", true)
	if nil != err {
		initializeLogEntry.WithField("file", "micro-ci.yaml").Error(err)
		os.Exit(1)
	}
	initializeLogEntry.Info("check docker version")
	v, err := docker.GetDockerVersion()
	if nil != err {
		initializeLogEntry.Error(err)
		os.Exit(1)
	}
	initializeLogEntry.WithField("version", v).Info("docker version detected")
}
