package types

import (
	"fmt"
)

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
	Framework string `yaml:"framework"`
	UpLocal string `yaml:"up_local,omitempty"`
	LocalHost string `yaml:"local_host,omitempty"`
	DockerHost string `yaml:"docker_host,omitempty"`
	DockerRepo string `yaml:"docker_repo,omitempty"`
	ClusterHost string `yaml:"cluster_host,omitempty"`
  DbPassword string `yaml:"db_password,omitempty"`
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
  APIHELM = "https://www.dropbox.com/scl/fi/ajfh7yugc00y9z84mmwxw/kubefs-helm-api.zip?rlkey=hthd6914sleelssvesjj0i9uc&st=1vgniqlf&dl=1"
)

var FRAMEWORKS = map[string][]string{
	"api": {"nest", "fast", "go"},
	"frontend": {"next", "sveltekit", "remix"},
	"database": {"cassandra", "redis", "elasticsearch"},
}

func GetHelmChart(dockerRepo string, name string, serviceType string, port int, ingressEnabled string, healthCheck string) string{
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
  host: ""
  tls: []

env: []
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
`, dockerRepo, name, serviceType, port, ingressEnabled, healthCheck, healthCheck)
}