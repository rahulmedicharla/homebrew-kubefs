/*
Copyright © 2025 Rahul Medicharla <rmedicharla@gmail.com>

*/
package cmd

import (
	"fmt"
	"strings"
	"github.com/spf13/cobra"
	"github.com/zalando/go-keyring"
	"github.com/rahulmedicharla/kubefs/utils"
)

// configCmd represents the config command
var configCmd = &cobra.Command{
	Use:   "config [command]",
	Short: "kubefs config - configure kubefs environment and auth configurations",
	Long: `kubefs config - configure kubefs environment and auth configurations
example: 
	kubefs config docker --flag
	`,
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Help()
	},
}

var dockerCmd = &cobra.Command{
	Use:   "docker",
	Short: "Configure Docker settings",
	Long:  `Configure Docker settings for kubefs
example: 
	kubefs config docker --flags
	`,
	Run: func(cmd *cobra.Command, args []string) {
		// get service information
		service := "docker"
		user := "kubefs"

		// Read remove flag
		remove, err := cmd.Flags().GetBool("remove")
		if err != nil {
			utils.PrintError(fmt.Sprintf("Error reading remove flag: %v", err.Error()))
			return
		}

		if remove {
			err := keyring.Delete(service, user)
			if err != nil {
				utils.PrintError(fmt.Sprintf("Error deleting Docker credentials: %v", err.Error()))
				return
			}
			utils.PrintSuccess("Docker credentials removed successfully")
		} else {
			username, err := utils.ReadInput("Enter Docker username: ")
			if err != nil {
				utils.PrintError(fmt.Sprintf("Error reading Docker username: %v", err.Error()))
			}

			pat, err := utils.ReadInput("Enter Docker PAT (https://docs.docker.com/security/for-developers/access-tokens/): ")
			if err != nil {
				utils.PrintError(fmt.Sprintf("Error reading Docker PAT: %v", err.Error()))
			}

			err = keyring.Set(service, user, fmt.Sprintf("%s:%s", username, pat))
			if err != nil {
				utils.PrintError(fmt.Sprintf("Error saving Docker credentials: %v", err))
				return
			}

			utils.PrintSuccess("Saving Docker credentials")
		}
	},
}

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List all configurations",
	Long:  `List all configurations for kubefs
example: 
	kubefs config list
	`,
	Run: func(cmd *cobra.Command, args []string) {
		user := "kubefs"
		services := []string{"docker"}

		fmt.Println("Listing all configurations \n")

		for _, service := range services {
			creds, err := keyring.Get(service, user)
			if err != nil {
				fmt.Printf("Error getting %s credentials | No credentials set: %v\n", service, err)
			} else {
				username, password := strings.Split(creds, ":")[0], strings.Split(creds, ":")[1]
				fmt.Printf("%s credentials:\n", service)
				fmt.Println("Username:", username)
				fmt.Println("Password/PAT:", password)
			}

			fmt.Println()
		}
	},
}


func init() {
	rootCmd.AddCommand(configCmd)
	configCmd.AddCommand(dockerCmd)
	configCmd.AddCommand(listCmd)
	configCmd.PersistentFlags().BoolP("remove", "r", false, "remove the associated configuration")
}