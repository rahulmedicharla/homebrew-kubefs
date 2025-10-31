/*
Copyright © 2025 Rahul Medicharla <rmedicharla@gmail.com>
*/
package cmd

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/rahulmedicharla/kubefs/types"
	"github.com/rahulmedicharla/kubefs/utils"
	"github.com/spf13/cobra"
)

// testCmd represents the test command
var testCmd = &cobra.Command{
	Use:   "test [command]",
	Short: "kubefs test - test your build environment in docker locally before deploying",
	Long: `kubefs test - test your build environment in docker locally before deploying
example:
	kubefs test all --flags,
	kubefs test resource <frontend> <api> <database> --flags,
	kubefs test addons <addons> --flags`,
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Help()
	},
}

var rawCompose = map[string]interface{}{
	"services": map[string]interface{}{},
	"networks": map[string]interface{}{
		"shared_network": map[string]string{
			"driver": "bridge",
		},
	},
	"volumes": map[string]interface{}{},
}

func testAddon(rawCompose *map[string]interface{}, addon *types.Addon) error {
	err := utils.RunCommand(fmt.Sprintf("docker pull %s", addon.DockerRepo), false, true)
	if err != nil {
		return err
	}

	service := map[string]interface{}{
		"image": addon.DockerRepo,
		"networks": []string{
			"shared_network",
		},
		"environment": []string{},
	}

	service["ports"] = []string{
		fmt.Sprintf("%v:%v", addon.Port, addon.Port),
	}

	var allowedHosts []string
	for _, name := range addon.Dependencies {
		resource, err := utils.GetResourceFromName(name)
		if err != nil {
			return err
		}
		allowedHosts = append(allowedHosts, resource.DockerHost)
	}

	env := service["environment"].([]string)
	env = append(env, addon.Environment...)

	switch addon.Name {
	case "oauth2":
		service["volumes"] = []string{
			"./addons/oauth2/private_key.pem:/etc/ssl/private/private_key.pem",
			"./addons/oauth2/public_key.pem:/etc/ssl/public/public_key.pem",
			"oauth2Store:/app/store",
		}

		env = append(env,
			fmt.Sprintf("ALLOWED_ORIGINS=%s", strings.Join(allowedHosts, ",")),
			fmt.Sprintf("PORT=%v", addon.Port),
			fmt.Sprintf("NAME=%s", utils.ManifestData.KubefsName),
		)

		(*rawCompose)["volumes"].(map[string]interface{})["oauth2Store"] = map[string]string{
			"driver": "local",
		}
	case "gateway":
		service["volumes"] = []string{
			"./addons/gateway/private_key.pem:/etc/ssl/private/private_key.pem",
			"./addons/gateway/public_key.pem:/etc/ssl/public/public_key.pem",
		}

		clients := make([]string, 0)
		for _, name := range addon.Dependencies {
			resource, err := utils.GetResourceFromName(name)
			if err != nil {
				return err
			}

			secret, err := base64.URLEncoding.DecodeString(resource.Environment["clientSecret"])
			if err != nil {
				return err
			}

			hash := sha256.Sum256(secret)
			clients = append(clients, fmt.Sprintf("%s:%s", resource.Environment["clientId"], hex.EncodeToString(hash[:])))
		}

		env = append(env,
			fmt.Sprintf("CLIENTS=%s", strings.Join(clients, ",")),
			fmt.Sprintf("ALLOWED_ORIGINS=%s", strings.Join(allowedHosts, ",")),
			fmt.Sprintf("PORT=%v", addon.Port),
			"DEBUG=1",
		)
	}
	service["environment"] = env

	(*rawCompose)["services"].(map[string]interface{})[addon.Name] = service

	return nil
}

func testResource(rawCompose *map[string]interface{}, name string, resource *types.Resource) error {
	err := utils.RunCommand(fmt.Sprintf("docker pull %s", resource.DockerRepo), true, true)
	if err != nil {
		return err
	}

	service := map[string]interface{}{
		"image": resource.DockerRepo,
		"ports": []string{fmt.Sprintf("%v:%v", resource.Port, resource.Port)},
		"networks": []string{
			"shared_network",
		},
		"environment": []string{},
		"volumes":     []string{},
	}

	if resource.Type != "database" {
		for rName, r := range utils.ManifestData.Resources {
			if r.Type == "database" {
				service["environment"] = append(service["environment"].([]string), fmt.Sprintf("%sHOST_READ=%s", rName, r.DockerHost))
			}
			service["environment"] = append(service["environment"].([]string), fmt.Sprintf("%sHOST=%s", rName, r.DockerHost))
		}

		for _, a := range resource.Dependents {
			addon, _ := utils.GetAddonFromName(a)
			service["environment"] = append(service["environment"].([]string), fmt.Sprintf("%sHOST=%s", a, addon.DockerHost))
		}

		envData, err := utils.ReadEnv(fmt.Sprintf("%s/.env", name))
		if err == nil {
			for _, line := range envData {
				service["environment"] = append(service["environment"].([]string), line)
			}
		}

		for key, env := range resource.Environment {
			service["environment"] = append(service["environment"].([]string), fmt.Sprintf("%s=%s", key, env))
		}

	} else {
		if resource.Framework == "redis" {
			service["environment"] = []string{fmt.Sprintf("REDIS_PASSWORD=%s", resource.Opts["password"]), fmt.Sprintf("REDIS_PORT_NUMBER=%v", resource.Port), fmt.Sprintf("REDIS_DATABASE=%s", resource.Opts["default-database"])}
			service["volumes"] = []string{"redis_data:/bitnami/redis/data"}
			(*rawCompose)["volumes"].(map[string]interface{})["redis_data"] = map[string]string{
				"driver": "local",
			}
		} else {
			service["environment"] = []string{fmt.Sprintf("POSTGRESQL_PASSWORD=%s", resource.Opts["password"]), fmt.Sprintf("POSTGRESQL_PORT_NUMBER=%v", resource.Port), fmt.Sprintf("POSTGRESQL_DATABASE=%s", resource.Opts["default-database"]), fmt.Sprintf("POSTGRESQL_USERNAME=%s", resource.Opts["user"])}
			service["volumes"] = []string{"postgresql_data:/bitnami/postgresql"}
			(*rawCompose)["volumes"].(map[string]interface{})["postgresql_data"] = map[string]string{
				"driver": "local",
			}
		}
	}

	(*rawCompose)["services"].(map[string]interface{})[name] = service
	return nil
}

var testAllCmd = &cobra.Command{
	Use:   "all",
	Short: "kubefs test all - test your entire build environment in docker locally before deploying",
	Long: `kubefs test all - test your entire build environment in docker locally before deploying
example:
	kubefs test all --flags
	`,
	Run: func(cmd *cobra.Command, args []string) {

		utils.PrintWarning("Testing all resources in docker")

		var errors []string
		var successes []string

		for name, resource := range utils.ManifestData.Resources {
			err := testResource(&rawCompose, name, &resource)
			if err != nil {
				errors = append(errors, name)
				continue
			}
			successes = append(successes, name)
		}

		for _, addon := range utils.ManifestData.Addons {
			err := testAddon(&rawCompose, &addon)
			if err != nil {
				errors = append(errors, addon.Name)
				continue
			}
			successes = append(successes, addon.Name)
		}

		err := utils.WriteYaml(&rawCompose, "docker-compose.yaml")
		if err != nil {
			utils.PrintError(fmt.Errorf("error writing docker-compose.yaml file. %v", err))
			return
		}

		utils.PrintWarning("Wrote docker-compose.yaml file")

		if len(errors) > 0 {
			utils.PrintError(fmt.Errorf("error including resources & addons %v", errors))
		}

		if len(successes) > 0 {
			utils.PrintInfo(fmt.Sprintf("Successfully included resources & addons %v", successes))
		}

		var onlyWrite bool
		var persist bool
		onlyWrite, _ = cmd.Flags().GetBool("only-write")
		persist, _ = cmd.Flags().GetBool("persist-data")

		if !onlyWrite {
			err := utils.RunCommand("docker compose up --remove-orphans", true, true)
			if err != nil {
				utils.PrintError(fmt.Errorf("eror running docker compose: %v", err))
				return
			}

			var command string
			if persist {
				command = "docker compose down"
			} else {
				command = "docker compose down -v --rmi all"
			}

			err = utils.RunCommand(command, true, true)
			if err != nil {
				utils.PrintError(fmt.Errorf("error stopping docker compose: %v", err))
				return
			}
		}
	},
}

var testResourceCmd = &cobra.Command{
	Use:   "resource [name ...]",
	Short: "kubefs test resource [name ...] - test listed resources & addons in docker locally before deploying",
	Long: `kubefs test resource [name ...] - test listed resource & addons in docker locally before deploying
example:
	kubefs test resource <frontend> <api> <database> --flags,
	kubefs test resource <frontend> --flags`,
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) < 1 {
			cmd.Help()
			return
		}

		addonNames, _ := cmd.Flags().GetString("with-addons")
		var addonsList []string
		if addonNames != "" {
			addonsList = strings.Split(addonNames, ",")
		}

		var errors []string
		var successes []string

		utils.PrintWarning(fmt.Sprintf("Testing resources %v in docker", args))

		for _, name := range args {
			resource, err := utils.GetResourceFromName(name)
			if err != nil {
				errors = append(errors, name)
				continue
			}

			err = testResource(&rawCompose, name, resource)
			if err != nil {
				errors = append(errors, name)
				continue
			}
			successes = append(successes, name)
		}

		for _, name := range addonsList {
			addon, err := utils.GetAddonFromName(name)
			if err != nil {
				errors = append(errors, name)
				continue
			}

			err = testAddon(&rawCompose, addon)
			if err != nil {
				errors = append(errors, name)
				continue
			}
			successes = append(successes, name)
		}

		err := utils.WriteYaml(&rawCompose, "docker-compose.yaml")
		if err != nil {
			utils.PrintError(fmt.Errorf("error writing docker-compose.yaml file. %v", err))
			return
		}

		utils.PrintWarning("Wrote docker-compose.yaml file")

		if len(errors) > 0 {
			utils.PrintError(fmt.Errorf("error including resources & addons %v", errors))
		}

		if len(successes) > 0 {
			utils.PrintInfo(fmt.Sprintf("Successfully included resources & addons %v", successes))
		}

		var onlyWrite bool
		var persist bool
		onlyWrite, _ = cmd.Flags().GetBool("only-write")
		persist, _ = cmd.Flags().GetBool("persist-data")

		if !onlyWrite {
			err := utils.RunCommand("docker compose up --remove-orphans", true, true)
			if err != nil {
				utils.PrintError(fmt.Errorf("error running docker compose: %v", err))
				return
			}

			var command string
			if persist {
				command = "docker compose down"
			} else {
				command = "docker compose down -v --rmi all"
			}

			err = utils.RunCommand(command, true, true)
			if err != nil {
				utils.PrintError(fmt.Errorf("error stopping docker compose: %v", err))
				return
			}
		}
	},
}

var testAddonCmd = &cobra.Command{
	Use:   "addons [name ...]",
	Short: "kubefs test addons [name ...] - test listed addons in docker locally before deploying",
	Long: `kubefs test addons [name ...] - test listed addons in docker locally before deploying
example:
	kubefs test addons <addon_name> <addon_name> --flags,
	kubefs test addons <addon_name> --flags`,
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) < 1 {
			cmd.Help()
			return
		}

		var errors []string
		var successes []string

		utils.PrintWarning(fmt.Sprintf("Testing resources %v in docker", args))

		for _, name := range args {
			addon, err := utils.GetAddonFromName(name)
			if err != nil {
				errors = append(errors, name)
				continue
			}

			err = testAddon(&rawCompose, addon)
			if err != nil {
				errors = append(errors, name)
				continue
			}
			successes = append(successes, name)
		}

		err := utils.WriteYaml(&rawCompose, "docker-compose.yaml")
		if err != nil {
			utils.PrintError(fmt.Errorf("error writing docker-compose.yaml file. %v", err))
			return
		}

		utils.PrintWarning("Wrote docker-compose.yaml file")

		if len(errors) > 0 {
			utils.PrintError(fmt.Errorf("error including resources & addons %v", errors))
		}

		if len(successes) > 0 {
			utils.PrintInfo(fmt.Sprintf("Successfully included resources & addons %v", successes))
		}

		var onlyWrite bool
		var persist bool
		onlyWrite, _ = cmd.Flags().GetBool("only-write")
		persist, _ = cmd.Flags().GetBool("persist-data")

		if !onlyWrite {
			err := utils.RunCommand("docker compose up --remove-orphans", true, true)
			if err != nil {
				utils.PrintError(fmt.Errorf("error running docker compose: %v", err))
				return
			}

			var command string
			if persist {
				command = "docker compose down"
			} else {
				command = "docker compose down -v --rmi all"
			}

			err = utils.RunCommand(command, true, true)
			if err != nil {
				utils.PrintError(fmt.Errorf("error stopping docker compose: %v", err))
				return
			}
		}
	},
}

func init() {
	rootCmd.AddCommand(testCmd)
	testCmd.AddCommand(testAllCmd)
	testCmd.AddCommand(testResourceCmd)
	testCmd.AddCommand(testAddonCmd)

	testCmd.PersistentFlags().BoolP("only-write", "w", false, "only create the docker compose file; dont start it")
	testCmd.PersistentFlags().BoolP("persist-data", "p", false, "persist images & volume data after testing")
	testResourceCmd.Flags().StringP("with-addons", "a", "", "include addons in the test")
}
