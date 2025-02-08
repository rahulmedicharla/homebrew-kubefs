package utils

import (
    _ "embed"
	"fmt"
    "os"
    "github.com/rahulmedicharla/kubefs/types"
    "gopkg.in/yaml.v3"
    "reflect"
    "encoding/json"
    "bufio"
)

var ManifestData types.Project
var ManifestStatus int

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

func GetAddonFromName(name string) *types.Addon {
    for _, addon := range ManifestData.Addons {
        if addon.Name == name {
            return &addon
        }
    }
    PrintError(fmt.Sprintf("Addon %s not found", name))
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

func ReadEnv(path string) (int, []string) {
    _, err := os.Stat(path)
    if os.IsNotExist(err) {
        return types.ERROR, nil
    }

    file, err := os.Open(path)
    if err != nil {
        PrintError(fmt.Sprintf("Error opening env file: %v", err))
        return types.ERROR, nil
    }
    defer file.Close()

    scanner := bufio.NewScanner(file)
    var envData []string
    for scanner.Scan() {
        line := scanner.Text()
        envData = append(envData, line)
    }

    return types.SUCCESS, envData


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
    resourceList := []types.Resource{}
    
    for i, resource := range project.Resources {
        if resource.Name != name {
            resourceList = append(resourceList, project.Resources[i])            
        }
    }
    project.Resources = resourceList
    return WriteManifest(project)
}

func RemoveAddon(project *types.Project, name string) int {
    addonList := []types.Addon{}
    
    for i, addon := range project.Addons {
        if addon.Name != name {
            addonList = append(addonList, project.Addons[i])            
        }
    }
    project.Addons = addonList
    return WriteManifest(project)
}

func ValidateProject() int{
    _, err := os.Stat("manifest.yaml")
    if os.IsNotExist(err) {
        ManifestStatus = types.ERROR
        return types.ERROR
    }
    return types.SUCCESS 
}

func VerifyName(name string) bool {
    for _, resource := range ManifestData.Resources {
        if resource.Name == name {
            PrintError(fmt.Sprintf("Resource %s already exists", name))
            return false
        }
    }

    for _, addon := range ManifestData.Addons {
        if addon.Name == name {
            PrintError(fmt.Sprintf("Addon %s already exists", name))
            return false
        }
    }

    return true
}

func VerifyPort(port int) bool {
    if port == 6000 || port == 8000 {
        PrintError("Port 6000 and 8000 are reserved")
        return false
    }

    for _, resource := range ManifestData.Resources {
        if resource.Port == port {
            PrintError(fmt.Sprintf("Port %d already in use %s", port, resource.Name))
            return false
        }
    }

    for _, addon := range ManifestData.Addons {
        if addon.Port == port {
            PrintError(fmt.Sprintf("Port %d already in use by %s", port, addon.Name))
            return false
        }
    }

    return true
}

func VerifyFramework(framework string, rType string) bool {
    for _, f := range types.FRAMEWORKS[rType] {
        if f == framework {
            return true
        }
    }
    PrintError(fmt.Sprintf("Framework %s not supported for %s", framework, rType))
    return false
}

func GetCurrentResourceNames() []string {
    var names []string
    for _, resource := range ManifestData.Resources {
        names = append(names, resource.Name)
    }
    return names
}