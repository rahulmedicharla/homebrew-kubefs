package utils

import (
    _ "embed"
	"fmt"
    "os"
    "github.com/rahulmedicharla/kubefs/types"
    "gopkg.in/yaml.v3"
    "reflect"
    "encoding/json"
)

var ManifestData types.Project
var ManifestStatus int

//go:embed kubefshelper 
var KubefsHelper []byte

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

func GetResourceFromName(name string) *types.Resource {
    for _, resource := range ManifestData.Resources {
        if resource.Name == name {
            return &resource
        }
    }
    return nil
}

func ReadYaml(path string) (int, map[string]interface{}) {
    data, err := os.ReadFile(path)
    if err != nil {
        PrintError(fmt.Sprintf("Error reading YAML file: %v", err))
        return types.ERROR, nil
    }

    var yamlData map[string]interface{}
    err = yaml.Unmarshal(data, &yamlData)
    if err != nil {
        PrintError(fmt.Sprintf("Error unmarshaling YAML: %v", err))
        return types.ERROR, nil
    }

    return types.SUCCESS, yamlData
}

func WriteYaml(data *map[string]interface{}, path string) int {
    yamlData, err := yaml.Marshal(data)
    if err != nil {
        PrintError(fmt.Sprintf("Error marshaling YAML: %v", err))
        return types.ERROR
    }

    err = os.WriteFile(path, yamlData, 0644)
    if err != nil {
        PrintError(fmt.Sprintf("Error writing YAML to file: %v", err))
        return types.ERROR
    }

    return types.SUCCESS
}

func ReadManifest() int{
    projectErr := ValidateProject()
    if projectErr == types.ERROR {
        return types.ERROR
    }

    data, err := os.ReadFile("manifest.yaml")
    if err != nil {
        PrintError(fmt.Sprintf("Error reading manifest: %v", err))
        return types.ERROR
    }

    err = yaml.Unmarshal(data, &ManifestData)
    if err != nil {
        PrintError(fmt.Sprintf("Error reading manifest: %v", err))
        return types.ERROR
    }

    ManifestStatus = types.SUCCESS
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

func ReadJson(path string) (int, map[string]interface{}) {
    data, err := os.ReadFile(path)
    if err != nil {
        PrintError(fmt.Sprintf("Error reading JSON file: %v", err))
        return types.ERROR, nil
    }

    var jsonData map[string]interface{}
    err = json.Unmarshal(data, &jsonData)
    if err != nil {
        PrintError(fmt.Sprintf("Error unmarshaling JSON: %v", err))
        return types.ERROR, nil
    }

    return types.SUCCESS, jsonData
}


func WriteJson(data map[string]interface{}, path string) int {
    jsonData, err := json.MarshalIndent(data, "", "  ")
    if err != nil {
        PrintError(fmt.Sprintf("Error marshaling JSON: %v", err))
        return types.ERROR
    }

    err = os.WriteFile(path, jsonData, 0644)
    if err != nil {
        PrintError(fmt.Sprintf("Error writing JSON to file: %v", err))
        return types.ERROR
    }

    return types.SUCCESS
}

func UpdateResource(project *types.Project, resource *types.Resource, field string, new_value string) int{
    for i, res := range project.Resources {
        if res.Name == resource.Name {
            reflect.ValueOf(&project.Resources[i]).Elem().FieldByName(field).SetString(new_value)
            return WriteManifest(project)
        }
    }
    return types.ERROR
}

func RemoveAll(project *types.Project) int {
    project.Resources = []types.Resource{}
    return WriteManifest(project)
}

func RemoveResource(project *types.Project, name string) int {
    for i, resource := range project.Resources {
        if resource.Name == name {
            project.Resources = append(project.Resources[:i], project.Resources[i+1:]...)
            return WriteManifest(project)
        }
    }
    return types.ERROR
}

func ValidateProject() int{
    _, err := os.Stat("manifest.yaml")
    if os.IsNotExist(err) {
        ManifestStatus = types.ERROR
        return types.ERROR
    }
    return types.SUCCESS 
}