package application

import (
	"context"
	"fmt"
	"slices"
	"sort"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/traP-jp/terraform-provider-neoshowcase/internal/neoshowcase"
	"github.com/traP-jp/terraform-provider-neoshowcase/internal/neoshowcase/gen"
	"github.com/traP-jp/terraform-provider-neoshowcase/internal/provider/ownership"
)

var (
	_ resource.Resource                   = (*applicationResource)(nil)
	_ resource.ResourceWithConfigure      = (*applicationResource)(nil)
	_ resource.ResourceWithImportState    = (*applicationResource)(nil)
	_ resource.ResourceWithValidateConfig = (*applicationResource)(nil)
)

type applicationResource struct {
	session *neoshowcase.Session
}

func New() resource.Resource {
	return &applicationResource{}
}

func (r *applicationResource) Metadata(_ context.Context, request resource.MetadataRequest, response *resource.MetadataResponse) {
	response.TypeName = request.ProviderTypeName + "_application"
}

func (r *applicationResource) Schema(_ context.Context, _ resource.SchemaRequest, response *resource.SchemaResponse) {
	response.Schema = resourceSchema()
}

func (r *applicationResource) Configure(_ context.Context, request resource.ConfigureRequest, response *resource.ConfigureResponse) {
	if request.ProviderData == nil {
		return
	}
	session, ok := request.ProviderData.(*neoshowcase.Session)
	if !ok {
		response.Diagnostics.AddError("Unexpected resource configuration", "Expected a NeoShowcase session from the provider.")
		return
	}
	r.session = session
}

func (r *applicationResource) ValidateConfig(ctx context.Context, request resource.ValidateConfigRequest, response *resource.ValidateConfigResponse) {
	var config resourceModel
	response.Diagnostics.Append(request.Config.Get(ctx, &config)...)
	if response.Diagnostics.HasError() {
		return
	}
	build, diagnostics := config.build(ctx)
	response.Diagnostics.Append(diagnostics...)
	if !config.Build.IsNull() && !config.Build.IsUnknown() {
		if err := validateBuild(build); err != nil {
			response.Diagnostics.AddAttributeError(path.Root("build"), "Invalid application build configuration", err.Error())
		}
	}
}

func (r *applicationResource) ImportState(ctx context.Context, request resource.ImportStateRequest, response *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), request, response)
}

func (r *applicationResource) Create(ctx context.Context, request resource.CreateRequest, response *resource.CreateResponse) {
	if !r.configured() {
		response.Diagnostics.AddError("NeoShowcase client is not configured", "Configure the NeoShowcase provider before managing this resource.")
		return
	}
	var plan resourceModel
	var config resourceModel
	response.Diagnostics.Append(request.Plan.Get(ctx, &plan)...)
	response.Diagnostics.Append(request.Config.Get(ctx, &config)...)
	if response.Diagnostics.HasError() {
		return
	}

	build, diagnostics := plan.build(ctx)
	response.Diagnostics.Append(diagnostics...)
	buildRequest, err := buildToAPI(build)
	if err != nil {
		response.Diagnostics.AddAttributeError(path.Root("build"), "Invalid application build configuration", err.Error())
		return
	}
	websites, diagnostics := urlsToAPI(ctx, plan.URLs)
	response.Diagnostics.Append(diagnostics...)
	ports, diagnostics := portForwardingsToAPI(ctx, plan.PortForwardings)
	response.Diagnostics.Append(diagnostics...)
	additionalOwnerIDs, diagnostics := stringsFromSet(ctx, plan.AdditionalOwnerIDs)
	response.Diagnostics.Append(diagnostics...)
	if response.Diagnostics.HasError() {
		return
	}
	authoritativeOwnerID := r.session.CurrentUser.GetId()
	if slices.Contains(additionalOwnerIDs, authoritativeOwnerID) {
		response.Diagnostics.AddAttributeError(path.Root("additional_owner_ids"), "Authoritative owner listed as an additional owner", "The provider user is added automatically and must not be included in additional_owner_ids.")
		return
	}

	// Always create stopped so owners and environment variables are complete
	// before a requested initial start.
	created, err := r.session.Client.CreateApplication(ctx, &gen.CreateApplicationRequest{
		Name: plan.Name.ValueString(), RepositoryId: plan.RepositoryID.ValueString(), RefName: plan.RefName.ValueString(),
		Config: buildRequest, Websites: websites, PortPublications: ports, StartOnCreate: false,
	})
	if err != nil {
		response.Diagnostics.AddError("Unable to create NeoShowcase application", err.Error())
		return
	}

	initialState, initialDiagnostics := flattenApplication(ctx, created, nil, plan.EnvironmentVariables, authoritativeOwnerID)
	response.Diagnostics.Append(initialDiagnostics...)
	response.Diagnostics.Append(response.State.Set(ctx, &initialState)...)
	if response.Diagnostics.HasError() {
		return
	}

	desiredOwners := ownership.Effective(authoritativeOwnerID, additionalOwnerIDs)
	actualOwners := ownership.Effective("", created.GetOwnerIds())
	if !slices.Equal(desiredOwners, actualOwners) {
		if err := r.session.Client.UpdateApplication(ctx, &gen.UpdateApplicationRequest{
			Id: created.GetId(), OwnerIds: &gen.UpdateApplicationRequest_UpdateOwners{OwnerIds: desiredOwners},
		}); err != nil {
			response.Diagnostics.AddError("Application created but owner update failed", err.Error())
			return
		}
	}

	planVariables, planDiagnostics := environmentVariablesFromMap(ctx, plan.EnvironmentVariables)
	configVariables, configDiagnostics := environmentVariablesFromMap(ctx, config.EnvironmentVariables)
	response.Diagnostics.Append(planDiagnostics...)
	response.Diagnostics.Append(configDiagnostics...)
	if response.Diagnostics.HasError() {
		return
	}
	r.reconcileEnvironmentVariables(ctx, created.GetId(), planVariables, configVariables, nil, &response.Diagnostics)
	if response.Diagnostics.HasError() {
		return
	}
	if plan.Running.ValueBool() {
		if err := r.session.Client.StartApplication(ctx, created.GetId()); err != nil {
			response.Diagnostics.AddError("Application created but could not be started", err.Error())
			return
		}
	}
	r.readAfterWrite(ctx, created.GetId(), plan.EnvironmentVariables, &response.State, &response.Diagnostics)
}

func (r *applicationResource) Read(ctx context.Context, request resource.ReadRequest, response *resource.ReadResponse) {
	if !r.configured() {
		response.Diagnostics.AddError("NeoShowcase client is not configured", "Configure the NeoShowcase provider before managing this resource.")
		return
	}
	var state resourceModel
	response.Diagnostics.Append(request.State.Get(ctx, &state)...)
	if response.Diagnostics.HasError() {
		return
	}
	application, err := r.session.Client.GetApplication(ctx, state.ID.ValueString())
	if neoshowcase.IsNotFound(err) {
		response.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		response.Diagnostics.AddError("Unable to read NeoShowcase application", err.Error())
		return
	}
	authoritativeOwnerID := r.session.CurrentUser.GetId()
	if err := ownership.ValidateAuthoritativeOwner(authoritativeOwnerID, application.GetOwnerIds()); err != nil {
		response.Diagnostics.AddError("Provider user is not the authoritative application owner", err.Error())
		return
	}
	environments, err := r.session.Client.GetApplicationEnvironmentVariables(ctx, application.GetId())
	if err != nil {
		response.Diagnostics.AddError("Unable to read NeoShowcase application environment variables", err.Error())
		return
	}
	refreshed, diagnostics := flattenApplication(ctx, application, environments, state.EnvironmentVariables, authoritativeOwnerID)
	response.Diagnostics.Append(diagnostics...)
	if !response.Diagnostics.HasError() {
		response.Diagnostics.Append(response.State.Set(ctx, &refreshed)...)
	}
}

func (r *applicationResource) Update(ctx context.Context, request resource.UpdateRequest, response *resource.UpdateResponse) {
	if !r.configured() {
		response.Diagnostics.AddError("NeoShowcase client is not configured", "Configure the NeoShowcase provider before managing this resource.")
		return
	}
	var plan resourceModel
	var state resourceModel
	var config resourceModel
	response.Diagnostics.Append(request.Plan.Get(ctx, &plan)...)
	response.Diagnostics.Append(request.State.Get(ctx, &state)...)
	response.Diagnostics.Append(request.Config.Get(ctx, &config)...)
	if response.Diagnostics.HasError() {
		return
	}

	authoritativeOwnerID := r.session.CurrentUser.GetId()
	effectiveOwnerIDs, diagnostics := stringsFromSet(ctx, state.EffectiveOwnerIDs)
	response.Diagnostics.Append(diagnostics...)
	if err := ownership.ValidateAuthoritativeOwner(authoritativeOwnerID, effectiveOwnerIDs); err != nil {
		response.Diagnostics.AddError("Provider user is not the authoritative application owner", err.Error())
		return
	}
	additionalOwnerIDs, diagnostics := stringsFromSet(ctx, plan.AdditionalOwnerIDs)
	response.Diagnostics.Append(diagnostics...)
	if slices.Contains(additionalOwnerIDs, authoritativeOwnerID) {
		response.Diagnostics.AddAttributeError(path.Root("additional_owner_ids"), "Authoritative owner listed as an additional owner", "The provider user is added automatically and must not be included in additional_owner_ids.")
		return
	}

	update := &gen.UpdateApplicationRequest{Id: state.ID.ValueString()}
	changed := false
	if !plan.Name.Equal(state.Name) {
		value := plan.Name.ValueString()
		update.Name = &value
		changed = true
	}
	if !plan.RefName.Equal(state.RefName) {
		value := plan.RefName.ValueString()
		update.RefName = &value
		changed = true
	}
	if !plan.Build.Equal(state.Build) {
		build, buildDiagnostics := plan.build(ctx)
		response.Diagnostics.Append(buildDiagnostics...)
		value, err := buildToAPI(build)
		if err != nil {
			response.Diagnostics.AddAttributeError(path.Root("build"), "Invalid application build configuration", err.Error())
			return
		}
		update.Config = value
		changed = true
	}
	if !plan.URLs.Equal(state.URLs) {
		value, valueDiagnostics := urlsToAPI(ctx, plan.URLs)
		response.Diagnostics.Append(valueDiagnostics...)
		update.Websites = &gen.UpdateApplicationRequest_UpdateWebsites{Websites: value}
		changed = true
	}
	if !plan.PortForwardings.Equal(state.PortForwardings) {
		value, valueDiagnostics := portForwardingsToAPI(ctx, plan.PortForwardings)
		response.Diagnostics.Append(valueDiagnostics...)
		update.PortPublications = &gen.UpdateApplicationRequest_UpdatePorts{PortPublications: value}
		changed = true
	}
	if !plan.AdditionalOwnerIDs.Equal(state.AdditionalOwnerIDs) || state.AuthoritativeOwnerID.ValueString() != authoritativeOwnerID {
		update.OwnerIds = &gen.UpdateApplicationRequest_UpdateOwners{OwnerIds: ownership.Effective(authoritativeOwnerID, additionalOwnerIDs)}
		changed = true
	}

	planVariables, planDiagnostics := environmentVariablesFromMap(ctx, plan.EnvironmentVariables)
	configVariables, configDiagnostics := environmentVariablesFromMap(ctx, config.EnvironmentVariables)
	stateVariables, stateDiagnostics := environmentVariablesFromMap(ctx, state.EnvironmentVariables)
	response.Diagnostics.Append(planDiagnostics...)
	response.Diagnostics.Append(configDiagnostics...)
	response.Diagnostics.Append(stateDiagnostics...)
	if response.Diagnostics.HasError() {
		return
	}

	// Reconcile environment variables before updating the application. An
	// application update schedules an asynchronous repository fetch and build;
	// writing variables first ensures that build observes the complete desired
	// environment when both change in the same apply.
	r.reconcileEnvironmentVariables(ctx, state.ID.ValueString(), planVariables, configVariables, stateVariables, &response.Diagnostics)
	if response.Diagnostics.HasError() {
		return
	}

	if changed {
		if err := r.session.Client.UpdateApplication(ctx, update); err != nil {
			response.Diagnostics.AddError("Unable to update NeoShowcase application", err.Error())
			return
		}
	}

	if !plan.Running.Equal(state.Running) {
		var err error
		if plan.Running.ValueBool() {
			err = r.session.Client.StartApplication(ctx, state.ID.ValueString())
		} else {
			err = r.session.Client.StopApplication(ctx, state.ID.ValueString())
		}
		if err != nil {
			response.Diagnostics.AddError("Unable to change NeoShowcase application running state", err.Error())
			return
		}
	}
	r.readAfterWrite(ctx, state.ID.ValueString(), plan.EnvironmentVariables, &response.State, &response.Diagnostics)
}

func (r *applicationResource) Delete(ctx context.Context, request resource.DeleteRequest, response *resource.DeleteResponse) {
	if !r.configured() {
		response.Diagnostics.AddError("NeoShowcase client is not configured", "Configure the NeoShowcase provider before managing this resource.")
		return
	}
	var state resourceModel
	response.Diagnostics.Append(request.State.Get(ctx, &state)...)
	if response.Diagnostics.HasError() {
		return
	}
	application, err := r.session.Client.GetApplication(ctx, state.ID.ValueString())
	if neoshowcase.IsNotFound(err) {
		return
	}
	if err != nil {
		response.Diagnostics.AddError("Unable to read NeoShowcase application before deletion", err.Error())
		return
	}
	if application.GetRunning() {
		if err := r.session.Client.StopApplication(ctx, application.GetId()); err != nil && !neoshowcase.IsNotFound(err) {
			response.Diagnostics.AddError("Unable to stop NeoShowcase application before deletion", err.Error())
			return
		}
	}
	if err := r.session.Client.DeleteApplication(ctx, application.GetId()); err != nil && !neoshowcase.IsNotFound(err) {
		response.Diagnostics.AddError("Unable to delete NeoShowcase application", err.Error())
	}
}

func (r *applicationResource) reconcileEnvironmentVariables(ctx context.Context, applicationID string, desired, configured, prior map[string]environmentVariableModel, diagnostics *diag.Diagnostics) {
	keys := make([]string, 0, len(prior)+len(desired))
	seen := make(map[string]struct{}, len(prior)+len(desired))
	for key := range prior {
		seen[key] = struct{}{}
		keys = append(keys, key)
	}
	for key := range desired {
		if _, ok := seen[key]; !ok {
			keys = append(keys, key)
		}
	}
	sort.Strings(keys)
	for _, key := range keys {
		desiredVariable, wanted := desired[key]
		priorVariable, existed := prior[key]
		if !wanted {
			if err := r.session.Client.DeleteApplicationEnvironmentVariable(ctx, applicationID, key); err != nil {
				diagnostics.AddError("Unable to delete NeoShowcase application environment variable", err.Error())
				return
			}
			continue
		}
		if existed && desiredVariable.Version.Equal(priorVariable.Version) {
			continue
		}
		configuredVariable, ok := configured[key]
		if !ok || configuredVariable.Value.IsNull() || configuredVariable.Value.IsUnknown() {
			diagnostics.AddAttributeError(path.Root("environment_variables").AtMapKey(key).AtName("value_wo"), "Missing environment variable value", fmt.Sprintf("Set value_wo to create or update environment variable %q.", key))
			return
		}
		if err := r.session.Client.SetApplicationEnvironmentVariable(ctx, applicationID, key, configuredVariable.Value.ValueString()); err != nil {
			diagnostics.AddError("Unable to set NeoShowcase application environment variable", err.Error())
			return
		}
	}
}

func (r *applicationResource) readAfterWrite(ctx context.Context, applicationID string, priorEnvironmentVariables types.Map, state *tfsdk.State, diagnostics *diag.Diagnostics) {
	application, err := r.session.Client.GetApplication(ctx, applicationID)
	if err != nil {
		diagnostics.AddError("Application was changed but could not be read", err.Error())
		return
	}
	if err := ownership.ValidateAuthoritativeOwner(r.session.CurrentUser.GetId(), application.GetOwnerIds()); err != nil {
		diagnostics.AddError("Application was changed but the provider user is no longer an owner", err.Error())
		return
	}
	environments, err := r.session.Client.GetApplicationEnvironmentVariables(ctx, applicationID)
	if err != nil {
		diagnostics.AddError("Application was changed but its environment variables could not be read", err.Error())
		return
	}
	refreshed, flattenDiagnostics := flattenApplication(ctx, application, environments, priorEnvironmentVariables, r.session.CurrentUser.GetId())
	diagnostics.Append(flattenDiagnostics...)
	if !diagnostics.HasError() {
		diagnostics.Append(state.Set(ctx, &refreshed)...)
	}
}

func (r *applicationResource) configured() bool {
	return r.session != nil && r.session.Client != nil && r.session.CurrentUser != nil
}
