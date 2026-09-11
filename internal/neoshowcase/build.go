package neoshowcase

import (
	"context"
	"fmt"

	"connectrpc.com/connect"

	"github.com/traP-jp/terraform-provider-neoshowcase/internal/neoshowcase/gen"
)

func (c *Client) GetApplicationBuilds(ctx context.Context, applicationID string) ([]*gen.Build, error) {
	response, err := c.rpc.GetBuilds(ctx, connect.NewRequest(&gen.ApplicationIdRequest{Id: applicationID}))
	if err != nil {
		return nil, fmt.Errorf("get builds for NeoShowcase application %q: %w", applicationID, err)
	}
	return response.Msg.GetBuilds(), nil
}

func (c *Client) GetBuild(ctx context.Context, buildID string) (*gen.Build, error) {
	response, err := c.rpc.GetBuild(ctx, connect.NewRequest(&gen.BuildIdRequest{BuildId: buildID}))
	if err != nil {
		return nil, fmt.Errorf("get NeoShowcase build %q: %w", buildID, err)
	}
	return response.Msg, nil
}
