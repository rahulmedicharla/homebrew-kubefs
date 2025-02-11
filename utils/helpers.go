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
    "errors"
)

var ManifestData types.Project
var ManifestStatus error

func PrintError(message string) {
	fmt.Printf("\033[31mError: %s\033[0m\n", message)
}

func PrintSuccess(message string) {
	fmt.Printf("\033[32mSuccess: %s\033[0m\n", message)
}

func PrintWarning(message string) {
	fmt.Printf("\033[33m%s\033[0m\n", message)
}

func GetResourceFromName(name string) (*types.Resource, error) {
    for _, resource := range ManifestData.Resources {
        if resource.Name == name {
            return &resource, nil
        }
    }
    return nil, errors.New(fmt.Sprintf("Resource %s not found", name))
}

func GetAddonFromName(name string) (*types.Addon, error) {
    for _, addon := range ManifestData.Addons {
        if addon.Name == name {
            return &addon, nil
        }
    }
    return nil, errors.New(fmt.Sprintf("Addon %s not found", name))
}

func WriteYaml(data *map[string]interface{}, path string) error {
    yamlData, err := yaml.Marshal(data)
    if err != nil {
        return err
    }

    err = os.WriteFile(path, yamlData, 0644)
    if err != nil {
        return err
    }

    return nil
}

func ReadManifest() error{
    projectErr := ValidateProject()
    if projectErr != nil {
        return projectErr
    }

    data, err := os.ReadFile("manifest.yaml")
    if err != nil {
        return err
    }

    err = yaml.Unmarshal(data, &ManifestData)
    if err != nil {
        return err
    }

    ManifestStatus = nil
    return nil
}

func WriteManifest(project *types.Project, path string) error{
    data, err := yaml.Marshal(project)
    if err != nil {
        return err
    }

    err = os.WriteFile(path, data, 0644)
    if err != nil {
        return err
    }

    return nil
}

func ReadJson(path string) (*map[string]interface{}, error) {
    data, err := os.ReadFile(path)
    if err != nil {
        return nil, err
    }

    var jsonData map[string]interface{}
    err = json.Unmarshal(data, &jsonData)
    if err != nil {
        return nil, err
    }

    return &jsonData, nil
}

func WriteJson(data map[string]interface{}, path string) error {
    jsonData, err := json.MarshalIndent(data, "", "  ")
    if err != nil {
        return err
    }

    err = os.WriteFile(path, jsonData, 0644)
    if err != nil {
        return err
    }

    return nil
}

func ReadEnv(path string) ([]string, error) {
    _, err := os.Stat(path)
    if os.IsNotExist(err) {
        return nil, err
    }

    file, err := os.Open(path)
    if err != nil {
        return nil, err
    }
    defer file.Close()

    scanner := bufio.NewScanner(file)
    var envData []string
    for scanner.Scan() {
        line := scanner.Text()
        envData = append(envData, line)
    }

    return envData, nil
}

func UpdateResource(project *types.Project, resource *types.Resource, field string, new_value string) error{
    for i, res := range project.Resources {
        if res.Name == resource.Name {
            reflect.ValueOf(&project.Resources[i]).Elem().FieldByName(field).SetString(new_value)
            return WriteManifest(project, "manifest.yaml")
        }
    }
    return errors.New("Resource not found")
}

func RemoveResource(project *types.Project, name string) error {
    resourceList := []types.Resource{}
    
    for i, resource := range project.Resources {
        if resource.Name != name {
            resourceList = append(resourceList, project.Resources[i])            
        }
    }
    project.Resources = resourceList
    return WriteManifest(project, "manifest.yaml")
}

func RemoveAddon(project *types.Project, name string) error {
    addonList := []types.Addon{}
    
    for i, addon := range project.Addons {
        if addon.Name != name {
            addonList = append(addonList, project.Addons[i])            
        }
    }
    project.Addons = addonList
    return WriteManifest(project, "manifest.yaml")
}

func ValidateProject() error{
    _, err := os.Stat("manifest.yaml")
    if os.IsNotExist(err) {
        ManifestStatus = errors.New("Not a valid kubefs project: use 'kubefs init' to create a new project")
        return ManifestStatus
    }
    return nil
}

func VerifyName(name string) error {
    for _, resource := range ManifestData.Resources {
        if resource.Name == name {
            return errors.New(fmt.Sprintf("Resource %s already exists", name))
        }
    }

    for _, addon := range ManifestData.Addons {
        if addon.Name == name {
            return errors.New(fmt.Sprintf("Addon %s already exists", name))
        }
    }

    return nil
}

func VerifyPort(port int) error {
    for _, resource := range ManifestData.Resources {
        if resource.Port == port {
            return errors.New(fmt.Sprintf("Port %d already in use by %s", port, resource.Name))
        }
    }

    for _, addon := range ManifestData.Addons {
        if addon.Port == port {
            return errors.New(fmt.Sprintf("Port %d already in use by %s", port, addon.Name))
        }
    }

    return nil
}

func VerifyFramework(framework string, rType string) error {
    for _, f := range types.FRAMEWORKS[rType] {
        if f == framework {
            return nil
        }
    }
    return errors.New(fmt.Sprintf("Framework %s not supported for %s", framework, rType))
}

func GetCurrentResourceNames() []string {
    var names []string
    for _, resource := range ManifestData.Resources {
        names = append(names, resource.Name)
    }
    return names
}

func GetHelmChart(dockerRepo string, name string, serviceType string, port int, ingressEnabled bool, ingressHost string, healthCheck string, replicaCount int) *map[string]interface{} {
    baseValuesChart := map[string]interface{}{
      "replicaCount": replicaCount,
      "image": map[string]interface{}{
        "repository": dockerRepo,
        "pullPolicy": "Always",
        "tag": "latest",
      },
      "imagePullSecrets": []string{},
      "namespace": name,
      "serviceAccount": map[string]interface{}{
        "create": true,
        "automount": true,
        "annotations": map[string]interface{}{},
        "name": "",
      },
      "service": map[string]interface{}{
        "type": serviceType,
        "port": port,
      },
      "ingress": map[string]interface{}{
        "enabled": ingressEnabled,
        "className": "nginx",
        "annotations": map[string]interface{}{
          "kubernetes.io/ingress.class": "nginx",
          "nginx.ingress.kubernetes.io/rewrite-target": "/",
        },
        "host": ingressHost,
        "tls": []string{},
      },
      "env": []interface{}{},
      "secrets": []interface{}{},
      "resources": map[string]interface{}{},
      "livenessProbe": map[string]interface{}{
        "httpGet": map[string]interface{}{
          "path": healthCheck,
          "port": "http",
        },
      },
      "readinessProbe": map[string]interface{}{
        "httpGet": map[string]interface{}{
          "path": healthCheck,
          "port": "http",
        },
      },
      "autoscaling": map[string]interface{}{
        "enabled": false,
        "minReplicas": 1,
        "maxReplicas": 100,
        "targetCPUUtilizationPercentage": 80,
      },
      "volumes": []interface{}{},
      "volumeMounts": []interface{}{},
      "nodeSelector": map[string]interface{}{},
      "tolerations": []string{},
      "affinity": map[string]interface{}{},
    }
  
    return &baseValuesChart
  
  }