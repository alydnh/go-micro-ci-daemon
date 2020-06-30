package yaml

type Registry struct {
	Type    string `yaml:"type"`
	Address string `yaml:"address"`
	Port    int    `yaml:"port"`
	UseSSL  bool   `yaml:"useSSL"`
}
