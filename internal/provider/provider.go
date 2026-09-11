package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/resource"

	currentuserdatasource "github.com/traP-jp/terraform-provider-neoshowcase/internal/provider/datasource/currentuser"
	systeminfodatasource "github.com/traP-jp/terraform-provider-neoshowcase/internal/provider/datasource/systeminfo"
	applicationresource "github.com/traP-jp/terraform-provider-neoshowcase/internal/provider/resource/application"
	repositoryresource "github.com/traP-jp/terraform-provider-neoshowcase/internal/provider/resource/repository"
)

const typeName = "neoshowcase"

var _ provider.Provider = (*neoShowcaseProvider)(nil)

type neoShowcaseProvider struct {
	version string
}

func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &neoShowcaseProvider{version: version}
	}
}

func (p *neoShowcaseProvider) Metadata(_ context.Context, _ provider.MetadataRequest, response *provider.MetadataResponse) {
	response.TypeName = typeName
	response.Version = p.version
}

func (p *neoShowcaseProvider) Resources(_ context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		repositoryresource.New,
		applicationresource.New,
	}
}

func (p *neoShowcaseProvider) DataSources(_ context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		currentuserdatasource.New,
		systeminfodatasource.New,
	}
}
