package application

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
)

const (
	buildTypeRuntimeBuildpack  = "runtime_buildpack"
	buildTypeRuntimeCommand    = "runtime_command"
	buildTypeRuntimeDockerfile = "runtime_dockerfile"
	buildTypeStaticBuildpack   = "static_buildpack"
	buildTypeStaticCommand     = "static_command"
	buildTypeStaticDockerfile  = "static_dockerfile"

	startupBehaviorUndefined   = "undefined"
	startupBehaviorLoadingPage = "loading_page"
	startupBehaviorBlocking    = "blocking"
)

var buildAttributeTypes = map[string]attr.Type{
	"type":                  types.StringType,
	"use_mariadb":           types.BoolType,
	"use_mongodb":           types.BoolType,
	"entrypoint":            types.StringType,
	"command":               types.StringType,
	"auto_shutdown_enabled": types.BoolType,
	"auto_shutdown_startup": types.StringType,
	"artifact_path":         types.StringType,
	"spa":                   types.BoolType,
	"context":               types.StringType,
	"base_image":            types.StringType,
	"build_command":         types.StringType,
	"dockerfile_name":       types.StringType,
}

var urlAttributeTypes = map[string]attr.Type{
	"fqdn":           types.StringType,
	"path_prefix":    types.StringType,
	"strip_prefix":   types.BoolType,
	"https":          types.BoolType,
	"h2c":            types.BoolType,
	"http_port":      types.Int64Type,
	"authentication": types.StringType,
}

var portForwardingAttributeTypes = map[string]attr.Type{
	"internet_port":    types.Int64Type,
	"application_port": types.Int64Type,
	"protocol":         types.StringType,
}

var environmentVariableAttributeTypes = map[string]attr.Type{
	"value_wo":         types.StringType,
	"value_wo_version": types.Int64Type,
}

type resourceModel struct {
	ID                            types.String `tfsdk:"id"`
	Name                          types.String `tfsdk:"name"`
	RepositoryID                  types.String `tfsdk:"repository_id"`
	RefName                       types.String `tfsdk:"ref_name"`
	Build                         types.Object `tfsdk:"build"`
	URLs                          types.Set    `tfsdk:"urls"`
	PortForwardings               types.Set    `tfsdk:"port_forwardings"`
	Running                       types.Bool   `tfsdk:"running"`
	EnvironmentVariables          types.Map    `tfsdk:"environment_variables"`
	SystemEnvironmentVariableKeys types.Set    `tfsdk:"system_environment_variable_keys"`
	AuthoritativeOwnerID          types.String `tfsdk:"authoritative_owner_id"`
	AdditionalOwnerIDs            types.Set    `tfsdk:"additional_owner_ids"`
	EffectiveOwnerIDs             types.Set    `tfsdk:"effective_owner_ids"`
	Commit                        types.String `tfsdk:"commit"`
	DeployType                    types.String `tfsdk:"deploy_type"`
	ContainerState                types.String `tfsdk:"container_state"`
	ContainerMessage              types.String `tfsdk:"container_message"`
	CurrentBuildID                types.String `tfsdk:"current_build_id"`
	LatestBuildStatus             types.String `tfsdk:"latest_build_status"`
	CreatedAt                     types.String `tfsdk:"created_at"`
	UpdatedAt                     types.String `tfsdk:"updated_at"`
}

type buildModel struct {
	Type                types.String `tfsdk:"type"`
	UseMariaDB          types.Bool   `tfsdk:"use_mariadb"`
	UseMongoDB          types.Bool   `tfsdk:"use_mongodb"`
	Entrypoint          types.String `tfsdk:"entrypoint"`
	Command             types.String `tfsdk:"command"`
	AutoShutdownEnabled types.Bool   `tfsdk:"auto_shutdown_enabled"`
	AutoShutdownStartup types.String `tfsdk:"auto_shutdown_startup"`
	ArtifactPath        types.String `tfsdk:"artifact_path"`
	SPA                 types.Bool   `tfsdk:"spa"`
	Context             types.String `tfsdk:"context"`
	BaseImage           types.String `tfsdk:"base_image"`
	BuildCommand        types.String `tfsdk:"build_command"`
	DockerfileName      types.String `tfsdk:"dockerfile_name"`
}

type urlModel struct {
	FQDN           types.String `tfsdk:"fqdn"`
	PathPrefix     types.String `tfsdk:"path_prefix"`
	StripPrefix    types.Bool   `tfsdk:"strip_prefix"`
	HTTPS          types.Bool   `tfsdk:"https"`
	H2C            types.Bool   `tfsdk:"h2c"`
	HTTPPort       types.Int64  `tfsdk:"http_port"`
	Authentication types.String `tfsdk:"authentication"`
}

type portForwardingModel struct {
	InternetPort    types.Int64  `tfsdk:"internet_port"`
	ApplicationPort types.Int64  `tfsdk:"application_port"`
	Protocol        types.String `tfsdk:"protocol"`
}

type environmentVariableModel struct {
	Value   types.String `tfsdk:"value_wo"`
	Version types.Int64  `tfsdk:"value_wo_version"`
}

func (m resourceModel) build(ctx context.Context) (buildModel, diag.Diagnostics) {
	var value buildModel
	if m.Build.IsNull() || m.Build.IsUnknown() {
		return value, nil
	}
	diagnostics := m.Build.As(ctx, &value, basetypes.ObjectAsOptions{})
	return value, diagnostics
}

func environmentVariablesFromMap(ctx context.Context, value types.Map) (map[string]environmentVariableModel, diag.Diagnostics) {
	variables := make(map[string]environmentVariableModel)
	if value.IsNull() || value.IsUnknown() {
		return variables, nil
	}
	diagnostics := value.ElementsAs(ctx, &variables, false)
	return variables, diagnostics
}
