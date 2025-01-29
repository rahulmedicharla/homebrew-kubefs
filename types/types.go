package types

type Project struct {
	KubefsName string `yaml:"name"`
	Version string `yaml:"version"`
	Description string `yaml:"description"`
	Resources []Resource `yaml:"resources"`
}

type Resource struct {
	Name string `yaml:"name"`
	Port int `yaml:"port"`
	Type string `yaml:"type"`
	Framework string `yaml:"framework,omitempty"`
	UpLocal string `yaml:"up_local"`
	LocalHost string `yaml:"local_host"`
	DockerHost string `yaml:"docker_host"`
	ClusterHost string `yaml:"cluster_host"`
}
const (
	ERROR = 0
	SUCCESS = 1
)

var FRAMEWORKS = map[string][]string{
	"api": {"koa", "fast", "go"},
	"frontend": {"react", "angular", "vue"},
	"database": {"cassandra", "mongodb"},
}