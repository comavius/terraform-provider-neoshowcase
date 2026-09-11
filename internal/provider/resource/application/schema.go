package application

import (
	"regexp"

	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/mapvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/setvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64default"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/setdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func resourceSchema() schema.Schema {
	emptyURLs := types.SetValueMust(types.ObjectType{AttrTypes: urlAttributeTypes}, []attr.Value{})
	emptyPortForwardings := types.SetValueMust(types.ObjectType{AttrTypes: portForwardingAttributeTypes}, []attr.Value{})
	emptyOwners := types.SetValueMust(types.StringType, []attr.Value{})

	return schema.Schema{
		MarkdownDescription: "Creates and manages a NeoShowcase application, including its user-defined environment variables.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{Computed: true, MarkdownDescription: "NeoShowcase application ID."},
			"name": schema.StringAttribute{
				Required: true, MarkdownDescription: "Application display name.",
				Validators: []validator.String{stringvalidator.RegexMatches(regexp.MustCompile(`\S`), "name must contain a non-whitespace character")},
			},
			"repository_id": schema.StringAttribute{
				Required: true, MarkdownDescription: "Repository used to build the application. NeoShowcase does not allow this value to be changed in place.",
				Validators:    []validator.String{stringvalidator.RegexMatches(regexp.MustCompile(`^\S+$`), "repository ID must not contain whitespace")},
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"ref_name": schema.StringAttribute{
				Required: true, MarkdownDescription: "Git ref built by NeoShowcase.",
				Validators: []validator.String{stringvalidator.RegexMatches(regexp.MustCompile(`\S`), "ref name must contain a non-whitespace character")},
			},
			"build": buildSchema(),
			"urls": schema.SetNestedAttribute{
				Optional: true, Computed: true, Default: setdefault.StaticValue(emptyURLs),
				MarkdownDescription: "Exact set of HTTP or HTTPS access URLs exposed by the application.",
				NestedObject: schema.NestedAttributeObject{Attributes: map[string]schema.Attribute{
					"fqdn": schema.StringAttribute{
						Required: true,
						Validators: []validator.String{stringvalidator.RegexMatches(
							regexp.MustCompile(`^[a-z0-9_-]+(\.[a-z0-9_-]+)*$`),
							"fqdn must contain only lowercase letters, digits, hyphens, underscores, and dots",
						)},
					},
					"path_prefix":  schema.StringAttribute{Optional: true, Computed: true, Default: stringdefault.StaticString("/")},
					"strip_prefix": schema.BoolAttribute{Optional: true, Computed: true, Default: booldefault.StaticBool(false)},
					"https":        schema.BoolAttribute{Optional: true, Computed: true, Default: booldefault.StaticBool(true)},
					"h2c":          schema.BoolAttribute{Optional: true, Computed: true, Default: booldefault.StaticBool(false)},
					"http_port": schema.Int64Attribute{
						Optional: true, Computed: true, Default: int64default.StaticInt64(80),
						Validators: []validator.Int64{int64validator.Between(1, 65535)},
					},
					"authentication": schema.StringAttribute{
						Optional: true, Computed: true, Default: stringdefault.StaticString("off"),
						Validators: []validator.String{stringvalidator.OneOf("off", "soft", "hard")},
					},
				}},
			},
			"port_forwardings": schema.SetNestedAttribute{
				Optional: true, Computed: true, Default: setdefault.StaticValue(emptyPortForwardings),
				MarkdownDescription: "Exact set of TCP or UDP port forwarding rules for the application.",
				NestedObject: schema.NestedAttributeObject{Attributes: map[string]schema.Attribute{
					"internet_port":    schema.Int64Attribute{Required: true, Validators: []validator.Int64{int64validator.Between(1, 65535)}},
					"application_port": schema.Int64Attribute{Required: true, Validators: []validator.Int64{int64validator.Between(1, 65535)}},
					"protocol": schema.StringAttribute{
						Required: true, Validators: []validator.String{stringvalidator.OneOf("tcp", "udp")},
					},
				}},
			},
			"running": schema.BoolAttribute{
				Optional: true, Computed: true, Default: booldefault.StaticBool(false),
				MarkdownDescription: "Desired application running state. The provider starts or stops the application to enforce this value.",
			},
			"environment_variables": schema.MapNestedAttribute{
				Optional: true, Sensitive: true,
				MarkdownDescription: "Exact map of user-defined environment variables. Keys beginning with `NS_` are reserved for NeoShowcase. Values are write-only and never stored in Terraform state.",
				Validators: []validator.Map{mapvalidator.KeysAre(
					stringvalidator.RegexMatches(regexp.MustCompile(`^[A-Z_][A-Z0-9_]*$`), "environment variable keys must use uppercase letters, digits, and underscores"),
					environmentVariableKeyValidator{},
				)},
				NestedObject: schema.NestedAttributeObject{Attributes: map[string]schema.Attribute{
					"value_wo": schema.StringAttribute{
						Required: true, Sensitive: true, WriteOnly: true,
						MarkdownDescription: "Write-only value. This value is never stored in Terraform state.",
					},
					"value_wo_version": schema.Int64Attribute{
						Optional: true, Computed: true, Default: int64default.StaticInt64(0),
						MarkdownDescription: "Increment this value to resend value_wo.",
					},
				}},
			},
			"system_environment_variable_keys": schema.SetAttribute{
				Computed: true, ElementType: types.StringType,
				MarkdownDescription: "Keys of environment variables generated by NeoShowcase. Their values are deliberately not stored in state.",
			},
			"authoritative_owner_id": schema.StringAttribute{Computed: true, MarkdownDescription: "ID of the provider user, which must always remain an owner."},
			"additional_owner_ids": schema.SetAttribute{
				Optional: true, Computed: true, Default: setdefault.StaticValue(emptyOwners), ElementType: types.StringType,
				MarkdownDescription: "Exact set of owners managed in addition to the provider user.",
				Validators:          []validator.Set{setvalidator.ValueStringsAre(stringvalidator.RegexMatches(regexp.MustCompile(`^\S+$`), "owner IDs must not contain whitespace"))},
			},
			"effective_owner_ids": schema.SetAttribute{Computed: true, ElementType: types.StringType, MarkdownDescription: "Actual owner set, including the authoritative provider user."},
			"commit":              schema.StringAttribute{Computed: true, MarkdownDescription: "Current application commit."},
			"deploy_type":         schema.StringAttribute{Computed: true, MarkdownDescription: "Application deployment type reported by NeoShowcase."},
			"container_state":     schema.StringAttribute{Computed: true, MarkdownDescription: "Current container state."},
			"container_message":   schema.StringAttribute{Computed: true, MarkdownDescription: "Current container status message."},
			"current_build_id":    schema.StringAttribute{Computed: true, MarkdownDescription: "Current build ID."},
			"latest_build_status": schema.StringAttribute{Computed: true, MarkdownDescription: "Latest build status, when available."},
			"created_at":          schema.StringAttribute{Computed: true, MarkdownDescription: "Creation time in RFC 3339 format."},
			"updated_at":          schema.StringAttribute{Computed: true, MarkdownDescription: "Last update time in RFC 3339 format."},
		},
	}
}

func buildSchema() schema.SingleNestedAttribute {
	return schema.SingleNestedAttribute{
		Required:            true,
		MarkdownDescription: "Build and deployment configuration. Fields not used by the selected type must keep their defaults.",
		Attributes: map[string]schema.Attribute{
			"type": schema.StringAttribute{
				Required: true,
				Validators: []validator.String{stringvalidator.OneOf(
					buildTypeRuntimeBuildpack, buildTypeRuntimeCommand, buildTypeRuntimeDockerfile,
					buildTypeStaticBuildpack, buildTypeStaticCommand, buildTypeStaticDockerfile,
				)},
			},
			"use_mariadb": schema.BoolAttribute{
				Optional: true, Computed: true, Default: booldefault.StaticBool(false),
				MarkdownDescription: "Provision a MariaDB database for a runtime application. Changing this value replaces the application.",
				PlanModifiers:       []planmodifier.Bool{boolplanmodifier.RequiresReplace()},
			},
			"use_mongodb": schema.BoolAttribute{
				Optional: true, Computed: true, Default: booldefault.StaticBool(false),
				MarkdownDescription: "Provision a MongoDB database for a runtime application. Changing this value replaces the application.",
				PlanModifiers:       []planmodifier.Bool{boolplanmodifier.RequiresReplace()},
			},
			"entrypoint":            schema.StringAttribute{Optional: true, Computed: true, Default: stringdefault.StaticString("")},
			"command":               schema.StringAttribute{Optional: true, Computed: true, Default: stringdefault.StaticString("")},
			"auto_shutdown_enabled": schema.BoolAttribute{Optional: true, Computed: true, Default: booldefault.StaticBool(false)},
			"auto_shutdown_startup": schema.StringAttribute{
				Optional: true, Computed: true, Default: stringdefault.StaticString(startupBehaviorUndefined),
				Validators: []validator.String{stringvalidator.OneOf(startupBehaviorUndefined, startupBehaviorLoadingPage, startupBehaviorBlocking)},
			},
			"artifact_path":   schema.StringAttribute{Optional: true, Computed: true, Default: stringdefault.StaticString("")},
			"spa":             schema.BoolAttribute{Optional: true, Computed: true, Default: booldefault.StaticBool(false)},
			"context":         schema.StringAttribute{Optional: true, Computed: true, Default: stringdefault.StaticString("")},
			"base_image":      schema.StringAttribute{Optional: true, Computed: true, Default: stringdefault.StaticString("")},
			"build_command":   schema.StringAttribute{Optional: true, Computed: true, Default: stringdefault.StaticString("")},
			"dockerfile_name": schema.StringAttribute{Optional: true, Computed: true, Default: stringdefault.StaticString("")},
		},
	}
}
