package main

import (
	"github.com/alydnh/go-micro-ci-common/yaml"
	"github.com/alydnh/go-micro-ci-daemon/docker"
	"github.com/sirupsen/logrus"
	"os"
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
}
