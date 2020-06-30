package docker

import (
	"context"
	"encoding/binary"
	"fmt"
	"github.com/alydnh/go-micro-ci-common/utils"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/client"
	"io"
	"strings"
)

func GetDockerVersion() (string, error) {
	if dockerClient, err := client.NewEnvClient(); nil != err {
		return utils.EmptyString, err
	} else if version, err := dockerClient.ServerVersion(context.Background()); nil != err {
		return utils.EmptyString, err
	} else {
		return version.Version, nil
	}
}

func EnsureNetworkMode(networkMode, driver string) error {
	dockerClient, err := client.NewEnvClient()
	if nil != err {
		return err
	}

	filterArgs := filters.NewArgs()
	filterArgs.Add("name", networkMode)
	if r, err := dockerClient.NetworkList(context.Background(), types.NetworkListOptions{
		Filters: filterArgs,
	}); nil != err {
		return err
	} else if len(r) > 0 {
		if strings.Compare(r[0].Driver, driver) != 0 {
			return fmt.Errorf("%s:网络驱动模式为%s, 预期为:%s", networkMode, r[0].Driver, driver)
		}
	} else if _, err := dockerClient.NetworkCreate(context.Background(), networkMode, types.NetworkCreate{
		Driver: driver,
	}); nil != err {
		return err
	}

	return nil
}

func DeleteImage(imageID string) error {
	if dockerClient, err := client.NewEnvClient(); nil != err {
		return err
	} else if resp, err := dockerClient.ImageRemove(context.Background(), imageID, types.ImageRemoveOptions{}); nil != err {
		return err
	} else if len(resp) == 0 {
		return fmt.Errorf("未能删除任何镜像")
	}
	return nil
}

// StdType is the type of standard stream
// a writer can multiplex to.
type StdType byte

const (
	// Stdin represents standard input stream type.
	Stdin StdType = iota
	// Stdout represents standard output stream type.
	Stdout
	// Stderr represents standard error steam type.
	Stderr
	// SystemErr represents errors originating from the system that make it
	// into the the multiplexed stream.
	SystemErr

	stdWriterPrefixLen = 8
	stdWriterFdIndex   = 0
	stdWriterSizeIndex = 4

	startingBufLen = 32*1024 + stdWriterPrefixLen + 1
)

// StdCopy is a modified version of io.Copy.
//
// StdCopy will demultiplex `src`, assuming that it contains two streams,
// previously multiplexed together using a StdWriter instance.
// As it reads from `src`, StdCopy will write to `dstout` and `dsterr`.
//
// StdCopy will read until it hits EOF on `src`. It will then return a nil error.
// In other words: if `err` is non nil, it indicates a real underlying error.
//
// `written` will hold the total number of bytes written to `dstout` and `dsterr`.
func StdCopy(ctx context.Context, dstOut, dstErr io.Writer, src io.Reader) (written int64, err error) {
	var (
		buf       = make([]byte, startingBufLen)
		bufLen    = len(buf)
		nr, nw    int
		er, ew    error
		out       io.Writer
		frameSize int
	)

	for {
		// Make sure we have at least a full header
		for nr < stdWriterPrefixLen {
			var nr2 int
			nr2, er = src.Read(buf[nr:])
			nr += nr2
			if er == io.EOF {
				if nr < stdWriterPrefixLen {
					return written, nil
				}
				break
			}
			if er != nil {
				return 0, er
			}
		}

		stream := StdType(buf[stdWriterFdIndex])
		// Check the first byte to know where to write
		switch stream {
		case Stdin:
			fallthrough
		case Stdout:
			// Write on stdout
			out = dstOut
		case Stderr:
			// Write on stderr
			out = dstErr
		case SystemErr:
			// If we're on SystemErr, we won't write anywhere.
			// NB: if this code changes later, make sure you don't try to write
			// to outstream if SystemErr is the stream
			out = nil
		default:
			return 0, fmt.Errorf("unrecognized input header: %d", buf[stdWriterFdIndex])
		}

		// Retrieve the size of the frame
		frameSize = int(binary.BigEndian.Uint32(buf[stdWriterSizeIndex : stdWriterSizeIndex+4]))

		// Check if the buffer is big enough to read the frame.
		// Extend it if necessary.
		if frameSize+stdWriterPrefixLen > bufLen {
			buf = append(buf, make([]byte, frameSize+stdWriterPrefixLen-bufLen+1)...)
			bufLen = len(buf)
		}

		// While the amount of bytes read is less than the size of the frame + header, we keep reading
		for nr < frameSize+stdWriterPrefixLen {
			var nr2 int
			nr2, er = src.Read(buf[nr:])
			nr += nr2
			if er == io.EOF {
				if nr < frameSize+stdWriterPrefixLen {
					return written, nil
				}
				break
			}
			if er != nil {
				return 0, er
			}
		}

		// we might have an error from the source mixed up in our multiplexed
		// stream. if we do, return it.
		if stream == SystemErr {
			return written, fmt.Errorf("error from daemon in stream: %s", string(buf[stdWriterPrefixLen:frameSize+stdWriterPrefixLen]))
		}

		// Write the retrieved frame (without header)
		nw, ew = out.Write(buf[stdWriterPrefixLen : frameSize+stdWriterPrefixLen])
		if ew != nil {
			return 0, ew
		}

		// If the frame has not been fully written: error
		if nw != frameSize {
			return 0, io.ErrShortWrite
		}
		written += int64(nw)

		// Move the rest of the buffer to the beginning
		copy(buf, buf[frameSize+stdWriterPrefixLen:])
		// Move the index
		nr -= frameSize + stdWriterPrefixLen

		select {
		case <-ctx.Done():
			return
		default:
		}
	}
}

func PromiseGo(f func() error) chan error {
	ch := make(chan error, 1)
	go func() {
		ch <- f()
	}()
	return ch
}
