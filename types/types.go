package types

import (
	"fmt"
)

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
  Opts map[string]string `yaml:"opts,omitempty"`
  Dependencies []string `yaml:"dependencies,omitempty"`
}

type ApiResponse struct {
	Token string `json:"token",omitempty`
	Detail string `json:"detail, omitempty"`
}

const (
	ERROR = 0
	SUCCESS = 1
)

const (
  HELMCHART = "https://www.dropbox.com/scl/fi/ysju5bkpup02eiy7b3qde/helm-template.zip?rlkey=9gzobe08xdugaymr7kz1kyt4o&st=eskuno42&dl=1"
)

var FRAMEWORKS = map[string][]string{
	"api": {"nest", "fast", "go"},
	"frontend": {"next", "sveltekit", "remix"},
	"database": {"cassandra", "redis"},
  "addons": {"oauth2"},
}

func GetHelmChart(dockerRepo string, name string, serviceType string, port int, ingressEnabled string, ingressHost string, healthCheck string) string{
  return fmt.Sprintf(`
replicaCount: 3
image:
  #CHANGE LINE BELOW
  repository: %s
  pullPolicy: Always
  tag: latest

imagePullSecrets: []
nameOverride: ""
fullnameOverride: ""

#CHANGE LINE BELOW
namespace: %s

serviceAccount:
  create: true
  automount: true
  annotations: {}
  name: ""

podAnnotations: {}
podLabels: {}

podSecurityContext: {}

securityContext: {}

service:
  #CHANGE BOTH LINES BELOW
  type: %s
  port: %v

ingress:
  #CHANGE LINE BELOW
  enabled: %s
  className: nginx
  annotations: 
    kubernetes.io/ingress.class: nginx
    nginx.ingress.kubernetes.io/rewrite-target: /
  #CHANGE LINE BELOW, ADD HOST PATH FOR FRONTEND INGRESS
  host: %s
  tls: []

env: []
secrets: []
resources: {}

livenessProbe:
  httpGet:
    path: %s
    port: http
readinessProbe:
  httpGet:
    path: %s
    port: http
autoscaling:
  enabled: false
  minReplicas: 1
  maxReplicas: 100
  targetCPUUtilizationPercentage: 80

volumes: []
volumeMounts: []
nodeSelector: {}
tolerations: []
affinity: {}
`, dockerRepo, name, serviceType, port, ingressEnabled, ingressHost, healthCheck, healthCheck)
}