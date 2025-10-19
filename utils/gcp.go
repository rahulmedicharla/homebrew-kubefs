package utils

import (
	"context"
	"fmt"
	resourcemanager "cloud.google.com/go/resourcemanager/apiv3"
	resourcemanagerpb "cloud.google.com/go/resourcemanager/apiv3/resourcemanagerpb"
	serviceusage "cloud.google.com/go/serviceusage/apiv1"
	serviceusagepb "cloud.google.com/go/serviceusage/apiv1/serviceusagepb"
	"google.golang.org/api/iterator"
)

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

func SetupGcp(ctx context.Context, projectName string) (error, *string) {
	//
	// Setup GCP project by verifying project existence or creating new project
	//
	
	// Verify project exists
	err, projectId := SearchGcpProjects(ctx, projectName)
	if err != nil {
		return fmt.Errorf("error verifying GCP project: %v", err), nil
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
		return fmt.Errorf("error enabling GCP services: %v", err), nil
	}

	return nil, projectId
}