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
  APIHELM = "https://www.dropbox.com/scl/fi/wjkg3ku50a0snqw1lbst5/kubefs_helm_api.zip?rlkey=onfprl9zft534c7n6n1puniye&st=0biwajb7&dl=1"
  FRONTENDHELM = "https://www.dropbox.com/scl/fi/p22jght8sl31i0609ftia/kubefs_helm_frontend.zip?rlkey=6c5tus3vwcr9aglemqsoh52ui&st=h2u1f4g9&dl=1"
  DBHELM = "https://www.dropbox.com/scl/fi/6ztaevp929v1ypdx9xsdm/kubefs_helm_db.zip?rlkey=lde5mi6pj12v9uu0lgwgfiqah&st=atyt5pt8&dl=1"
)

var FRAMEWORKS = map[string][]string{
	"api": {"koa", "fast", "go"},
	"frontend": {"react", "angular", "vue"},
	"database": {"cassandra", "mongodb"},
}

func GetHelmChart() string{
  
}

func GetMongoCompose(port int) string {
	return fmt.Sprintf(`
services:
  container:
    image: mongo:latest
    ports:
      - "%v:27017"
    environment: []
    volumes:
      - mongo_data:/data/db
    networks:
      - shared_network

  setup:
    image: mongo:latest
    command: |
      bash -c '
      until mongosh --host container --port 27017 --eval "db.runCommand({ ping: 1 })"; do
        echo "Waiting for MongoDB to be ready...";
        sleep 5;
      done;
      mongosh --host container --port 27017 --eval "db.getSiblingDB(\"default\").createCollection(\"default\")"
      '
    depends_on:
      - container
    restart: "no"
    networks:
      - shared_network

volumes:
  mongo_data:

networks:
  shared_network:
    external: true
`, port)
}

func GetCassandraCompose(port int) string {
	return fmt.Sprintf(`
services:
  container:
    image: cassandra:latest
    ports:
      - "%v:9042"
    environment:
      - CASSANDRA_CLUSTER_NAME=cluster
    volumes:
      - cassandra_data:/var/lib/cassandra
    networks:
      - shared_network
  setup:
    image: cassandra:latest
    command: |
      bash -c "
      until cqlsh container 9042 -e 'describe keyspaces'; do
        echo 'Waiting for cassandra to be ready...';
        sleep 5;
      done;
      cqlsh container 9042 -e \"CREATE KEYSPACE default WITH replication = {'class': 'SimpleStrategy', 'replication_factor': 1}\""
    depends_on:
      - container
    restart: "no"
    networks:
      - shared_network

volumes:
  cassandra_data:

networks:
  shared_network:
    external: true
`, port)
}