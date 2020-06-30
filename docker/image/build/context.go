package build

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/docker/docker/pkg/fileutils"
	"github.com/pkg/errors"
)

const (
	// DefaultDockerfileName is the Default filename with Docker commands, read by docker build
	DefaultDockerfileName string = "Dockerfile"
)

// ValidateContextDirectory checks if all the contents of the directory
// can be read and returns an error if some files can't be read
// symlinks which point to non-existing files don't trigger an error
func ValidateContextDirectory(srcPath string, excludes []string) error {
	contextRoot, err := getContextRoot(srcPath)
	if err != nil {
		return err
	}
	return filepath.Walk(contextRoot, func(filePath string, f os.FileInfo, err error) error {
		if err != nil {
			if os.IsPermission(err) {
				return errors.Errorf("can't stat '%s'", filePath)
			}
			if os.IsNotExist(err) {
				return nil
			}
			return err
		}

		// skip this directory/file if it's not in the path, it won't get added to the context
		if relFilePath, err := filepath.Rel(contextRoot, filePath); err != nil {
			return err
		} else if skip, err := fileutils.Matches(relFilePath, excludes); err != nil {
			return err
		} else if skip {
			if f.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		// skip checking if symlinks point to non-existing files, such symlinks can be useful
		// also skip named pipes, because they hanging on open
		if f.Mode()&(os.ModeSymlink|os.ModeNamedPipe) != 0 {
			return nil
		}

		if !f.IsDir() {
			currentFile, err := os.Open(filePath)
			if err != nil && os.IsPermission(err) {
				return errors.Errorf("no permission to read from '%s'", filePath)
			}
			currentFile.Close()
		}
		return nil
	})
}

// GetContextFromLocalDir uses the given local directory as context for a
// `docker build`. Returns the absolute path to the local context directory,
// the relative path of the dockerfile in that context directory, and a non-nil
// error on success.
func GetContextFromLocalDir(localDir, dockerfileName string) (absContextDir, relDockerfile string, err error) {
	// When using a local context directory, when the Dockerfile is specified
	// with the `-f/--file` option then it is considered relative to the
	// current directory and not the context directory.
	if dockerfileName != "" && dockerfileName != "-" {
		if dockerfileName, err = filepath.Abs(dockerfileName); err != nil {
			return "", "", errors.Errorf("unable to get absolute path to Dockerfile: %v", err)
		}
	}

	return getDockerfileRelPath(localDir, dockerfileName)
}

// getDockerfileRelPath uses the given context directory for a `docker build`
// and returns the absolute path to the context directory, the relative path of
// the dockerfile in that context directory, and a non-nil error on success.
func getDockerfileRelPath(givenContextDir, givenDockerfile string) (absContextDir, relDockerfile string, err error) {
	if absContextDir, err = filepath.Abs(givenContextDir); err != nil {
		return "", "", errors.Errorf("unable to get absolute context directory of given context directory %q: %v", givenContextDir, err)
	}

	// The context dir might be a symbolic link, so follow it to the actual
	// target directory.
	//
	// FIXME. We use isUNC (always false on non-Windows platforms) to workaround
	// an issue in golang. On Windows, EvalSymLinks does not work on UNC file
	// paths (those starting with \\). This hack means that when using links
	// on UNC paths, they will not be followed.
	if !isUNC(absContextDir) {
		absContextDir, err = filepath.EvalSymlinks(absContextDir)
		if err != nil {
			return "", "", errors.Errorf("unable to evaluate symlinks in context path: %v", err)
		}
	}

	stat, err := os.Lstat(absContextDir)
	if err != nil {
		return "", "", errors.Errorf("unable to stat context directory %q: %v", absContextDir, err)
	}

	if !stat.IsDir() {
		return "", "", errors.Errorf("context must be a directory: %s", absContextDir)
	}

	absDockerfile := givenDockerfile
	if absDockerfile == "" {
		// No -f/--file was specified so use the default relative to the
		// context directory.
		absDockerfile = filepath.Join(absContextDir, DefaultDockerfileName)

		// Just to be nice ;-) look for 'dockerfile' too but only
		// use it if we found it, otherwise ignore this check
		if _, err = os.Lstat(absDockerfile); os.IsNotExist(err) {
			altPath := filepath.Join(absContextDir, strings.ToLower(DefaultDockerfileName))
			if _, err = os.Lstat(altPath); err == nil {
				absDockerfile = altPath
			}
		}
	} else if absDockerfile == "-" {
		absDockerfile = filepath.Join(absContextDir, DefaultDockerfileName)
	}

	// If not already an absolute path, the Dockerfile path should be joined to
	// the base directory.
	if !filepath.IsAbs(absDockerfile) {
		absDockerfile = filepath.Join(absContextDir, absDockerfile)
	}

	// Evaluate symlinks in the path to the Dockerfile too.
	//
	// FIXME. We use isUNC (always false on non-Windows platforms) to workaround
	// an issue in golang. On Windows, EvalSymLinks does not work on UNC file
	// paths (those starting with \\). This hack means that when using links
	// on UNC paths, they will not be followed.
	if givenDockerfile != "-" {
		if !isUNC(absDockerfile) {
			absDockerfile, err = filepath.EvalSymlinks(absDockerfile)
			if err != nil {
				return "", "", errors.Errorf("unable to evaluate symlinks in Dockerfile path: %v", err)

			}
		}

		if _, err := os.Lstat(absDockerfile); err != nil {
			if os.IsNotExist(err) {
				return "", "", errors.Errorf("Cannot locate Dockerfile: %q", absDockerfile)
			}
			return "", "", errors.Errorf("unable to stat Dockerfile: %v", err)
		}
	}

	if relDockerfile, err = filepath.Rel(absContextDir, absDockerfile); err != nil {
		return "", "", errors.Errorf("unable to get relative Dockerfile path: %v", err)
	}

	if strings.HasPrefix(relDockerfile, ".."+string(filepath.Separator)) {
		return "", "", errors.Errorf("The Dockerfile (%s) must be within the build context (%s)", givenDockerfile, givenContextDir)
	}

	return absContextDir, relDockerfile, nil
}

// isUNC returns true if the path is UNC (one starting \\). It always returns
// false on Linux.
func isUNC(path string) bool {
	return runtime.GOOS == "windows" && strings.HasPrefix(path, `\\`)
}
