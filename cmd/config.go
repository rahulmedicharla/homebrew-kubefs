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
	Long: `kubefs config - configure kubefs environment and auth configurations`,
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Help()
	},
}

var gcpCmd = &cobra.Command{
	Use:   "gcp",
	Short: "Configure GCP settings",
	Long:  `Configure GCP settings for kubefs`,
	Run: func(cmd *cobra.Command, args []string) {
		service := "gcp"
		user := "kubefs"

		// Read remove flag
		remove, err := cmd.Flags().GetBool("remove")
		if err != nil {
			fmt.Println("Error reading remove flag:", err)
			return
		}

		if remove {
			err := keyring.Delete(service, user)
			if err != nil {
				utils.PrintError(fmt.Sprintf("Error deleting GCP credentials: %v", err))
				return
			}
			utils.PrintSuccess("GCP credentials removed successfully")
		} else {
			var input string

			fmt.Print("Enter GCP username: ")
			fmt.Scanln(&input)
			username := strings.TrimSpace(input)
			fmt.Print("Enter GCP password: ")
			fmt.Scanln(&input)
			password := strings.TrimSpace(input)

			err := keyring.Set(service, user, fmt.Sprintf("%s:%s", username, password))
			if err != nil {
				utils.PrintError(fmt.Sprintf("Error saving GCP credentials: %v", err))
				return
			}

			utils.PrintSuccess("Saving GCP credentials")
		}
	},
}

var awsCmd = &cobra.Command{
	Use:   "aws",
	Short: "Configure AWS settings",
	Long:  `Configure AWS settings for kubefs`,
	Run: func(cmd *cobra.Command, args []string) {
		service := "aws"
		user := "kubefs"

		// Read remove flag
		remove, err := cmd.Flags().GetBool("remove")
		if err != nil {
			utils.PrintError(fmt.Sprintf("Error reading remove flag: %v", err))
			return
		}

		if remove {
			err := keyring.Delete(service, user)
			if err != nil {
				utils.PrintError(fmt.Sprintf("Error deleting AWS credentials: %v", err))
				return
			}
			utils.PrintSuccess("AWS credentials removed successfully")
		} else {
			var input string

			fmt.Print("Enter AWS username: ")
			fmt.Scanln(&input)
			username := strings.TrimSpace(input)
			fmt.Print("Enter AWS password: ")
			fmt.Scanln(&input)
			password := strings.TrimSpace(input)

			err := keyring.Set(service, user, fmt.Sprintf("%s:%s", username, password))
			if err != nil {
				utils.PrintError(fmt.Sprintf("Error saving AWS credentials: %v", err))
				return
			}

			utils.PrintSuccess("Saving AWS credentials")
		}
	},
}

var azureCmd = &cobra.Command{
	Use:   "azure",
	Short: "Configure Azure settings",
	Long:  `Configure Azure settings for kubefs`,
	Run: func(cmd *cobra.Command, args []string) {
		service := "azure"
		user := "kubefs"

		// Read remove flag
		remove, err := cmd.Flags().GetBool("remove")
		if err != nil {
			utils.PrintError(fmt.Sprintf("Error reading remove flag: %v", err))
			return
		}

		if remove {
			err := keyring.Delete(service, user)
			if err != nil {
				utils.PrintError(fmt.Sprintf("Error deleting Azure credentials: %v", err))
				return
			}
			utils.PrintSuccess("Azure credentials removed successfully")
		} else {
			var input string

			fmt.Print("Enter Azure username: ")
			fmt.Scanln(&input)
			username := strings.TrimSpace(input)
			fmt.Print("Enter Azure password: ")
			fmt.Scanln(&input)
			password := strings.TrimSpace(input)

			err := keyring.Set(service, user, fmt.Sprintf("%s:%s", username, password))
			if err != nil {
				utils.PrintError(fmt.Sprintf("Error saving Azure credentials: %v", err))
				return
			}

			utils.PrintSuccess("Saving Azure credentials")
		}
	},
}

var dockerCmd = &cobra.Command{
	Use:   "docker",
	Short: "Configure Docker settings",
	Long:  `Configure Docker settings for kubefs`,
	Run: func(cmd *cobra.Command, args []string) {
		// get service information
		service := "docker"
		user := "kubefs"

		// Read remove flag
		remove, err := cmd.Flags().GetBool("remove")
		if err != nil {
			utils.PrintError(fmt.Sprintf("Error reading remove flag: %v", err))
			return
		}

		if remove {
			err := keyring.Delete(service, user)
			if err != nil {
				utils.PrintError(fmt.Sprintf("Error deleting Docker credentials: %v", err))
				return
			}
			utils.PrintSuccess("Docker credentials removed successfully")
		} else {
			var input string

			fmt.Print("Enter Docker username: ")
			fmt.Scanln(&input)
			username := strings.TrimSpace(input)
			fmt.Print("Enter Docker password: ")
			fmt.Scanln(&input)
			password := strings.TrimSpace(input)

			err := keyring.Set(service, user, fmt.Sprintf("%s:%s", username, password))
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
	Long:  `List all configurations for kubefs`,
	Run: func(cmd *cobra.Command, args []string) {
		user := "kubefs"
		services := []string{"docker", "aws", "azure", "gcp"}

		fmt.Println("Listing all configurations \n")

		for _, service := range services {
			creds, err := keyring.Get(service, user)
			if err != nil {
				fmt.Printf("Error getting %s credentials | No credentials set: %v\n", service, err)
			} else {
				username, password := strings.Split(creds, ":")[0], strings.Split(creds, ":")[1]
				fmt.Printf("%s credentials:\n", service)
				fmt.Println("Username:", username)
				fmt.Println("Password:", password)
			}

			fmt.Println()
		}
	},
}


func init() {
	rootCmd.AddCommand(configCmd)
	configCmd.AddCommand(awsCmd)
	configCmd.AddCommand(azureCmd)
	configCmd.AddCommand(dockerCmd)
	configCmd.AddCommand(gcpCmd)
	configCmd.AddCommand(listCmd)
	configCmd.PersistentFlags().BoolP("remove", "r", false, "remove the associated configuration")
}