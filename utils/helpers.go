package utils

import (
	"fmt"
    "os"
    "github.com/rahulmedicharla/kubefs/types"
    "gopkg.in/yaml.v3"
)

func PrintError(message string) {
	fmt.Printf("\033[31mError: %s\033[0m\n", message)
}

func PrintSuccess(message string) {
	fmt.Printf("\033[32mSuccess: %s\033[0m\n", message)
}

func PrintWarning(message string) {
	fmt.Printf("\033[33m%s\033[0m\n", message)
}

func Contains(slice []string, item string) bool {
    for _, s := range slice {
        if s == item {
            return true
        }
    }
    return false
}

func ReadManifest(project *types.Project) int{
    data, err := os.ReadFile("manifest.yaml")
    if err != nil {
        PrintError(fmt.Sprintf("Error reading manifest: %v", err))
        return types.ERROR
    }

    err = yaml.Unmarshal(data, project)
    if err != nil {
        PrintError(fmt.Sprintf("Error reading manifest: %v", err))
        return types.ERROR
    }

    return types.SUCCESS
}

func WriteManifest(project *types.Project) int{
    data, err := yaml.Marshal(project)
    if err != nil {
        PrintError(fmt.Sprintf("Error writing manifest: %v", err))
        return types.ERROR
    }

    err = os.WriteFile("manifest.yaml", data, 0644)
    if err != nil {
        PrintError(fmt.Sprintf("Error writing manifest: %v", err))
        return types.ERROR
    }

    return types.SUCCESS
}

func ValidateProject() int{
    _, err := os.Stat("manifest.yaml")
    if os.IsNotExist(err) {
        PrintError("Not a valid kubefs project: use 'kubefs init' to create a new project")
        return types.ERROR
    }
    return types.SUCCESS 
}