package repository

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
)

const (
	authMethodNone  = "none"
	authMethodBasic = "basic"
	authMethodSSH   = "ssh"
)

var authAttributeTypes = map[string]attr.Type{
	"method":              types.StringType,
	"username":            types.StringType,
	"password_wo":         types.StringType,
	"password_wo_version": types.Int64Type,
}

type resourceModel struct {
	ID                   types.String `tfsdk:"id"`
	Name                 types.String `tfsdk:"name"`
	URL                  types.String `tfsdk:"url"`
	HTMLURL              types.String `tfsdk:"html_url"`
	Auth                 types.Object `tfsdk:"auth"`
	AuthoritativeOwnerID types.String `tfsdk:"authoritative_owner_id"`
	AdditionalOwnerIDs   types.Set    `tfsdk:"additional_owner_ids"`
	EffectiveOwnerIDs    types.Set    `tfsdk:"effective_owner_ids"`
}

type authModel struct {
	Method          types.String `tfsdk:"method"`
	Username        types.String `tfsdk:"username"`
	Password        types.String `tfsdk:"password_wo"`
	PasswordVersion types.Int64  `tfsdk:"password_wo_version"`
}

func (m resourceModel) auth(ctx context.Context) (authModel, diag.Diagnostics) {
	var auth authModel
	if m.Auth.IsNull() || m.Auth.IsUnknown() {
		return auth, nil
	}
	diagnostics := m.Auth.As(ctx, &auth, basetypes.ObjectAsOptions{})
	return auth, diagnostics
}
