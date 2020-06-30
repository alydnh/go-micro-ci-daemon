package docker

import (
	"fmt"
	"os"
	goSignal "os/signal"
	"runtime"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/signal"
	"github.com/sirupsen/logrus"
	"golang.org/x/net/context"
)

// resizeTtyTo resizes tty to specific height and width
func resizeTtyTo(ctx context.Context, client client.ContainerAPIClient, id string, height, width uint, isExec bool) {
	if height == 0 && width == 0 {
		return
	}

	options := types.ResizeOptions{
		Height: height,
		Width:  width,
	}

	var err error
	if isExec {
		err = client.ContainerExecResize(ctx, id, options)
	} else {
		err = client.ContainerResize(ctx, id, options)
	}

	if err != nil {
		logrus.Debugf("Error resize: %s", err)
	}
}

// MonitorTtySize updates the container tty size when the terminal tty changes size
func MonitorTtySize(ctx context.Context, streams Streams, id string, isExec bool) error {
	dockerClient, err := client.NewEnvClient()
	if nil != err {
		return err
	}
	resizeTty := func() {
		height, width := streams.Out().GetTtySize()
		resizeTtyTo(ctx, dockerClient, id, height, width, isExec)
	}

	resizeTty()

	if runtime.GOOS == "windows" {
		go func() {
			prevH, prevW := streams.Out().GetTtySize()
			for {
				time.Sleep(time.Millisecond * 250)
				h, w := streams.Out().GetTtySize()

				if prevW != w || prevH != h {
					resizeTty()
				}
				prevH = h
				prevW = w
			}
		}()
	} else {
		sigChan := make(chan os.Signal, 1)
		goSignal.Notify(sigChan, signal.SIGWINCH)
		go func() {
			for range sigChan {
				resizeTty()
			}
		}()
	}
	return nil
}

// ForwardAllSignals forwards signals to the container
func ForwardAllSignals(ctx context.Context, client *client.Client, streams Streams, cid string) chan os.Signal {
	sigChan := make(chan os.Signal, 128)
	signal.CatchAll(sigChan)
	go func() {
		for s := range sigChan {
			if s == signal.SIGCHLD || s == signal.SIGPIPE {
				continue
			}
			var sig string
			for sigStr, sigN := range signal.SignalMap {
				if sigN == s {
					sig = sigStr
					break
				}
			}
			if sig == "" {
				_, _ = fmt.Fprintf(streams.Err(), "Unsupported signal: %v. Discarding.\n", s)
				continue
			}

			if err := client.ContainerKill(ctx, cid, sig); err != nil {
				logrus.Debugf("Error sending signal: %s", err)
			}
		}
	}()
	return sigChan
}
