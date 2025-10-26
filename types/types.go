package types

type Project struct {
	KubefsName string `yaml:"name"`
	Version string `yaml:"version"`
	Description string `yaml:"description"`
	Resources []Resource `yaml:"resources"`
  	Addons []Addon `yaml:"addons"`
	CloudConfig []CloudConfig `yaml:"cloud_config"`
}

type Resource struct {
	Name string `yaml:"name"`
	Port int `yaml:"port"`
	Type string `yaml:"type"`
	Framework string `yaml:"framework"`
	AttachCommand map[string]string `yaml:"attach_command"`
	UpLocal string `yaml:"up_local",omitempty`
	LocalHost string `yaml:"local_host",omitempty`
	DockerHost string `yaml:"docker_host",omitempty`
	DockerRepo string `yaml:"docker_repo",omitempty`
	ClusterHost string `yaml:"cluster_host",omitempty`
	ClusterHostRead string `yaml:"cluster_host_read",omitempty`
	Dependents []string `yaml:"dependents",omitempty`
	Opts map[string]string `yaml:"opts",omitempty`
}

type Addon struct {
  Name string `yaml:"name"`
  Port int `yaml:"port"`
  DockerRepo string `yaml:"docker_repo"`
  LocalHost string `yaml:"local_host"`
  DockerHost string `yaml:"docker_host"`
  ClusterHost string `yaml:"cluster_host"`
  Dependencies []string `yaml:"dependencies",omitempty`
  Environment []string `yaml:"environment",omitempty`
}

type CloudConfig struct {
	Provider string `yaml:"provider"`
	ProjectId string `yaml:"project_id", omitempty`
	ProjectName string `yaml:"project_name", omitempty`
	Region string `yaml:"region", omitempty`
	ClusterName string `yaml:"cluster_name",omitempty`
}

type ApiResponse struct {
	Token string `json:"token",omitempty`
	Detail string `json:"detail",omitempty`
}
const (
	DOCKER_LOGIN_ENDPOINT = "https://hub.docker.com/v2/users/login/"
	DOCKER_REPO_ENDPOINT = "https://hub.docker.com/v2/repositories/"
)
var FRAMEWORKS = map[string][]string{
	"api": {"nest", "fast", "gin"},
	"frontend": {"next", "sveltekit", "remix"},
	"database": {"postgresql", "redis"},
	"addons": {"oauth2"},
}