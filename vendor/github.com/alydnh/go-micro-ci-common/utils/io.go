package utils

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
)

func IsEmptyFolder(name string) (bool, error) {
	f, err := os.Open(name)
	if err != nil {
		return false, err
	}
	defer f.Close()

	_, err = f.Readdirnames(1) // Or f.Readdir(1)
	if err == io.EOF {
		return true, nil
	}
	return false, err // Either not empty or error, suits both cases
}

func DirectoryExists(path string) bool {
	f, err := os.Stat(path)
	if os.IsNotExist(err) {
		return false
	}
	return f.IsDir()
}

func FileExists(path string) bool {
	f, err := os.Stat(path)
	if os.IsNotExist(err) {
		return false
	}
	return !f.IsDir()
}

func EnsureDirectory(path string, createIfNotExists bool) error {
	if f, err := os.Stat(path); nil != err {
		if os.IsNotExist(err) {
			if createIfNotExists {
				if err = os.MkdirAll(path, os.FileMode(0775)); nil != err {
					return err
				}
			} else {
				return fmt.Errorf(path, "not exists")
			}
		} else {
			return err
		}
	} else if !f.IsDir() {
		return fmt.Errorf(path, "is not a valid directory")
	}
	return nil
}

const (
	// tmpPermissionForDirectory makes the destination directory writable,
	// so that stuff can be copied recursively even if any original directory is NOT writable.
	// See https://github.com/otiai10/copy/pull/9 for more information.
	tmpPermissionForDirectory = os.FileMode(0755)
)

// Copy copies src to dest, doesn't matter if src is a directory or a file
func Copy(src, dest string) error {
	info, err := os.Lstat(src)
	if err != nil {
		return err
	}
	return copy(src, dest, info)
}

// copy dispatches copy-funcs according to the mode.
// Because this "copy" could be called recursively,
// "info" MUST be given here, NOT nil.
func copy(src, dest string, info os.FileInfo) error {
	if info.Mode()&os.ModeSymlink != 0 {
		return linkCopy(src, dest, info)
	}
	if info.IsDir() {
		return directoryCopy(src, dest, info)
	}
	return fileCopy(src, dest, info)
}

// fileCopy is for just a file,
// with considering existence of parent directory
// and file permission.
func fileCopy(src, dest string, info os.FileInfo) error {

	if err := os.MkdirAll(filepath.Dir(dest), os.ModePerm); err != nil {
		return err
	}

	f, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer f.Close()

	if err = os.Chmod(f.Name(), info.Mode()); err != nil {
		return err
	}

	s, err := os.Open(src)
	if err != nil {
		return err
	}
	defer s.Close()

	_, err = io.Copy(f, s)
	return err
}

// directoryCopy is for a directory,
// with scanning contents inside the directory
// and pass everything to "copy" recursively.
func directoryCopy(srcdir, destdir string, info os.FileInfo) error {

	originalMode := info.Mode()

	// Make dest dir with 0755 so that everything writable.
	if err := os.MkdirAll(destdir, tmpPermissionForDirectory); err != nil {
		return err
	}
	// Recover dir mode with original one.
	defer os.Chmod(destdir, originalMode)

	contents, err := ioutil.ReadDir(srcdir)
	if err != nil {
		return err
	}

	for _, content := range contents {
		cs, cd := filepath.Join(srcdir, content.Name()), filepath.Join(destdir, content.Name())
		if err := copy(cs, cd, content); err != nil {
			// If any error, exit immediately
			return err
		}
	}

	return nil
}

// linkCopy is for a symlink,
// with just creating a new symlink by replicating src symlink.
func linkCopy(src, dest string, info os.FileInfo) error {
	src, err := os.Readlink(src)
	if err != nil {
		return err
	}
	return os.Symlink(src, dest)
}
