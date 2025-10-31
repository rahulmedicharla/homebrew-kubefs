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
	"v8.run/go/exp/util/maps"
)

// addonsCmd represents the addons command
var addonsCmd = &cobra.Command{
	Use:   "addons [command]",
	Short: "kubefs addons - manage addons in project",
	Long: `kubefs addons - manage addons in project
example:
	kubefs addons enable --flags,
	kubefs addons disable --flags,
	kubefs addons manage --flags
	kubefs addons list
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

func connectGatewayToResource(addonName string, resourceNames maps.Set[string], errors *[]string, successes *[]string) maps.Set[string] {
	validAttachedResourceNames := maps.NewSet[string]()

	for _, n := range maps.Keys(resourceNames) {
		resource, err := utils.GetResourceFromName(n)
		if err != nil {
			*errors = append(*errors, fmt.Sprintf("%s:%s", addonName, n))
			fmt.Println(*errors)
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
		validAttachedResourceNames.Insert(n)
		*successes = append(*successes, fmt.Sprintf("%s:%s", addonName, n))
	}
	return validAttachedResourceNames
}

func constructGatewayAddon(addonName string, port int, resourceNames maps.Set[string], errors *[]string, successes *[]string) (*types.Addon, error) {

	err := generateKeyFiles(addonName)
	if err != nil {
		return nil, err
	}

	validAttachedResourceNames := connectGatewayToResource(addonName, resourceNames, errors, successes)

	newAddon := types.Addon{
		Port:         port,
		DockerRepo:   "rmedicharla/gateway",
		LocalHost:    fmt.Sprintf("http://localhost:%v", port),
		DockerHost:   fmt.Sprintf("http://gateway:%v", port),
		ClusterHost:  "http://gateway-deploy.gateway.svc.cluster.local",
		Dependencies: maps.Keys(validAttachedResourceNames),
		Environment: []string{
			"PRIVATE_KEY_PATH=/etc/ssl/private/private_key.pem",
			"PUBLIC_KEY_PATH=/etc/ssl/public/public_key.pem",
		},
	}

	return &newAddon, nil
}

func connectOauth2ToResource(addonName string, resourceNames maps.Set[string], errors *[]string, successes *[]string) maps.Set[string] {
	validAttachedResourceNames := maps.NewSet[string]()

	for _, n := range maps.Keys(resourceNames) {
		resource, err := utils.GetResourceFromName(n)
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
		validAttachedResourceNames.Insert(n)
		*successes = append(*successes, fmt.Sprintf("%s:%s", addonName, n))
	}

	return validAttachedResourceNames
}

func constructOauth2Addon(addonName string, port int, resourceNames maps.Set[string], errors *[]string, successes *[]string) (*types.Addon, error) {
	var twoFa bool
	err := utils.ReadInput("Would you like to enable 2FA for this oauth2 addon (y/n): ", &twoFa)
	if err != nil {
		return nil, err
	}

	err = generateKeyFiles(addonName)
	if err != nil {
		return nil, err
	}

	validAttachedResourceNames := connectOauth2ToResource(addonName, resourceNames, errors, successes)

	newAddon := types.Addon{
		Port:         port,
		DockerRepo:   "rmedicharla/auth",
		LocalHost:    fmt.Sprintf("http://localhost:%v", port),
		DockerHost:   fmt.Sprintf("http://oauth2:%v", port),
		ClusterHost:  "http://oauth2-deploy.oauth2.svc.cluster.local",
		Dependencies: maps.Keys(validAttachedResourceNames),
		Environment:  []string{"TWO_FACTOR_AUTH=" + fmt.Sprintf("%v", twoFa)},
	}

	return &newAddon, nil
}

func disconnectAddonFromResource(addonName string, dependencies maps.Set[string], errors *[]string, successes *[]string) maps.Set[string] {
	validDisconnectResourceNames := maps.NewSet[string]()

	for _, n := range maps.Keys(dependencies) {
		resource, err := utils.GetResourceFromName(n)
		if err != nil {
			utils.PrintError(err)
			*errors = append(*errors, fmt.Sprintf("%s:%s", addonName, n))
			continue
		}

		var newDependents []string
		for _, dep := range resource.Dependents {
			if dep != addonName {
				newDependents = append(newDependents, dep)
			}
		}

		// remove clientID and clientSecret if gateway
		if addonName == "gateway" {
			delete(resource.Environment, "clientId")
			delete(resource.Environment, "clientSecret")
		}

		resource.Dependents = newDependents
		err = utils.UpdateResource(&utils.ManifestData, n, resource)
		if err != nil {
			utils.PrintError(err)
			*errors = append(*errors, fmt.Sprintf("%s:%s", addonName, n))
			continue
		}

		validDisconnectResourceNames.Insert(n)
		*successes = append(*successes, fmt.Sprintf("%s:%s", addonName, n))
	}

	return validDisconnectResourceNames
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

		var errors []string
		var successes []string

		utils.PrintInfo(fmt.Sprintf("Enabling addons %v", args))

		for _, addon := range args {
			splitAddon := strings.Split(addon, ":")
			if len(splitAddon) != 2 {
				utils.PrintError(fmt.Errorf("addon %s configured incorrectly. should be <addon-name>:<port>", addon))
				return
			}

			name := splitAddon[0]
			addonPort := splitAddon[1]

			utils.PrintInfo(fmt.Sprintf("Enabling addon [%s]", name))

			if err := utils.VerifyFramework(name, "addons"); err != nil {
				utils.PrintError(err)
				errors = append(errors, name)
				continue
			}

			if err := utils.VerifyName(name); err != nil {
				utils.PrintError(err)
				errors = append(errors, name)
				continue
			}

			port, err := strconv.Atoi(addonPort)
			if err != nil {
				utils.PrintError(err)
				errors = append(errors, name)
				continue
			}

			if err = utils.VerifyPort(port); err != nil {
				utils.PrintError(err)
				errors = append(errors, name)
				continue
			}

			var resources string
			err = utils.ReadInput(fmt.Sprintf("What resource(s) would you like the to be attached to this oauth2 adddon (comma seperated) %v: ", utils.GetCurrentResourceNames()), &resources)
			if err != nil {
				utils.PrintError(err)
				errors = append(errors, name)
				continue
			}

			resourceNames := maps.NewSet(strings.Split(resources, ",")...)

			var newAddon *types.Addon
			resourceErrors := make([]string, 0)
			resourceSuccesses := make([]string, 0)

			switch name {
			case "oauth2":
				newAddon, err = constructOauth2Addon(name, port, resourceNames, &resourceErrors, &resourceSuccesses)
				if err != nil {
					utils.PrintError(err)
					errors = append(errors, name)
					continue
				}
			case "gateway":
				newAddon, err = constructGatewayAddon(name, port, resourceNames, &resourceErrors, &resourceSuccesses)
				if err != nil {
					utils.PrintError(err)
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

			utils.ManifestData.Addons[name] = *newAddon
		}

		err := utils.WriteManifest(&utils.ManifestData, "manifest.yaml")
		if err != nil {
			utils.PrintError(err)
			return
		}

		if len(errors) > 0 {
			utils.PrintError(fmt.Errorf("error enabling addons %v", errors))
		}

		if len(successes) > 0 {
			utils.PrintInfo(fmt.Sprintf("addon %v enabled successfully", successes))
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

		var errors []string
		var successes []string

		utils.PrintInfo(fmt.Sprintf("Disabling addons %v", args))

		for _, name := range args {

			addon, err := utils.GetAddonFromName(name)
			if err != nil {
				utils.PrintError(err)
				errors = append(errors, name)
				continue
			}

			resourceErrors := make([]string, 0)
			resourceSuccesses := make([]string, 0)

			// make set so no duplicates
			disableList := maps.NewSet(addon.Dependencies...)

			disconnectAddonFromResource(name, disableList, &resourceErrors, &resourceSuccesses)

			err = utils.RunCommand(fmt.Sprintf("rm -rf addons/%s", name), false, true)
			if err != nil {
				utils.PrintError(err)
				errors = append(errors, name)
				continue
			}

			err = utils.RemoveAddon(&utils.ManifestData, name)
			if err != nil {
				utils.PrintError(err)
				errors = append(errors, name)
				continue
			}

			if len(resourceErrors) > 0 {
				errors = append(errors, resourceErrors...)
			}
			if len(resourceSuccesses) > 0 {
				successes = append(successes, resourceSuccesses...)
			}
		}

		if len(errors) > 0 {
			utils.PrintError(fmt.Errorf("error disabled addons %v", errors))
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

var addonsManageCmd = &cobra.Command{
	Use:   "manage",
	Short: "kubefs addons manage - manage attached resources to addons in project",
	Long: `kubefs addons manage - manage attached resource to addons in project
example:
	kubefs addons manage <addonName> --add <resourceName>
	kubefs addons manage <addonName> --remove <resourceName>
`,
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) < 1 {
			cmd.Help()
			return
		}

		name := args[0]
		addon, err := utils.GetAddonFromName(name)
		if err != nil {
			utils.PrintError(err)
			return
		}

		errors := make([]string, 0)
		successes := make([]string, 0)

		var addResources, removeReources string
		addResources, _ = cmd.Flags().GetString("add")
		removeReources, _ = cmd.Flags().GetString("remove")

		addList := maps.NewSet[string]()
		removeList := maps.NewSet[string]()
		if len(addResources) > 0 {
			addList = maps.NewSet(strings.Split(addResources, ",")...)
		}
		if len(removeReources) > 0 {
			removeList = maps.NewSet(strings.Split(removeReources, ",")...)
		}

		// remove any duplicates and get current addon dependencies
		currentList := maps.NewSet(addon.Dependencies...)

		// remove any dependencies that are being added & already in use
		// remove any dependencies that are being removed & not already in use
		errors = append(errors, maps.Keys(addList.Intersection(currentList))...)
		errors = append(errors, maps.Keys(removeList.Subtract(currentList))...)
		addList = addList.Subtract(currentList)
		removeList = removeList.Intersection(currentList)

		var addResourceNames maps.Set[string]
		var removeResourceNames maps.Set[string]

		switch name {
		case "oauth2":
			addResourceNames = connectOauth2ToResource(name, addList, &errors, &successes)
		case "gateway":
			addResourceNames = connectGatewayToResource(name, addList, &errors, &successes)
		}
		removeResourceNames = disconnectAddonFromResource(name, removeList, &errors, &successes)

		//update manifest
		newDependencyList := (currentList.Union(addResourceNames)).Subtract(removeResourceNames)
		addon.Dependencies = maps.Keys(newDependencyList)

		err = utils.UpdateAddons(&utils.ManifestData, name, addon)
		if err != nil {
			utils.PrintError(err)
			return
		}

		if len(errors) > 0 {
			utils.PrintError(fmt.Errorf("error managing addons %v", errors))
		}

		if len(successes) > 0 {
			utils.PrintInfo(fmt.Sprintf("managed addons %v", successes))
		}

	},
}

func init() {
	rootCmd.AddCommand(addonsCmd)
	addonsCmd.AddCommand(addonsEnableCmd)
	addonsCmd.AddCommand(addonsDisableCmd)
	addonsCmd.AddCommand(addonsListCmd)
	addonsCmd.AddCommand(addonsManageCmd)

	addonsManageCmd.Flags().StringP("add", "a", "", "resources to add to specified addon (comma seperated)")
	addonsManageCmd.Flags().StringP("remove", "r", "", "resources to remove from specified addon (comma seperated)")
}
