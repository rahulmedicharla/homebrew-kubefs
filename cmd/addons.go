/*
Copyright © 2025 Rahul Medicharla <rmedicharla@gmail.com>

*/
package cmd

import (
	"fmt"
	"github.com/spf13/cobra"
	"github.com/rahulmedicharla/kubefs/types"
	"github.com/rahulmedicharla/kubefs/utils"
	"strings"
	"strconv"
	"os/exec"
	"os"
	"reflect"
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

var addonsEnableCmd = &cobra.Command{
	Use:   "enable",
	Short: "kubefs addons enable - enable addons in project",
	Long: `kubefs addons enable - enable listed (comma seperated) addons in project
example:
	kubefs addons enable -a <addon-name:port>
	kubefs addons enable -a <addon-name:port>,<addon-name:port>
`,
	Run: func(cmd *cobra.Command, args []string) {
		if utils.ManifestStatus == types.ERROR {
			utils.PrintError("Not a valid kubefs project: use 'kubefs init' to create a new project")
			return
		}

		addons, _ := cmd.Flags().GetString("addon")
		addonList := strings.Split(addons, ",")

		var errors []string
		var successes []string

		utils.PrintWarning(fmt.Sprintf("Enabling addons %v", addonList))

		for _, addon := range addonList {
			name := strings.Split(addon, ":")[0]
			addonPort := strings.Split(addon, ":")[1]

			if !utils.VerifyFramework(name, "addons") {
				errors = append(errors, name)
				continue
			}

			if !utils.VerifyName(name) {
				errors = append(errors, name)
				continue
			}

			port, err := strconv.Atoi(addonPort)
			if err != nil || !utils.VerifyPort(port) {
				errors = append(errors, name)
				continue
			}

			var newAddon types.Addon
			if name == "oauth2" {
				var input string
				fmt.Print(fmt.Sprintf("What resource would you like to be redirected to after authentication?%v: ", utils.GetCurrentResourceNames()))
				fmt.Scanln(&input)
				
				redirectResource := utils.GetResourceFromName(input)
				if redirectResource == nil {
					utils.PrintError(fmt.Sprintf("Resource %s not found", input))
					errors = append(errors, name)
					continue
				}
				
				fmt.Print(fmt.Sprintf("What path would you like to be redirected to on %s after authentication? (ex. /): ", redirectResource.Name))
				fmt.Scanln(&input)
				redirectPath := input

				fmt.Print(fmt.Sprintf("What resource would you like to to have confirm the auth tokens?%v: ", utils.GetCurrentResourceNames()))
				fmt.Scanln(&input)
				
				confirmResource := utils.GetResourceFromName(input)
				if confirmResource == nil {
					utils.PrintError(fmt.Sprintf("Resource %s not found", input))
					errors = append(errors, name)
					continue
				}

				commands := []string{
					fmt.Sprintf("mkdir addons/%s", name),
					fmt.Sprintf("openssl genrsa -out addons/%s/private_key.pem -aes256 -passout pass:kubefs", name),
					fmt.Sprintf("openssl rsa -passin pass:kubefs -pubout -in addons/%s/private_key.pem -out addons/%s/public_key.pem", name, name),
				}

				var isErr bool
				isErr = false
				for _, command := range commands {
					cmd := exec.Command("sh", "-c", command)
					cmd.Stderr = os.Stderr
					err := cmd.Run()
					fmt.Println(err)
					if err != nil {
						utils.PrintError(fmt.Sprintf("Error enabling addon %s", name))
						errors = append(errors, name)
						isErr = true
						break
					}
				}
				if isErr {
					continue
				}

				newAddon = types.Addon{
					Name: name,
					Port: port,
					DockerRepo: "rmedicharla/auth",
					LocalHost: fmt.Sprintf("http://localhost:%s", addonPort),
					DockerHost: fmt.Sprintf("http://oauth2:%s", addonPort),
					ClusterHost: fmt.Sprintf("http://oauth2-deploy.oauth2.svc.cluster.local:%s", addonPort),
					Env: []string{
						fmt.Sprintf("PORT=%s", addonPort), 
						fmt.Sprintf("REDIRECT_RESOURCE=%s", redirectResource.Name),
						fmt.Sprintf("REDIRECT_PATH=%s", redirectPath),
						fmt.Sprintf("CONFIRM_RESOURCE=%s", confirmResource.Name),
					},
				}

				utils.ManifestData.Addons = append(utils.ManifestData.Addons, newAddon)
				successes = append(successes, name)
			}
		}

		utils.WriteManifest(&utils.ManifestData)

		if len(errors) > 0 {
			utils.PrintError(fmt.Sprintf("Error enabling addons %v", errors))
		}

		if len(successes) > 0 {
			utils.PrintSuccess(fmt.Sprintf("Addon %v enabled successfully", successes))
		}
	},
}

var addonsDisableCmd = &cobra.Command{
	Use:   "disable",
	Short: "kubefs addons disable - disable addons in project",
	Long: `kubefs addons disable - disable listed (comma seperated) addons in project
example:
	kubefs addons disable -a <addon-name>
	kubefs addons disable -a <addon-name>,<addon-name>
`,
	Run: func(cmd *cobra.Command, args []string) {
		if utils.ManifestStatus == types.ERROR {
			utils.PrintError("Not a valid kubefs project: use 'kubefs init' to create a new project")
			return
		}

		addons, _ := cmd.Flags().GetString("addon")
		addonList := strings.Split(addons, ",")

		var errors []string
		var successes []string

		utils.PrintWarning(fmt.Sprintf("Disabling addons %v", addonList))

		for _, name := range addonList {

			addon := utils.GetAddonFromName(name)
			if addon == nil {
				errors = append(errors, name)
				continue
			}

			cmd := exec.Command("sh", "-c", fmt.Sprintf("rm -rf addons/%s", name))
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
			err := cmd.Run()
			if err != nil {
				utils.PrintError(fmt.Sprintf("Error disabling addon %s", name))
				errors = append(errors, name)
				continue
			}

			utils.RemoveAddon(&utils.ManifestData, name)
			successes = append(successes, name)
		}

		if len(errors) > 0 {
			utils.PrintError(fmt.Sprintf("Error enabling addons %v", errors))
		}

		if len(successes) > 0 {
			utils.PrintSuccess(fmt.Sprintf("Addon %v enabled successfully", successes))
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
		if utils.ManifestStatus == types.ERROR {
			utils.PrintError("Not a valid kubefs project: use 'kubefs init' to create a new project")
			return
		}

		utils.PrintWarning("Listing addons")

		for _, addon := range utils.ManifestData.Addons {
			addonValue := reflect.ValueOf(addon)
			addonType := reflect.TypeOf(addon)
			for i := 0; i < addonValue.NumField(); i++ {
				field := addonType.Field(i)
				value := addonValue.Field(i)
				fmt.Printf("%s: %v\n", field.Name, value)
			}
			fmt.Println("\n")
		}
	},
}


func init() {
	rootCmd.AddCommand(addonsCmd)
	addonsCmd.AddCommand(addonsEnableCmd)
	addonsCmd.AddCommand(addonsDisableCmd)
	addonsCmd.AddCommand(addonsListCmd)

	addonsEnableCmd.Flags().StringP("addon", "a", "", "addon name and port. Format <addon-name:port>. Supported addons: [oauth2]")
	addonsDisableCmd.Flags().StringP("addon", "a", "", "addon name. Supported addons: [oauth2]")

	addonsEnableCmd.MarkFlagRequired("addon")
	addonsDisableCmd.MarkFlagRequired("addon")
}
