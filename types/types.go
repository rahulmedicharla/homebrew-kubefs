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
	UpDocker string `yaml:"up_docker,omitempty"`
	ClusterHost string `yaml:"cluster_host,omitempty"`
  DbUsername string `yaml:"db_username,omitempty"`
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
  APIHELM = "https://www.dropbox.com/scl/fi/whubjnyg1adtql8mk1x7k/kubefs-helm-api.zip?rlkey=jsj29bn7mn9o30e2atc49cugq&st=37v4al9g&dl=1"
  FRONTENDHELM = "https://www.dropbox.com/scl/fi/svzju9j1anh4bbkavjarh/kubefs-helm-frontend.zip?rlkey=yqq9w05ilki3db2jr48dlm54g&st=c2pgcdro&dl=1"
  DBHELM = "https://www.dropbox.com/scl/fi/osr60qcihytmosu3vqqvg/kubefs-helm-db.zip?rlkey=0xgzcbvo54ung88abyg3rdxrq&st=ete907jx&dl=1"
)

var FRAMEWORKS = map[string][]string{
	"api": {"koa", "fast", "go"},
	"frontend": {"react", "angular", "vue"},
	"database": {"cassandra", "redis"},
}

func GetHelmChart(dockerRepo string, name string, serviceType string, port int, ingressEnabled string) string{
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

resources: {}

livenessProbe:
  httpGet:
    path: /
    port: http
readinessProbe:
  httpGet:
    path: /
    port: http

kubefsHelper:
  volumeMounts:
    - name: traefik-config
      mountPath: /config
  volumes:
    - name: traefik-config
      configMap:
        name: traefik-config
  port: 6000
  env: []
  livenessProbe:
    initialDelaySeconds: 5
    periodSeconds: 5
    httpGet:
      path: /health
      port: 6000
  readinessProbe:
    initialDelaySeconds: 5
    periodSeconds: 5
    httpGet:
      path: /health
      port: 6000

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
`, dockerRepo, name, serviceType, port, ingressEnabled)
}

func GetRedisCompose(port int, password string) string {
	return fmt.Sprintf(`
services:
  container:
    image: bitnami/redis:latest
    ports:
      - "%v:6379"
    environment: 
      - REDIS_PASSWORD=%s
    volumes:
      - redis_data:/bitnami/redis/data
    networks:
      - shared_network

volumes:
  redis_data:
    driver: local

networks:
  shared_network:
    external: true
`, port, password)
}

func GetCassandraCompose(port int, username string, password string) string {
	return fmt.Sprintf(`
services:
  container:
    image: bitnami/cassandra:latest
    ports:
      - "%v:9042"
    volumes:
      - cassandra_data:/bitnami
    environment:
      - CASSANDRA_USER=%s
      - CASSANDRA_PASSWORD=%s
    networks:
      - shared_network

volumes:
  cassandra_data:

networks:
  shared_network:
    external: true
`, port, username, password)
}