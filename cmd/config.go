/*
Copyright © 2025 Rahul Medicharla <rmedicharla@gmail.com>

*/
package cmd

import (
	"context"
	"fmt"
	"strings"
	"github.com/spf13/cobra"
	"github.com/zalando/go-keyring"
	"github.com/rahulmedicharla/kubefs/utils"
	"github.com/rahulmedicharla/kubefs/types"
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

var gcpCmd = &cobra.Command{
	Use:   "gcp",
	Short: "Configure GCP settings",
	Long:  `Configure GCP settings for kubefs
example: 
	kubefs config gcp --flags
	`,
	Run: func(cmd *cobra.Command, args []string) {
		// authenticate via gcloud cli

		remove, err := cmd.Flags().GetBool("remove")
		if err != nil {
			utils.PrintError(fmt.Sprintf("Error reading remove flag: %v", err.Error()))
			return
		}

		if remove {
			// Revoke gcloud authentication
			err = utils.RunCommand("gcloud auth revoke", true, true)
			if err != nil {
				utils.PrintError(fmt.Sprintf("Error revoking GCP authentication: %v", err.Error()))
				return
			}

			err = utils.RemoveCloudConfig(&utils.ManifestData, "gcp")
			if err != nil {
				utils.PrintError(fmt.Sprintf("Error removing GCP configuration from manifest: %v", err.Error()))
				return
			}

			utils.PrintSuccess("GCP authentication revoked successfully")
		} else {

			// Authenticate and enable with GCP using gcloud CLI
			err = utils.AuthenticateGCP()
			if err != nil {
				utils.PrintError(fmt.Sprintf("Error authenticating with GCP: %v", err.Error()))
				return
			}

			// gather configuration details
			var projectName string
			ctx := context.Background()

			err = utils.ReadInput("Enter GCP Project Id: ", &projectName)
			if err != nil {
				utils.PrintError(fmt.Sprintf("Error reading GCP Project Id: %v", err.Error()))
				return
			}
			
			// Setup GCP
			err, projectId, clusterName, region := utils.SetupGcp(ctx, projectName)
			if err != nil {
				utils.PrintError(err.Error())
				return
			}
			
			// Save GCP configuration
			cloudConfig := types.CloudConfig{
				Provider: "gcp",
				ProjectId: *projectId,
				ProjectName: projectName,
				ClusterName: *clusterName,
				Region: *region,
			}

			err, _ = utils.VerifyGcpProject()
			if err == nil {
				// Update existing config
				err = utils.UpdateCloudConfig(&utils.ManifestData, "gcp", &cloudConfig)
				if err != nil {
					utils.PrintError(fmt.Sprintf("Error updating GCP configuration to manifest: %v", err.Error()))
					return
				}
				
				utils.PrintSuccess(fmt.Sprintf("GCP Project updated successfully: %s", projectName))
				return
			}

			// Add new config
			utils.ManifestData.CloudConfig = append(utils.ManifestData.CloudConfig, cloudConfig)
			err = utils.WriteManifest(&utils.ManifestData, "manifest.yaml")
			if err != nil {
				utils.PrintError(fmt.Sprintf("Error saving GCP configuration to manifest: %v", err.Error()))
				return
			}

			utils.PrintSuccess(fmt.Sprintf("GCP Project configured successfully: %s", projectName))
		
		}
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
			var username, pat string
			err := utils.ReadInput("Enter Docker username: ", &username)
			if err != nil {
				utils.PrintError(fmt.Sprintf("Error reading Docker username: %v", err.Error()))
			}

			err = utils.ReadInput("Enter Docker PAT (https://docs.docker.com/security/for-developers/access-tokens/): ", &pat)
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

		fmt.Println("Cloud Configurations:")
		for _, config := range utils.ManifestData.CloudConfig {
			fmt.Printf("Provider: %s\n", config.Provider)
			fmt.Printf("Project ID: %s\n", config.ProjectId)
			fmt.Printf("Cluster Name: %s\n", config.ClusterName)
			fmt.Println()
		}
	},
}


func init() {
	rootCmd.AddCommand(configCmd)
	configCmd.AddCommand(listCmd)

	configCmd.AddCommand(dockerCmd)
	configCmd.AddCommand(gcpCmd)

	configCmd.PersistentFlags().BoolP("remove", "r", false, "remove the associated configuration")
}