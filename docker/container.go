package docker

import (
	"archive/tar"
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/alydnh/go-micro-ci-common/utils"
	"github.com/alydnh/go-micro-ci-daemon/docker/image/build"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/events"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/api/types/versions"
	"github.com/docker/docker/builder/dockerignore"
	"github.com/docker/docker/cli"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/archive"
	"github.com/docker/docker/pkg/fileutils"
	"github.com/docker/docker/pkg/jsonmessage"
	"github.com/docker/docker/pkg/progress"
	"github.com/docker/docker/pkg/streamformatter"
	"github.com/docker/docker/pkg/stringid"
	"github.com/docker/go-connections/nat"
	"github.com/pkg/errors"
	"io"
	"io/ioutil"
	"net/http/httputil"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"
)

func NewContainer(name string, id *string) *Container {
	return &Container{
		name:   name,
		ports:  make([]*portInfo, 0, 1),
		mounts: make(map[string]string),
		envs:   make(map[string]string),
		id:     id,
	}
}

type Container struct {
	id          *string
	name        string
	ports       []*portInfo
	networkMode *string
	mounts      map[string]string
	envs        map[string]string
	args        []string
	image       *struct {
		name string
		ref  *string
	}
	registryAuth string
}

func (c *Container) SetAuthConfig(host, userName, password string) *Container {
	config := types.AuthConfig{
		Username:      userName,
		Password:      password,
		ServerAddress: host,
	}
	bytes, _ := json.Marshal(config)
	c.registryAuth = base64.URLEncoding.EncodeToString(bytes)
	return c
}

func (c *Container) AddPort(portName, hostIP string, hostPort int) {
	c.ports = append(c.ports, &portInfo{
		portName: portName,
		hostIP:   hostIP,
		hostPort: strconv.Itoa(hostPort),
	})
}

func (c *Container) SetEnv(env map[string]string) *Container {
	for key, value := range env {
		c.envs[key] = value
	}
	return c
}

func (c *Container) SetArgs(args []string) *Container {
	c.args = args
	return c
}

func (c *Container) SetMounts(mounts map[string]string) *Container {
	c.mounts = mounts
	return c
}

func (c *Container) SetNetwork(networkMode *string) *Container {
	c.networkMode = networkMode
	return c
}

func (c Container) QualifiedName() string {
	if nil == c.id {
		return c.name
	}

	return fmt.Sprintf("%s(%s)", c.name, *c.id)
}

func (c Container) Name() string {
	return c.name
}

func (c *Container) Exists() (bool, error) {
	if id, err := c.EnsureID(); nil != err {
		return false, err
	} else {
		return !utils.EmptyOrWhiteSpace(id), nil
	}
}

func (c *Container) IsRunning() (bool, error) {
	id, err := c.EnsureID()
	if nil != err {
		return false, err
	}
	if utils.EmptyOrWhiteSpace(id) {
		return false, nil
	}

	if json, err := c.inspect(); nil != err {
		return false, err
	} else {
		return json.State.Running, nil
	}
}

func (c Container) ID() *string {
	return c.id
}

func (c *Container) BuildImage(name string, credentials map[string]*Credential, dockerBuildContent, contextDir string, out io.Writer) error {

	if dockerClient, err := client.NewEnvClient(); nil != err {
		return err
	} else if contextDir, relDockerfile, err := build.GetContextFromLocalDir(contextDir, "-"); nil != err {
		return err
	} else {
		relDockerfile = archive.CanonicalTarNameForPath(relDockerfile)
		f, err := os.Open(filepath.Join(contextDir, ".dockerignore"))
		if err != nil && !os.IsNotExist(err) {
			return err
		}
		defer f.Close()

		var excludes []string
		if err == nil {
			excludes, err = dockerignore.ReadAll(f)
			if err != nil {
				return err
			}
		}

		if err := build.ValidateContextDirectory(contextDir, excludes); err != nil {
			return errors.Errorf("Error checking context: '%s'.", err)
		}

		if err := build.ValidateContextDirectory(contextDir, excludes); err != nil {
			return errors.Errorf("Error checking context: '%s'.", err)
		}

		if keep, _ := fileutils.Matches(".dockerignore", excludes); keep {
			excludes = append(excludes, "!.dockerignore")
			excludes = append(excludes, "!"+relDockerfile)
		}

		dockerfileCtx := ioutil.NopCloser(strings.NewReader(dockerBuildContent))

		if buildCtx, err := archive.TarWithOptions(contextDir, &archive.TarOptions{
			Compression:     0,
			ExcludePatterns: excludes,
		}); err != nil {
			return err
		} else if buildCtx, relDockerfile, err = addDockerfileToBuildContext(dockerfileCtx, buildCtx); nil != err {
			return err
		} else {
			// Setup an upload progress bar
			progressOutput := streamformatter.NewProgressOutput(out)
			stream := NewOutStream(out)
			if !stream.IsTerminal() {
				progressOutput = &lastProgressOutput{output: progressOutput}
			}
			var body io.Reader = progress.NewProgressReader(buildCtx, progressOutput, 0, "", "发送数据给docker daemon")
			authConfigs := make(map[string]types.AuthConfig)
			for host, credential := range credentials {
				authConfigs[host] = types.AuthConfig{
					Username:      credential.UserName,
					Password:      credential.Password,
					ServerAddress: host,
				}
			}
			buildOptions := types.ImageBuildOptions{
				Tags:        []string{name},
				Remove:      true,
				AuthConfigs: authConfigs,
				NoCache:     true,
				Dockerfile:  relDockerfile,
			}

			if response, err := dockerClient.ImageBuild(context.Background(), body, buildOptions); nil != err {
				return err
			} else {
				defer response.Body.Close()
				stream := NewOutStream(out)
				if err := DisplayJSONMessagesStream(response.Body, out, stream.FD(), stream.IsTerminal(), nil); nil != err {
					if jsonErr, ok := err.(*jsonmessage.JSONError); ok {
						// If no error code is set, default to 1
						if jsonErr.Code == 0 {
							jsonErr.Code = 1
						}

						return cli.StatusError{Status: jsonErr.Message, StatusCode: jsonErr.Code}
					}

					return err
				}
			}
		}
	}

	return nil
}

func (c *Container) EnsureImageByID(id string) error {
	if dockerClient, err := client.NewEnvClient(); nil != err {
		return err
	} else if image, _, err := dockerClient.ImageInspectWithRaw(context.Background(), id); nil != err {
		return err
	} else if len(image.RepoTags) == 0 {
		return fmt.Errorf("镜像无可用标签")
	} else {
		c.image = &struct {
			name string
			ref  *string
		}{name: image.RepoTags[0], ref: nil}
	}
	return nil
}

func (c *Container) GetImageID() (string, error) {
	if exists, err := c.Exists(); nil != err {
		return utils.EmptyString, err
	} else if !exists {
		return utils.EmptyString, fmt.Errorf("容器不存在")
	}

	if json, err := c.inspect(); nil != err {
		return utils.EmptyString, err
	} else {
		return json.Image, nil
	}
}

func (c *Container) EnsureImage(name string, ref *string, out io.Writer) (string, error) {
	filterArgs := filters.NewArgs()
	filterArgs.Add("reference", name)
	referenceName := name
	if nil != ref {
		referenceName = *ref
	}
	if dockerClient, err := client.NewEnvClient(); nil != err {
		return utils.EmptyString, err
	} else if summary, err := dockerClient.ImageList(context.Background(), types.ImageListOptions{
		Filters: filterArgs,
	}); nil != err {
		return utils.EmptyString, err
	} else if len(summary) > 0 {
		c.image = &struct {
			name string
			ref  *string
		}{
			name: name,
			ref:  ref,
		}
		return summary[0].ID, nil
	} else if reader, err := dockerClient.ImagePull(context.Background(), referenceName, types.ImagePullOptions{
		RegistryAuth: c.registryAuth,
	}); nil != err {
		return utils.EmptyString, err
	} else {
		defer reader.Close()
		stream := NewOutStream(out)
		if err = DisplayJSONMessagesToStream(reader, stream, nil); nil != err {
			return utils.EmptyString, err
		}
		return c.EnsureImage(name, ref, out)
	}
}

func (c *Container) EnsureID() (string, error) {
	if nil != c.id {
		return *c.id, nil
	}

	filterArgs := filters.NewArgs()
	filterArgs.Add("name", c.name)
	if dockerClient, err := client.NewEnvClient(); nil != err {
		return utils.EmptyString, err
	} else if containers, err := dockerClient.ContainerList(context.Background(), types.ContainerListOptions{
		Filters: filterArgs,
		All:     true,
	}); nil != err {
		return utils.EmptyString, err
	} else if !utils.EmptyArray(containers) {
		v, ok := utils.FirstOrDefault(containers, func(tc types.Container) bool {
			return utils.Any(tc.Names, func(name string) bool {
				return strings.Compare(strings.TrimLeft(name, "/"), c.name) == 0
			})
		})
		if ok {
			id := v.(types.Container).ID
			c.id = &id
			return *c.id, nil
		}
	}
	return utils.EmptyString, nil
}

func (c *Container) EnsureContainer(attachStdin, attachStdout, attachStderr bool, consoleSize [2]uint) error {
	containerID, err := c.EnsureID()
	if nil != err {
		return err
	}

	if utils.EmptyOrWhiteSpace(containerID) {
		ctx := context.Background()
		dockerClient, err := client.NewEnvClient()
		if nil != err {
			return err
		}
		containerConfig, hostConfig := c.getRunningConfigs()
		containerConfig.StdinOnce = true
		containerConfig.AttachStdout = attachStdout
		containerConfig.AttachStderr = attachStderr
		containerConfig.AttachStdin = attachStdin
		hostConfig.ConsoleSize = consoleSize
		if resp, err := dockerClient.ContainerCreate(ctx, &containerConfig, &hostConfig, nil, c.name); nil != err {
			return err
		} else {
			c.id = &resp.ID
		}
	}

	return nil
}

func (c *Container) EnsureContainerRunning(attachStdin, attachStdout, attachStderr bool, consoleSize [2]uint) error {
	containerID, err := c.EnsureID()
	if nil != err {
		return err
	}

	running, err := c.IsRunning()
	if nil != err {
		return err
	}

	ctx := context.Background()
	dockerClient, err := client.NewEnvClient()
	if nil != err {
		return err
	}

	if !utils.EmptyOrWhiteSpace(containerID) && !running {
		if err := dockerClient.ContainerStart(ctx, containerID, types.ContainerStartOptions{}); nil != err {
			return err
		}
		return nil
	}

	if err := c.EnsureContainer(attachStdin, attachStdout, attachStderr, consoleSize); nil != err {
		return err
	}

	return c.EnsureContainerRunning(attachStdin, attachStdout, attachStderr, consoleSize)
}

func (c *Container) Rename(newName string) error {
	exists, err := c.Exists()
	if nil != err {
		return err
	}
	if !exists {
		return fmt.Errorf("容器:%s不存在", c.QualifiedName())
	}

	if strings.Compare(newName, c.name) != 0 {
		dockerClient, err := client.NewEnvClient()
		if nil != err {
			return err
		}

		if err = dockerClient.ContainerRename(context.Background(), *c.id, newName); nil != err {
			return err
		}
		c.name = newName
	}

	return nil
}

func (c *Container) StopAndRename(newName string) error {
	exists, err := c.Exists()
	if nil != err {
		return err
	}
	if !exists {
		return nil
	}

	running, err := c.IsRunning()
	if nil != err {
		return err
	}

	dockerClient, err := client.NewEnvClient()
	if nil != err {
		return err
	}

	if running {
		if err = dockerClient.ContainerStop(context.Background(), *c.id, nil); nil != err {
			return err
		}
	}

	if strings.Compare(newName, c.name) != 0 {
		if err = dockerClient.ContainerRename(context.Background(), *c.id, newName); nil != err {
			return err
		}
	}

	c.id = nil
	return nil
}

func (c *Container) Purge() error {
	containerID := c.id
	if err := c.StopAndRename(c.name); nil != err {
		return err
	}

	dockerClient, err := client.NewEnvClient()
	if nil != err {
		return err
	}

	if err := dockerClient.ContainerRemove(context.Background(), *containerID, types.ContainerRemoveOptions{
		RemoveVolumes: true,
		Force:         true,
	}); nil != err {
		return err
	}

	c.id = nil
	return nil
}

func (c *Container) Attach(ctx context.Context, tty bool, streams Streams, stdin io.ReadCloser, stdOut, stderr io.Writer, errCh *chan error) (close func(), errAttach error) {
	id, err := c.EnsureID()
	if nil != err {
		return nil, err
	}

	dockerClient, errAttach := client.NewEnvClient()
	if nil != errAttach {
		return nil, errAttach
	}

	options := types.ContainerAttachOptions{
		Stream: true,
		Stdin:  nil != stdin,
		Stdout: nil != stdOut,
		Stderr: nil != stderr,
	}

	resp, errAttach := dockerClient.ContainerAttach(ctx, id, options)
	if nil != errAttach && errAttach != httputil.ErrPersistEOF {
		return nil, errAttach
	}

	*errCh = PromiseGo(func() error {
		if errHijack := holdHijackedConnection(ctx, streams, false, stdin, stdOut, stderr, resp); errHijack != nil {
			return errHijack
		}
		return errAttach
	})

	return resp.Close, nil
}

func (c *Container) WaitExitOrRemoved(ctx context.Context, detach context.CancelFunc, writerExitChan ExitChan) (exitChan ExitChan, err error) {
	id, err := c.EnsureID()
	if nil != err {
		return nil, err
	}

	dockerClient, err := client.NewEnvClient()
	if nil != err {
		return nil, err
	}

	var removeErr error
	exitChan = make(ExitChan)
	exitResult := struct {
		ExitCode int
		Message  string
	}{125, utils.EmptyString}

	// Get events via Events API
	f := filters.NewArgs()
	f.Add("type", "container")
	f.Add("container", id)
	options := types.EventsOptions{
		Filters: f,
	}
	eventCtx, cancel := context.WithCancel(ctx)
	eventChan, errChan := dockerClient.Events(eventCtx, options)

	eventProcessor := func(e events.Message) bool {
		stopProcessing := false
		switch e.Status {
		case "die":
			if v, ok := e.Actor.Attributes["exitCode"]; ok {
				code, err := strconv.Atoi(v)
				if err == nil {
					exitResult.ExitCode = code
				}
			}

			// TODO: if c.autoRemove
			if true {
				stopProcessing = true
			} else {
				// If we are talking to an older daemon, `AutoRemove` is not supported.
				// We need to fall back to the old behavior, which is client-side removal
				if versions.LessThan(dockerClient.ClientVersion(), "1.25") {
					go func() {
						removeErr = dockerClient.ContainerRemove(ctx, id, types.ContainerRemoveOptions{RemoveVolumes: true})
						if removeErr != nil {
							cancel() // cancel the event Q
						}
					}()
				}
			}
		case "detach":
			exitResult.ExitCode = 0
			stopProcessing = true
		case "destroy":
			stopProcessing = true
		}
		return stopProcessing
	}

	go func() {
		defer func() {
			if nil != detach {
				detach()
			}
			exitChan <- exitResult // must always send an exit code or the caller will block
			cancel()
		}()

		for {
			select {
			case <-eventCtx.Done():
				if removeErr != nil {
					return
				}
			case evt := <-eventChan:
				if eventProcessor(evt) {
					return
				}
			case <-errChan:
				return
			case writerExitResult := <-writerExitChan:
				exitResult.ExitCode = writerExitResult.ExitCode
				exitResult.Message = writerExitResult.Message
				return
			}
		}
	}()

	return exitChan, nil
}

func holdHijackedConnection(ctx context.Context, streams Streams, tty bool, inputStream io.ReadCloser, outputStream, errorStream io.Writer, resp types.HijackedResponse) error {
	var (
		err         error
		restoreOnce sync.Once
	)
	if inputStream != nil && tty {
		if err := setRawTerminal(streams); err != nil {
			return err
		}
		defer func() {
			restoreOnce.Do(func() {
				restoreTerminal(streams, inputStream)
			})
		}()
	}

	receiveStdout := make(chan error, 1)
	if outputStream != nil || errorStream != nil {
		go func() {
			// When TTY is ON, use regular copy
			if tty && outputStream != nil {
				_, err = io.Copy(outputStream, resp.Reader)
				// we should restore the terminal as soon as possible once connection end
				// so any following print messages will be in normal type.
				if inputStream != nil {
					restoreOnce.Do(func() {
						restoreTerminal(streams, inputStream)
					})
				}
			} else {
				_, err = StdCopy(ctx, outputStream, errorStream, resp.Reader)
			}

			receiveStdout <- err
		}()
	}

	stdinDone := make(chan struct{})
	go func() {
		if inputStream != nil {
			io.Copy(resp.Conn, inputStream)
			// we should restore the terminal as soon as possible once connection end
			// so any following print messages will be in normal type.
			if tty {
				restoreOnce.Do(func() {
					restoreTerminal(streams, inputStream)
				})
			}
		}

		_ = resp.CloseWrite()
		close(stdinDone)
	}()

	select {
	case err := <-receiveStdout:
		if err != nil {
			return err
		}
	case <-stdinDone:
		if outputStream != nil || errorStream != nil {
			select {
			case err := <-receiveStdout:
				if err != nil {
					return err
				}
			case <-ctx.Done():
			}
		}
	case <-ctx.Done():
	}

	return nil
}

func setRawTerminal(streams Streams) error {
	if err := streams.In().SetRawTerminal(); err != nil {
		return err
	}
	return streams.Out().SetRawTerminal()
}

func restoreTerminal(streams Streams, in io.Closer) error {
	streams.In().RestoreTerminal()
	streams.Out().RestoreTerminal()
	// WARNING: DO NOT REMOVE THE OS CHECKS !!!
	// For some reason this Close call blocks on darwin..
	// As the client exits right after, simply discard the close
	// until we find a better solution.
	//
	// This can also cause the client on Windows to get stuck in Win32 CloseHandle()
	// in some cases. See https://github.com/docker/docker/issues/28267#issuecomment-288237442
	// Tracked internally at Microsoft by VSO #11352156. In the
	// Windows case, you hit this if you are using the native/v2 console,
	// not the "legacy" console, and you start the client in a new window. eg
	// `start docker run --rm -it microsoft/nanoserver cmd /s /c echo foobar`
	// will hang. Remove start, and it won't repro.
	if in != nil && runtime.GOOS != "darwin" && runtime.GOOS != "windows" {
		return in.Close()
	}
	return nil
}

func (c Container) getRunningConfigs() (container.Config, container.HostConfig) {
	var (
		mounts      []mount.Mount = nil
		envs        []string      = nil
		portMaps    nat.PortMap
		networkMode container.NetworkMode
	)

	exposedPorts := make(nat.PortSet)
	if !utils.EmptyArray(c.ports) {
		portMaps = make(nat.PortMap)
		for _, port := range c.ports {
			natPort := nat.Port(port.portName)
			exposedPorts[natPort] = struct{}{}
			portMaps[natPort] = []nat.PortBinding{
				{
					HostIP:   port.hostIP,
					HostPort: port.hostPort,
				},
			}
		}
	}

	if nil != c.mounts {
		mounts = make([]mount.Mount, 0, len(c.mounts))
		for containerPath, hostPath := range c.mounts {
			mounts = append(mounts, mount.Mount{
				Type:   mount.TypeBind,
				Source: hostPath,
				Target: containerPath,
			})
		}
	}

	if nil != c.envs {
		envs = make([]string, 0, len(c.envs))
		for key, value := range c.envs {
			envs = append(envs, fmt.Sprintf("%s=%s", key, value))
		}
	}

	containerConfig := container.Config{
		Hostname:     c.name,
		ExposedPorts: exposedPorts,
		Env:          envs,
		Cmd:          c.args,
		Image:        c.image.name,
	}

	if nil != c.networkMode {
		networkMode = container.NetworkMode(*c.networkMode)
	}

	hostConfig := container.HostConfig{
		NetworkMode:  networkMode,
		PortBindings: portMaps,
		Mounts:       mounts,
	}

	return containerConfig, hostConfig
}

func (c *Container) inspect() (*types.ContainerJSON, error) {
	if dockerClient, err := client.NewEnvClient(); nil != err {
		return nil, err
	} else if json, err := dockerClient.ContainerInspect(context.Background(), *c.id); nil != err {
		return nil, err
	} else {
		return &json, err
	}
}

type portInfo struct {
	portName string
	hostIP   string
	hostPort string
}

// lastProgressOutput is the same as progress.Output except
// that it only output with the last update. It is used in
// non terminal scenarios to suppress verbose messages
type lastProgressOutput struct {
	output progress.Output
}

// WriteProgress formats progress information from a ProgressReader.
func (out *lastProgressOutput) WriteProgress(progress progress.Progress) error {
	if !progress.LastUpdate {
		return nil
	}

	return out.output.WriteProgress(progress)
}

func addDockerfileToBuildContext(dockerfileCtx io.ReadCloser, buildCtx io.ReadCloser) (io.ReadCloser, string, error) {
	file, err := ioutil.ReadAll(dockerfileCtx)
	dockerfileCtx.Close()
	if err != nil {
		return nil, "", err
	}
	now := time.Now()
	hdrTmpl := &tar.Header{
		Mode:       0600,
		Uid:        0,
		Gid:        0,
		ModTime:    now,
		Typeflag:   tar.TypeReg,
		AccessTime: now,
		ChangeTime: now,
	}
	randomName := ".dockerfile." + stringid.GenerateRandomID()[:20]

	buildCtx = archive.ReplaceFileTarWrapper(buildCtx, map[string]archive.TarModifierFunc{
		// Add the dockerfile with a random filename
		randomName: func(_ string, h *tar.Header, content io.Reader) (*tar.Header, []byte, error) {
			return hdrTmpl, file, nil
		},
		// Update .dockerignore to include the random filename
		".dockerignore": func(_ string, h *tar.Header, content io.Reader) (*tar.Header, []byte, error) {
			if h == nil {
				h = hdrTmpl
			}

			b := &bytes.Buffer{}
			if content != nil {
				if _, err := b.ReadFrom(content); err != nil {
					return nil, nil, err
				}
			} else {
				b.WriteString(".dockerignore")
			}
			b.WriteString("\n" + randomName + "\n")
			return h, b.Bytes(), nil
		},
	})
	return buildCtx, randomName, nil
}

//type ContainerDiff struct {
//	Name        string
//	Path        string
//	FieldType   ContainerDiffFieldType
//	MapKey      interface{}
//	SourceValue interface{}
//	TargetValue interface{}
//	DiffType    ContainerDiffType
//}
//
//type ContainerDiffType string
//
//var (
//	ContainerDiffValueChanged      ContainerDiffType = "Changed"
//	ContainerDiffNewMapKey         ContainerDiffType = "NewMapKey"
//	ContainerDiffNonExistMapKey    ContainerDiffType = "NonExistMapKey"
//	ContainerDiffNewArrayItem      ContainerDiffType = "NewArrayItem"
//	ContainerDiffNonExistArrayItem ContainerDiffType = "NonExistArrayItem"
//)
//
//type ContainerDiffFieldType string
//
//var (
//	ContainerDiffArrayField  ContainerDiffFieldType = "Field"
//	ContainerDiffMapField    ContainerDiffFieldType = "Map"
//	ContainerDiffStringField ContainerDiffFieldType = "String"
//	ContainerDiffIntField    ContainerDiffFieldType = "Int"
//)

//func (c Container) Changed() ([]string, error) {
//	json, err := c.inspect()
//	if nil != err {
//		return nil, err
//	}
//
//	targetContainer, targetHost := json.Config, json.HostConfig
//	sourceContainer, sourceHost := c.getRunningConfigs()
//
//	diffs := make([]string, 0, 10)
//
//	if strings.Compare(sourceContainer.Hostname, targetContainer.Hostname) != 0 {
//		diffs = append(diffs, fmt.Sprintf("[主机名][新容器:%s][当前容器:%s]", sourceContainer.Hostname, targetContainer.Hostname))
//	}
//
//	if strings.Compare(sourceContainer.Image, targetContainer.Image) != 0 {
//		diffs = append(diffs, fmt.Sprintf("[镜像名][新容器:%s][当前容器:%s]", sourceContainer.Image, targetContainer.Image))
//	}
//
//	sourceExposedPorts := make(nat.PortSet)
//	targetExposedPorts := make(nat.PortSet)
//	if nil != sourceContainer.ExposedPorts {
//		sourceExposedPorts = sourceContainer.ExposedPorts
//	}
//	if nil != targetContainer.ExposedPorts {
//		targetExposedPorts = targetContainer.ExposedPorts
//	}
//	for port := range sourceExposedPorts {
//		if !utils.Any(utils.MapKeys(targetExposedPorts), func(targetPort nat.Port) bool { return port == targetPort }) {
//			diffs = append(diffs, fmt.Sprintf("[暴露端口][新容器:%s][当前容器:未设置]", port))
//		}
//	}
//
//	sourceEnv := make([]string, 0)
//	targetEnv := make([]string, 0)
//	if nil != sourceContainer.BuildEnvs {
//		sourceEnv = sourceContainer.BuildEnvs
//	}
//	if nil != targetContainer.BuildEnvs {
//		targetEnv = targetContainer.BuildEnvs
//	}
//	for _, env := range sourceEnv {
//		if !utils.Any(targetEnv, func(targetEnv string) bool { return env == targetEnv }) {
//			diffs = append(diffs, fmt.Sprintf("[环境变量][新容器:%s][当前容器:未设置]", env))
//		}
//	}
//	//for _, env := range targetEnv {
//	//	if !utils.Any(sourceEnv, func(sourceEnv string) bool { return env == sourceEnv }) {
//	//		diffs = append(diffs, fmt.Sprintf("[环境变量][新容器:未设置][当前容器:%s]", env))
//	//	}
//	//}
//
//	sourceCmd := make([]string, 0)
//	targetCmd := make([]string, 0)
//	if nil != sourceContainer.Cmd {
//		sourceCmd = sourceContainer.Cmd
//	}
//	if nil != targetContainer.Cmd {
//		targetCmd = targetContainer.Cmd
//	}
//	for _, cmd := range sourceCmd {
//		if !utils.Any(targetCmd, func(targetCmd string) bool { return cmd == targetCmd }) {
//			diffs = append(diffs, fmt.Sprintf("[运行参数][新容器:%s][当前容器:未设置]", cmd))
//		}
//	}
//	//for _, cmd := range targetCmd {
//	//	if !utils.Any(sourceCmd, func(targetCmd string) bool { return cmd == targetCmd }) {
//	//		diffs = append(diffs, fmt.Sprintf("[运行参数][新容器:未设置][当前容器:%s]", cmd))
//	//	}
//	//}
//
//	if strings.Compare(string(sourceHost.NetworkMode), string(targetHost.NetworkMode)) != 0 {
//		diffs = append(diffs, fmt.Sprintf("[网络模式][新容器:%s][当前容器:%s]", sourceHost.NetworkMode, targetHost.NetworkMode))
//	}
//
//	sourcePortBindingMap := make(nat.PortMap)
//	targetPortBindingMap := make(nat.PortMap)
//	if nil != sourceHost.PortBindings {
//		sourcePortBindingMap = sourceHost.PortBindings
//	}
//	if nil != targetHost.PortBindings {
//		targetPortBindingMap = targetHost.PortBindings
//	}
//	for port, sourcePortBindings := range sourcePortBindingMap {
//		if targetPortBindings, ok := targetPortBindingMap[port]; !ok {
//			diffs = append(diffs, fmt.Sprintf("[端口映射][新容器:%s][当前容器:未设置]", port))
//		} else {
//			for _, binding := range sourcePortBindings {
//				if !utils.Any(targetPortBindings, func(targetBinding nat.PortBinding) bool {
//					return strings.Compare(binding.HostPort, targetBinding.HostPort) == 0 &&
//						strings.Compare(binding.HostIP, targetBinding.HostIP) == 0
//				}) {
//					diffs = append(diffs, fmt.Sprintf("[端口映射-%s][新容器:%s:%s][当前容器:未设置]", port, binding.HostIP, binding.HostPort))
//				}
//			}
//			for _, binding := range targetPortBindings {
//				if !utils.Any(sourcePortBindings, func(targetBinding nat.PortBinding) bool {
//					return strings.Compare(binding.HostPort, targetBinding.HostPort) == 0 &&
//						strings.Compare(binding.HostIP, targetBinding.HostIP) == 0
//				}) {
//					diffs = append(diffs, fmt.Sprintf("[端口映射-%s][新容器:未设置][当前容器:%s:%s]", port, binding.HostIP, binding.HostPort))
//				}
//			}
//		}
//	}
//
//	for port, targetPortBindings := range targetPortBindingMap {
//		if sourcePortBindings, ok := sourcePortBindingMap[port]; !ok {
//			diffs = append(diffs, fmt.Sprintf("[端口映射][新容器:未设置][当前容器:%s]", port))
//		} else {
//			for _, binding := range targetPortBindings {
//				if !utils.Any(sourcePortBindings, func(targetBinding nat.PortBinding) bool {
//					return strings.Compare(binding.HostPort, targetBinding.HostPort) == 0 &&
//						strings.Compare(binding.HostIP, targetBinding.HostIP) == 0
//				}) {
//					diffs = append(diffs, fmt.Sprintf("[端口映射-%s][新容器:%s:%s][当前容器:未设置]", port, binding.HostIP, binding.HostPort))
//				}
//			}
//			for _, binding := range sourcePortBindings {
//				if !utils.Any(targetPortBindings, func(targetBinding nat.PortBinding) bool {
//					return strings.Compare(binding.HostPort, targetBinding.HostPort) == 0 &&
//						strings.Compare(binding.HostIP, targetBinding.HostIP) == 0
//				}) {
//					diffs = append(diffs, fmt.Sprintf("[端口映射-%s][新容器:未设置][当前容器:%s:%s]", port, binding.HostIP, binding.HostPort))
//				}
//			}
//		}
//	}
//
//	sourceMounts := make([]mount.Mount, 0)
//	targetMounts := make([]mount.Mount, 0)
//	if nil != sourceHost.Mounts {
//		sourceMounts = sourceHost.Mounts
//	}
//	if nil != targetHost.Mounts {
//		targetMounts = sourceHost.Mounts
//	}
//	for _, sourceMount := range sourceMounts {
//		if !utils.Any(targetMounts, func(targetMount mount.Mount) bool {
//			return strings.Compare(sourceMount.Source, targetMount.Source) == 0 &&
//				strings.Compare(sourceMount.Target, targetMount.Target) == 0
//		}) {
//			diffs = append(diffs, fmt.Sprintf("[数据绑定][新容器:%s=>%s][当前容器:未设置]", sourceMount.Source, sourceMount.Target))
//		}
//	}
//	for _, targetMount := range targetMounts {
//		if !utils.Any(sourceMounts, func(sourceMount mount.Mount) bool {
//			return strings.Compare(targetMount.Source, sourceMount.Source) == 0 &&
//				strings.Compare(targetMount.Target, sourceMount.Target) == 0
//		}) {
//			diffs = append(diffs, fmt.Sprintf("[数据绑定][新容器:未设置][当前容器:%s=>%s]", targetMount.Source, targetMount.Target))
//		}
//	}
//
//	return diffs, nil
//}
