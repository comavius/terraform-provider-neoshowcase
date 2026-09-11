package neoshowcase

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"connectrpc.com/connect"
	"google.golang.org/protobuf/types/known/emptypb"

	"github.com/traP-jp/terraform-provider-neoshowcase/internal/neoshowcase/gen"
	"github.com/traP-jp/terraform-provider-neoshowcase/internal/neoshowcase/gen/genconnect"
)

type testAPIService struct {
	genconnect.UnimplementedAPIServiceHandler
	testing    *testing.T
	repository *gen.Repository
}

func (s testAPIService) GetMe(_ context.Context, request *connect.Request[emptypb.Empty]) (*connect.Response[gen.User], error) {
	s.testing.Helper()
	if got, want := request.Header().Get(DefaultAuthHeader), "terraform"; got != want {
		s.testing.Errorf("auth header = %q, want %q", got, want)
	}
	if got, want := request.Header().Get("X-Test-Header"), "test-value"; got != want {
		s.testing.Errorf("additional header = %q, want %q", got, want)
	}
	return connect.NewResponse(&gen.User{Id: "user-id", Name: "terraform"}), nil
}

func (s *testAPIService) CreateRepository(_ context.Context, request *connect.Request[gen.CreateRepositoryRequest]) (*connect.Response[gen.Repository], error) {
	s.testing.Helper()
	if request.Msg.GetAuth().GetNone() == nil {
		s.testing.Error("create repository auth is not NONE")
	}
	s.repository = &gen.Repository{
		Id:         "repository-id",
		Name:       request.Msg.GetName(),
		Url:        request.Msg.GetUrl(),
		HtmlUrl:    "https://example.com/repository",
		AuthMethod: gen.Repository_NONE,
		OwnerIds:   []string{"provider-user"},
	}
	return connect.NewResponse(s.repository), nil
}

func (s *testAPIService) GetRepository(_ context.Context, request *connect.Request[gen.RepositoryIdRequest]) (*connect.Response[gen.Repository], error) {
	s.testing.Helper()
	if s.repository == nil || request.Msg.GetRepositoryId() != s.repository.GetId() {
		return nil, connect.NewError(connect.CodeNotFound, nil)
	}
	return connect.NewResponse(s.repository), nil
}

func (s *testAPIService) UpdateRepository(_ context.Context, request *connect.Request[gen.UpdateRepositoryRequest]) (*connect.Response[emptypb.Empty], error) {
	s.testing.Helper()
	if s.repository == nil || request.Msg.GetId() != s.repository.GetId() {
		return nil, connect.NewError(connect.CodeNotFound, nil)
	}
	if request.Msg.Name != nil {
		s.repository.Name = request.Msg.GetName()
	}
	if request.Msg.Url != nil {
		s.repository.Url = request.Msg.GetUrl()
	}
	if request.Msg.OwnerIds != nil {
		s.repository.OwnerIds = request.Msg.GetOwnerIds().GetOwnerIds()
	}
	return connect.NewResponse(&emptypb.Empty{}), nil
}

func (s *testAPIService) DeleteRepository(_ context.Context, request *connect.Request[gen.RepositoryIdRequest]) (*connect.Response[emptypb.Empty], error) {
	s.testing.Helper()
	if s.repository == nil || request.Msg.GetRepositoryId() != s.repository.GetId() {
		return nil, connect.NewError(connect.CodeNotFound, nil)
	}
	s.repository = nil
	return connect.NewResponse(&emptypb.Empty{}), nil
}

func TestClientGetMe(t *testing.T) {
	t.Parallel()

	path, handler := genconnect.NewAPIServiceHandler(&testAPIService{testing: t})
	mux := httptest.NewServer(withPath(path, handler))
	t.Cleanup(mux.Close)

	client, err := NewClient(Options{
		Endpoint: mux.URL + "/",
		User:     "terraform",
		AdditionalHeaders: map[string]string{
			"X-Test-Header": "test-value",
		},
		HTTPClient: mux.Client(),
	})
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	user, err := client.GetMe(context.Background())
	if err != nil {
		t.Fatalf("GetMe() error = %v", err)
	}
	if got, want := user.GetId(), "user-id"; got != want {
		t.Fatalf("user ID = %q, want %q", got, want)
	}
}

func TestClientRepositoryLifecycle(t *testing.T) {
	t.Parallel()

	service := &testAPIService{testing: t}
	path, handler := genconnect.NewAPIServiceHandler(service)
	server := httptest.NewServer(withPath(path, handler))
	t.Cleanup(server.Close)

	client, err := NewClient(Options{
		Endpoint:   server.URL,
		User:       "terraform",
		HTTPClient: server.Client(),
	})
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	created, err := client.CreateRepository(context.Background(), &gen.CreateRepositoryRequest{
		Name: "repository",
		Url:  "https://example.com/repository.git",
		Auth: &gen.CreateRepositoryAuth{Auth: &gen.CreateRepositoryAuth_None{None: &emptypb.Empty{}}},
	})
	if err != nil {
		t.Fatalf("CreateRepository() error = %v", err)
	}
	if got, want := created.GetId(), "repository-id"; got != want {
		t.Fatalf("repository ID = %q, want %q", got, want)
	}

	name := "renamed"
	if err := client.UpdateRepository(context.Background(), &gen.UpdateRepositoryRequest{
		Id:   created.GetId(),
		Name: &name,
		OwnerIds: &gen.UpdateRepositoryRequest_UpdateOwners{
			OwnerIds: []string{"additional-user", "provider-user"},
		},
	}); err != nil {
		t.Fatalf("UpdateRepository() error = %v", err)
	}

	repository, err := client.GetRepository(context.Background(), created.GetId())
	if err != nil {
		t.Fatalf("GetRepository() error = %v", err)
	}
	if got, want := repository.GetName(), name; got != want {
		t.Errorf("repository name = %q, want %q", got, want)
	}
	if got, want := len(repository.GetOwnerIds()), 2; got != want {
		t.Errorf("owner count = %d, want %d", got, want)
	}

	if err := client.DeleteRepository(context.Background(), created.GetId()); err != nil {
		t.Fatalf("DeleteRepository() error = %v", err)
	}
	_, err = client.GetRepository(context.Background(), created.GetId())
	if !IsNotFound(err) {
		t.Fatalf("GetRepository() after deletion error = %v, want not found", err)
	}
}

func withPath(path string, handler http.Handler) http.Handler {
	mux := http.NewServeMux()
	mux.Handle(path, handler)
	return mux
}
