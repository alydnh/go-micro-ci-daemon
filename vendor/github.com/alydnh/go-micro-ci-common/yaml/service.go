package yaml

import (
	"fmt"
	"github.com/alydnh/go-micro-ci-common/utils"
)

type Service struct {
	name         string
	BaseService  string            `yaml:"baseService"`
	Image        *Image            `yaml:"image"`
	ExposedPorts ExposedPorts      `yaml:"exposedPorts"`
	Env          map[string]string `yaml:"env"`
	Mounts       Mounts            `yaml:"mounts"`
	Args         []string          `yaml:"args"`
	Build        *Build            `yaml:"build"`
	MetaData     map[string]string `yaml:"metadata"`
	Assertions   *Assertions       `yaml:"assertions"`
	DependsOn    []string          `yaml:"dependsOn"`
	Tags         map[string]string `yaml:"tags"`
	DataSource   DataSource        `yaml:"dataSource"`
	isThird      bool
	snapShot     *Service
}

func (y Service) Name() string {
	return y.name
}

func (y Service) HasBaseService() bool {
	return !utils.EmptyOrWhiteSpace(y.BaseService)
}

func (y Service) Equals(serviceYaml *Service, git, build, image bool) (equals bool) {

	equals = true

	if git {
		if equals, _ = utils.EqualValues(y.Build.Git, serviceYaml.Build.Git); equals {
			if equals, _ = utils.EqualValues(y.Build.Tag, serviceYaml.Build.Tag); equals {
				equals, _ = utils.EqualValues(y.Build.Branch, serviceYaml.Build.Branch)
			}
		}
	}

	if !equals {
		return
	}

	if build {
		if equals, _ = utils.EqualValues(y.Build.TargetFolder, serviceYaml.Build.TargetFolder); equals {
			if equals, _ = utils.EqualValues(y.Build.Env, serviceYaml.Build.Env); equals {
				if equals, _ = utils.EqualValues(y.Build.Scripts, serviceYaml.Build.Scripts); equals {
					equals, _ = utils.EqualValues(y.Build.AdditionalFiles, serviceYaml.Build.AdditionalFiles)
				}
			}
		}
	}

	if !equals {
		return
	}

	if image {
		equals, _ = utils.EqualValues(y.Image, serviceYaml.Image)
	}

	return
}

func (y *Service) ApplyBaseService(baseService *Service) {
	snapShot := y.snapShot
	*y = *baseService.Clone()
	y.BaseService = snapShot.BaseService
	y.name = snapShot.name
	if nil != snapShot.Tags && len(snapShot.Tags) > 0 {
		y.Tags = snapShot.Tags
	}
	if nil != snapShot.Build {
		y.Build = snapShot.Build
	}
	if nil != snapShot.Args && len(snapShot.Args) > 0 {
		y.Args = snapShot.Args
	}
	if nil != snapShot.DependsOn && len(snapShot.DependsOn) > 0 {
		y.DependsOn = snapShot.DependsOn
	}
	if nil != snapShot.Env && len(snapShot.Env) > 0 {
		y.Env = snapShot.Env
	}
	if nil != snapShot.Assertions {
		y.Assertions = snapShot.Assertions
	}
	if nil != snapShot.Mounts && len(snapShot.Mounts) > 0 {
		y.Mounts = snapShot.Mounts
	}
	if nil != snapShot.ExposedPorts && len(snapShot.ExposedPorts) > 0 {
		y.ExposedPorts = snapShot.ExposedPorts
	}
	if nil != snapShot.DataSource && len(snapShot.DataSource) > 0 {
		y.DataSource = snapShot.DataSource
	}
	y.isThird = snapShot.isThird
}

func (y *Service) Initialize(name string, isThird, doValidate bool) error {
	y.name = name
	y.isThird = isThird
	if y.HasBaseService() {
		y.snapShot = y.Clone()
	}
	if nil == y.ExposedPorts {
		y.ExposedPorts = make(ExposedPorts)
	}
	if nil == y.Env {
		y.Env = make(map[string]string)
	}
	if nil == y.Mounts {
		y.Mounts = make(Mounts)
	}
	if nil == y.Args {
		y.Args = make([]string, 0)
	}
	if nil == y.MetaData {
		y.MetaData = make(map[string]string)
	}
	if nil == y.DataSource {
		y.DataSource = make(DataSource)
	}
	if nil == y.Assertions {
		y.Assertions = &Assertions{
			Successes: make([]string, 0),
			Fails:     make([]string, 0),
			Lines:     100,
			Timeout:   "10s",
		}
	}
	if nil == y.DependsOn {
		y.DependsOn = make([]string, 0)
	}
	if nil == y.Tags {
		y.Tags = make(map[string]string)
	}

	if doValidate {
		if nil == y.Image {
			if !y.HasBaseService() {
				return fmt.Errorf("%s.image不能为空", name)
			}
		} else if err := y.Image.Validate(); nil != err {
			return fmt.Errorf("%s.image.%s", name, err.Error())
		}

		if nil != y.MetaData && len(y.MetaData) > 0 {
			return fmt.Errorf("%s.metadata只读", name)
		}
	}

	if nil != y.Build {
		if err := y.Build.Initialize(); nil != err {
			return fmt.Errorf("%s.build.%s", name, err.Error())
		}
	}

	for portName, port := range y.ExposedPorts {
		if err := port.Validate(PortName(portName)); nil != err {
			return fmt.Errorf("%s.exposePorts.%s.%s", name, portName, err.Error())
		}
	}

	if err := y.Assertions.Initialize(doValidate); nil != err {
		return fmt.Errorf("%s.assertions.%s", name, err.Error())
	}

	return nil
}

func (y *Service) IsThird() bool {
	return y.isThird
}

func (y Service) Clone() *Service {
	cloned := &Service{
		name:        y.name,
		BaseService: y.BaseService,
		isThird:     y.isThird,
	}

	if nil != y.Image {
		cloned.Image = y.Image.Clone()
	}

	if nil != y.ExposedPorts {
		cloned.ExposedPorts = make(ExposedPorts)
		for name, port := range y.ExposedPorts {
			cloned.ExposedPorts[name] = port.Clone()
		}
	}

	if nil != y.Env {
		cloned.Env = utils.CopyMap(y.Env).(map[string]string)
	}

	if nil != y.Mounts {
		cloned.Mounts = Mounts(utils.CopyMap(y.Mounts).(map[MountContainerPath]MountHostPath))
	}

	if nil != y.Args {
		cloned.Args = append([]string{}, y.Args...)
	}

	if nil != y.Build {
		cloned.Build = y.Build.Clone()
	}

	if nil != y.MetaData {
		cloned.MetaData = utils.CopyMap(y.MetaData).(map[string]string)
	}

	if nil != y.Assertions {
		cloned.Assertions = y.Assertions.Clone()
	}

	if nil != y.DependsOn {
		cloned.DependsOn = append([]string{}, y.DependsOn...)
	}

	if nil != y.Tags {
		cloned.Tags = utils.CopyMap(y.Tags).(map[string]string)
	}

	if nil != y.DataSource {
		cloned.DataSource = y.DataSource.Clone()
	}

	return cloned
}
