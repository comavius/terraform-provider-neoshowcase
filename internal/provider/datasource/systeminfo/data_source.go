package systeminfo

import (
	"context"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/traP-jp/terraform-provider-neoshowcase/internal/neoshowcase"
	"github.com/traP-jp/terraform-provider-neoshowcase/internal/neoshowcase/gen"
)

var (
	_ datasource.DataSource              = (*systemInfoDataSource)(nil)
	_ datasource.DataSourceWithConfigure = (*systemInfoDataSource)(nil)
)

var sshAttributeTypes = map[string]attr.Type{
	"host": types.StringType,
	"port": types.Int64Type,
}

var domainAttributeTypes = map[string]attr.Type{
	"domain":          types.StringType,
	"exclude_domains": types.SetType{ElemType: types.StringType},
	"auth_available":  types.BoolType,
	"already_bound":   types.BoolType,
}

var portAttributeTypes = map[string]attr.Type{
	"start_port": types.Int64Type,
	"end_port":   types.Int64Type,
	"protocol":   types.StringType,
}

var linkAttributeTypes = map[string]attr.Type{
	"name": types.StringType,
	"url":  types.StringType,
}

type systemInfoDataSource struct {
	session *neoshowcase.Session
}

type systemInfoModel struct {
	PublicKey       types.String `tfsdk:"public_key"`
	SSH             types.Object `tfsdk:"ssh"`
	Domains         types.Set    `tfsdk:"domains"`
	Ports           types.Set    `tfsdk:"ports"`
	AdditionalLinks types.Set    `tfsdk:"additional_links"`
	Version         types.String `tfsdk:"version"`
	Revision        types.String `tfsdk:"revision"`
}

func New() datasource.DataSource {
	return &systemInfoDataSource{}
}

func (d *systemInfoDataSource) Metadata(_ context.Context, request datasource.MetadataRequest, response *datasource.MetadataResponse) {
	response.TypeName = request.ProviderTypeName + "_system_info"
}

func (d *systemInfoDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, response *datasource.SchemaResponse) {
	response.Schema = schema.Schema{
		MarkdownDescription: "Returns deployment capabilities and version information reported by NeoShowcase.",
		Attributes: map[string]schema.Attribute{
			"public_key": schema.StringAttribute{Computed: true, MarkdownDescription: "System-wide SSH deploy public key."},
			"ssh": schema.SingleNestedAttribute{
				Computed: true,
				Attributes: map[string]schema.Attribute{
					"host": schema.StringAttribute{Computed: true},
					"port": schema.Int64Attribute{Computed: true},
				},
			},
			"domains": schema.SetNestedAttribute{
				Computed: true,
				NestedObject: schema.NestedAttributeObject{Attributes: map[string]schema.Attribute{
					"domain":          schema.StringAttribute{Computed: true},
					"exclude_domains": schema.SetAttribute{Computed: true, ElementType: types.StringType},
					"auth_available":  schema.BoolAttribute{Computed: true},
					"already_bound":   schema.BoolAttribute{Computed: true},
				}},
			},
			"ports": schema.SetNestedAttribute{
				Computed: true,
				NestedObject: schema.NestedAttributeObject{Attributes: map[string]schema.Attribute{
					"start_port": schema.Int64Attribute{Computed: true},
					"end_port":   schema.Int64Attribute{Computed: true},
					"protocol":   schema.StringAttribute{Computed: true},
				}},
			},
			"additional_links": schema.SetNestedAttribute{
				Computed: true,
				NestedObject: schema.NestedAttributeObject{Attributes: map[string]schema.Attribute{
					"name": schema.StringAttribute{Computed: true},
					"url":  schema.StringAttribute{Computed: true},
				}},
			},
			"version":  schema.StringAttribute{Computed: true},
			"revision": schema.StringAttribute{Computed: true},
		},
	}
}

func (d *systemInfoDataSource) Configure(_ context.Context, request datasource.ConfigureRequest, response *datasource.ConfigureResponse) {
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

func (d *systemInfoDataSource) Read(ctx context.Context, _ datasource.ReadRequest, response *datasource.ReadResponse) {
	if d.session == nil || d.session.Client == nil {
		response.Diagnostics.AddError("NeoShowcase client is not configured", "Configure the NeoShowcase provider before reading this data source.")
		return
	}

	info, err := d.session.Client.GetSystemInfo(ctx)
	if err != nil {
		response.Diagnostics.AddError("Unable to read NeoShowcase system information", err.Error())
		return
	}

	state, diagnostics := flattenSystemInfo(info)
	response.Diagnostics.Append(diagnostics...)
	if response.Diagnostics.HasError() {
		return
	}
	response.Diagnostics.Append(response.State.Set(ctx, &state)...)
}

func flattenSystemInfo(info *gen.SystemInfo) (systemInfoModel, diag.Diagnostics) {
	var diagnostics diag.Diagnostics
	if info == nil {
		diagnostics.AddError("Invalid NeoShowcase response", "GetSystemInfo returned an empty response.")
		return systemInfoModel{}, diagnostics
	}

	ssh := types.ObjectNull(sshAttributeTypes)
	if info.GetSsh() != nil {
		var sshDiagnostics diag.Diagnostics
		ssh, sshDiagnostics = types.ObjectValue(sshAttributeTypes, map[string]attr.Value{
			"host": types.StringValue(info.GetSsh().GetHost()),
			"port": types.Int64Value(int64(info.GetSsh().GetPort())),
		})
		diagnostics.Append(sshDiagnostics...)
	}

	domains := make([]attr.Value, 0, len(info.GetDomains()))
	for _, domain := range info.GetDomains() {
		excluded, excludedDiagnostics := stringSet(domain.GetExcludeDomains())
		diagnostics.Append(excludedDiagnostics...)
		value, valueDiagnostics := types.ObjectValue(domainAttributeTypes, map[string]attr.Value{
			"domain":          types.StringValue(domain.GetDomain()),
			"exclude_domains": excluded,
			"auth_available":  types.BoolValue(domain.GetAuthAvailable()),
			"already_bound":   types.BoolValue(domain.GetAlreadyBound()),
		})
		diagnostics.Append(valueDiagnostics...)
		domains = append(domains, value)
	}
	domainSet, domainDiagnostics := types.SetValue(types.ObjectType{AttrTypes: domainAttributeTypes}, domains)
	diagnostics.Append(domainDiagnostics...)

	ports := make([]attr.Value, 0, len(info.GetPorts()))
	for _, port := range info.GetPorts() {
		value, valueDiagnostics := types.ObjectValue(portAttributeTypes, map[string]attr.Value{
			"start_port": types.Int64Value(int64(port.GetStartPort())),
			"end_port":   types.Int64Value(int64(port.GetEndPort())),
			"protocol":   types.StringValue(protocolName(port.GetProtocol())),
		})
		diagnostics.Append(valueDiagnostics...)
		ports = append(ports, value)
	}
	portSet, portDiagnostics := types.SetValue(types.ObjectType{AttrTypes: portAttributeTypes}, ports)
	diagnostics.Append(portDiagnostics...)

	links := make([]attr.Value, 0, len(info.GetAdditionalLinks()))
	for _, link := range info.GetAdditionalLinks() {
		value, valueDiagnostics := types.ObjectValue(linkAttributeTypes, map[string]attr.Value{
			"name": types.StringValue(link.GetName()),
			"url":  types.StringValue(link.GetUrl()),
		})
		diagnostics.Append(valueDiagnostics...)
		links = append(links, value)
	}
	linkSet, linkDiagnostics := types.SetValue(types.ObjectType{AttrTypes: linkAttributeTypes}, links)
	diagnostics.Append(linkDiagnostics...)

	return systemInfoModel{
		PublicKey:       types.StringValue(info.GetPublicKey()),
		SSH:             ssh,
		Domains:         domainSet,
		Ports:           portSet,
		AdditionalLinks: linkSet,
		Version:         types.StringValue(info.GetVersion()),
		Revision:        types.StringValue(info.GetRevision()),
	}, diagnostics
}

func stringSet(values []string) (types.Set, diag.Diagnostics) {
	elements := make([]attr.Value, 0, len(values))
	for _, value := range values {
		elements = append(elements, types.StringValue(value))
	}
	return types.SetValue(types.StringType, elements)
}

func protocolName(protocol gen.PortPublicationProtocol) string {
	return strings.ToLower(protocol.String())
}
