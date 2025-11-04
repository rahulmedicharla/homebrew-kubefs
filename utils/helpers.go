package utils

import (
	"bufio"
	_ "embed"
	"encoding/json"
	"fmt"
	"os"

	"github.com/rahulmedicharla/kubefs/types"
	log "github.com/sirupsen/logrus"
	"gopkg.in/yaml.v3"
)

var ManifestData types.Project
var ManifestStatus error

// LOGGER FUNCTIONS
func PrintError(err error) {
	log.Error(err.Error())
}

func PrintInfo(message string) {
	log.Info(message)
}

func PrintWarning(message string) {
	log.Warn(message)
}

// RETURNS A POINTER TO RESOURCE/ADDON/CLOUD_CONFIG
func GetResourceFromName(name string) (*types.Resource, error) {
	resource, ok := ManifestData.Resources[name]
	if !ok {
		return nil, fmt.Errorf("resource [%s] not found, create using 'kubefs create'", name)
	}
	return &resource, nil
}

func GetAddonFromName(name string) (*types.Addon, error) {
	addon, ok := ManifestData.Addons[name]
	if !ok {
		return nil, fmt.Errorf("addon [%s] not found, enable using 'kubefs addons enable'", name)
	}
	return &addon, nil
}

func GetCloudConfigFromProvider(provider string) (*types.CloudConfig, error) {
	cloudConfig, ok := ManifestData.CloudConfig[provider]
	if !ok {
		return nil, fmt.Errorf("cloud config [%s] not found, enable using 'kubefs config'", provider)
	}
	return &cloudConfig, nil
}

func GetCurrentResourceNames() []string {
	var names []string
	for key := range ManifestData.Resources {
		names = append(names, key)
	}
	return names
}

// READ AND WRITE FILES
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

func ReadManifest() error {
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

func WriteManifest(project *types.Project, path string) error {
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

// UPDATES A POINTER TO RESOURCE/ADDON/CLOUD_CONFIG
func UpdateResource(project *types.Project, name string, resource *types.Resource) error {
	project.Resources[name] = *resource
	return WriteManifest(project, "manifest.yaml")
}

func UpdateAddons(project *types.Project, name string, addon *types.Addon) error {
	project.Addons[name] = *addon
	return WriteManifest(project, "manifest.yaml")
}

func UpdateCloudConfig(project *types.Project, provider string, config *types.CloudConfig) error {
	project.CloudConfig[provider] = *config
	return WriteManifest(project, "manifest.yaml")

}

// DELETS POINTERS TO RESOURCE/ADDON/CLOUD_CONFIG
func RemoveResource(project *types.Project, name string) error {
	delete(project.Resources, name)
	return WriteManifest(project, "manifest.yaml")
}

func RemoveAddon(project *types.Project, name string) error {
	delete(project.Addons, name)
	return WriteManifest(project, "manifest.yaml")
}

func RemoveCloudConfig(project *types.Project, provider string) error {
	delete(project.CloudConfig, provider)
	return WriteManifest(project, "manifest.yaml")
}

func RemoveClusterName(config *types.CloudConfig, clusterName string) ([]string, error) {
	newConfig := make([]string, 0)
	for _, name := range config.ClusterNames {
		if name != clusterName {
			newConfig = append(newConfig, name)
		}
	}
	return newConfig, nil
}

// VALIDATION
func ValidateProject() error {
	_, err := os.Stat("manifest.yaml")
	if os.IsNotExist(err) {
		return fmt.Errorf("not a valid kubefs project. Use 'kubefs init to setup'")
	}
	return nil
}

func VerifyName(name string) error {
	_, ok := ManifestData.Resources[name]
	if ok {
		return fmt.Errorf("resource [%s] already exists. Try another name", name)
	}

	_, ok = ManifestData.Resources[name]
	if ok {
		return fmt.Errorf("addon [%s] already exists. Try another name", name)
	}
	return nil
}

func VerifyPort(port int) error {
	for name, resource := range ManifestData.Resources {
		if resource.Port == port {
			return fmt.Errorf("port [%d] already in use by resource [%s]. Try another port", port, name)
		}
	}

	for name, addon := range ManifestData.Addons {
		if addon.Port == port {
			return fmt.Errorf("port [%d] already in use by addon [%s]. Try another port", port, name)
		}
	}

	return nil
}

func VerifyFramework(framework string, rType string) error {
	exists := types.FRAMEWORKS[rType].Contains(framework)
	if !exists {
		return fmt.Errorf("framework [%s] not supported by kubefs. Try another framework", framework)
	}
	return nil
}

func VerifyTarget(target string) error {
	exists := types.TARGETS.Contains(target)
	if !exists {
		return fmt.Errorf("invalid deployment target [%s]. Supported targets are %v", target, types.TARGETS)
	}
	return nil
}

func VerifyClusterName(provider string, config *types.CloudConfig, clusterName string) error {
	for _, n := range config.ClusterNames {
		if n == clusterName {
			return nil
		}
	}
	return fmt.Errorf("cluster name [%s] not found in [%s]", clusterName, provider)
}

func GetHelmChart(dockerRepo string, name string, serviceType string, port int, ingressEnabled bool, ingressHost string, healthCheck string, replicaCount int) *map[string]interface{} {
	baseValuesChart := map[string]interface{}{
		"replicaCount": replicaCount,
		"image": map[string]interface{}{
			"repository": dockerRepo,
			"pullPolicy": "Always",
			"tag":        "latest",
		},
		"imagePullSecrets": []string{},
		"namespace":        name,
		"serviceAccount": map[string]interface{}{
			"create":      true,
			"automount":   true,
			"annotations": map[string]interface{}{},
			"name":        "",
		},
		"service": map[string]interface{}{
			"type": serviceType,
			"port": port,
		},
		"ingress": map[string]interface{}{
			"enabled":   ingressEnabled,
			"className": "nginx",
			"annotations": map[string]interface{}{
				"kubernetes.io/ingress.class":                "nginx",
				"nginx.ingress.kubernetes.io/rewrite-target": "/",
			},
			"host": ingressHost,
			"tls":  []string{},
		},
		"env":       []interface{}{},
		"secrets":   []interface{}{},
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
			"enabled":                        false,
			"minReplicas":                    1,
			"maxReplicas":                    100,
			"targetCPUUtilizationPercentage": 80,
		},
		"volumes":      []interface{}{},
		"volumeMounts": []interface{}{},
		"nodeSelector": map[string]interface{}{},
		"tolerations":  []string{},
		"affinity":     map[string]interface{}{},
	}

	return &baseValuesChart

}
