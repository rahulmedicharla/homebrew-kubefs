package utils

import (
	"fmt"
	"github.com/rahulmedicharla/kubefs/types"
)

func ProvisionMinikubeCluster(clusterName string) error {
	PrintSuccess("Provisioning Minikube cluster & enabling required addons...")
	commands := []string{
		fmt.Sprintf("minikube start -p %s", clusterName),
		fmt.Sprintf("minikube profile %s", clusterName),
		"minikube addons enable ingress",
		"minikube addons enable metrics-server",
	}

	return RunMultipleCommands(commands, true, true)
}

func DeleteMinikubeCluster(config *types.CloudConfig, clusterName string) error {
	return RunCommand(fmt.Sprintf("minikube delete -p %s", clusterName), true, true)
}

func PauseMinikubeCluster(config *types.CloudConfig, clusterName string) error {
	return RunCommand(fmt.Sprintf("minikube stop -p %s", clusterName), true, true)
}

func StartMinikubeCluster(config *types.CloudConfig, clusterName string) error {
	return RunCommand(fmt.Sprintf("minikube start -p %s", clusterName), true, true)
}

func GetMinikubeContext(config *types.CloudConfig) error {
	return RunCommand(fmt.Sprintf("minikube profile %s", config.MainCluster), true, true)
}