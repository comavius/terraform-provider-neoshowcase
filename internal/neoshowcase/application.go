package neoshowcase

import (
	"context"
	"fmt"

	"connectrpc.com/connect"

	"github.com/traP-jp/terraform-provider-neoshowcase/internal/neoshowcase/gen"
)

func (c *Client) CreateApplication(ctx context.Context, request *gen.CreateApplicationRequest) (*gen.Application, error) {
	response, err := c.rpc.CreateApplication(ctx, connect.NewRequest(request))
	if err != nil {
		return nil, fmt.Errorf("create NeoShowcase application: %w", err)
	}
	return response.Msg, nil
}

func (c *Client) GetApplication(ctx context.Context, applicationID string) (*gen.Application, error) {
	response, err := c.rpc.GetApplication(ctx, connect.NewRequest(&gen.ApplicationIdRequest{Id: applicationID}))
	if err != nil {
		return nil, fmt.Errorf("get NeoShowcase application %q: %w", applicationID, err)
	}
	return response.Msg, nil
}

func (c *Client) UpdateApplication(ctx context.Context, request *gen.UpdateApplicationRequest) error {
	_, err := c.rpc.UpdateApplication(ctx, connect.NewRequest(request))
	if err != nil {
		return fmt.Errorf("update NeoShowcase application %q: %w", request.GetId(), err)
	}
	return nil
}

func (c *Client) DeleteApplication(ctx context.Context, applicationID string) error {
	_, err := c.rpc.DeleteApplication(ctx, connect.NewRequest(&gen.ApplicationIdRequest{Id: applicationID}))
	if err != nil {
		return fmt.Errorf("delete NeoShowcase application %q: %w", applicationID, err)
	}
	return nil
}

func (c *Client) GetApplicationEnvironmentVariables(ctx context.Context, applicationID string) ([]*gen.ApplicationEnvVar, error) {
	response, err := c.rpc.GetEnvVars(ctx, connect.NewRequest(&gen.ApplicationIdRequest{Id: applicationID}))
	if err != nil {
		return nil, fmt.Errorf("get NeoShowcase application %q environment variables: %w", applicationID, err)
	}
	return response.Msg.GetVariables(), nil
}

func (c *Client) SetApplicationEnvironmentVariable(ctx context.Context, applicationID, key, value string) error {
	_, err := c.rpc.SetEnvVar(ctx, connect.NewRequest(&gen.SetApplicationEnvVarRequest{ApplicationId: applicationID, Key: key, Value: value}))
	if err != nil {
		return fmt.Errorf("set NeoShowcase application %q environment variable %q: %w", applicationID, key, err)
	}
	return nil
}

func (c *Client) DeleteApplicationEnvironmentVariable(ctx context.Context, applicationID, key string) error {
	_, err := c.rpc.DeleteEnvVar(ctx, connect.NewRequest(&gen.DeleteApplicationEnvVarRequest{ApplicationId: applicationID, Key: key}))
	if err != nil {
		return fmt.Errorf("delete NeoShowcase application %q environment variable %q: %w", applicationID, key, err)
	}
	return nil
}

func (c *Client) StartApplication(ctx context.Context, applicationID string) error {
	_, err := c.rpc.StartApplication(ctx, connect.NewRequest(&gen.ApplicationIdRequest{Id: applicationID}))
	if err != nil {
		return fmt.Errorf("start NeoShowcase application %q: %w", applicationID, err)
	}
	return nil
}

func (c *Client) StopApplication(ctx context.Context, applicationID string) error {
	_, err := c.rpc.StopApplication(ctx, connect.NewRequest(&gen.ApplicationIdRequest{Id: applicationID}))
	if err != nil {
		return fmt.Errorf("stop NeoShowcase application %q: %w", applicationID, err)
	}
	return nil
}
