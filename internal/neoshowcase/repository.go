package neoshowcase

import (
	"context"
	"fmt"

	"connectrpc.com/connect"

	"github.com/traP-jp/terraform-provider-neoshowcase/internal/neoshowcase/gen"
)

func (c *Client) CreateRepository(ctx context.Context, request *gen.CreateRepositoryRequest) (*gen.Repository, error) {
	response, err := c.rpc.CreateRepository(ctx, connect.NewRequest(request))
	if err != nil {
		return nil, fmt.Errorf("create NeoShowcase repository: %w", err)
	}
	return response.Msg, nil
}

func (c *Client) GetOwnedRepositoryByURL(ctx context.Context, repositoryURL string) (*gen.Repository, error) {
	response, err := c.rpc.GetRepositories(ctx, connect.NewRequest(&gen.GetRepositoriesRequest{
		Scope: gen.GetRepositoriesRequest_MINE,
	}))
	if err != nil {
		return nil, fmt.Errorf("list owned NeoShowcase repositories: %w", err)
	}
	for _, repository := range response.Msg.GetRepositories() {
		if repository.GetUrl() == repositoryURL {
			return repository, nil
		}
	}
	return nil, nil
}

func (c *Client) GetRepository(ctx context.Context, repositoryID string) (*gen.Repository, error) {
	response, err := c.rpc.GetRepository(ctx, connect.NewRequest(&gen.RepositoryIdRequest{RepositoryId: repositoryID}))
	if err != nil {
		return nil, fmt.Errorf("get NeoShowcase repository %q: %w", repositoryID, err)
	}
	return response.Msg, nil
}

func (c *Client) UpdateRepository(ctx context.Context, request *gen.UpdateRepositoryRequest) error {
	_, err := c.rpc.UpdateRepository(ctx, connect.NewRequest(request))
	if err != nil {
		return fmt.Errorf("update NeoShowcase repository %q: %w", request.GetId(), err)
	}
	return nil
}

func (c *Client) DeleteRepository(ctx context.Context, repositoryID string) error {
	_, err := c.rpc.DeleteRepository(ctx, connect.NewRequest(&gen.RepositoryIdRequest{RepositoryId: repositoryID}))
	if err != nil {
		return fmt.Errorf("delete NeoShowcase repository %q: %w", repositoryID, err)
	}
	return nil
}
