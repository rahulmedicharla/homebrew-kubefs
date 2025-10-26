package utils

import (
	"fmt"
	"github.com/rahulmedicharla/kubefs/types"
)

func SetupMinikube(clusterName string) error {
	PrintSuccess("Starting Minikube cluster & enabling required addons...")
	commands := []string{
		fmt.Sprintf("minikube start -p %s", clusterName),
		fmt.Sprintf("minikube profile %s", clusterName),
		"minikube addons enable ingress",
		"minikube addons enable metrics-server",
		"minikube stop",
	}

	return RunMultipleCommands(commands, true, true)
}

func UpdateMinikubeContext(config *types.CloudConfig) error {
	return RunCommand(fmt.Sprintf("minikube profile %s", config.ClusterName), true, true)
}

func GetMinikubeCluster(config *types.CloudConfig) error {
	err := RunCommand(fmt.Sprintf("minikube start -p %s", config.ClusterName), true, true)
	if err != nil {
		return fmt.Errorf("failed to start minikube cluster %v", err.Error())
	}

	return UpdateMinikubeContext(config)
}

func DeleteMinikubeCluster(config *types.CloudConfig) error {
	return RunCommand(fmt.Sprintf("minikube delete -p %s", config.ClusterName), true, true)
}