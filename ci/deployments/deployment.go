package deployments

import (
	"context"
	"fmt"
	"github.com/alydnh/go-micro-ci-common/logs"
	"github.com/alydnh/go-micro-ci-common/utils"
	"github.com/alydnh/go-micro-ci-common/yaml"
	"github.com/alydnh/go-micro-ci-daemon/ci"
	"github.com/alydnh/go-micro-ci-daemon/docker"
	"github.com/sirupsen/logrus"
	"go.uber.org/atomic"
	"io/ioutil"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

func FromService(containerName string) (*Deployment, error) {
	service := ci.CI.GetService(containerName)
	fileName := filepath.Join(ci.MicroCIDeploymentFolderPath, service.Name())
	deployment, err := yaml.ReadDeployment(fileName)
	if nil != err && !os.IsNotExist(err) {
		return nil, err
	}
	return &Deployment{
		deployment:  deployment,
		service:     service,
		container:   docker.NewContainer(containerName, nil),
		imageRef:    ci.GetServiceImageRef(service),
		serviceEnv:  ci.GetServiceEnvironments(service),
		networkMode: ci.GetNetworkMode(),
	}, nil
}

var DeploymentNotChangedErr = fmt.Errorf("deployment has no changed with deployed docker container")

type Deployment struct {
	deployment        *yaml.Deployment
	service           *yaml.Service
	container         *docker.Container
	scope             *logs.LogrusScope
	imageRef          *string
	serviceEnv        map[string]string
	networkMode       string
	originalContainer *docker.Container
}

func (d *Deployment) SetLogrusScope(scope *logs.LogrusScope) {
	d.scope = scope
}

func (d Deployment) logrusScope() *logs.LogrusScope {
	if nil != d.scope {
		return d.scope
	}
	return &logs.LogrusScope{Entry: ci.DefaultLogger.WithField("deployment", d.service.Name())}
}

func (d *Deployment) Deploy() error {
	return d.logrusScope().Call(d.checkExists).Then(d.ensureImage).Then(d.run).Then(d.purge, d.originalContainer).Then(d.save).GetError()
}

func (d *Deployment) checkExists(ls *logs.LogrusScope) error {
	ls.Info("checking existed docker container...")
	if nil != d.deployment {
		if !d.deployment.Equals(d.service.Args, d.serviceEnv, d.service.ExposedPorts, d.service.Mounts) {
			ls = ls.WithField("reason", "Environment, Arguments, Port Mapping, Mounts")
		} else if utils.EmptyOrWhiteSpace(d.deployment.DockerImageID) || utils.EmptyOrWhiteSpace(d.deployment.ContainerID) {
			ls = ls.WithField("reason", "DOCKER_CONTAINER_INFO_NOT_FOUND")
		} else if exists, err := d.container.Exists(); nil != err {
			return err
		} else if !exists {
			ls = ls.WithField("reason", "DOCKER_CONTAINER_NOT_FOUND")
		} else if containerID, err := d.container.EnsureID(); nil != err {
			return err
		} else if strings.Compare(containerID, d.deployment.ContainerID) != 0 {
			ls = ls.WithField("reason", "DOCKER_CONTAINER_ID")
		} else if image, err := d.container.GetImageID(); nil != err {
			return err
		} else if strings.Compare(image, d.deployment.DockerImageID) != 0 {
			ls = ls.WithField("reason", "IMAGE_ID")
		} else if running, err := d.container.IsRunning(); nil != err {
			return err
		} else if !running {
			ls = ls.WithField("reason", "NOT_RUNNING")
		} else {
			return DeploymentNotChangedErr
		}
		d.deployment = nil
	}

	ls.Info("need redeployment")
	return nil
}

func (d *Deployment) run(ls *logs.LogrusScope) (err error) {
	d.container.SetEnv(d.serviceEnv)
	d.container.SetNetwork(&d.networkMode)
	if nil != d.service.ExposedPorts {
		for name, portYaml := range d.service.ExposedPorts {
			d.container.AddPort(string(name), portYaml.HostIP, portYaml.HostPort)
		}
	}

	if !utils.EmptyArray(d.service.Args) {
		d.container.SetArgs(d.service.Args)
	}

	var (
		originalContainerRunning bool
		consoleSize              = [2]uint{0, 0}
	)

	return ls.WithField("qualifiedName", d.container.QualifiedName()).Handle(func(ls *logs.LogrusScope) (result interface{}, err error) {
		existsContainerName, existsContainerID := utils.EmptyString, utils.EmptyString
		containerExists, err := d.container.Exists()
		if nil != err {
			return
		}
		if containerExists {
			existsContainerID = *d.container.ID()
			ls.Info("exists, rolling update...")
			if originalContainerRunning, err = d.container.IsRunning(); nil != err {
				return
			}
			existsContainerName = fmt.Sprintf("%s-%s", d.container.Name(), utils.ToDatetimeStringWithoutDash(time.Now()))
			ls.WithField("existsContainerName", existsContainerName).Info("renaming...")
			if err = d.container.StopAndRename(existsContainerName); nil != err {
				return
			}
			d.originalContainer = docker.NewContainer(existsContainerName, &existsContainerID)
		}

		if runtime.GOOS == "windows" {
			consoleSize[0], consoleSize[1] = docker.NewOutStream(logs.NewWriter(ls)).GetTtySize()
		}

		if err = d.container.EnsureContainer(false, true, true, consoleSize); nil != err {
			return
		}

		ctx, cancelFunc := context.WithCancel(context.Background())
		writeExitedChan := make(docker.ExitChan)
		var (
			errCh     chan error
			streams   docker.Streams
			exitChan  docker.ExitChan
			detach    context.CancelFunc
			closeFunc func()
		)
		timeouts := d.service.Assertions.GetTimeout()
		maxLogLines := d.service.Assertions.GetLines()
		attach := timeouts > 0 || maxLogLines > 0
		if attach {
			writer := logs.NewWriter(ls)
			lineCount := atomic.NewInt32(0)
			stdout := docker.CreateConditionWriter(writer, maxLogLines, lineCount, writeExitedChan, d.service.Assertions.Keywords()...)
			stderr := docker.CreateConditionWriter(writer, maxLogLines, lineCount, writeExitedChan, d.service.Assertions.Keywords()...)
			stdinWithCloser := ioutil.NopCloser(os.Stdin)
			streams = docker.NewStreams(stdinWithCloser, stdout, stderr)
			if timeouts > 0 {
				ctx, detach = context.WithDeadline(context.Background(), time.Now().Add(timeouts))
			}
			closeFunc, err = d.container.Attach(ctx, false, streams, stdinWithCloser, stdout, stderr, &errCh)
			if nil != err {
				return
			}
			if nil != closeFunc {
				defer closeFunc()
			}
		}

		ls.Info("running...")
		if exitChan, err = d.container.WaitExitOrRemoved(ctx, detach, writeExitedChan); nil != err {
			return
		}

		if err = d.container.EnsureContainerRunning(false, false, false, consoleSize); nil != err {
			if attach {
				cancelFunc()
				<-errCh
			}
			return
		}

		ls.Info(*d.container.ID())
		if attach && streams.Out().IsTerminal() {
			if err = docker.MonitorTtySize(ctx, streams, *d.container.ID(), false); nil != err {
				err = fmt.Errorf("error monitoring TTY size: %s", err.Error())
				return
			}
		}

		if nil != errCh {
			if err = <-errCh; nil != err {
				return
			}
		}

		containerExitResult := <-exitChan
		if containerExitResult.ExitCode != 0 {
			if containerExitResult.ExitCode == docker.ExitStopKeyword {
				if d.service.Assertions.IsSuccess(containerExitResult.Message) {
					ls.Info("container started")
					return
				}
			}

			err = fmt.Errorf("container start failed: %d %s", containerExitResult.ExitCode, containerExitResult.Message)
		}
		return

	}).OnError(func(lastErr error, ls *logs.LogrusScope) error {
		r := ls.WithField("runError", lastErr.Error()).
			Call(d.purge, d.container).
			Then(d.recover, d.originalContainer, originalContainerRunning)
		if r.HasError() {
			ls.Error(r.GetError())
		}
		return lastErr
	})
}

func (d *Deployment) purge(container *docker.Container, ls *logs.LogrusScope) error {
	if nil != container {
		ls.Info("purging container...")
		err := container.Purge()
		if nil != err {
			ls.Error("purge container failed with error:", err)
			return err
		}
	}
	return nil
}

func (d *Deployment) recover(container *docker.Container, runFlag bool, ls *logs.LogrusScope) (err error) {
	if nil != container {
		ls.WithFields(logrus.Fields{
			"originalName": container.QualifiedName(),
			"targetName":   d.container.Name(),
		}).Info("renaming...")
		err = container.Rename(d.container.Name())
		if nil != err {
			ls.Error(err)
		}

		if runFlag {
			ls.WithField("containerName", container.QualifiedName()).Info("recover starting...")
			err = container.EnsureContainerRunning(false, false, false, [2]uint{0, 0})
			if nil != err {
				ls.Error(err)
			}
		}
	}

	return
}

func (d *Deployment) ensureImage(ls *logs.LogrusScope) (err error) {
	ls.WithFields(logrus.Fields{
		"imageRef":   d.imageRef,
		"imageName:": d.service.Image.Name,
	}).Info("ensure docker image")
	if credential := ci.GetCredential(d.service); nil != credential {
		d.container.SetAuthConfig(credential.Host, credential.UserName, credential.Password)
	}
	_, err = d.container.EnsureImage(d.service.Image.Name, d.imageRef, logs.NewWriter(ls))
	return
}

func (d *Deployment) save(ls *logs.LogrusScope) (err error) {
	path := filepath.Join(ci.MicroCIDeploymentFolderPath, d.service.Name())
	ls.WithField("fileName", path).Info("save deployment")
	deployedImageID := utils.EmptyString
	if deployedImageID, err = d.container.GetImageID(); nil != err {
		return err
	}

	deployment := yaml.CreateDeployment(
		d.service.Name(),
		*d.container.ID(),
		d.container.Name(),
		deployedImageID,
		d.service.Args,
		d.serviceEnv,
		d.service.ExposedPorts,
		d.service.Mounts)
	err = deployment.SaveToFile(path)
	if nil == err {
		return err
	}
	d.deployment = deployment
	return
}
