package yaml

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"github.com/alydnh/go-micro-ci-common/utils"
	"gopkg.in/yaml.v3"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
)

func CreateManifest(build *Build, targetPath string, headHash [20]byte) (*Manifest, error) {
	m := &Manifest{
		HeadHash:        hex.EncodeToString(headHash[:]),
		Files:           make([]*ManifestFile, 0),
		Scripts:         build.Scripts,
		AdditionalFiles: build.AdditionalFiles,
		BuildEnvs:       build.Env,
		targetPath:      targetPath,
	}

	if err := filepath.Walk(targetPath, func(path string, info os.FileInfo, err error) error {
		if nil != err {
			return err
		}
		if !info.IsDir() {
			return m.addFile(targetPath, path)
		}
		return nil
	}); nil != err {
		return nil, err
	}

	return m, nil
}

func ReadManifest(path string) (*Manifest, error) {
	m := &Manifest{}
	if bytes, err := ioutil.ReadFile(path); nil != err {
		return nil, err
	} else if err := yaml.Unmarshal(bytes, m); nil != err {
		return nil, err
	}
	m.targetPath = filepath.Dir(path)
	return m, nil
}

type Manifest struct {
	HeadHash        string            `yaml:"headHash"`
	Files           []*ManifestFile   `yaml:"files"`
	Scripts         []string          `yaml:"scripts"`
	AdditionalFiles []string          `yaml:"additionalFiles"`
	BuildEnvs       map[string]string `yaml:"buildEnvs"`
	DockerImageID   *string           `yaml:"dockerImageID"`
	Image           *Image            `yaml:"image"`
	targetPath      string
}

func (m Manifest) CheckCRC() (bool, error) {
	for _, file := range m.Files {
		path := filepath.Join(m.targetPath, file.Path)
		if !utils.FileExists(path) {
			return false, nil
		}
		if crc, err := m.getCRC(path); nil != err || strings.Compare(crc, file.CRC) != 0 {
			return false, err
		}
	}
	return true, nil
}

func (m Manifest) BuildEquals(build *Build) bool {
	if equal, _ := utils.EqualValues(build.Env, m.BuildEnvs); !equal {
		return false
	}
	if equal, _ := utils.EqualValues(build.AdditionalFiles, m.AdditionalFiles); !equal {
		return false
	}
	if equal, _ := utils.EqualValues(build.Scripts, m.Scripts); !equal {
		return false
	}

	return true
}

func (m Manifest) HeadEquals(headHash [20]byte) bool {
	return strings.Compare(hex.EncodeToString(headHash[:]), m.HeadHash) == 0
}

func (m Manifest) ImageEquals(image *Image) bool {
	if nil == m.Image {
		return false
	}

	if equal, _ := utils.EqualValues(image, m.Image); !equal {
		return false
	}

	return true
}

func (m *Manifest) ApplyImage(imageID string, image *Image) {
	m.DockerImageID = &imageID
	m.Image = image
}

func (m Manifest) SaveToFile(fileName string) error {
	if bytes, err := yaml.Marshal(m); nil != err {
		return err
	} else {
		return ioutil.WriteFile(fileName, bytes, os.FileMode(0775))
	}
}

func (m Manifest) FilesCount() int {
	return len(m.Files)
}

func (m *Manifest) addFile(root, path string) error {
	crc, err := m.getCRC(path)
	if nil != err {
		return err
	}
	manifestFile := &ManifestFile{
		Path: strings.TrimPrefix(path, root),
		CRC:  crc,
	}
	m.Files = append(m.Files, manifestFile)
	return nil
}

func (m Manifest) getCRC(path string) (string, error) {
	f, err := os.Open(path)
	if nil != err {
		return utils.EmptyString, fmt.Errorf("打开文件:%s 失败:%s", path, err.Error())
	}
	defer f.Close()
	hash := md5.New()
	if _, err := io.Copy(hash, f); nil != err {
		return utils.EmptyString, fmt.Errorf("读取文件:%s 失败:%s", path, err.Error())
	}
	return hex.EncodeToString(hash.Sum(nil)), nil
}

type ManifestFile struct {
	Path string `yaml:"path"`
	CRC  string `yaml:"crc"`
}
