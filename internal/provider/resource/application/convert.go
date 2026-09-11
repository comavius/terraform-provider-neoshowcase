package application

import (
	"context"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/traP-jp/terraform-provider-neoshowcase/internal/neoshowcase/gen"
	"github.com/traP-jp/terraform-provider-neoshowcase/internal/provider/ownership"
)

func validateBuild(build buildModel) error {
	if build.Type.IsNull() || build.Type.IsUnknown() {
		return fmt.Errorf("build.type must be known")
	}
	typeName := build.Type.ValueString()
	useMariaDB := knownBool(build.UseMariaDB)
	useMongoDB := knownBool(build.UseMongoDB)
	entrypoint := knownString(build.Entrypoint, "")
	command := knownString(build.Command, "")
	autoShutdownEnabled := knownBool(build.AutoShutdownEnabled)
	autoShutdownStartup := knownString(build.AutoShutdownStartup, startupBehaviorUndefined)
	artifactPath := knownString(build.ArtifactPath, "")
	spa := knownBool(build.SPA)
	contextPath := knownString(build.Context, "")
	baseImage := knownString(build.BaseImage, "")
	buildCommand := knownString(build.BuildCommand, "")
	dockerfileName := knownString(build.DockerfileName, "")
	runtime := strings.HasPrefix(typeName, "runtime_")
	static := strings.HasPrefix(typeName, "static_")
	if !runtime && !static {
		return fmt.Errorf("unsupported build type %q", typeName)
	}

	if runtime {
		if artifactPath != "" || spa {
			return fmt.Errorf("artifact_path and spa can only be used by static build types")
		}
		if autoShutdownEnabled && autoShutdownStartup == startupBehaviorUndefined {
			return fmt.Errorf("auto_shutdown_startup must be loading_page or blocking when auto_shutdown_enabled is true")
		}
	} else {
		if useMariaDB || useMongoDB || entrypoint != "" || command != "" || autoShutdownEnabled || autoShutdownStartup != startupBehaviorUndefined {
			return fmt.Errorf("runtime settings can only be used by runtime build types")
		}
		if strings.TrimSpace(artifactPath) == "" {
			return fmt.Errorf("artifact_path is required for static build types")
		}
	}

	switch typeName {
	case buildTypeRuntimeBuildpack, buildTypeStaticBuildpack:
		if baseImage != "" || buildCommand != "" || dockerfileName != "" {
			return fmt.Errorf("base_image, build_command, and dockerfile_name cannot be used by buildpack build types")
		}
	case buildTypeRuntimeCommand, buildTypeStaticCommand:
		if contextPath != "" || dockerfileName != "" {
			return fmt.Errorf("context and dockerfile_name cannot be used by command build types")
		}
	case buildTypeRuntimeDockerfile, buildTypeStaticDockerfile:
		if strings.TrimSpace(dockerfileName) == "" {
			return fmt.Errorf("dockerfile_name is required for Dockerfile build types")
		}
		if baseImage != "" || buildCommand != "" {
			return fmt.Errorf("base_image and build_command cannot be used by Dockerfile build types")
		}
	}
	if typeName == buildTypeRuntimeCommand && baseImage == "" && entrypoint == "" && command == "" {
		return fmt.Errorf("runtime_command requires at least one of base_image, entrypoint, or command")
	}
	return nil
}

func knownString(value types.String, fallback string) string {
	if value.IsNull() || value.IsUnknown() {
		return fallback
	}
	return value.ValueString()
}

func knownBool(value types.Bool) bool {
	return !value.IsNull() && !value.IsUnknown() && value.ValueBool()
}

func buildToAPI(build buildModel) (*gen.ApplicationConfig, error) {
	if err := validateBuild(build); err != nil {
		return nil, err
	}
	runtime := &gen.RuntimeConfig{
		UseMariadb: build.UseMariaDB.ValueBool(),
		UseMongodb: build.UseMongoDB.ValueBool(),
		Entrypoint: build.Entrypoint.ValueString(),
		Command:    build.Command.ValueString(),
		AutoShutdown: &gen.AutoShutdownConfig{
			Enabled: build.AutoShutdownEnabled.ValueBool(),
			Startup: startupBehaviorToAPI(build.AutoShutdownStartup.ValueString()),
		},
	}
	static := &gen.StaticConfig{ArtifactPath: build.ArtifactPath.ValueString(), Spa: build.SPA.ValueBool()}

	config := &gen.ApplicationConfig{}
	switch build.Type.ValueString() {
	case buildTypeRuntimeBuildpack:
		config.BuildConfig = &gen.ApplicationConfig_RuntimeBuildpack{RuntimeBuildpack: &gen.BuildConfigRuntimeBuildpack{RuntimeConfig: runtime, Context: build.Context.ValueString()}}
	case buildTypeRuntimeCommand:
		config.BuildConfig = &gen.ApplicationConfig_RuntimeCmd{RuntimeCmd: &gen.BuildConfigRuntimeCmd{RuntimeConfig: runtime, BaseImage: build.BaseImage.ValueString(), BuildCmd: build.BuildCommand.ValueString()}}
	case buildTypeRuntimeDockerfile:
		config.BuildConfig = &gen.ApplicationConfig_RuntimeDockerfile{RuntimeDockerfile: &gen.BuildConfigRuntimeDockerfile{RuntimeConfig: runtime, DockerfileName: build.DockerfileName.ValueString(), Context: build.Context.ValueString()}}
	case buildTypeStaticBuildpack:
		config.BuildConfig = &gen.ApplicationConfig_StaticBuildpack{StaticBuildpack: &gen.BuildConfigStaticBuildpack{StaticConfig: static, Context: build.Context.ValueString()}}
	case buildTypeStaticCommand:
		config.BuildConfig = &gen.ApplicationConfig_StaticCmd{StaticCmd: &gen.BuildConfigStaticCmd{StaticConfig: static, BaseImage: build.BaseImage.ValueString(), BuildCmd: build.BuildCommand.ValueString()}}
	case buildTypeStaticDockerfile:
		config.BuildConfig = &gen.ApplicationConfig_StaticDockerfile{StaticDockerfile: &gen.BuildConfigStaticDockerfile{StaticConfig: static, DockerfileName: build.DockerfileName.ValueString(), Context: build.Context.ValueString()}}
	default:
		return nil, fmt.Errorf("unsupported build type %q", build.Type.ValueString())
	}
	return config, nil
}

func buildFromAPI(config *gen.ApplicationConfig) (types.Object, diag.Diagnostics) {
	var diagnostics diag.Diagnostics
	if config == nil || config.BuildConfig == nil {
		diagnostics.AddError("Invalid NeoShowcase application", "Application build configuration was empty.")
		return types.ObjectNull(buildAttributeTypes), diagnostics
	}

	build := buildModel{
		UseMariaDB:          types.BoolValue(false),
		UseMongoDB:          types.BoolValue(false),
		Entrypoint:          types.StringValue(""),
		Command:             types.StringValue(""),
		AutoShutdownEnabled: types.BoolValue(false),
		AutoShutdownStartup: types.StringValue(startupBehaviorUndefined),
		ArtifactPath:        types.StringValue(""),
		SPA:                 types.BoolValue(false),
		Context:             types.StringValue(""),
		BaseImage:           types.StringValue(""),
		BuildCommand:        types.StringValue(""),
		DockerfileName:      types.StringValue(""),
	}

	setRuntime := func(value *gen.RuntimeConfig) {
		if value == nil {
			return
		}
		build.UseMariaDB = types.BoolValue(value.GetUseMariadb())
		build.UseMongoDB = types.BoolValue(value.GetUseMongodb())
		build.Entrypoint = types.StringValue(value.GetEntrypoint())
		build.Command = types.StringValue(value.GetCommand())
		if value.GetAutoShutdown() != nil {
			build.AutoShutdownEnabled = types.BoolValue(value.GetAutoShutdown().GetEnabled())
			build.AutoShutdownStartup = types.StringValue(strings.ToLower(value.GetAutoShutdown().GetStartup().String()))
		}
	}
	setStatic := func(value *gen.StaticConfig) {
		if value == nil {
			return
		}
		build.ArtifactPath = types.StringValue(value.GetArtifactPath())
		build.SPA = types.BoolValue(value.GetSpa())
	}

	switch value := config.BuildConfig.(type) {
	case *gen.ApplicationConfig_RuntimeBuildpack:
		build.Type = types.StringValue(buildTypeRuntimeBuildpack)
		setRuntime(value.RuntimeBuildpack.GetRuntimeConfig())
		build.Context = types.StringValue(value.RuntimeBuildpack.GetContext())
	case *gen.ApplicationConfig_RuntimeCmd:
		build.Type = types.StringValue(buildTypeRuntimeCommand)
		setRuntime(value.RuntimeCmd.GetRuntimeConfig())
		build.BaseImage = types.StringValue(value.RuntimeCmd.GetBaseImage())
		build.BuildCommand = types.StringValue(value.RuntimeCmd.GetBuildCmd())
	case *gen.ApplicationConfig_RuntimeDockerfile:
		build.Type = types.StringValue(buildTypeRuntimeDockerfile)
		setRuntime(value.RuntimeDockerfile.GetRuntimeConfig())
		build.DockerfileName = types.StringValue(value.RuntimeDockerfile.GetDockerfileName())
		build.Context = types.StringValue(value.RuntimeDockerfile.GetContext())
	case *gen.ApplicationConfig_StaticBuildpack:
		build.Type = types.StringValue(buildTypeStaticBuildpack)
		setStatic(value.StaticBuildpack.GetStaticConfig())
		build.Context = types.StringValue(value.StaticBuildpack.GetContext())
	case *gen.ApplicationConfig_StaticCmd:
		build.Type = types.StringValue(buildTypeStaticCommand)
		setStatic(value.StaticCmd.GetStaticConfig())
		build.BaseImage = types.StringValue(value.StaticCmd.GetBaseImage())
		build.BuildCommand = types.StringValue(value.StaticCmd.GetBuildCmd())
	case *gen.ApplicationConfig_StaticDockerfile:
		build.Type = types.StringValue(buildTypeStaticDockerfile)
		setStatic(value.StaticDockerfile.GetStaticConfig())
		build.DockerfileName = types.StringValue(value.StaticDockerfile.GetDockerfileName())
		build.Context = types.StringValue(value.StaticDockerfile.GetContext())
	default:
		diagnostics.AddError("Unsupported NeoShowcase application build", fmt.Sprintf("Unsupported build configuration %T.", config.BuildConfig))
		return types.ObjectNull(buildAttributeTypes), diagnostics
	}

	value, valueDiagnostics := types.ObjectValueFrom(context.Background(), buildAttributeTypes, build)
	diagnostics.Append(valueDiagnostics...)
	return value, diagnostics
}

func startupBehaviorToAPI(value string) gen.AutoShutdownConfig_StartupBehavior {
	switch value {
	case startupBehaviorLoadingPage:
		return gen.AutoShutdownConfig_LOADING_PAGE
	case startupBehaviorBlocking:
		return gen.AutoShutdownConfig_BLOCKING
	default:
		return gen.AutoShutdownConfig_UNDEFINED
	}
}

func urlsToAPI(ctx context.Context, value types.Set) ([]*gen.CreateWebsiteRequest, diag.Diagnostics) {
	var diagnostics diag.Diagnostics
	if value.IsNull() || value.IsUnknown() {
		return nil, diagnostics
	}
	var urls []urlModel
	diagnostics.Append(value.ElementsAs(ctx, &urls, false)...)
	result := make([]*gen.CreateWebsiteRequest, 0, len(urls))
	for _, url := range urls {
		authentication, err := authenticationToAPI(url.Authentication.ValueString())
		if err != nil {
			diagnostics.AddError("Invalid URL authentication", err.Error())
			continue
		}
		result = append(result, &gen.CreateWebsiteRequest{
			Fqdn: url.FQDN.ValueString(), PathPrefix: url.PathPrefix.ValueString(),
			StripPrefix: url.StripPrefix.ValueBool(), Https: url.HTTPS.ValueBool(), H2C: url.H2C.ValueBool(),
			HttpPort: int32(url.HTTPPort.ValueInt64()), Authentication: authentication,
		})
	}
	return result, diagnostics
}

func urlsFromAPI(websites []*gen.Website) (types.Set, diag.Diagnostics) {
	var diagnostics diag.Diagnostics
	elements := make([]attr.Value, 0, len(websites))
	for _, website := range websites {
		value, valueDiagnostics := types.ObjectValue(urlAttributeTypes, map[string]attr.Value{
			"fqdn": types.StringValue(website.GetFqdn()), "path_prefix": types.StringValue(website.GetPathPrefix()),
			"strip_prefix": types.BoolValue(website.GetStripPrefix()), "https": types.BoolValue(website.GetHttps()),
			"h2c": types.BoolValue(website.GetH2C()), "http_port": types.Int64Value(int64(website.GetHttpPort())),
			"authentication": types.StringValue(strings.ToLower(website.GetAuthentication().String())),
		})
		diagnostics.Append(valueDiagnostics...)
		elements = append(elements, value)
	}
	value, valueDiagnostics := types.SetValue(types.ObjectType{AttrTypes: urlAttributeTypes}, elements)
	diagnostics.Append(valueDiagnostics...)
	return value, diagnostics
}

func authenticationToAPI(value string) (gen.AuthenticationType, error) {
	switch value {
	case "off":
		return gen.AuthenticationType_OFF, nil
	case "soft":
		return gen.AuthenticationType_SOFT, nil
	case "hard":
		return gen.AuthenticationType_HARD, nil
	default:
		return gen.AuthenticationType_OFF, fmt.Errorf("unsupported authentication type %q", value)
	}
}

func portForwardingsToAPI(ctx context.Context, value types.Set) ([]*gen.PortPublication, diag.Diagnostics) {
	var diagnostics diag.Diagnostics
	if value.IsNull() || value.IsUnknown() {
		return nil, diagnostics
	}
	var portForwardings []portForwardingModel
	diagnostics.Append(value.ElementsAs(ctx, &portForwardings, false)...)
	result := make([]*gen.PortPublication, 0, len(portForwardings))
	for _, port := range portForwardings {
		protocol := gen.PortPublicationProtocol_TCP
		if port.Protocol.ValueString() == "udp" {
			protocol = gen.PortPublicationProtocol_UDP
		}
		result = append(result, &gen.PortPublication{InternetPort: int32(port.InternetPort.ValueInt64()), ApplicationPort: int32(port.ApplicationPort.ValueInt64()), Protocol: protocol})
	}
	return result, diagnostics
}

func portForwardingsFromAPI(ports []*gen.PortPublication) (types.Set, diag.Diagnostics) {
	var diagnostics diag.Diagnostics
	elements := make([]attr.Value, 0, len(ports))
	for _, port := range ports {
		value, valueDiagnostics := types.ObjectValue(portForwardingAttributeTypes, map[string]attr.Value{
			"internet_port": types.Int64Value(int64(port.GetInternetPort())), "application_port": types.Int64Value(int64(port.GetApplicationPort())),
			"protocol": types.StringValue(strings.ToLower(port.GetProtocol().String())),
		})
		diagnostics.Append(valueDiagnostics...)
		elements = append(elements, value)
	}
	value, valueDiagnostics := types.SetValue(types.ObjectType{AttrTypes: portForwardingAttributeTypes}, elements)
	diagnostics.Append(valueDiagnostics...)
	return value, diagnostics
}

func flattenApplication(ctx context.Context, application *gen.Application, environments []*gen.ApplicationEnvVar, priorEnvironmentVariables types.Map, authoritativeOwnerID string) (resourceModel, diag.Diagnostics) {
	var diagnostics diag.Diagnostics
	if application == nil {
		diagnostics.AddError("Invalid NeoShowcase response", "Application response was empty.")
		return resourceModel{}, diagnostics
	}
	build, buildDiagnostics := buildFromAPI(application.GetConfig())
	diagnostics.Append(buildDiagnostics...)
	urls, urlDiagnostics := urlsFromAPI(application.GetWebsites())
	diagnostics.Append(urlDiagnostics...)
	portForwardings, portForwardingDiagnostics := portForwardingsFromAPI(application.GetPortPublications())
	diagnostics.Append(portForwardingDiagnostics...)
	variables, systemKeys, environmentDiagnostics := flattenEnvironmentVariables(ctx, environments, priorEnvironmentVariables)
	diagnostics.Append(environmentDiagnostics...)
	additionalOwners, additionalDiagnostics := stringSet(ownership.Additional(authoritativeOwnerID, application.GetOwnerIds()))
	diagnostics.Append(additionalDiagnostics...)
	effectiveOwners, effectiveDiagnostics := stringSet(application.GetOwnerIds())
	diagnostics.Append(effectiveDiagnostics...)

	latestBuildStatus := types.StringNull()
	if application.LatestBuildStatus != nil {
		latestBuildStatus = types.StringValue(strings.ToLower(application.GetLatestBuildStatus().String()))
	}
	return resourceModel{
		ID: types.StringValue(application.GetId()), Name: types.StringValue(application.GetName()),
		RepositoryID: types.StringValue(application.GetRepositoryId()), RefName: types.StringValue(application.GetRefName()), Build: build,
		URLs: urls, PortForwardings: portForwardings, Running: types.BoolValue(application.GetRunning()),
		EnvironmentVariables: variables, SystemEnvironmentVariableKeys: systemKeys,
		AuthoritativeOwnerID: types.StringValue(authoritativeOwnerID), AdditionalOwnerIDs: additionalOwners, EffectiveOwnerIDs: effectiveOwners,
		Commit: types.StringValue(application.GetCommit()), DeployType: types.StringValue(strings.ToLower(application.GetDeployType().String())),
		ContainerState: types.StringValue(strings.ToLower(application.GetContainer().String())), ContainerMessage: types.StringValue(application.GetContainerMessage()),
		CurrentBuildID: types.StringValue(application.GetCurrentBuild()), LatestBuildStatus: latestBuildStatus,
		CreatedAt: timestampValue(application.GetCreatedAt()), UpdatedAt: timestampValue(application.GetUpdatedAt()),
	}, diagnostics
}

func flattenEnvironmentVariables(ctx context.Context, environments []*gen.ApplicationEnvVar, prior types.Map) (types.Map, types.Set, diag.Diagnostics) {
	var diagnostics diag.Diagnostics
	priorVariables, priorDiagnostics := environmentVariablesFromMap(ctx, prior)
	diagnostics.Append(priorDiagnostics...)
	userVariables := make(map[string]attr.Value)
	systemKeys := make([]string, 0)
	for _, environment := range environments {
		if environment.GetSystem() {
			systemKeys = append(systemKeys, environment.GetKey())
			continue
		}
		version := types.Int64Value(0)
		if previous, ok := priorVariables[environment.GetKey()]; ok && !previous.Version.IsNull() && !previous.Version.IsUnknown() {
			version = previous.Version
		}
		value, valueDiagnostics := types.ObjectValue(environmentVariableAttributeTypes, map[string]attr.Value{
			"value_wo": types.StringNull(), "value_wo_version": version,
		})
		diagnostics.Append(valueDiagnostics...)
		userVariables[environment.GetKey()] = value
	}
	variables, variableDiagnostics := types.MapValue(types.ObjectType{AttrTypes: environmentVariableAttributeTypes}, userVariables)
	diagnostics.Append(variableDiagnostics...)
	if len(userVariables) == 0 && prior.IsNull() {
		variables = types.MapNull(types.ObjectType{AttrTypes: environmentVariableAttributeTypes})
	}
	slices.Sort(systemKeys)
	systemSet, systemDiagnostics := stringSet(systemKeys)
	diagnostics.Append(systemDiagnostics...)
	return variables, systemSet, diagnostics
}

func stringSet(values []string) (types.Set, diag.Diagnostics) {
	values = slices.Clone(values)
	slices.Sort(values)
	elements := make([]attr.Value, 0, len(values))
	for _, value := range values {
		elements = append(elements, types.StringValue(value))
	}
	return types.SetValue(types.StringType, elements)
}

func stringsFromSet(ctx context.Context, value types.Set) ([]string, diag.Diagnostics) {
	if value.IsNull() || value.IsUnknown() {
		return nil, nil
	}
	var result []string
	diagnostics := value.ElementsAs(ctx, &result, false)
	slices.Sort(result)
	return result, diagnostics
}

func timestampValue(value *timestamppb.Timestamp) types.String {
	if value == nil || !value.IsValid() {
		return types.StringNull()
	}
	return types.StringValue(value.AsTime().UTC().Format(time.RFC3339Nano))
}
