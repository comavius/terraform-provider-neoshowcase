package repository

import (
	"regexp"

	"github.com/hashicorp/terraform-plugin-framework-validators/setvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64default"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/setdefault"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func resourceSchema() schema.Schema {
	return schema.Schema{
		MarkdownDescription: "Registers and manages a Git repository in NeoShowcase.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "NeoShowcase repository ID.",
			},
			"name": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Repository display name.",
				Validators:          []validator.String{stringvalidator.RegexMatches(regexp.MustCompile(`\S`), "name must contain a non-whitespace character")},
			},
			"url": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Git clone URL.",
				Validators:          []validator.String{stringvalidator.RegexMatches(regexp.MustCompile(`\S`), "URL must contain a non-whitespace character")},
			},
			"html_url": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Human-readable repository URL reported by NeoShowcase.",
			},
			"auth": schema.SingleNestedAttribute{
				Required:            true,
				MarkdownDescription: "Repository authentication. SSH uses the system-wide NeoShowcase deploy key in this provider version.",
				Attributes: map[string]schema.Attribute{
					"method": schema.StringAttribute{
						Required:            true,
						MarkdownDescription: "Authentication method: `none`, `basic`, or `ssh`.",
						Validators:          []validator.String{stringvalidator.OneOf(authMethodNone, authMethodBasic, authMethodSSH)},
					},
					"username": schema.StringAttribute{
						Optional:            true,
						MarkdownDescription: "Username used by BASIC authentication.",
					},
					"password_wo": schema.StringAttribute{
						Optional:            true,
						Sensitive:           true,
						WriteOnly:           true,
						MarkdownDescription: "Write-only BASIC authentication password. This value is never stored in Terraform state.",
					},
					"password_wo_version": schema.Int64Attribute{
						Optional:            true,
						Computed:            true,
						Default:             int64default.StaticInt64(0),
						MarkdownDescription: "Increment this value to rotate `password_wo`.",
					},
				},
			},
			"authoritative_owner_id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "ID of the provider user, which must always remain an owner.",
			},
			"additional_owner_ids": schema.SetAttribute{
				Optional:            true,
				Computed:            true,
				ElementType:         types.StringType,
				Default:             setdefault.StaticValue(types.SetValueMust(types.StringType, []attr.Value{})),
				MarkdownDescription: "Exact set of owners managed in addition to the provider user.",
				Validators: []validator.Set{
					setvalidator.ValueStringsAre(stringvalidator.RegexMatches(regexp.MustCompile(`^\S+$`), "owner IDs must not contain whitespace")),
				},
			},
			"effective_owner_ids": schema.SetAttribute{
				Computed:            true,
				ElementType:         types.StringType,
				MarkdownDescription: "Actual owner set, including the authoritative provider user.",
			},
		},
	}
}
