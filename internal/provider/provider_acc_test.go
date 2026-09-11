//go:build acceptance

package provider

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"regexp"
	"slices"
	"sync"
	"testing"

	"connectrpc.com/connect"
	frameworkproviderserver "github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/emptypb"

	"github.com/traP-jp/terraform-provider-neoshowcase/internal/neoshowcase/gen"
	"github.com/traP-jp/terraform-provider-neoshowcase/internal/neoshowcase/gen/genconnect"
)

func TestAccRepositoryLifecycle(t *testing.T) {
	service := &acceptanceAPIService{}
	path, handler := genconnect.NewAPIServiceHandler(service)
	mux := http.NewServeMux()
	mux.Handle(path, handler)
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: map[string]func() (tfprotov6.ProviderServer, error){
			"neoshowcase": frameworkproviderserver.NewProtocol6WithError(New("acceptance")()),
		},
		CheckDestroy: func(_ *terraform.State) error {
			if repository := service.snapshot(); repository != nil {
				return fmt.Errorf("repository %q still exists", repository.GetId())
			}
			return nil
		},
		Steps: []resource.TestStep{
			{
				Config: acceptanceRepositoryConfig(server.URL, "public", "none", 0, ""),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("neoshowcase_repository.test", "name", "public"),
					resource.TestCheckResourceAttr("neoshowcase_repository.test", "auth.method", "none"),
					resource.TestCheckResourceAttr("neoshowcase_repository.test", "authoritative_owner_id", acceptanceOwnerID),
					resource.TestCheckResourceAttr("neoshowcase_repository.test", "additional_owner_ids.#", "1"),
					resource.TestCheckResourceAttr("neoshowcase_repository.test", "effective_owner_ids.#", "2"),
					resource.TestCheckTypeSetElemAttr("neoshowcase_repository.test", "effective_owner_ids.*", acceptanceOwnerID),
				),
			},
			{
				Config: acceptanceRepositoryConfig(server.URL, "private", "basic", 1, "first-secret"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("neoshowcase_repository.test", "name", "private"),
					resource.TestCheckResourceAttr("neoshowcase_repository.test", "auth.method", "basic"),
					resource.TestCheckResourceAttr("neoshowcase_repository.test", "auth.username", "git"),
					resource.TestCheckResourceAttr("neoshowcase_repository.test", "auth.password_wo_version", "1"),
					resource.TestCheckNoResourceAttr("neoshowcase_repository.test", "auth.password_wo"),
					checkAcceptancePassword(service, "first-secret"),
				),
			},
			{
				Config: acceptanceRepositoryConfig(server.URL, "private", "basic", 2, "rotated-secret"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("neoshowcase_repository.test", "auth.password_wo_version", "2"),
					resource.TestCheckNoResourceAttr("neoshowcase_repository.test", "auth.password_wo"),
					checkAcceptancePassword(service, "rotated-secret"),
				),
			},
			{
				PreConfig: func() {
					service.setOwners([]string{"additional-owner"})
				},
				Config:      acceptanceRepositoryConfig(server.URL, "private", "basic", 2, "rotated-secret"),
				ExpectError: regexp.MustCompile("Provider user is not the authoritative repository owner"),
			},
			{
				PreConfig: func() {
					service.setOwners([]string{"additional-owner", acceptanceOwnerID})
				},
				Config: acceptanceRepositoryConfig(server.URL, "private", "basic", 2, "rotated-secret"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("neoshowcase_repository.test", "effective_owner_ids.#", "2"),
				),
			},
			{
				ResourceName:            "neoshowcase_repository.test",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"auth.username", "auth.password_wo_version"},
			},
		},
	})
}

const acceptanceOwnerID = "provider-owner"

func acceptanceRepositoryConfig(endpoint, name, method string, passwordVersion int, password string) string {
	auth := `auth = { method = "none" }`
	if method == "basic" {
		auth = fmt.Sprintf(`
  auth = {
    method              = "basic"
    username            = "git"
    password_wo         = %q
    password_wo_version = %d
  }`, password, passwordVersion)
	}

	return fmt.Sprintf(`
provider "neoshowcase" {
  endpoint = %q
  user     = "terraform"
}

resource "neoshowcase_repository" "test" {
  name = %q
  url  = "https://example.com/repository.git"

  %s

  additional_owner_ids = ["additional-owner"]
}
`, endpoint, name, auth)
}

func checkAcceptancePassword(service *acceptanceAPIService, want string) resource.TestCheckFunc {
	return func(_ *terraform.State) error {
		if got := service.password(); got != want {
			return fmt.Errorf("repository password = %q, want %q", got, want)
		}
		return nil
	}
}

type acceptanceAPIService struct {
	genconnect.UnimplementedAPIServiceHandler

	mu            sync.Mutex
	repository    *gen.Repository
	basicPassword string
}

func (s *acceptanceAPIService) GetMe(context.Context, *connect.Request[emptypb.Empty]) (*connect.Response[gen.User], error) {
	return connect.NewResponse(&gen.User{Id: acceptanceOwnerID, Name: "terraform"}), nil
}

func (s *acceptanceAPIService) CreateRepository(_ context.Context, request *connect.Request[gen.CreateRepositoryRequest]) (*connect.Response[gen.Repository], error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.repository = &gen.Repository{
		Id:       "repository-id",
		Name:     request.Msg.GetName(),
		Url:      request.Msg.GetUrl(),
		HtmlUrl:  "https://example.com/repository",
		OwnerIds: []string{acceptanceOwnerID},
	}
	s.setAuth(request.Msg.GetAuth())
	return connect.NewResponse(proto.Clone(s.repository).(*gen.Repository)), nil
}

func (s *acceptanceAPIService) GetRepository(_ context.Context, request *connect.Request[gen.RepositoryIdRequest]) (*connect.Response[gen.Repository], error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.repository == nil || request.Msg.GetRepositoryId() != s.repository.GetId() {
		return nil, connect.NewError(connect.CodeNotFound, errors.New("repository not found"))
	}
	return connect.NewResponse(proto.Clone(s.repository).(*gen.Repository)), nil
}

func (s *acceptanceAPIService) UpdateRepository(_ context.Context, request *connect.Request[gen.UpdateRepositoryRequest]) (*connect.Response[emptypb.Empty], error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.repository == nil || request.Msg.GetId() != s.repository.GetId() {
		return nil, connect.NewError(connect.CodeNotFound, errors.New("repository not found"))
	}
	if request.Msg.Name != nil {
		s.repository.Name = request.Msg.GetName()
	}
	if request.Msg.Url != nil {
		s.repository.Url = request.Msg.GetUrl()
	}
	if request.Msg.Auth != nil {
		s.setAuth(request.Msg.GetAuth())
	}
	if request.Msg.OwnerIds != nil {
		s.repository.OwnerIds = slices.Clone(request.Msg.GetOwnerIds().GetOwnerIds())
	}
	return connect.NewResponse(&emptypb.Empty{}), nil
}

func (s *acceptanceAPIService) DeleteRepository(_ context.Context, request *connect.Request[gen.RepositoryIdRequest]) (*connect.Response[emptypb.Empty], error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.repository == nil || request.Msg.GetRepositoryId() != s.repository.GetId() {
		return nil, connect.NewError(connect.CodeNotFound, errors.New("repository not found"))
	}
	s.repository = nil
	return connect.NewResponse(&emptypb.Empty{}), nil
}

func (s *acceptanceAPIService) setAuth(auth *gen.CreateRepositoryAuth) {
	s.basicPassword = ""
	switch {
	case auth.GetBasic() != nil:
		s.repository.AuthMethod = gen.Repository_BASIC
		s.basicPassword = auth.GetBasic().GetPassword()
	case auth.GetSsh() != nil:
		s.repository.AuthMethod = gen.Repository_SSH
	default:
		s.repository.AuthMethod = gen.Repository_NONE
	}
}

func (s *acceptanceAPIService) snapshot() *gen.Repository {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.repository == nil {
		return nil
	}
	return proto.Clone(s.repository).(*gen.Repository)
}

func (s *acceptanceAPIService) password() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.basicPassword
}

func (s *acceptanceAPIService) setOwners(ownerIDs []string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.repository != nil {
		s.repository.OwnerIds = slices.Clone(ownerIDs)
	}
}
