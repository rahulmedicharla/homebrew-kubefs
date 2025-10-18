package utils

import (
	"context"
	"fmt"
	resourcemanager "cloud.google.com/go/resourcemanager/apiv3"
	resourcemanagerpb "cloud.google.com/go/resourcemanager/apiv3/resourcemanagerpb"
	"google.golang.org/api/iterator"
)

func AuthenticateGCP() error {
	PrintSuccess("Starting GCP authentication process... Opening link in browser")
	err := RunCommand("gcloud auth application-default login", true, true)
	if err != nil {
		return err
	}

	PrintSuccess("Authenticated with GCP successfully")
	return nil
}

func VerifyGCPProject(ctx context.Context, projectName string) error {
	// 
	// Verify the project name exists in GCP, else create new project
	// 

	client, err := resourcemanager.NewProjectsClient(ctx)
	if err != nil {
		return fmt.Errorf("failed to create projects client: %v", err) 
	}
	defer client.Close()

	req := &resourcemanagerpb.SearchProjectsRequest{
		// Query: fmt.Sprintf("name:%s", projectName),
	}
	it := client.SearchProjects(ctx, req)

	for {
		project, err := it.Next()

		if err == iterator.Done {
			break
		}
		if err != nil {
			return fmt.Errorf("failed to list projects: %v", err)
		}


		if project.GetProjectId() == projectName {
			PrintSuccess(fmt.Sprintf("Project %s found in GCP", projectName))
			return nil // Project found
		}
	}

	PrintWarning(fmt.Sprintf("Project %s not found in GCP... Creating one now", projectName))

	createProjReq := &resourcemanagerpb.CreateProjectRequest{
		Project: &resourcemanagerpb.Project{
			ProjectId: projectName,
		},
	}

	operation, err := client.CreateProject(ctx, createProjReq)
	if err != nil {
		return fmt.Errorf("failed to create project: %v", err)
	}

	_, err = operation.Wait(ctx)
	if err != nil {
		return fmt.Errorf("failed to wait for project creation: %v", err)
	}

	PrintSuccess(fmt.Sprintf("Project %s created successfully", projectName))
	return nil
}