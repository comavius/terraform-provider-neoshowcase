package currentuser

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/traP-jp/terraform-provider-neoshowcase/internal/neoshowcase"
)

var (
	_ datasource.DataSource              = (*currentUserDataSource)(nil)
	_ datasource.DataSourceWithConfigure = (*currentUserDataSource)(nil)
)

type currentUserDataSource struct {
	session *neoshowcase.Session
}

type currentUserModel struct {
	ID        types.String `tfsdk:"id"`
	Name      types.String `tfsdk:"name"`
	Admin     types.Bool   `tfsdk:"admin"`
	AvatarURL types.String `tfsdk:"avatar_url"`
}

func New() datasource.DataSource {
	return &currentUserDataSource{}
}

func (d *currentUserDataSource) Metadata(_ context.Context, request datasource.MetadataRequest, response *datasource.MetadataResponse) {
	response.TypeName = request.ProviderTypeName + "_current_user"
}

func (d *currentUserDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, response *datasource.SchemaResponse) {
	response.Schema = schema.Schema{
		MarkdownDescription: "Returns the user configured for this NeoShowcase provider instance.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "NeoShowcase user ID.",
			},
			"name": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "NeoShowcase user name.",
			},
			"admin": schema.BoolAttribute{
				Computed:            true,
				MarkdownDescription: "Whether the user is an administrator.",
			},
			"avatar_url": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "User avatar URL.",
			},
		},
	}
}

func (d *currentUserDataSource) Configure(_ context.Context, request datasource.ConfigureRequest, response *datasource.ConfigureResponse) {
	if request.ProviderData == nil {
		return
	}

	session, ok := request.ProviderData.(*neoshowcase.Session)
	if !ok {
		response.Diagnostics.AddError("Unexpected data source configuration", "Expected a NeoShowcase session from the provider.")
		return
	}
	d.session = session
}

func (d *currentUserDataSource) Read(ctx context.Context, _ datasource.ReadRequest, response *datasource.ReadResponse) {
	if d.session == nil || d.session.CurrentUser == nil {
		response.Diagnostics.AddError("NeoShowcase client is not configured", "Configure the NeoShowcase provider before reading this data source.")
		return
	}

	user := d.session.CurrentUser
	state := currentUserModel{
		ID:        types.StringValue(user.GetId()),
		Name:      types.StringValue(user.GetName()),
		Admin:     types.BoolValue(user.GetAdmin()),
		AvatarURL: types.StringValue(user.GetAvatarUrl()),
	}
	response.Diagnostics.Append(response.State.Set(ctx, &state)...)
}
