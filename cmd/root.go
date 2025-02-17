/*
Copyright © 2025 Rahul Medicharla <rmedicharla@gmail.com>

*/
package cmd

import (
	"os"
	"github.com/spf13/cobra"
	"github.com/rahulmedicharla/kubefs/utils"
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "kubefs",
	Short: "kubefs -  a cli tool to create & deploy full stack applications onto kubernetes clusters",
	Long: `kubefs - a cli tool to create & deploy full stack applications onto kubernetes clusters`,
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	err := utils.ReadManifest()
	if err != nil {
		utils.PrintError(err.Error())
		os.Exit(1)
	}
	utils.GetHttpClient()
}


