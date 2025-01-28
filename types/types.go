package types

type Project struct {
	KubefsName string `yaml:"name"`
	Version string `yaml:"version"`
	Description string `yaml:"description"`
	Resources []Resource `yaml:"resources"`
}

type Resource struct {
	Name string `yaml:"name"`
	Type string `yaml:"type"`
	Target string `yaml:"target"`
}