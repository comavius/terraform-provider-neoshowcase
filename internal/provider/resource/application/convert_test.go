package application

import (
	"context"
	"reflect"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"

	"github.com/traP-jp/terraform-provider-neoshowcase/internal/neoshowcase/gen"
)

func TestBuildRoundTrip(t *testing.T) {
	t.Parallel()

	tests := map[string]buildModel{
		buildTypeRuntimeBuildpack: withBuildType(buildTypeRuntimeBuildpack, func(value *buildModel) {
			value.Context = types.StringValue("service")
			value.Entrypoint = types.StringValue("/app/server")
			value.AutoShutdownEnabled = types.BoolValue(true)
			value.AutoShutdownStartup = types.StringValue(startupBehaviorBlocking)
		}),
		buildTypeRuntimeCommand: withBuildType(buildTypeRuntimeCommand, func(value *buildModel) {
			value.BaseImage = types.StringValue("alpine:3")
			value.BuildCommand = types.StringValue("go build ./...")
			value.Command = types.StringValue("./server")
		}),
		buildTypeRuntimeDockerfile: withBuildType(buildTypeRuntimeDockerfile, func(value *buildModel) {
			value.DockerfileName = types.StringValue("Dockerfile.runtime")
			value.Context = types.StringValue(".")
			value.UseMariaDB = types.BoolValue(true)
		}),
		buildTypeStaticBuildpack: withBuildType(buildTypeStaticBuildpack, func(value *buildModel) {
			value.ArtifactPath = types.StringValue("dist")
			value.Context = types.StringValue("web")
			value.SPA = types.BoolValue(true)
		}),
		buildTypeStaticCommand: withBuildType(buildTypeStaticCommand, func(value *buildModel) {
			value.ArtifactPath = types.StringValue("public")
			value.BaseImage = types.StringValue("node:24")
			value.BuildCommand = types.StringValue("npm run build")
		}),
		buildTypeStaticDockerfile: withBuildType(buildTypeStaticDockerfile, func(value *buildModel) {
			value.ArtifactPath = types.StringValue("output")
			value.DockerfileName = types.StringValue("Dockerfile.static")
			value.Context = types.StringValue("frontend")
		}),
	}

	for name, original := range tests {
		original := original
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			apiValue, err := buildToAPI(original)
			if err != nil {
				t.Fatalf("buildToAPI() error = %v", err)
			}
			flattened, diagnostics := buildFromAPI(apiValue)
			if diagnostics.HasError() {
				t.Fatalf("buildFromAPI() diagnostics = %v", diagnostics)
			}
			var roundTrip buildModel
			diagnostics = flattened.As(context.Background(), &roundTrip, basetypes.ObjectAsOptions{})
			if diagnostics.HasError() {
				t.Fatalf("Object.As() diagnostics = %v", diagnostics)
			}
			if !reflect.DeepEqual(roundTrip, original) {
				t.Fatalf("round trip = %#v, want %#v", roundTrip, original)
			}
		})
	}
}

func TestValidateBuild(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		build buildModel
	}{
		{"static without artifact", withBuildType(buildTypeStaticBuildpack, nil)},
		{"dockerfile without name", withBuildType(buildTypeRuntimeDockerfile, nil)},
		{"runtime with static setting", withBuildType(buildTypeRuntimeBuildpack, func(value *buildModel) { value.SPA = types.BoolValue(true) })},
		{"static with runtime setting", withBuildType(buildTypeStaticBuildpack, func(value *buildModel) {
			value.ArtifactPath = types.StringValue("dist")
			value.Command = types.StringValue("serve")
		})},
		{"auto shutdown without startup", withBuildType(buildTypeRuntimeBuildpack, func(value *buildModel) { value.AutoShutdownEnabled = types.BoolValue(true) })},
	}
	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			if err := validateBuild(test.build); err == nil {
				t.Fatal("validateBuild() error = nil")
			}
		})
	}
}

func TestFlattenEnvironmentVariables(t *testing.T) {
	t.Parallel()

	priorEntry, diagnostics := types.ObjectValue(environmentVariableAttributeTypes, map[string]attr.Value{
		"value_wo": types.StringNull(), "value_wo_version": types.Int64Value(7),
	})
	if diagnostics.HasError() {
		t.Fatal(diagnostics)
	}
	prior := types.MapValueMust(types.ObjectType{AttrTypes: environmentVariableAttributeTypes}, map[string]attr.Value{"TOKEN": priorEntry})
	variables, systemKeys, diagnostics := flattenEnvironmentVariables(context.Background(), []*gen.ApplicationEnvVar{
		{Key: "TOKEN", Value: "secret"},
		{Key: "MARIADB_PASSWORD", Value: "system-secret", System: true},
	}, prior)
	if diagnostics.HasError() {
		t.Fatalf("flattenEnvironmentVariables() diagnostics = %v", diagnostics)
	}
	var got map[string]environmentVariableModel
	diagnostics = variables.ElementsAs(context.Background(), &got, false)
	if diagnostics.HasError() {
		t.Fatal(diagnostics)
	}
	if len(got) != 1 || got["TOKEN"].Version.ValueInt64() != 7 || !got["TOKEN"].Value.IsNull() {
		t.Fatalf("user variables = %#v", got)
	}
	var keys []string
	diagnostics = systemKeys.ElementsAs(context.Background(), &keys, false)
	if diagnostics.HasError() || !reflect.DeepEqual(keys, []string{"MARIADB_PASSWORD"}) {
		t.Fatalf("system keys = %#v, diagnostics = %v", keys, diagnostics)
	}
}

func defaultBuild() buildModel {
	return buildModel{
		UseMariaDB: types.BoolValue(false), UseMongoDB: types.BoolValue(false),
		Entrypoint: types.StringValue(""), Command: types.StringValue(""),
		AutoShutdownEnabled: types.BoolValue(false), AutoShutdownStartup: types.StringValue(startupBehaviorUndefined),
		ArtifactPath: types.StringValue(""), SPA: types.BoolValue(false), Context: types.StringValue(""),
		BaseImage: types.StringValue(""), BuildCommand: types.StringValue(""), DockerfileName: types.StringValue(""),
	}
}

func withBuildType(typeName string, modify func(*buildModel)) buildModel {
	value := defaultBuild()
	value.Type = types.StringValue(typeName)
	if modify != nil {
		modify(&value)
	}
	return value
}
