package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type providerModel struct {
	Endpoint           types.String `tfsdk:"endpoint"`
	User               types.String `tfsdk:"user"`
	AuthHeader         types.String `tfsdk:"auth_header"`
	AdditionalHeaders  types.Map    `tfsdk:"additional_headers"`
	InsecureSkipVerify types.Bool   `tfsdk:"insecure_skip_verify"`
}

func (p *neoShowcaseProvider) Schema(_ context.Context, _ provider.SchemaRequest, response *provider.SchemaResponse) {
	response.Schema = schema.Schema{
		MarkdownDescription: "Manages applications and related configuration in NeoShowcase.",
		Attributes: map[string]schema.Attribute{
			"endpoint": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "NeoShowcase Gateway base URL. May also be set with `NEOSHOWCASE_ENDPOINT`.",
			},
			"user": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "User name placed in the trusted proxy authentication header. May also be set with `NEOSHOWCASE_USER`.",
			},
			"auth_header": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "Trusted proxy authentication header. Defaults to `X-Showcase-User` and may also be set with `NEOSHOWCASE_AUTH_HEADER`.",
			},
			"additional_headers": schema.MapAttribute{
				Optional:            true,
				Sensitive:           true,
				ElementType:         types.StringType,
				MarkdownDescription: "Additional HTTP headers sent with every API request.",
			},
			"insecure_skip_verify": schema.BoolAttribute{
				Optional:            true,
				MarkdownDescription: "Disables TLS certificate verification. This should only be used in development environments.",
			},
		},
	}
}
