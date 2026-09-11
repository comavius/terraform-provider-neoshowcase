package repository

import (
	"context"
	"slices"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/traP-jp/terraform-provider-neoshowcase/internal/neoshowcase"
	"github.com/traP-jp/terraform-provider-neoshowcase/internal/neoshowcase/gen"
	"github.com/traP-jp/terraform-provider-neoshowcase/internal/provider/ownership"
)

func (r *repositoryResource) Create(ctx context.Context, request resource.CreateRequest, response *resource.CreateResponse) {
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

	auth, diagnostics := config.auth(ctx)
	response.Diagnostics.Append(diagnostics...)
	authRequest, err := createAuthRequest(auth, true)
	if err != nil {
		response.Diagnostics.AddAttributeError(path.Root("auth"), "Invalid repository authentication", err.Error())
		return
	}

	additionalOwnerIDs, diagnostics := stringsFromSet(ctx, plan.AdditionalOwnerIDs)
	response.Diagnostics.Append(diagnostics...)
	authoritativeOwnerID := r.session.CurrentUser.GetId()
	if slices.Contains(additionalOwnerIDs, authoritativeOwnerID) {
		response.Diagnostics.AddAttributeError(path.Root("additional_owner_ids"), "Authoritative owner listed as an additional owner", "The provider user is added automatically and must not be included in additional_owner_ids.")
		return
	}

	created, err := r.session.Client.CreateRepository(ctx, &gen.CreateRepositoryRequest{
		Name: plan.Name.ValueString(),
		Url:  plan.URL.ValueString(),
		Auth: authRequest,
	})
	if err != nil {
		response.Diagnostics.AddError("Unable to create NeoShowcase repository", err.Error())
		return
	}

	// Persist the server-assigned ID before the follow-up owner update so a
	// partial failure does not orphan a repository from Terraform state.
	initialState, initialDiagnostics := flattenRepository(ctx, created, plan.Auth, authoritativeOwnerID)
	response.Diagnostics.Append(initialDiagnostics...)
	response.Diagnostics.Append(response.State.Set(ctx, &initialState)...)
	if response.Diagnostics.HasError() {
		return
	}

	desiredOwners := ownership.Effective(authoritativeOwnerID, additionalOwnerIDs)
	actualOwners := ownership.Effective("", created.GetOwnerIds())
	if !slices.Equal(desiredOwners, actualOwners) {
		if err := r.session.Client.UpdateRepository(ctx, &gen.UpdateRepositoryRequest{
			Id:       created.GetId(),
			OwnerIds: &gen.UpdateRepositoryRequest_UpdateOwners{OwnerIds: desiredOwners},
		}); err != nil {
			response.Diagnostics.AddError("Repository created but owner update failed", err.Error())
			return
		}
	}

	r.readAfterWrite(ctx, created.GetId(), plan.Auth, &response.State, &response.Diagnostics)
}

func (r *repositoryResource) Read(ctx context.Context, request resource.ReadRequest, response *resource.ReadResponse) {
	if !r.configured() {
		response.Diagnostics.AddError("NeoShowcase client is not configured", "Configure the NeoShowcase provider before managing this resource.")
		return
	}

	var state resourceModel
	response.Diagnostics.Append(request.State.Get(ctx, &state)...)
	if response.Diagnostics.HasError() {
		return
	}

	repository, err := r.session.Client.GetRepository(ctx, state.ID.ValueString())
	if neoshowcase.IsNotFound(err) {
		response.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		response.Diagnostics.AddError("Unable to read NeoShowcase repository", err.Error())
		return
	}

	authoritativeOwnerID := r.session.CurrentUser.GetId()
	if err := ownership.ValidateAuthoritativeOwner(authoritativeOwnerID, repository.GetOwnerIds()); err != nil {
		response.Diagnostics.AddError("Provider user is not the authoritative repository owner", err.Error())
		return
	}

	refreshed, diagnostics := flattenRepository(ctx, repository, state.Auth, authoritativeOwnerID)
	response.Diagnostics.Append(diagnostics...)
	if response.Diagnostics.HasError() {
		return
	}
	response.Diagnostics.Append(response.State.Set(ctx, &refreshed)...)
}

func (r *repositoryResource) Update(ctx context.Context, request resource.UpdateRequest, response *resource.UpdateResponse) {
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

	planAuth, diagnostics := plan.auth(ctx)
	response.Diagnostics.Append(diagnostics...)
	stateAuth, diagnostics := state.auth(ctx)
	response.Diagnostics.Append(diagnostics...)
	configAuth, diagnostics := config.auth(ctx)
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
	effectiveOwnerIDs := effectiveOwnersFromState(ctx, state, &response.Diagnostics)
	if response.Diagnostics.HasError() {
		return
	}
	if err := ownership.ValidateAuthoritativeOwner(authoritativeOwnerID, effectiveOwnerIDs); err != nil {
		response.Diagnostics.AddError("Provider user is not the authoritative repository owner", err.Error())
		return
	}

	update := &gen.UpdateRepositoryRequest{Id: state.ID.ValueString()}
	changed := false
	if !plan.Name.Equal(state.Name) {
		value := plan.Name.ValueString()
		update.Name = &value
		changed = true
	}
	if !plan.URL.Equal(state.URL) {
		value := plan.URL.ValueString()
		update.Url = &value
		changed = true
	}
	if authChanged(planAuth, stateAuth) {
		authRequest, err := createAuthRequest(configAuth, true)
		if err != nil {
			response.Diagnostics.AddAttributeError(path.Root("auth"), "Invalid repository authentication update", err.Error())
			return
		}
		update.Auth = authRequest
		changed = true
	}
	if !plan.AdditionalOwnerIDs.Equal(state.AdditionalOwnerIDs) || state.AuthoritativeOwnerID.ValueString() != authoritativeOwnerID {
		update.OwnerIds = &gen.UpdateRepositoryRequest_UpdateOwners{
			OwnerIds: ownership.Effective(authoritativeOwnerID, additionalOwnerIDs),
		}
		changed = true
	}

	if changed {
		if err := r.session.Client.UpdateRepository(ctx, update); err != nil {
			response.Diagnostics.AddError("Unable to update NeoShowcase repository", err.Error())
			return
		}
	}

	r.readAfterWrite(ctx, state.ID.ValueString(), plan.Auth, &response.State, &response.Diagnostics)
}

func (r *repositoryResource) Delete(ctx context.Context, request resource.DeleteRequest, response *resource.DeleteResponse) {
	if !r.configured() {
		response.Diagnostics.AddError("NeoShowcase client is not configured", "Configure the NeoShowcase provider before managing this resource.")
		return
	}

	var state resourceModel
	response.Diagnostics.Append(request.State.Get(ctx, &state)...)
	if response.Diagnostics.HasError() {
		return
	}

	err := r.session.Client.DeleteRepository(ctx, state.ID.ValueString())
	if err != nil && !neoshowcase.IsNotFound(err) {
		response.Diagnostics.AddError("Unable to delete NeoShowcase repository", err.Error())
	}
}

func authChanged(plan, state authModel) bool {
	return !plan.Method.Equal(state.Method) ||
		!plan.Username.Equal(state.Username) ||
		!plan.PasswordVersion.Equal(state.PasswordVersion)
}

func (r *repositoryResource) readAfterWrite(ctx context.Context, repositoryID string, priorAuth types.Object, state *tfsdk.State, diagnostics *diag.Diagnostics) {
	repository, err := r.session.Client.GetRepository(ctx, repositoryID)
	if err != nil {
		diagnostics.AddError("Repository was changed but could not be read", err.Error())
		return
	}
	if err := ownership.ValidateAuthoritativeOwner(r.session.CurrentUser.GetId(), repository.GetOwnerIds()); err != nil {
		diagnostics.AddError("Repository was changed but the provider user is no longer an owner", err.Error())
		return
	}

	refreshed, flattenDiagnostics := flattenRepository(ctx, repository, priorAuth, r.session.CurrentUser.GetId())
	diagnostics.Append(flattenDiagnostics...)
	if diagnostics.HasError() {
		return
	}
	diagnostics.Append(state.Set(ctx, &refreshed)...)
}

func effectiveOwnersFromState(ctx context.Context, state resourceModel, diagnostics *diag.Diagnostics) []string {
	owners, ownerDiagnostics := stringsFromSet(ctx, state.EffectiveOwnerIDs)
	diagnostics.Append(ownerDiagnostics...)
	return owners
}
