/*
Copyright © 2025 Rahul Medicharla <rmedicharla@gmail.com>
*/
package cmd

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"reflect"
	"strconv"
	"strings"

	"github.com/google/uuid"
	"github.com/rahulmedicharla/kubefs/types"
	"github.com/rahulmedicharla/kubefs/utils"
	"github.com/spf13/cobra"
)

// addonsCmd represents the addons command
var addonsCmd = &cobra.Command{
	Use:   "addons [command]",
	Short: "kubefs addons - manage addons in project",
	Long: `kubefs addons - manage addons in project
example:
	kubefs addons enable --flags,
	kubefs addons disable --flags,
`,
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Help()
	},
}

func generateKeyFiles(addonName string) error {
	commands := []string{
		fmt.Sprintf("mkdir addons/%s", addonName),
		fmt.Sprintf("openssl genrsa -out addons/%s/private_key.pem", addonName),
		fmt.Sprintf("openssl rsa -pubout -in addons/%s/private_key.pem -out addons/%s/public_key.pem", addonName, addonName),
	}

	return utils.RunMultipleCommands(commands, false, true)
}

func constructGatewayAddon(addonName string, port int, resourceNames []string, errors *[]string, successes *[]string) (*types.Addon, error) {
	var validAttachedResourceNames []string

	err := generateKeyFiles(addonName)
	if err != nil {
		return nil, err
	}

	for _, n := range resourceNames {
		err, resource := utils.GetResourceFromName(n)
		if err != nil {
			*errors = append(*errors, fmt.Sprintf("%s:%s", addonName, n))
			continue
		}

		// update resource dependents & generate client_id and secret
		client_id := uuid.New()
		secret := make([]byte, 32)
		rand.Read(secret)
		client_secret := base64.URLEncoding.EncodeToString(secret)

		resource.Dependents = append(resource.Dependents, addonName)
		if resource.Environment == nil {
			resource.Environment = make(map[string]string, 0)
		}
		resource.Environment["clientId"] = client_id.String()
		resource.Environment["clientSecret"] = client_secret

		err = utils.UpdateResource(&utils.ManifestData, n, resource)
		if err != nil {
			*errors = append(*errors, fmt.Sprintf("%s:%s", addonName, n))
			continue
		}
		validAttachedResourceNames = append(validAttachedResourceNames, n)
		*successes = append(*successes, fmt.Sprintf("%s:%s", addonName, n))
	}

	newAddon := types.Addon{
		Name:         addonName,
		Port:         port,
		DockerRepo:   "rmedicharla/gateway",
		LocalHost:    fmt.Sprintf("http://localhost:%v", port),
		DockerHost:   fmt.Sprintf("http://gateway:%v", port),
		ClusterHost:  "http://gateway-deploy.gateway.svc.cluster.local",
		Dependencies: validAttachedResourceNames,
		Environment: []string{
			"PRIVATE_KEY_PATH=/etc/ssl/private/private_key.pem",
			"PUBLIC_KEY_PATH=/etc/ssl/public/public_key.pem",
		},
	}

	return &newAddon, nil
}

func constructOauth2Addon(addonName string, port int, resourceNames []string, errors *[]string, successes *[]string) (*types.Addon, error) {
	var twoFa bool
	err := utils.ReadInput("Would you like to enable 2FA for this oauth2 addon (y/n): ", &twoFa)
	if err != nil {
		return nil, err
	}

	err = generateKeyFiles(addonName)
	if err != nil {
		return nil, err
	}

	var validAttachedResourceNames []string
	for _, n := range resourceNames {
		err, resource := utils.GetResourceFromName(n)
		if err != nil {
			*errors = append(*errors, fmt.Sprintf("%s:%s", addonName, n))
			continue
		}

		resource.Dependents = append(resource.Dependents, addonName)
		err = utils.UpdateResource(&utils.ManifestData, n, resource)
		if err != nil {
			*errors = append(*errors, fmt.Sprintf("%s:%s", addonName, n))
			continue
		}
		validAttachedResourceNames = append(validAttachedResourceNames, n)
		*successes = append(*successes, fmt.Sprintf("%s:%s", addonName, n))
	}

	newAddon := types.Addon{
		Name:         addonName,
		Port:         port,
		DockerRepo:   "rmedicharla/auth",
		LocalHost:    fmt.Sprintf("http://localhost:%v", port),
		DockerHost:   fmt.Sprintf("http://oauth2:%v", port),
		ClusterHost:  "http://oauth2-deploy.oauth2.svc.cluster.local",
		Dependencies: validAttachedResourceNames,
		Environment:  []string{"TWO_FACTOR_AUTH=" + fmt.Sprintf("%v", twoFa)},
	}

	return &newAddon, nil
}

var addonsEnableCmd = &cobra.Command{
	Use:   "enable",
	Short: "kubefs addons enable - enable addons in project",
	Long: `kubefs addons enable - enable listed addons in project
example:
	kubefs addons enable <addon-name:port>
	kubefs addons enable <addon-name:port> <addon-name:port>
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

		var errors []string
		var successes []string

		utils.PrintInfo(fmt.Sprintf("Enabling addons %v", args))

		for _, addon := range args {
			name := strings.Split(addon, ":")[0]
			addonPort := strings.Split(addon, ":")[1]

			utils.PrintInfo(fmt.Sprintf("Enabling addon [%s]", name))

			if err := utils.VerifyFramework(name, "addons"); err != nil {
				utils.PrintError(err.Error())
				errors = append(errors, name)
				continue
			}

			if err := utils.VerifyName(name); err != nil {
				utils.PrintError(err.Error())
				errors = append(errors, name)
				continue
			}

			port, err := strconv.Atoi(addonPort)
			if err != nil {
				utils.PrintError(err.Error())
				errors = append(errors, name)
				continue
			}

			if err = utils.VerifyPort(port); err != nil {
				utils.PrintError(err.Error())
				errors = append(errors, name)
				continue
			}

			var resources string
			err = utils.ReadInput(fmt.Sprintf("What resource(s) would you like the to be attached to this oauth2 adddon (comma seperated) %v: ", utils.GetCurrentResourceNames()), &resources)
			if err != nil {
				utils.PrintError(err.Error())
				errors = append(errors, name)
				continue
			}

			resourceNames := strings.Split(resources, ",")

			var newAddon *types.Addon
			resourceErrors := make([]string, 0)
			resourceSuccesses := make([]string, 0)
			if name == "oauth2" {
				newAddon, err = constructOauth2Addon(name, port, resourceNames, &resourceErrors, &resourceSuccesses)
				if err != nil {
					utils.PrintError(err.Error())
					errors = append(errors, name)
					continue
				}

			} else if name == "gateway" {
				newAddon, err = constructGatewayAddon(name, port, resourceNames, &resourceErrors, &resourceSuccesses)
				if err != nil {
					utils.PrintError(err.Error())
					errors = append(errors, name)
					continue
				}
			}

			if len(resourceErrors) > 0 {
				errors = append(errors, resourceErrors...)
			}
			if len(resourceSuccesses) > 0 {
				successes = append(successes, resourceSuccesses...)
			}

			utils.ManifestData.Addons = append(utils.ManifestData.Addons, *newAddon)
		}

		utils.WriteManifest(&utils.ManifestData, "manifest.yaml")

		if len(errors) > 0 {
			utils.PrintError(fmt.Sprintf("Error enabling addons %v", errors))
		}

		if len(successes) > 0 {
			utils.PrintInfo(fmt.Sprintf("Addon %v enabled successfully", successes))
		}
	},
}

var addonsDisableCmd = &cobra.Command{
	Use:   "disable",
	Short: "kubefs addons disable - disable addons in project",
	Long: `kubefs addons disable - disable listed addons in project
example:
	kubefs addons disable <addon-name>
	kubefs addons disable <addon-name> <addon-name>
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

		var errors []string
		var successes []string

		utils.PrintInfo(fmt.Sprintf("Disabling addons %v", args))

		for _, name := range args {

			err, addon := utils.GetAddonFromName(name)
			if err != nil {
				utils.PrintError(err.Error())
				errors = append(errors, name)
				continue
			}

			err = utils.RunCommand(fmt.Sprintf("rm -rf addons/%s", name), false, true)
			if err != nil {
				utils.PrintError(err.Error())
				errors = append(errors, name)
				continue
			}

			err = utils.RemoveAddon(&utils.ManifestData, name)
			if err != nil {
				utils.PrintError(err.Error())
				errors = append(errors, name)
				continue
			}

			for _, dependent := range addon.Dependencies {
				err, resource := utils.GetResourceFromName(dependent)
				if err != nil {
					utils.PrintError(err.Error())
					errors = append(errors, name)
					continue
				}

				var newDependents []string
				for _, dep := range resource.Dependents {
					if dep != name {
						newDependents = append(newDependents, dep)
					}
				}

				// remove clientID and clientSecret if gateway
				if addon.Name == "gateway" {
					delete(resource.Opts, "clientId")
					delete(resource.Opts, "clientSecret")
				}

				resource.Dependents = newDependents
				err = utils.UpdateResource(&utils.ManifestData, dependent, resource)
				if err != nil {
					utils.PrintError(err.Error())
					errors = append(errors, name)
					continue
				}
			}

			successes = append(successes, name)
		}

		if len(errors) > 0 {
			utils.PrintError(fmt.Sprintf("Error disabled addons %v", errors))
		}

		if len(successes) > 0 {
			utils.PrintInfo(fmt.Sprintf("Addon %v disabled successfully", successes))
		}
	},
}

var addonsListCmd = &cobra.Command{
	Use:   "list",
	Short: "kubefs addons list - list addons in project",
	Long: `kubefs addons list - list addons in project
example:
	kubefs addons list
`,
	Run: func(cmd *cobra.Command, args []string) {
		if utils.ManifestStatus != nil {
			utils.PrintError(utils.ManifestStatus.Error())
		}

		utils.PrintInfo("Listing addons")

		for _, addon := range utils.ManifestData.Addons {
			addonValue := reflect.ValueOf(addon)
			addonType := reflect.TypeOf(addon)
			for i := 0; i < addonValue.NumField(); i++ {
				field := addonType.Field(i)
				value := addonValue.Field(i)
				fmt.Printf("%s: %v\n", field.Name, value)
			}
			fmt.Println()
		}
	},
}

func init() {
	rootCmd.AddCommand(addonsCmd)
	addonsCmd.AddCommand(addonsEnableCmd)
	addonsCmd.AddCommand(addonsDisableCmd)
	addonsCmd.AddCommand(addonsListCmd)

}
