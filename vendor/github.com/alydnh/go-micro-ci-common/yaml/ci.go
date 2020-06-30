package yaml

import (
	"bytes"
	"fmt"
	"github.com/alydnh/go-micro-ci-common/utils"
	"gopkg.in/yaml.v3"
	"io/ioutil"
	"os"
	"strings"
	"text/template"
)

type CI struct {
	Variables             map[string]string   `yaml:"variables"`
	CommonEnvs            map[string]string   `yaml:"commonEnvs"`
	ThirdServices         map[string]*Service `yaml:"thirdServices"`
	Services              map[string]*Service `yaml:"services"`
	Registry              *Registry           `yaml:"registry"`
	Credentials           []*Credential       `yaml:"credentials"`
	Metadata              map[string]string   `yaml:"metadata"`
	CIName                string              `yaml:"name"`
	sequencedServiceNames []string
}

func OpenCI(path string, doValidate bool) (ci *CI, err error) {
	if _, err := os.Stat(path); nil != err {
		return nil, err
	}
	ci = &CI{}
	templateBuffer := &bytes.Buffer{}
	var data []byte
	data, err = ioutil.ReadFile(path)
	if nil != err {
		return
	}
	if err = yaml.Unmarshal(data, ci); nil != err {
		return
	}

	if nil != ci.Variables && len(ci.Variables) > 0 {
		var tpl *template.Template
		tpl, err = template.New("t").Parse(string(data))
		if nil != err {
			return
		}
		if err = tpl.Execute(templateBuffer, ci.Variables); nil != err {
			return
		}
		if err = yaml.Unmarshal(templateBuffer.Bytes(), &ci); nil != err {
			return
		}
		if err := ci.Initialize(doValidate); nil != err {
			return nil, err
		}
	}

	return
}

func NewCI(name string) *CI {
	return &CI{
		Variables:             make(map[string]string),
		CommonEnvs:            make(map[string]string),
		ThirdServices:         make(map[string]*Service),
		Services:              make(map[string]*Service),
		Credentials:           make([]*Credential, 0),
		Metadata:              make(map[string]string),
		CIName:                name,
		sequencedServiceNames: make([]string, 0),
	}
}

func (y CI) GetSequencedContainerNames() []string {
	return utils.Select(y.sequencedServiceNames, y.GetContainerName).([]string)
}

func (y CI) GetContainerName(serviceName string) string {
	return fmt.Sprintf("%s-%s", y.Name(), serviceName)
}

func (y CI) GetService(name string) *Service {
	if service, ok := y.ThirdServices[name]; ok {
		return service
	} else if service, ok = y.Services[name]; ok {
		return service
	}

	serviceName := strings.TrimPrefix(name, fmt.Sprintf("%s-", y.Name()))
	if service, ok := y.ThirdServices[serviceName]; ok {
		return service
	}
	return y.Services[serviceName]
}

func (y *CI) SetMetadata(key, value string) {
	if nil == y.Metadata {
		y.Metadata = make(map[string]string)
	}
	if utils.EmptyOrWhiteSpace(value) {
		delete(y.Metadata, key)
	} else {
		y.Metadata[key] = value
	}
}

func (y CI) GetMetadata(key string) string {
	if nil == y.Metadata {
		return utils.EmptyString
	}
	return y.Metadata[key]
}

func (y CI) Name() string {
	return y.CIName
}

func (y *CI) AddOrUpdateService(service *Service) {
	if service.IsThird() {
		y.ThirdServices[service.Name()] = service
	} else {
		y.Services[service.Name()] = service
	}
}

func (y *CI) RemoveService(service *Service) {
	if service.IsThird() {
		delete(y.ThirdServices, service.Name())
	} else {
		delete(y.Services, service.Name())
	}
}

func (y *CI) Initialize(doValidate bool) error {
	if nil == y.Metadata {
		y.Metadata = make(map[string]string)
	}

	if doValidate {
		if len(y.Metadata) > 0 {
			return fmt.Errorf("metadata只读")
		}
	}

	if nil == y.CommonEnvs {
		y.CommonEnvs = make(map[string]string)
	}

	if nil != y.Credentials {
		for index, credential := range y.Credentials {
			if err := credential.Validate(); nil != err {
				return fmt.Errorf("credentials[%d].%s", index, err.Error())
			}
		}
	} else {
		y.Credentials = make([]*Credential, 0)
	}

	serviceNames := make(map[string]bool)

	if nil == y.ThirdServices {
		y.ThirdServices = make(map[string]*Service)
	}

	context := &dependsOnContext{
		processed:   make([]*Service, 0, 10),
		unprocessed: make([]*Service, 0, 10),
		chain:       make([]*Service, 0),
	}
	if len(y.ThirdServices) > 0 {
		for name, service := range y.ThirdServices {
			if err := service.Initialize(name, true, doValidate); nil != err {
				return fmt.Errorf("thirdServices.%s", err.Error())
			}
			if _, ok := serviceNames[name]; ok {
				return fmt.Errorf("thirdService.%s 名字重复", name)
			}
			serviceNames[name] = true
			context.add(service)
		}

		for _, service := range y.ThirdServices {
			if service.HasBaseService() {
				targetService, ok := y.ThirdServices[service.BaseService]
				if !ok {
					return fmt.Errorf("thirdService.%s.baseService:%s 未找到", service.Name(), service.BaseService)
				}
				if strings.Compare(targetService.Name(), service.Name()) == 0 {
					return fmt.Errorf("thirdService.%s.baseService:%s 不能是服务自身", service.Name(), service.BaseService)
				}
				if targetService.HasBaseService() {
					return fmt.Errorf("thirdService.%s.baseService:%s 不能也是引用服务", service.Name(), service.BaseService)
				}
				service.ApplyBaseService(targetService)
			}
		}
	}

	if doValidate && (nil == y.Services || len(y.Services) == 0) {
		return fmt.Errorf("未找到services定义")
	}

	for name, service := range y.Services {
		if err := service.Initialize(name, false, doValidate); nil != err {
			return fmt.Errorf("services.%s", err.Error())
		}
		if _, ok := serviceNames[name]; ok {
			return fmt.Errorf("services.%s 名字重复", name)
		}
		serviceNames[name] = true
		context.add(service)
	}

	for _, service := range y.Services {
		if service.HasBaseService() {
			targetService, ok := y.Services[service.BaseService]
			if !ok {
				return fmt.Errorf("service.%s.baseService:%s 未找到", service.Name(), service.BaseService)
			}
			if strings.Compare(targetService.Name(), service.Name()) == 0 {
				return fmt.Errorf("service.%s.baseService:%s 不能是服务自身", service.Name(), service.BaseService)
			}
			if targetService.HasBaseService() {
				return fmt.Errorf("service.%s.baseService:%s 不能也是引用服务", service.Name(), service.BaseService)
			}
			service.ApplyBaseService(targetService)
		}
	}

	for !context.finished() {
		for _, service := range context.unprocessed {
			if err := service.processDependsOn(context); nil != err {
				return err
			} else {
				break
			}
		}
	}

	y.sequencedServiceNames = utils.Select(context.processed, func(s *Service) string { return s.name }).([]string)
	return nil
}

func (y CI) GetCredential(name string, ref *string) *Credential {
	if v, ok := utils.FirstOrDefault(y.Credentials, func(c *Credential) bool {
		return c.Match(name, ref)
	}); ok {
		return v.(*Credential)
	}
	return nil
}

func (y CI) Clone() *CI {
	cloned := &CI{
		Variables:             utils.CopyMap(y.Variables).(map[string]string),
		CommonEnvs:            utils.CopyMap(y.CommonEnvs).(map[string]string),
		ThirdServices:         make(map[string]*Service),
		Services:              make(map[string]*Service),
		Credentials:           make([]*Credential, 0),
		sequencedServiceNames: append([]string{}, y.sequencedServiceNames...),
	}

	if nil != y.ThirdServices {
		for name, service := range y.ThirdServices {
			cloned.ThirdServices[name] = service.Clone()
		}
	}

	if nil != y.Services {
		for name, service := range y.Services {
			cloned.Services[name] = service.Clone()
		}
	}

	if nil != y.Credentials {
		cloned.Credentials = make([]*Credential, 0, len(y.Credentials))
		for _, credential := range y.Credentials {
			cloned.Credentials = append(cloned.Credentials, credential.Clone())
		}
	}

	return cloned
}
