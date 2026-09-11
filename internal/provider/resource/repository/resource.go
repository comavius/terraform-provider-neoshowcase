package repository

import (
	"context"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"

	"github.com/traP-jp/terraform-provider-neoshowcase/internal/neoshowcase"
)

var (
	_ resource.Resource                   = (*repositoryResource)(nil)
	_ resource.ResourceWithConfigure      = (*repositoryResource)(nil)
	_ resource.ResourceWithImportState    = (*repositoryResource)(nil)
	_ resource.ResourceWithValidateConfig = (*repositoryResource)(nil)
)

type repositoryResource struct {
	session *neoshowcase.Session
}

func New() resource.Resource {
	return &repositoryResource{}
}

func (r *repositoryResource) Metadata(_ context.Context, request resource.MetadataRequest, response *resource.MetadataResponse) {
	response.TypeName = request.ProviderTypeName + "_repository"
}

func (r *repositoryResource) Schema(_ context.Context, _ resource.SchemaRequest, response *resource.SchemaResponse) {
	response.Schema = resourceSchema()
}

func (r *repositoryResource) Configure(_ context.Context, request resource.ConfigureRequest, response *resource.ConfigureResponse) {
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

func (r *repositoryResource) ValidateConfig(ctx context.Context, request resource.ValidateConfigRequest, response *resource.ValidateConfigResponse) {
	var config resourceModel
	response.Diagnostics.Append(request.Config.Get(ctx, &config)...)
	if response.Diagnostics.HasError() || config.Auth.IsNull() || config.Auth.IsUnknown() {
		return
	}

	auth, diagnostics := config.auth(ctx)
	response.Diagnostics.Append(diagnostics...)
	if response.Diagnostics.HasError() || auth.Method.IsNull() || auth.Method.IsUnknown() {
		return
	}

	switch auth.Method.ValueString() {
	case authMethodBasic:
		if auth.Username.IsNull() || auth.Username.IsUnknown() || strings.TrimSpace(auth.Username.ValueString()) == "" {
			response.Diagnostics.AddAttributeError(path.Root("auth").AtName("username"), "Missing BASIC authentication username", "Set auth.username when auth.method is basic.")
		}
	case authMethodNone, authMethodSSH:
		if !auth.Username.IsNull() && !auth.Username.IsUnknown() {
			response.Diagnostics.AddAttributeError(path.Root("auth").AtName("username"), "Unexpected authentication username", "auth.username can only be set when auth.method is basic.")
		}
		if !auth.Password.IsNull() && !auth.Password.IsUnknown() {
			response.Diagnostics.AddAttributeError(path.Root("auth").AtName("password_wo"), "Unexpected authentication password", "auth.password_wo can only be set when auth.method is basic.")
		}
	}
}

func (r *repositoryResource) ImportState(ctx context.Context, request resource.ImportStateRequest, response *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), request, response)
}

func (r *repositoryResource) configured() bool {
	return r.session != nil && r.session.Client != nil && r.session.CurrentUser != nil
}
