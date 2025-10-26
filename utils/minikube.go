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

func GetMinikubeContext(config *types.CloudConfig) error {
	return RunCommand(fmt.Sprintf("minikube profile %s", config.ClusterName), true, true)
}

func GetMinikubeCluster(config *types.CloudConfig) error {
	err := GetMinikubeContext(config)	
	if err != nil {
		startErr := RunCommand(fmt.Sprintf("minikube start -p %s", config.ClusterName), true, true)
		if startErr != nil {
			return fmt.Errorf("failed to start minikube cluster %v", err.Error())
		}
		return GetMinikubeContext(config)
	}

	return nil

}

func DeleteMinikubeCluster(config *types.CloudConfig) error {
	return RunCommand(fmt.Sprintf("minikube delete -p %s", config.ClusterName), true, true)
}