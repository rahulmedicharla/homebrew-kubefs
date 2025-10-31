package types

type Project struct {
	KubefsName  string              `yaml:"name"`
	Version     string              `yaml:"version"`
	Description string              `yaml:"description"`
	Resources   map[string]Resource `yaml:"resources"`
	Addons      []Addon             `yaml:"addons"`
	CloudConfig []CloudConfig       `yaml:"cloud_config"`
}

type Resource struct {
	Port            int               `yaml:"port"`
	Type            string            `yaml:"type"`
	Framework       string            `yaml:"framework"`
	AttachCommand   map[string]string `yaml:"attach_command"`
	UpLocal         string            `yaml:"up_local,omitempty"`
	LocalHost       string            `yaml:"local_host,omitempty"`
	DockerHost      string            `yaml:"docker_host,omitempty"`
	DockerRepo      string            `yaml:"docker_repo,omitempty"`
	ClusterHost     string            `yaml:"cluster_host,omitempty"`
	ClusterHostRead string            `yaml:"cluster_host_read,omitempty"`
	Dependents      []string          `yaml:"dependents,omitempty"`
	Opts            map[string]string `yaml:"opts,omitempty"`
	Environment     map[string]string `yaml:"environment,omitempty"`
}

type Addon struct {
	Name         string   `yaml:"name"`
	Port         int      `yaml:"port"`
	DockerRepo   string   `yaml:"docker_repo"`
	LocalHost    string   `yaml:"local_host"`
	DockerHost   string   `yaml:"docker_host"`
	ClusterHost  string   `yaml:"cluster_host"`
	Dependencies []string `yaml:"dependencies,omitempty"`
	Environment  []string `yaml:"environment,omitempty"`
}

type CloudConfig struct {
	Provider     string   `yaml:"provider"`
	ProjectId    string   `yaml:"project_id,omitempty"`
	ProjectName  string   `yaml:"project_name,omitempty"`
	Region       string   `yaml:"region,omitempty"`
	ClusterNames []string `yaml:"cluster_names,omitempty"`
	MainCluster  string   `yaml:"main_cluster,omitempty"`
}

type ApiResponse struct {
	Token  string `json:"token,omitempty"`
	Detail string `json:"detail,omitempty"`
}

const (
	DOCKER_LOGIN_ENDPOINT = "https://hub.docker.com/v2/users/login/"
	DOCKER_REPO_ENDPOINT  = "https://hub.docker.com/v2/repositories/"
)

var FRAMEWORKS = map[string][]string{
	"api":      {"nest", "fast", "gin"},
	"frontend": {"next", "sveltekit", "remix"},
	"database": {"postgresql", "redis"},
	"addons":   {"oauth2", "gateway"},
}

var TARGETS = []string{
	"minikube",
	"gcp",
}
