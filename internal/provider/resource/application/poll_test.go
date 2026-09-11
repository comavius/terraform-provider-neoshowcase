package application

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/types"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/traP-jp/terraform-provider-neoshowcase/internal/neoshowcase/gen"
)

type pollingBuildClient struct {
	buildLists [][]*gen.Build
	builds     []*gen.Build
	listCalls  int
	getCalls   int
}

func (c *pollingBuildClient) GetApplicationBuilds(context.Context, string) ([]*gen.Build, error) {
	index := min(c.listCalls, len(c.buildLists)-1)
	c.listCalls++
	return c.buildLists[index], nil
}

func (c *pollingBuildClient) GetBuild(context.Context, string) (*gen.Build, error) {
	index := min(c.getCalls, len(c.builds)-1)
	c.getCalls++
	return c.builds[index], nil
}

func TestWaitForNewBuildPollsUntilSuccess(t *testing.T) {
	t.Parallel()

	oldBuild := &gen.Build{Id: "old", Commit: "commit", Status: gen.BuildStatus_SUCCEEDED, QueuedAt: timestamppb.New(time.Unix(1, 0))}
	queuedBuild := &gen.Build{Id: "new", Commit: "commit", Status: gen.BuildStatus_QUEUED, QueuedAt: timestamppb.New(time.Unix(2, 0))}
	succeededBuild := &gen.Build{Id: "new", Commit: "commit", Status: gen.BuildStatus_SUCCEEDED, QueuedAt: queuedBuild.GetQueuedAt()}
	client := &pollingBuildClient{
		buildLists: [][]*gen.Build{{oldBuild}, {oldBuild, queuedBuild}},
		builds:     []*gen.Build{succeededBuild},
	}

	final, err := waitForNewBuildAtInterval(context.Background(), client, "application", "commit", map[string]struct{}{"old": {}}, time.Millisecond)
	if err != nil {
		t.Fatalf("waitForNewBuildAtInterval() error = %v", err)
	}
	if final.GetId() != "new" || final.GetStatus() != gen.BuildStatus_SUCCEEDED {
		t.Fatalf("waitForNewBuildAtInterval() = %#v", final)
	}
	if client.listCalls != 2 || client.getCalls != 1 {
		t.Fatalf("poll calls = (%d lists, %d gets), want (2, 1)", client.listCalls, client.getCalls)
	}
}

func TestWaitForBuildRejectsUnsuccessfulTerminalStatus(t *testing.T) {
	t.Parallel()

	for _, status := range []gen.BuildStatus{gen.BuildStatus_FAILED, gen.BuildStatus_CANCELLED, gen.BuildStatus_SKIPPED} {
		t.Run(status.String(), func(t *testing.T) {
			t.Parallel()
			client := &pollingBuildClient{}
			_, err := waitForBuildAtInterval(context.Background(), client, &gen.Build{Id: "build", Status: status}, time.Millisecond)
			if err == nil {
				t.Fatalf("waitForBuildAtInterval() error = nil for %s", status)
			}
		})
	}
}

func TestWaitForNewBuildHonorsContext(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	client := &pollingBuildClient{buildLists: [][]*gen.Build{{}}}

	_, err := waitForNewBuildAtInterval(ctx, client, "application", "", nil, time.Hour)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("waitForNewBuildAtInterval() error = %v, want context.Canceled", err)
	}
}

func TestEnvironmentVariablesDiffer(t *testing.T) {
	t.Parallel()

	versionOne := environmentVariableModel{Version: types.Int64Value(1)}
	versionTwo := environmentVariableModel{Version: types.Int64Value(2)}
	if environmentVariablesDiffer(map[string]environmentVariableModel{"TOKEN": versionOne}, map[string]environmentVariableModel{"TOKEN": versionOne}) {
		t.Fatal("environmentVariablesDiffer() = true for equal versions")
	}
	if !environmentVariablesDiffer(map[string]environmentVariableModel{"TOKEN": versionTwo}, map[string]environmentVariableModel{"TOKEN": versionOne}) {
		t.Fatal("environmentVariablesDiffer() = false for changed version")
	}
	if !environmentVariablesDiffer(map[string]environmentVariableModel{}, map[string]environmentVariableModel{"TOKEN": versionOne}) {
		t.Fatal("environmentVariablesDiffer() = false for removed variable")
	}
}
