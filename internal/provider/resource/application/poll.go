package application

import (
	"context"
	"fmt"
	"time"

	"github.com/traP-jp/terraform-provider-neoshowcase/internal/neoshowcase/gen"
)

const (
	applicationOperationTimeout = 10 * time.Minute
	buildPollInterval           = 11 * time.Second
)

type buildClient interface {
	GetApplicationBuilds(context.Context, string) ([]*gen.Build, error)
	GetBuild(context.Context, string) (*gen.Build, error)
}

func snapshotBuildIDs(ctx context.Context, client buildClient, applicationID string) (map[string]struct{}, error) {
	builds, err := client.GetApplicationBuilds(ctx, applicationID)
	if err != nil {
		return nil, err
	}
	ids := make(map[string]struct{}, len(builds))
	for _, build := range builds {
		ids[build.GetId()] = struct{}{}
	}
	return ids, nil
}

func waitForNewBuild(ctx context.Context, client buildClient, applicationID, commit string, excluded map[string]struct{}) (*gen.Build, error) {
	return waitForNewBuildAtInterval(ctx, client, applicationID, commit, excluded, buildPollInterval)
}

func waitForNewBuildAtInterval(ctx context.Context, client buildClient, applicationID, commit string, excluded map[string]struct{}, interval time.Duration) (*gen.Build, error) {
	for {
		builds, err := client.GetApplicationBuilds(ctx, applicationID)
		if err != nil {
			return nil, err
		}
		if build := newestMatchingBuild(builds, commit, excluded); build != nil {
			return waitForBuildAtInterval(ctx, client, build, interval)
		}
		if err := waitForPoll(ctx, interval); err != nil {
			return nil, err
		}
	}
}

func waitForBuildAtInterval(ctx context.Context, client buildClient, initial *gen.Build, interval time.Duration) (*gen.Build, error) {
	build := initial
	for !terminalBuildStatus(build.GetStatus()) {
		if err := waitForPoll(ctx, interval); err != nil {
			return nil, err
		}
		var err error
		build, err = client.GetBuild(ctx, build.GetId())
		if err != nil {
			return nil, err
		}
	}
	if build.GetStatus() != gen.BuildStatus_SUCCEEDED {
		return build, fmt.Errorf("NeoShowcase build %q finished with status %s", build.GetId(), build.GetStatus())
	}
	return build, nil
}

func newestMatchingBuild(builds []*gen.Build, commit string, excluded map[string]struct{}) *gen.Build {
	var newest *gen.Build
	for _, build := range builds {
		if commit != "" && build.GetCommit() != commit {
			continue
		}
		if _, skip := excluded[build.GetId()]; skip {
			continue
		}
		if newest == nil || queuedAfter(build, newest) {
			newest = build
		}
	}
	return newest
}

func queuedAfter(candidate, current *gen.Build) bool {
	if candidate.GetQueuedAt() == nil {
		return false
	}
	if current.GetQueuedAt() == nil {
		return true
	}
	return candidate.GetQueuedAt().AsTime().After(current.GetQueuedAt().AsTime())
}

func terminalBuildStatus(status gen.BuildStatus) bool {
	switch status {
	case gen.BuildStatus_SUCCEEDED, gen.BuildStatus_FAILED, gen.BuildStatus_CANCELLED, gen.BuildStatus_SKIPPED:
		return true
	default:
		return false
	}
}

func waitForPoll(ctx context.Context, interval time.Duration) error {
	timer := time.NewTimer(interval)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}
