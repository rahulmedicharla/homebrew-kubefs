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

func PrintError(err error) {
	log.Error(err.Error())
}

func PrintInfo(message string) {
	log.Info(message)
}

func PrintWarning(message string) {
	log.Warn(message)
}

func GetResourceFromName(name string) (*types.Resource, error) {
	for _, resource := range ManifestData.Resources {
		if resource.Name == name {
			return &resource, nil
		}
	}
	return nil, fmt.Errorf("resource [%s] not found, create using 'kubefs create'", name)
}

func GetAddonFromName(name string) (*types.Addon, error) {
	for _, addon := range ManifestData.Addons {
		if addon.Name == name {
			return &addon, nil
		}
	}
	return nil, fmt.Errorf("addon [%s] not found, enable using 'kubefs addons enable'", name)
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

func UpdateCloudConfig(project *types.Project, provider string, config *types.CloudConfig) error {
	for i, conf := range project.CloudConfig {
		if conf.Provider == provider {
			project.CloudConfig[i] = *config
			return WriteManifest(project, "manifest.yaml")
		}
	}
	return fmt.Errorf("cloud config for [%s] not setup. Use 'kubefs config' to setup", provider)
}

func UpdateResource(project *types.Project, name string, resource *types.Resource) error {
	for i, res := range project.Resources {
		if res.Name == name {
			project.Resources[i] = *resource
			return WriteManifest(project, "manifest.yaml")
		}
	}
	return fmt.Errorf("resource [%s] not found. Use 'kubefs create' to setup", name)
}

func UpdateAddons(project *types.Project, name string, addon *types.Addon) error {
	for i, ad := range project.Addons {
		if ad.Name == name {
			project.Addons[i] = *addon
			return WriteManifest(project, "manifest.yaml")
		}
	}
	return fmt.Errorf("addon [%s] not found. Use 'kubefs addons enable' to setup", name)
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

func RemoveCloudConfig(project *types.Project, provider string) error {
	configList := []types.CloudConfig{}

	for i, config := range project.CloudConfig {
		if config.Provider != provider {
			configList = append(configList, project.CloudConfig[i])
		}
	}
	project.CloudConfig = configList
	return WriteManifest(project, "manifest.yaml")
}

func ValidateProject() error {
	_, err := os.Stat("manifest.yaml")
	if os.IsNotExist(err) {
		ManifestStatus = fmt.Errorf("not a valid kubefs project. Use 'kubefs init to setup'")
		return ManifestStatus
	}
	return nil
}

func VerifyName(name string) error {
	for _, resource := range ManifestData.Resources {
		if resource.Name == name {
			return fmt.Errorf("resource [%s] already exists. Try another name", name)
		}
	}

	for _, addon := range ManifestData.Addons {
		if addon.Name == name {
			return fmt.Errorf("addon [%s] already exists. Try another name", name)
		}
	}

	return nil
}

func VerifyPort(port int) error {
	for _, resource := range ManifestData.Resources {
		if resource.Port == port {
			return fmt.Errorf("port [%d] already in use by resource [%s]. Try another port", port, resource.Name)
		}
	}

	for _, addon := range ManifestData.Addons {
		if addon.Port == port {
			return fmt.Errorf("port [%d] already in use by addon [%s]. Try another port", port, addon.Name)
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
	return fmt.Errorf("framework [%s] not supported by kubefs. Try another framework", framework)
}

func VerifyTarget(target string) error {
	for _, t := range types.TARGETS {
		if t == target {
			return nil
		}
	}
	return fmt.Errorf("invalid deployment target [%s]. Supported targets are %v", target, types.TARGETS)
}

func VerifyCloudConfig(provider string) (*types.CloudConfig, error) {
	for _, config := range ManifestData.CloudConfig {
		if config.Provider == provider {
			return &config, nil
		}
	}
	return nil, fmt.Errorf("cloud config [%s] not setup. Setup using 'kubefs config'", provider)
}

func VerifyClusterName(config *types.CloudConfig, clusterName string) error {
	for _, name := range config.ClusterNames {
		if name == clusterName {
			return nil
		}
	}
	return fmt.Errorf("cluster name [%s] not found in [%s]", clusterName, config.Provider)
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
