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

func GetGCPClusterContext(gcpConfig *types.CloudConfig) error {
	return RunCommand(fmt.Sprintf("gcloud container clusters get-credentials %s --location %s", gcpConfig.MainCluster, gcpConfig.Region), true, true)
}

func DeleteGCPCluster(gcpConfig *types.CloudConfig, clusterName string) error {
	ctx := context.Background()
	client, err := container.NewClusterManagerClient(ctx)
	if err != nil {
		return fmt.Errorf("failed to create cluster manager client: %v", err)
	}
	defer client.Close()

	req := &containerpb.DeleteClusterRequest{
		Name: fmt.Sprintf("%s/locations/%s/clusters/%s", gcpConfig.ProjectId, gcpConfig.Region, clusterName),
	}

	operation, err := client.DeleteCluster(ctx, req)
	if err != nil {
		return fmt.Errorf("failed to delete cluster: %v", err)
	}

	operationId := operation.GetName()

	for operation.GetStatus() != containerpb.Operation_DONE {
		if operation.GetStatus() == containerpb.Operation_ABORTING {
			return fmt.Errorf("failed to delete cluster: operation aborted")
		}
		PrintWarning(fmt.Sprintf("Waiting for GKE cluster deletion to complete... %s", operation.GetStatus().String()))
		time.Sleep(10 * time.Second)

		getOperationReq := &containerpb.GetOperationRequest{
			Name: fmt.Sprintf("%s/locations/%s/operations/%s", gcpConfig.ProjectId, gcpConfig.Region, operationId),
		}
		operation, err = client.GetOperation(ctx, getOperationReq)
		if err != nil {
			return fmt.Errorf("failed to get operation status: %v", err)
		}
	}

	PrintInfo(fmt.Sprintf("GKE Cluster %s deleted successfully from GCP", clusterName))

	return nil
}

func ProvisionGcpCluster(gcpConfig *types.CloudConfig, clusterName string) error {
	PrintWarning(fmt.Sprintf("Creating new cluster with default values...", clusterName))

	ctx := context.Background()
	c, err := container.NewClusterManagerClient(ctx)
	if err != nil {
		return fmt.Errorf("failed to create cluster manager client: %v", err)
	}
	defer c.Close()

	// Create new cluster if not found
	cluster := &containerpb.Cluster{
		Name: clusterName,
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

	operationId := operation.GetName()

	for operation.GetStatus() != containerpb.Operation_DONE {
		// Check for aborting status
		if operation.GetStatus() == containerpb.Operation_ABORTING {
			return fmt.Errorf("failed to create cluster: operation aborted")
		}

		// Print progress status
		PrintWarning(fmt.Sprintf("Waiting for GKE cluster creation to complete... %s", operation.GetStatus().String()))
		time.Sleep(10 * time.Second)

		// Get updated operation status
		getOperationReq := &containerpb.GetOperationRequest{
			Name: fmt.Sprintf("%s/locations/%s/operations/%s", gcpConfig.ProjectId, gcpConfig.Region, operationId),
		}
		operation, err = c.GetOperation(ctx, getOperationReq)
		if err != nil {
			return fmt.Errorf("failed to get operation status: %v", err)
		}
	}

	PrintInfo(fmt.Sprintf("GKE Cluster %s created successfully in GCP; Installing dependencies...", clusterName))

	commands := []string{
		fmt.Sprintf("gcloud container clusters get-credentials %s --location %s", clusterName, gcpConfig.Region),
		"kubectl apply -f https://github.com/kubernetes-sigs/metrics-server/releases/latest/download/components.yaml",
		"helm install ingress-nginx ingress-nginx/ingress-nginx --create-namespace --namespace ingress-nginx --set controller.service.annotations.service.beta.kubernetes.io/azure-load-balancer-health-probe-request-path=/healthz --set controller.service.externalTrafficPolicy=Local",
	}

	return RunMultipleCommands(commands, true, true)
}

func AuthenticateGCP() error {
	commands:= []string{
		"gcloud auth login --update-adc",
		"gcloud components install gke-gcloud-auth-plugin",
	}

	PrintInfo("Starting GCP authentication process... Opening link in browser")
	err := RunMultipleCommands(commands, true, true)
	if err != nil {
		return err
	}

	PrintInfo("Authenticated with GCP successfully")
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
			PrintInfo(fmt.Sprintf("Project %s found in GCP", projectName))
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

	PrintInfo(fmt.Sprintf("Enabled GCP services %v for project: %s", services, projectName))
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

func SetupGcp(ctx context.Context, projectName string) (error, *string, *string) {
	//
	// Setup GCP project by verifying project existence or creating new project
	//
	
	// Verify project exists
	err, projectId := SearchGcpProjects(ctx, projectName)
	if err != nil {
		return fmt.Errorf("error verifying GCP project: %v", err), nil, nil
	}
	PrintInfo(fmt.Sprintf("Found GCP Project: %s", projectName))

	// Enable required services
	services := []string{
		"compute.googleapis.com",
		"container.googleapis.com",
	}

	PrintInfo(fmt.Sprintf("Enabling required GCP services for project: %s", projectName))

	err = EnableGcpServices(ctx, *projectId, services)
	if err != nil {
		return fmt.Errorf("error enabling GCP services: %v", err), nil, nil
	}

	var region string

	err = ReadInput("Enter GKE Cluster Region: ", &region)
	if err != nil {
		return fmt.Errorf("error reading GKE Cluster Region: %v", err), nil, nil
	}

	err = VerifyRegion(ctx, projectId, region)
	if err != nil {
		return fmt.Errorf("error verifying GKE Cluster Region: %v", err), nil, nil
	}

	return nil, projectId, &region
}