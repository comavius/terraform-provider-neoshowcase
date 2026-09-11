package repository

import (
	"context"
	"fmt"
	"slices"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"google.golang.org/protobuf/types/known/emptypb"

	"github.com/traP-jp/terraform-provider-neoshowcase/internal/neoshowcase/gen"
	"github.com/traP-jp/terraform-provider-neoshowcase/internal/provider/ownership"
)

func createAuthRequest(auth authModel, requirePassword bool) (*gen.CreateRepositoryAuth, error) {
	if auth.Method.IsNull() || auth.Method.IsUnknown() {
		return nil, fmt.Errorf("authentication method must be known")
	}

	switch auth.Method.ValueString() {
	case authMethodNone:
		return &gen.CreateRepositoryAuth{Auth: &gen.CreateRepositoryAuth_None{None: &emptypb.Empty{}}}, nil
	case authMethodSSH:
		return &gen.CreateRepositoryAuth{Auth: &gen.CreateRepositoryAuth_Ssh{Ssh: &gen.CreateRepositoryAuthSSH{}}}, nil
	case authMethodBasic:
		if auth.Username.IsNull() || auth.Username.IsUnknown() || auth.Username.ValueString() == "" {
			return nil, fmt.Errorf("auth.username is required for BASIC authentication")
		}
		if requirePassword && (auth.Password.IsNull() || auth.Password.IsUnknown() || auth.Password.ValueString() == "") {
			return nil, fmt.Errorf("auth.password_wo is required when creating or changing BASIC authentication")
		}
		return &gen.CreateRepositoryAuth{Auth: &gen.CreateRepositoryAuth_Basic{Basic: &gen.CreateRepositoryAuthBasic{
			Username: auth.Username.ValueString(),
			Password: auth.Password.ValueString(),
		}}}, nil
	default:
		return nil, fmt.Errorf("unsupported authentication method %q", auth.Method.ValueString())
	}
}

func authMethodFromAPI(method gen.Repository_AuthMethod) (string, error) {
	switch method {
	case gen.Repository_NONE:
		return authMethodNone, nil
	case gen.Repository_BASIC:
		return authMethodBasic, nil
	case gen.Repository_SSH:
		return authMethodSSH, nil
	default:
		return "", fmt.Errorf("unsupported NeoShowcase authentication method %s", method.String())
	}
}

func flattenRepository(ctx context.Context, repository *gen.Repository, priorAuth types.Object, authoritativeOwnerID string) (resourceModel, diag.Diagnostics) {
	var diagnostics diag.Diagnostics
	if repository == nil {
		diagnostics.AddError("Invalid NeoShowcase response", "Repository response was empty.")
		return resourceModel{}, diagnostics
	}

	method, err := authMethodFromAPI(repository.GetAuthMethod())
	if err != nil {
		diagnostics.AddError("Unsupported repository authentication", err.Error())
		return resourceModel{}, diagnostics
	}

	auth := authModel{
		Method:          types.StringValue(method),
		Username:        types.StringNull(),
		Password:        types.StringNull(),
		PasswordVersion: types.Int64Value(0),
	}
	if !priorAuth.IsNull() && !priorAuth.IsUnknown() {
		var prior authModel
		diagnostics.Append(priorAuth.As(ctx, &prior, basetypes.ObjectAsOptions{})...)
		if !prior.Method.IsNull() && !prior.Method.IsUnknown() && prior.Method.ValueString() == method {
			auth.Username = prior.Username
			auth.PasswordVersion = prior.PasswordVersion
		}
	}
	authValue, authDiagnostics := types.ObjectValue(authAttributeTypes, map[string]attr.Value{
		"method":              auth.Method,
		"username":            auth.Username,
		"password_wo":         types.StringNull(),
		"password_wo_version": auth.PasswordVersion,
	})
	diagnostics.Append(authDiagnostics...)

	additionalOwnerIDs := ownership.Additional(authoritativeOwnerID, repository.GetOwnerIds())
	additionalOwners, additionalDiagnostics := stringSet(additionalOwnerIDs)
	diagnostics.Append(additionalDiagnostics...)
	effectiveOwners, effectiveDiagnostics := stringSet(repository.GetOwnerIds())
	diagnostics.Append(effectiveDiagnostics...)

	return resourceModel{
		ID:                   types.StringValue(repository.GetId()),
		Name:                 types.StringValue(repository.GetName()),
		URL:                  types.StringValue(repository.GetUrl()),
		HTMLURL:              types.StringValue(repository.GetHtmlUrl()),
		Auth:                 authValue,
		AuthoritativeOwnerID: types.StringValue(authoritativeOwnerID),
		AdditionalOwnerIDs:   additionalOwners,
		EffectiveOwnerIDs:    effectiveOwners,
	}, diagnostics
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

func stringsFromSet(ctx context.Context, set types.Set) ([]string, diag.Diagnostics) {
	if set.IsNull() || set.IsUnknown() {
		return nil, nil
	}
	var values []string
	diagnostics := set.ElementsAs(ctx, &values, false)
	slices.Sort(values)
	return values, diagnostics
}
