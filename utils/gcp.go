package utils

import (
	"context"
	"fmt"
	"time"
	resourcemanager "cloud.google.com/go/resourcemanager/apiv3"
	resourcemanagerpb "cloud.google.com/go/resourcemanager/apiv3/resourcemanagerpb"
	serviceusage "cloud.google.com/go/serviceusage/apiv1"
	serviceusagepb "cloud.google.com/go/serviceusage/apiv1/serviceusagepb"
	container "cloud.google.com/go/container/apiv1"
	containerpb "cloud.google.com/go/container/apiv1/containerpb"
	compute "cloud.google.com/go/compute/apiv1"
	computepb "cloud.google.com/go/compute/apiv1/computepb"
	"google.golang.org/api/iterator"
	"github.com/rahulmedicharla/kubefs/types"
)

func VerifyGcpProject() (error, *types.CloudConfig) {
	//
	// Verify if the user is authenticated with GCP using gcloud CLI
	//

	if ManifestData.CloudConfig != nil {
		for _, config := range ManifestData.CloudConfig {
			if config.Provider == "gcp" {
				return nil, &config
			}
		}
	}

	return fmt.Errorf("GCP project not setup. Please run 'kubefs config gcp' to setup GCP project."), nil
}

func GetOrCreateGCPCluster(ctx context.Context, gcpConfig *types.CloudConfig) error {
	c, err := container.NewClusterManagerClient(ctx)
	if err != nil {
		return fmt.Errorf("failed to create cluster manager client: %v", err)
	}
	defer c.Close()

	req := &containerpb.ListClustersRequest{
		Parent: fmt.Sprintf("%s/locations/%s", gcpConfig.ProjectId, gcpConfig.Region),
	}
	resp, err := c.ListClusters(ctx, req)
	if err != nil {
		return fmt.Errorf("failed to list clusters: %v", err)
	}

	for _, cluster := range resp.GetClusters() {
		if cluster.GetName() == gcpConfig.ClusterName {
			PrintSuccess(fmt.Sprintf("GKE Cluster %s found in GCP", gcpConfig.ClusterName))
			return nil
		}
	}

	PrintWarning(fmt.Sprintf("GKE Cluster %s not found in GCP. Creating new cluster with default values...", gcpConfig.ClusterName))

	// Create new cluster if not found
	cluster := &containerpb.Cluster{
		Name: gcpConfig.ClusterName,
		Autopilot: &containerpb.Autopilot{
			Enabled: true,
		},
	}

	createClusterReq := &containerpb.CreateClusterRequest{
		Parent: fmt.Sprintf("%s/locations/%s", gcpConfig.ProjectId, gcpConfig.Region),
		Cluster: cluster,
	}

	operation, err := c.CreateCluster(ctx, createClusterReq)
	if err != nil {
		return fmt.Errorf("failed to create cluster: %v", err)
	}

	progress := operation.GetProgress()
	for progress.GetStatus() != containerpb.Operation_DONE {
		if progress.GetStatus() == containerpb.Operation_ABORTING {
			return fmt.Errorf("failed to create cluster: operation aborted")
		}
		PrintWarning(fmt.Sprintf("Waiting for GKE cluster creation to complete... %s", progress.GetStatus().String()))
		time.Sleep(10 * time.Second)
		progress = operation.GetProgress()
	}

	PrintSuccess(fmt.Sprintf("GKE Cluster %s created successfully in GCP", gcpConfig.ClusterName))
	return nil
}

func AuthenticateGCP() error {
	commands:= []string{
		"gcloud auth login --update-adc",
		"gcloud components install gke-gcloud-auth-plugin",
	}

	PrintSuccess("Starting GCP authentication process... Opening link in browser")
	err := RunMultipleCommands(commands, true, true)
	if err != nil {
		return err
	}

	PrintSuccess("Authenticated with GCP successfully")
	return nil
}

func SearchGcpProjects(ctx context.Context, projectName string) (error, *string) {
	//
	// Search for existing GCP projects with the given project ID
	// Returns projectName projects/id
	//

	client, err := resourcemanager.NewProjectsClient(ctx)
	if err != nil {
		return fmt.Errorf("failed to create projects client: %v", err), nil
	}
	defer client.Close()

	req := &resourcemanagerpb.SearchProjectsRequest{
		Query: fmt.Sprintf("projectId:%s", projectName),
	}
	it := client.SearchProjects(ctx, req)

	for {
		project, err := it.Next()

		if err == iterator.Done {
			break
		}
		if err != nil {
			return fmt.Errorf("failed to list projects: %v", err), nil
		}

		if project.GetProjectId() == projectName {
			PrintSuccess(fmt.Sprintf("Project %s found in GCP", projectName))
			projectName := project.GetName()
			return nil, &projectName
		}
	}

	return fmt.Errorf("project %s not found in GCP.", projectName), nil
}

func EnableGcpServices(ctx context.Context, projectName string, services []string) error {
	//
	// Enable required GCP services for the project
	//
	client, err := serviceusage.NewClient(ctx)
	if err != nil {
		return fmt.Errorf("failed to create service usage client: %v", err)
	}
	defer client.Close()

	for _, service := range services {
		req := &serviceusagepb.EnableServiceRequest{
			Name: fmt.Sprintf("%s/services/%s", projectName, service),
		}
		
		operation, err := client.EnableService(ctx, req)
		if err != nil {
			return fmt.Errorf("failed to enable service %s: %v", service, err)
		}

		_, err = operation.Wait(ctx)
		if err != nil {
			return fmt.Errorf("failed to wait for service %s to be enabled: %v", service, err)
		}

	}

	PrintSuccess(fmt.Sprintf("Enabled GCP services %v for project: %s", services, projectName))
	return nil
}

func VerifyRegion(ctx context.Context, projectId *string, region string) error {
	//
	// Verify if the given region is valid in GCP
	//

	client, err := compute.NewRegionsRESTClient(ctx)
	if err != nil {
		return fmt.Errorf("failed to create regions client: %v", err)
	}
	defer client.Close()

	req := &computepb.ListRegionsRequest{
		Project: *projectId,
	}

	it := client.List(ctx, req)
	regions := []string{}
	for {
		r, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return fmt.Errorf("failed to list regions: %v", err)
		}

		regions = append(regions, r.GetName())

		if r.GetName() == region {
			return nil
		}
	}

	return fmt.Errorf("region %s not found in GCP. Available regions: %v", region, regions)
}

func SetupGcp(ctx context.Context, projectName string) (error, *string, *string, *string) {
	//
	// Setup GCP project by verifying project existence or creating new project
	//
	
	// Verify project exists
	err, projectId := SearchGcpProjects(ctx, projectName)
	if err != nil {
		return fmt.Errorf("error verifying GCP project: %v", err), nil, nil, nil
	}
	PrintSuccess(fmt.Sprintf("Found GCP Project: %s", projectName))

	// Enable required services
	services := []string{
		"compute.googleapis.com",
		"container.googleapis.com",
	}

	PrintSuccess(fmt.Sprintf("Enabling required GCP services for project: %s", projectName))

	err = EnableGcpServices(ctx, *projectId, services)
	if err != nil {
		return fmt.Errorf("error enabling GCP services: %v", err), nil, nil, nil
	}

	var clusterName, region string

	err = ReadInput("Enter GKE Cluster Name: ", &clusterName)
	if err != nil {
		return fmt.Errorf("error reading GKE Cluster Name: %v", err), nil, nil, nil
	}

	err = ReadInput("Enter GKE Cluster Region: ", &region)
	if err != nil {
		return fmt.Errorf("error reading GKE Cluster Region: %v", err), nil, nil, nil
	}

	err = VerifyRegion(ctx, projectId, region)
	if err != nil {
		return fmt.Errorf("error verifying GKE Cluster Region: %v", err), nil, nil, nil
	}

	return nil, projectId, &clusterName, &region
}