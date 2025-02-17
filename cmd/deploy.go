/*
Copyright © 2025 Rahul Medicharla <rmedicharla@gmail.com>

*/
package cmd

import (
	"fmt"
	"github.com/spf13/cobra"
	"github.com/rahulmedicharla/kubefs/utils"
	"github.com/rahulmedicharla/kubefs/types"
	"strings"
	"github.com/thanhpk/randstr"
)

// deployCmd represents the deploy command
var deployCmd = &cobra.Command{
	Use:   "deploy [command]",
	Short: "kubefs deploy - create helm charts & deploy the build targets onto the cluster",
	Long: `kubefs deploy - create helm charts & deploy the build targets onto the cluster
example:
	kubefs deploy all --flags,
	kubefs deploy resource <frontend> <api> <database> --flags,
	kubefs deploy addons <addon-name> <addon-name> --flags,`,
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Help()
	},
}

func deployAddon(addon *types.Addon, onlyHelmify bool, onlyDeploy bool) error {
	err := utils.RunCommand(fmt.Sprintf("docker pull %s", addon.DockerRepo), true, true)
	if err != nil {
		return err
	}
	pass := randstr.String(16)

	if !onlyDeploy {
		// helmify
		var valuesYaml map[string]interface{}
		if addon.Name == "oauth2"{
			if err = utils.DownloadZip(types.OAUTH2CHART, "addons/oauth2"); err != nil {
				return err
			}
			valuesYaml = *utils.GetHelmChart(addon.DockerRepo, addon.Name, "ClusterIP", addon.Port, false, "", "/health", 3)

			env := valuesYaml["env"].([]interface{})
			var allowedOrigins []string
			for _, n := range addon.Dependencies {
				attachedResource, err := utils.GetResourceFromName(n)
				if err != nil {
					return err
				}

				allowedOrigins = append(allowedOrigins, attachedResource.ClusterHost)
			}
			
			env = append(env, map[string]interface{}{
				"name": "ALLOWED_ORIGINS", 
				"value": strings.Join(allowedOrigins, ","),
			}, map[string]interface{}{
				"name": "PORT",
				"value": fmt.Sprintf("%v", addon.Port),
			}, map[string]interface{}{
				"name": "MODE",
				"value": "release",
			}, map[string]interface{}{
				"name": "NAME",
				"value": utils.ManifestData.KubefsName,
			}, map[string]interface{}{
				"name": "WRITE_CONNECTION_STRING",
				"value": fmt.Sprintf("postgresql://postgres:%s@auth-data-postgresql-primary:5432/auth?sslmode=verify-ca&sslrootcert=/etc/ssl/certs/tls/ca.crt", pass),
			}, map[string]interface{}{
				"name": "READ_CONNECTION_STRING",
				"value": fmt.Sprintf("postgresql://postgres:%s@auth-data-postgresql-read:5432/auth?sslmode=verify-ca&sslrootcert=/etc/ssl/certs/tls/ca.crt", pass),
			})

			for _, envVar := range addon.Environment{
				env = append(env, map[string]interface{}{"name": strings.Split(envVar, "=")[0], "value": strings.Split(envVar, "=")[1]})
			}

			valuesYaml["env"] = env

			valuesYaml["secrets"] = []interface{}{
				map[string]interface{}{
					"name": "public_key.pem",
					"value": "files/public_key.pem",
				},
				map[string]interface{}{
					"name": "private_key.pem",
					"value": "files/private_key.pem",
				},
			}

			err = utils.RunCommand(fmt.Sprintf("mkdir -p addons/oauth2/deploy/files && cp addons/oauth2/public_key.pem addons/oauth2/deploy/files && cp addons/oauth2/private_key.pem addons/oauth2/deploy/files"), true, true)
			if err != nil {
				return err
			}

			valuesYaml["volumes"] = []interface{}{
				map[string]interface{}{
					"name": "keys",
					"secret": map[string]string{
						"secretName": "oauth2-deploy-secret",
					},
				},
				map[string]interface{}{
					"name": "tls",
					"secret": map[string]string{
						"secretName": "auth-data-postgresql-crt",
					},
				},
			}

			valuesYaml["volumeMounts"] = []interface{}{
				map[string]string{
					"name": "keys",
					"mountPath": "/etc/ssl/private/private_key.pem",
					"subPath": "private_key.pem",
				},
				map[string]string{
					"name": "keys",
					"mountPath": "/etc/ssl/public/public_key.pem",
					"subPath": "public_key.pem",
				},
				map[string]string{
					"name": "tls",
					"mountPath": "/etc/ssl/certs/tls",
				},
			}

			err = utils.WriteYaml(&valuesYaml, "addons/oauth2/deploy/values.yaml")
			if err != nil {
				return err
			}
		}
	}
	if !onlyHelmify {
		// deploy
		if addon.Name == "oauth2"{
			configs := []string{
				"--set namespaceOverride=oauth2",
				"--set auth.postgresPassword=" + pass,
				"--set architecture=replication",
				"--set auth.database=auth",
				"--set readReplicas.replicaCount=3",
				"--set primary.persistence.size=1Gi",
				"--set readReplicas.persistence.size=1Gi",
				"--set primary.initdb.scripts.\"init-user\\.sql\"=\"CREATE TABLE IF NOT EXISTS accounts (uid UUID PRIMARY KEY\\, email TEXT\\, password TEXT\\, secret TEXT);CREATE TABLE IF NOT EXISTS refreshTokens (uid UUID PRIMARY KEY\\, token TEXT);CREATE TABLE IF NOT EXISTS twoFactorAuth (email TEXT PRIMARY KEY\\, secret TEXT);\"",
				"--set tls.enabled=true",
				"--set tls.autoGenerated=true",
			}

			commandBuilder := strings.Builder{}
			commandBuilder.WriteString("helm upgrade --install auth-data")
			for _, c := range configs {
				commandBuilder.WriteString(fmt.Sprintf(" %s", c))
			}
			commandBuilder.WriteString(" oci://registry-1.docker.io/bitnamicharts/postgresql")

			commands := []string{
				"helm upgrade --install oauth2 addons/oauth2/deploy",
				commandBuilder.String(),
			}

			err = utils.RunMultipleCommands(commands, true, true)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func deployUnique(resource *types.Resource, onlyHelmify bool, onlyDeploy bool) error {
	err := utils.RunCommand(fmt.Sprintf("docker pull %s", resource.DockerRepo), true, true)
	if err != nil {
		return err
	}

	if !onlyDeploy {
		// helmify
		if resource.Type == "database"{
			// database
			var cmds []string
			if resource.Framework == "postgresql"{
				cmds = append(cmds, 
					fmt.Sprintf("(cd %s; rm -rf deploy; helm pull oci://registry-1.docker.io/bitnamicharts/postgresql --untar && mv postgresql deploy)", resource.Name),
				)
			}else{
				cmds = append(cmds, 
					fmt.Sprintf("(cd %s; rm -rf deploy; helm pull oci://registry-1.docker.io/bitnamicharts/redis --untar && mv redis deploy)", resource.Name),
				)
			}

			cmds = append(cmds,fmt.Sprintf("echo 'connect using kubefs attach' > %s/deploy/templates/NOTES.txt", resource.Name))

			err = utils.RunMultipleCommands(cmds, true, true)
			if err != nil {
				return err
			}

		}else{
			// api or frontend
			if err = utils.DownloadZip(types.HELMCHART, resource.Name); err != nil {
				return err
			}

			var valuesYaml map[string]interface{}
			if resource.Type == "api"{
				// api
				valuesYaml = *utils.GetHelmChart(resource.DockerRepo, resource.Name, "ClusterIP", resource.Port, false, "", "/health", 3)
			}else{
				// frontend
				valuesYaml = *utils.GetHelmChart(resource.DockerRepo, resource.Name, "NodePort", resource.Port, true, resource.Opts["host-domain"], "/", 3)
			}

			env := valuesYaml["env"].([]interface{})
			for _, r := range utils.ManifestData.Resources {
				if r.Type == "database"{
					env = append(env, map[string]interface{}{"name": fmt.Sprintf("%sHOST_READ", r.Name), "value": r.ClusterHostRead})
				}
				env = append(env, map[string]interface{}{"name": fmt.Sprintf("%sHOST", r.Name), "value": r.ClusterHost})
			}

			for _, a := range resource.Dependents{
				addon, _ := utils.GetAddonFromName(a)
				env = append(env, map[string]interface{}{"name": fmt.Sprintf("%sHOST", a), "value": addon.ClusterHost})
			}
			valuesYaml["env"] = env
		
			envData, err := utils.ReadEnv(fmt.Sprintf("%s/.env", resource.Name))			
			if err == nil {
				secrets := valuesYaml["secrets"].([]interface{})
				for _,line := range envData {
					secrets = append(secrets, map[string]interface{}{"name": strings.Split(line, "=")[0], "value": strings.Split(line, "=")[1], "secretRef": fmt.Sprintf("%s-deploy-secret", resource.Name)})
				}
				valuesYaml["secrets"] = secrets
			}

			err = utils.WriteYaml(&valuesYaml, fmt.Sprintf("%s/deploy/values.yaml", resource.Name))
			if err != nil {
				return err
			}
		}
	}

	if !onlyHelmify {
		// deploy
		commandBuilder := strings.Builder{}
		if resource.Type == "database"{
			var configs []string
			if resource.Framework == "postgresql"{
				configs = []string{
					"--set primary.persistence.size=" + fmt.Sprintf("%v", resource.Opts["persistence"]),
					"--set readReplicas.persistence.size=" + fmt.Sprintf("%v", resource.Opts["persistence"]),
					"--set primary.service.ports.postgresql=80",
					"--set readReplicas.service.ports.postgresql=80",
					"--set architecture=replication",
					"--set readReplicas.replicaCount=3",
					"--set auth.database=" + resource.Opts["default-database"],
					"--set auth.username=" + resource.Opts["user"],
					"--set auth.password=" + resource.Opts["password"],
					"--set namespaceOverride=" + resource.Name,
				}
			}else{
				configs = []string{
					"--set master.persistence.size=" + fmt.Sprintf("%v", resource.Opts["persistence"]),
					"--set replica.persistence.size=" + fmt.Sprintf("%v", resource.Opts["persistence"]),
					"--set master.service.ports.redis=80",
					"--set replica.service.ports.redis=80",
					"--set auth.password=" + resource.Opts["password"],
					"--set namespaceOverride=" + resource.Name,
				}
			}

			commandBuilder.WriteString(fmt.Sprintf("kubectl create namespace %s; helm upgrade --install %s %s/deploy", resource.Name, resource.Name, resource.Name))
			for _, c := range configs {
				commandBuilder.WriteString(fmt.Sprintf(" %s", c))
			}
		}else{
			// api or frontend
			commandBuilder.WriteString(fmt.Sprintf("helm upgrade --install %s %s/deploy", resource.Name, resource.Name))
		}

		fmt.Println(commandBuilder.String())

		err = utils.RunCommand(commandBuilder.String(), true, true)
		if err != nil {
			return err
		}
	}

	return nil

}

var deployAllCmd = &cobra.Command{
	Use:   "all",
	Short: "kubefs deploy all - create helm charts & deploy the build targets onto the cluster for all resources",
	Long: `kubefs deploy all - create helm charts & deploy the build targets onto the cluster for all resources
example: 
	kubefs deploy all --flags,
	`,
	Run: func(cmd *cobra.Command, args []string) {
		if utils.ManifestStatus != nil {
			utils.PrintError(utils.ManifestStatus.Error())
			return
		}

		var onlyHelmify, onlyDeploy bool
		onlyHelmify, _ = cmd.Flags().GetBool("only-helmify")
		onlyDeploy, _ = cmd.Flags().GetBool("only-deploy")

		var errors []string
		var successes []string
		var hosts []string

        utils.PrintWarning("Deploying all resources & addons")

        for _, resource := range utils.ManifestData.Resources {
			err := deployUnique(&resource, onlyHelmify, onlyDeploy)
			if err != nil {
				utils.PrintError(fmt.Sprintf("Error deploying resource %s. %v", resource.Name, err.Error()))
				errors = append(errors, resource.Name)
				continue
			}

			successes = append(successes, resource.Name)
			if resource.Type == "frontend" {
				hosts = append(hosts, resource.Opts["host-domain"])
			}
        }

		for _, addon := range utils.ManifestData.Addons {
			err := deployAddon(&addon, onlyHelmify, onlyDeploy)
			if err != nil {
				utils.PrintError(fmt.Sprintf("Error deploying addon %s. %v", addon.Name, err.Error()))
				errors = append(errors, addon.Name)
				continue
			}
			successes = append(successes, addon.Name)
		}

		if len(errors) > 0 {
			utils.PrintError(fmt.Sprintf("Error deploying resource %v", errors))
		}

		if len(successes) > 0 {
			utils.PrintSuccess(fmt.Sprintf("Resource %v deployed successfully", successes))
		}

		if len(hosts) > 0 {
			utils.PrintWarning(fmt.Sprintf("Frontend resources are available at %v", hosts))
		}
	},
}

var deployResourceCmd = &cobra.Command{
	Use:   "resource [name ...]",
	Short: "kubefs deploy resource [name ...] - create helm charts & deploy the build targets onto the cluster for listed resource",
	Long: `kubefs deploy resource [name ...] - create helm charts & deploy the build targets onto the cluster for listed resource
example: 
	kubefs deploy resource <frontend> <api> <database> --flags,
	kubefs deploy resource <frontend> --flags
	`,
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) < 1 {
			cmd.Help()
			return
		}

		if utils.ManifestStatus != nil {
			utils.PrintError(utils.ManifestStatus.Error())
			return
		}

		var onlyHelmify, onlyDeploy bool
		onlyHelmify, _ = cmd.Flags().GetBool("only-helmify")
		onlyDeploy, _ = cmd.Flags().GetBool("only-deploy")
		
		addons, _ := cmd.Flags().GetString("with-addons")
		var addonList []string
		if addons != "" {
			addonList = strings.Split(addons, ",")
		}

		var successes []string
		var errors []string
		var hosts []string

		utils.PrintWarning(fmt.Sprintf("Deploying resource %v", args))
		utils.PrintWarning(fmt.Sprintf("Including addons %v", addonList))

		for _, name := range args {
			var resource *types.Resource
			resource, err := utils.GetResourceFromName(name)

			if err != nil {
				utils.PrintError(err.Error())
				errors = append(errors, name)
				continue
			}

			err = deployUnique(resource, onlyHelmify, onlyDeploy)
			if err != nil {
				utils.PrintError(fmt.Sprintf("Error deploying resource %s. %v", name, err.Error()))
				errors = append(errors, name)
				continue
			}

			successes = append(successes, name)
			if resource.Type == "frontend" {
				hosts = append(hosts, resource.Opts["host-domain"])
			}

		}

		for _, addon := range addonList {
			if addon == ""{
				continue
			}
			var addonResource *types.Addon
			addonResource, err := utils.GetAddonFromName(addon)
			if err != nil {
				utils.PrintError(err.Error())
				errors = append(errors, addon)
				continue
			}

			err = deployAddon(addonResource, onlyHelmify, onlyDeploy)
			if err != nil {
				utils.PrintError(fmt.Sprintf("Error deploying addon %s. %v", addon, err.Error()))
				errors = append(errors, addon)
				continue
			}

			successes = append(successes, addon)
		}

		if len(errors) > 0 {
			utils.PrintError(fmt.Sprintf("Error deploying resource %v", errors))
		}

		if len(successes) > 0 {
			utils.PrintSuccess(fmt.Sprintf("Resource %v deployed successfully", successes))
		}

		if len(hosts) > 0 {
			utils.PrintWarning(fmt.Sprintf("Frontend resources are available at %v", hosts))
		}
	},
}

var deployAddonCmd = &cobra.Command{
	Use:   "addons [name ...]",
	Short: "kubefs deploy addon [name ...] - create helm charts & deploy the build targets onto the cluster for listed addon",
	Long: `kubefs deploy addon [name ...] - create helm charts & deploy the build targets onto the cluster for listed addon
example:
	kubefs deploy addon <addon-name> <addon-name> --flags,
	kubefs deploy addon <addon-name> --flags,
	`,
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) < 1 {
			cmd.Help()
			return
		}

		if utils.ManifestStatus != nil {
			utils.PrintError(utils.ManifestStatus.Error())
			return
		}

		var onlyHelmify, onlyDeploy bool
		onlyHelmify, _ = cmd.Flags().GetBool("only-helmify")
		onlyDeploy, _ = cmd.Flags().GetBool("only-deploy")

		var successes []string
		var errors []string

		utils.PrintWarning(fmt.Sprintf("Deploying addons %v", args))

		for _, addon := range args {
			var addonResource *types.Addon
			addonResource, err := utils.GetAddonFromName(addon)
			if err != nil {
				utils.PrintError(err.Error())
				errors = append(errors, addon)
				continue
			}

			err = deployAddon(addonResource, onlyHelmify, onlyDeploy)
			if err != nil {
				utils.PrintError(fmt.Sprintf("Error deploying addon %s. %v", addon, err.Error()))
				errors = append(errors, addon)
				continue
			}

			successes = append(successes, addon)
		}

		if len(errors) > 0 {
			utils.PrintError(fmt.Sprintf("Error deploying resource %v", errors))
		}

		if len(successes) > 0 {
			utils.PrintSuccess(fmt.Sprintf("Resource %v deployed successfully", successes))
		}
	},
}

func init() {
	rootCmd.AddCommand(deployCmd)
	deployCmd.AddCommand(deployAllCmd)
	deployCmd.AddCommand(deployResourceCmd)
	deployCmd.AddCommand(deployAddonCmd)

	deployCmd.PersistentFlags().BoolP("only-helmify", "w", false, "only helmify the resources")
	deployCmd.PersistentFlags().BoolP("only-deploy", "d", false, "only deploy the resources")
	
	deployResourceCmd.Flags().StringP("with-addons", "a", "", "addons to be included in deployment (comma separated)")

}
