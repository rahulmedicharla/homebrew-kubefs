package types

type Project struct {
	KubefsName string `yaml:"name"`
	Version string `yaml:"version"`
	Description string `yaml:"description"`
	Resources []Resource `yaml:"resources"`
  Addons []Addon `yaml:"addons"`
}

type Resource struct {
	Name string `yaml:"name"`
	Port int `yaml:"port"`
	Type string `yaml:"type"`
	Framework string `yaml:"framework"`
	UpLocal string `yaml:"up_local,omitempty"`
	LocalHost string `yaml:"local_host,omitempty"`
	DockerHost string `yaml:"docker_host,omitempty"`
	DockerRepo string `yaml:"docker_repo,omitempty"`
	ClusterHost string `yaml:"cluster_host,omitempty"`
	DbPassword string `yaml:"db_password,omitempty"`
	UrlHost string `yaml:"url_host,omitempty"`
}

type Addon struct {
  Name string `yaml:"name"`
  Port int `yaml:"port"`
  DockerRepo string `yaml:"docker_repo"`
  LocalHost string `yaml:"local_host"`
  DockerHost string `yaml:"docker_host"`
  ClusterHost string `yaml:"cluster_host"`
  Opts map[string][]string `yaml:"opts,omitempty"`
  Dependencies []string `yaml:"dependencies,omitempty"`
}

type ApiResponse struct {
	Token string `json:"token",omitempty`
	Detail string `json:"detail,omitempty"`
}

const (
	ERROR = 0
	SUCCESS = 1
)

const (
	DOCKER_LOGIN_ENDPOINT = "https://hub.docker.com/v2/users/login/"
	DOCKER_REPO_ENDPOINT = "https://hub.docker.com/v2/repositories/"
)

const (
  HELMCHART = "https://www.dropbox.com/scl/fi/ysju5bkpup02eiy7b3qde/helm-template.zip?rlkey=9gzobe08xdugaymr7kz1kyt4o&st=eskuno42&dl=1"
  OAUTH2CHART = "https://www.dropbox.com/scl/fi/0jgd41yu5az584gd9me5i/kubefs-oauth-helm.zip?rlkey=a0y3mllr431dl8xedaz7q3x3z&st=uqka20an&dl=1"
)

var FRAMEWORKS = map[string][]string{
	"api": {"nest", "fast", "go"},
	"frontend": {"next", "sveltekit", "remix"},
	"database": {"cassandra", "redis"},
  "addons": {"oauth2"},
}