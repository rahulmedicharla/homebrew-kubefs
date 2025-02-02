/*
Copyright © 2025 Rahul Medicharla <rmedicharla@gmail.com>

*/
package cmd

import (
	"fmt"
	"github.com/spf13/cobra"
	"github.com/rahulmedicharla/kubefs/utils"
	"github.com/rahulmedicharla/kubefs/types"
	"os/exec"
	"os"
	"net/http"
	"io"
)

// deployCmd represents the deploy command
var deployCmd = &cobra.Command{
	Use:   "deploy [command]",
	Short: "kubefs deploy - create helm charts & deploy the build targets onto the cluster",
	Long: `kubefs deploy - create helm charts & deploy the build targets onto the cluster`,
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Help()
	},
}

func downloadZip(url string, name string) int {
	cmd := exec.Command("sh", "-c", fmt.Sprintf("(cd %s && rm -rf deploy)", name))
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmdErr := cmd.Run()
	if cmdErr != nil {
		utils.PrintError(fmt.Sprintf("Error removing old helm charts: %v", cmdErr))
		return types.ERROR
	}

	resp, err := http.Get(url)
	if err != nil {
		utils.PrintError(fmt.Sprintf("Error downloading helm chart: %v", err))
		return types.ERROR
	}

	defer resp.Body.Close()

	out, err := os.Create(fmt.Sprintf("%s/helm.zip", name))
	if err != nil {
		utils.PrintError(fmt.Sprintf("Error creating helm chart: %v", err))
		return types.ERROR
	}
	defer out.Close()

	_, err = io.Copy(out, resp.Body)
	if err != nil {
		utils.PrintError(fmt.Sprintf("Error copying helm chart: %v", err))
		return types.ERROR
	}

	cmd = exec.Command("sh", "-c", fmt.Sprintf("(cd %s && unzip helm.zip -d deploy && rm -rf helm.zip deploy/__MACOSX && echo '')", name))
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmdErr = cmd.Run()
	if cmdErr != nil {
		utils.PrintError(fmt.Sprintf("Error unzipping helm chart: %v", cmdErr))
		return types.ERROR
	}

	return types.SUCCESS
}

func deployUnique(resource *types.Resource, onlyHelmify bool, onlyDeploy bool, target string) int {
	cmd := exec.Command("sh", "-c", fmt.Sprintf("docker pull %s", resource.DockerRepo))
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err := cmd.Run()
	if err != nil {
		utils.PrintError(fmt.Sprintf("Docker image %s not found. Run 'kubefs compile' to set docker images", resource.Name))
		return types.ERROR
	}

	if !onlyDeploy {
		// helmify

		if resource.Type == "api"{
			err := downloadZip(types.APIHELM, resource.Name)
			if err == types.ERROR {
				return types.ERROR
			}

			defaultYaml := types.GetHelmChart(resource.DockerRepo, resource.Name, "ClusterIP", resource.Port, "false", "/health")
			fileWriteErr := os.WriteFile(fmt.Sprintf("%s/deploy/values.yaml", resource.Name), []byte(defaultYaml), 0644)

			if fileWriteErr != nil {
				return types.ERROR
			}

			err, valuesYaml := utils.ReadYaml(fmt.Sprintf("%s/deploy/values.yaml", resource.Name))
			if err == types.ERROR {
				return types.ERROR
			}
			env := valuesYaml["env"].([]interface{})
			for _, r := range utils.ManifestData.Resources {
				env = append(env, map[string]interface{}{"name": fmt.Sprintf("%sHOST", r.Name), "value": r.ClusterHost})
			}
			valuesYaml["env"] = env

			err = utils.WriteYaml(&valuesYaml, fmt.Sprintf("%s/deploy/values.yaml", resource.Name))
			if err == types.ERROR {
				return types.ERROR
			}

		}else if resource.Type == "frontend"{
			err := downloadZip(types.FRONTENDHELM, resource.Name)
			if err == types.ERROR {
				return types.ERROR
			}

			defaultYaml := types.GetHelmChart(resource.DockerRepo, resource.Name, "LoadBalancer", 80, "true", "/")
			fileWriteErr := os.WriteFile(fmt.Sprintf("%s/deploy/values.yaml", resource.Name), []byte(defaultYaml), 0644)

			if fileWriteErr != nil {
				return types.ERROR
			}

			err, valuesYaml := utils.ReadYaml(fmt.Sprintf("%s/deploy/values.yaml", resource.Name))
			if err == types.ERROR {
				return types.ERROR
			}
			for _, r := range utils.ManifestData.Resources {
				valuesYaml["kubefsHelper"].(map[string]interface{})["env"] = append(valuesYaml["kubefsHelper"].(map[string]interface{})["env"].([]interface{}), map[string]interface{}{"name": fmt.Sprintf("%sHOST", r.Name), "value": r.ClusterHost})
			}

			err = utils.WriteYaml(&valuesYaml, fmt.Sprintf("%s/deploy/values.yaml", resource.Name))
			if err == types.ERROR {
				return types.ERROR
			}
		}else{
			var cmd *exec.Cmd

			if resource.Framework == "cassandra"{
				cmd = exec.Command("sh", "-c", fmt.Sprintf("helm pull oci://registry-1.docker.io/bitnamicharts/cassandra --untar && mv cassandra %s/deploy", resource.Name))
			}else{
				cmd = exec.Command("sh", "-c", fmt.Sprintf("helm pull oci://registry-1.docker.io/bitnamicharts/redis --untar && mv redis %s/deploy", resource.Name))
			}

			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
			cmdErr := cmd.Run()
			if cmdErr != nil {
				utils.PrintError(fmt.Sprintf("Error pulling helm chart: %v", cmdErr))
				return types.ERROR
			}

		}
	}

	if !onlyHelmify {
		// deploy
		if target == "local" {
			var cmd *exec.Cmd

			if resource.Type == "database"{
				if resource.Framework == "cassandra"{
					cmd = exec.Command("sh", "-c", fmt.Sprintf("kubectl create namespace %s; helm upgrade --install %s %s/deploy --set service.ports.cql=%v --set persistence.size=1Gi --namespace %s", resource.Name, resource.Name, resource.Name, resource.Port, resource.Name))
				}else{
					cmd = exec.Command("sh", "-c", fmt.Sprintf("kubectl create namespace %s; helm upgrade --install %s %s/deploy --set service.ports.redis=%v --set persistence.size=1Gi --namespace %s", resource.Name, resource.Name, resource.Name, resource.Port, resource.Name))
				}
			}else{
				cmd = exec.Command("sh", "-c", fmt.Sprintf("helm upgrade --install %s %s/deploy", resource.Name, resource.Name))
			}

			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
			cmdErr := cmd.Run()
			if cmdErr != nil {
				utils.PrintError(fmt.Sprintf("Error deploying resource %s: %v", resource.Name, cmdErr))
				return types.ERROR
			}
		}
	}

	return types.SUCCESS

}

var deployAllCmd = &cobra.Command{
	Use:   "all",
	Short: "kubefs deploy all - create helm charts & deploy the build targets onto the cluster for all resources",
	Long: `kubefs deploy all - create helm charts & deploy the build targets onto the cluster for all resources`,
	Run: func(cmd *cobra.Command, args []string) {
		var onlyHelmify, onlyDeploy bool
		var target string
		onlyHelmify, _ = cmd.Flags().GetBool("only-helmify")
		onlyDeploy, _ = cmd.Flags().GetBool("only-deploy")
		target, _ = cmd.Flags().GetString("target")

		if utils.ManifestStatus == types.ERROR {
			utils.PrintError("Not a valid kubefs project: use 'kubefs init' to create a new project")
			return
		}

		if target != "local" && target != "aws" && target != "gcp" && target != "azure" {
			utils.PrintError("Invalid target cluster: use 'local', 'aws', 'gcp', or 'azure'")
			return
		}

        utils.PrintWarning("Deploying all resources")

        for _, resource := range utils.ManifestData.Resources {
			err := deployUnique(&resource, onlyHelmify, onlyDeploy, target)
			if err == types.ERROR {
				utils.PrintError(fmt.Sprintf("Error deploying resource %s", resource.Name))
			}
        }

        utils.PrintSuccess("All resources deployed successfully")
	},
}

var deployResourceCmd = &cobra.Command{
	Use:   "resource [name]",
	Short: "kubefs deploy resource - create helm charts & deploy the build targets onto the cluster for a specific resource",
	Long: `kubefs deploy resource - create helm charts & deploy the build targets onto the cluster for a specific resource`,
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) < 1 {
			cmd.Help()
			return
		}

		if utils.ManifestStatus == types.ERROR {
			utils.PrintError("Not a valid kubefs project: use 'kubefs init' to create a new project")
			return
		}

		var onlyHelmify, onlyDeploy bool
		var target string
		onlyHelmify, _ = cmd.Flags().GetBool("only-helmify")
		onlyDeploy, _ = cmd.Flags().GetBool("only-deploy")
		target, _ = cmd.Flags().GetString("target")

		if target != "local" && target != "aws" && target != "gcp" && target != "azure" {
			utils.PrintError("Invalid target cluster: use 'local', 'aws', 'gcp', or 'azure'")
			return
		}

        name := args[0]
        utils.PrintWarning(fmt.Sprintf("Deploying resource %s", name))

		var resource *types.Resource
		resource = utils.GetResourceFromName(name)

		if resource == nil {
			utils.PrintError(fmt.Sprintf("Resource %s not found", name))
			return
		}

		err := deployUnique(resource, onlyHelmify, onlyDeploy, target)
		if err == types.ERROR {
			utils.PrintError(fmt.Sprintf("Error deploying resource %s", name))
			return
		}

        utils.PrintSuccess(fmt.Sprintf("Resource %s deployed successfully", name))
	},
}

func init() {
	rootCmd.AddCommand(deployCmd)
	deployCmd.AddCommand(deployAllCmd)
	deployCmd.AddCommand(deployResourceCmd)

	deployCmd.PersistentFlags().StringP("target", "t", "local", "target cluster to deploy the resources onto [local|aws|gcp|azure]")
	deployCmd.PersistentFlags().BoolP("only-helmify", "w", false, "only helmify the resources")
	deployCmd.PersistentFlags().BoolP("only-deploy", "d", false, "only deploy the resources")

}
