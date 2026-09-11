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

const testSessionCookie = "neoshowcase_session=session-value; other_cookie=other-value"

type testAPIService struct {
	genconnect.UnimplementedAPIServiceHandler
	testing              *testing.T
	repository           *gen.Repository
	application          *gen.Application
	environmentVariables map[string]string
	builds               []*gen.Build
}

func (s *testAPIService) CreateApplication(_ context.Context, request *connect.Request[gen.CreateApplicationRequest]) (*connect.Response[gen.Application], error) {
	s.testing.Helper()
	s.application = &gen.Application{
		Id: "application-id", Name: request.Msg.GetName(), RepositoryId: request.Msg.GetRepositoryId(),
		RefName: request.Msg.GetRefName(), Config: request.Msg.GetConfig(), Websites: nil,
		PortPublications: request.Msg.GetPortPublications(), Running: request.Msg.GetStartOnCreate(), OwnerIds: []string{"provider-user"},
	}
	s.environmentVariables = make(map[string]string)
	return connect.NewResponse(s.application), nil
}

func (s *testAPIService) GetApplication(_ context.Context, request *connect.Request[gen.ApplicationIdRequest]) (*connect.Response[gen.Application], error) {
	s.testing.Helper()
	if s.application == nil || request.Msg.GetId() != s.application.GetId() {
		return nil, connect.NewError(connect.CodeNotFound, nil)
	}
	return connect.NewResponse(s.application), nil
}

func (s *testAPIService) UpdateApplication(_ context.Context, request *connect.Request[gen.UpdateApplicationRequest]) (*connect.Response[emptypb.Empty], error) {
	s.testing.Helper()
	if s.application == nil || request.Msg.GetId() != s.application.GetId() {
		return nil, connect.NewError(connect.CodeNotFound, nil)
	}
	if request.Msg.Name != nil {
		s.application.Name = request.Msg.GetName()
	}
	if request.Msg.RefName != nil {
		s.application.RefName = request.Msg.GetRefName()
	}
	if request.Msg.Config != nil {
		s.application.Config = request.Msg.GetConfig()
	}
	if request.Msg.OwnerIds != nil {
		s.application.OwnerIds = request.Msg.GetOwnerIds().GetOwnerIds()
	}
	return connect.NewResponse(&emptypb.Empty{}), nil
}

func (s *testAPIService) DeleteApplication(_ context.Context, request *connect.Request[gen.ApplicationIdRequest]) (*connect.Response[emptypb.Empty], error) {
	s.testing.Helper()
	if s.application == nil || request.Msg.GetId() != s.application.GetId() {
		return nil, connect.NewError(connect.CodeNotFound, nil)
	}
	s.application = nil
	s.environmentVariables = nil
	s.builds = nil
	return connect.NewResponse(&emptypb.Empty{}), nil
}

func (s *testAPIService) GetBuilds(_ context.Context, request *connect.Request[gen.ApplicationIdRequest]) (*connect.Response[gen.GetBuildsResponse], error) {
	s.testing.Helper()
	if s.application == nil || request.Msg.GetId() != s.application.GetId() {
		return nil, connect.NewError(connect.CodeNotFound, nil)
	}
	return connect.NewResponse(&gen.GetBuildsResponse{Builds: s.builds}), nil
}

func (s *testAPIService) GetBuild(_ context.Context, request *connect.Request[gen.BuildIdRequest]) (*connect.Response[gen.Build], error) {
	s.testing.Helper()
	for _, build := range s.builds {
		if build.GetId() == request.Msg.GetBuildId() {
			return connect.NewResponse(build), nil
		}
	}
	return nil, connect.NewError(connect.CodeNotFound, nil)
}

func (s *testAPIService) GetEnvVars(_ context.Context, request *connect.Request[gen.ApplicationIdRequest]) (*connect.Response[gen.ApplicationEnvVars], error) {
	s.testing.Helper()
	if s.application == nil || request.Msg.GetId() != s.application.GetId() {
		return nil, connect.NewError(connect.CodeNotFound, nil)
	}
	variables := make([]*gen.ApplicationEnvVar, 0, len(s.environmentVariables))
	for key, value := range s.environmentVariables {
		variables = append(variables, &gen.ApplicationEnvVar{ApplicationId: s.application.GetId(), Key: key, Value: value})
	}
	return connect.NewResponse(&gen.ApplicationEnvVars{Variables: variables}), nil
}

func (s *testAPIService) SetEnvVar(_ context.Context, request *connect.Request[gen.SetApplicationEnvVarRequest]) (*connect.Response[emptypb.Empty], error) {
	s.testing.Helper()
	if s.application == nil || request.Msg.GetApplicationId() != s.application.GetId() {
		return nil, connect.NewError(connect.CodeNotFound, nil)
	}
	s.environmentVariables[request.Msg.GetKey()] = request.Msg.GetValue()
	return connect.NewResponse(&emptypb.Empty{}), nil
}

func (s *testAPIService) DeleteEnvVar(_ context.Context, request *connect.Request[gen.DeleteApplicationEnvVarRequest]) (*connect.Response[emptypb.Empty], error) {
	s.testing.Helper()
	if s.application == nil || request.Msg.GetApplicationId() != s.application.GetId() {
		return nil, connect.NewError(connect.CodeNotFound, nil)
	}
	delete(s.environmentVariables, request.Msg.GetKey())
	return connect.NewResponse(&emptypb.Empty{}), nil
}

func (s *testAPIService) StartApplication(_ context.Context, request *connect.Request[gen.ApplicationIdRequest]) (*connect.Response[emptypb.Empty], error) {
	s.testing.Helper()
	if s.application == nil || request.Msg.GetId() != s.application.GetId() {
		return nil, connect.NewError(connect.CodeNotFound, nil)
	}
	s.application.Running = true
	return connect.NewResponse(&emptypb.Empty{}), nil
}

func (s *testAPIService) StopApplication(_ context.Context, request *connect.Request[gen.ApplicationIdRequest]) (*connect.Response[emptypb.Empty], error) {
	s.testing.Helper()
	if s.application == nil || request.Msg.GetId() != s.application.GetId() {
		return nil, connect.NewError(connect.CodeNotFound, nil)
	}
	s.application.Running = false
	return connect.NewResponse(&emptypb.Empty{}), nil
}

func (s testAPIService) GetMe(_ context.Context, request *connect.Request[emptypb.Empty]) (*connect.Response[gen.User], error) {
	s.testing.Helper()
	if got, want := request.Header().Get("Cookie"), testSessionCookie; got != want {
		s.testing.Errorf("Cookie header = %q, want %q", got, want)
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

func (s *testAPIService) GetRepositories(_ context.Context, request *connect.Request[gen.GetRepositoriesRequest]) (*connect.Response[gen.GetRepositoriesResponse], error) {
	s.testing.Helper()
	if got, want := request.Msg.GetScope(), gen.GetRepositoriesRequest_MINE; got != want {
		s.testing.Errorf("repository scope = %s, want %s", got, want)
	}
	repositories := []*gen.Repository(nil)
	if s.repository != nil {
		repositories = append(repositories, s.repository)
	}
	return connect.NewResponse(&gen.GetRepositoriesResponse{Repositories: repositories}), nil
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
		Endpoint:      mux.URL + "/",
		SessionCookie: testSessionCookie,
		HTTPClient:    mux.Client(),
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

func TestNewClientRequiresSessionCookie(t *testing.T) {
	t.Parallel()

	_, err := NewClient(Options{Endpoint: "https://ns.trap.jp"})
	if err == nil {
		t.Fatal("NewClient() error = nil, want missing session cookie error")
	}
}

func TestNewClientRejectsSessionCookieWithNewline(t *testing.T) {
	t.Parallel()

	_, err := NewClient(Options{
		Endpoint:      "https://ns.trap.jp",
		SessionCookie: "session=value\r\nX-Showcase-User: attacker",
	})
	if err == nil {
		t.Fatal("NewClient() error = nil, want invalid session cookie error")
	}
}

func TestClientRepositoryLifecycle(t *testing.T) {
	t.Parallel()

	service := &testAPIService{testing: t}
	path, handler := genconnect.NewAPIServiceHandler(service)
	server := httptest.NewServer(withPath(path, handler))
	t.Cleanup(server.Close)

	client, err := NewClient(Options{
		Endpoint:      server.URL,
		SessionCookie: testSessionCookie,
		HTTPClient:    server.Client(),
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
	found, err := client.GetOwnedRepositoryByURL(context.Background(), created.GetUrl())
	if err != nil {
		t.Fatalf("GetOwnedRepositoryByURL() error = %v", err)
	}
	if found.GetId() != created.GetId() {
		t.Fatalf("found repository ID = %q, want %q", found.GetId(), created.GetId())
	}
	notFound, err := client.GetOwnedRepositoryByURL(context.Background(), "https://example.com/missing.git")
	if err != nil {
		t.Fatalf("GetOwnedRepositoryByURL() missing error = %v", err)
	}
	if notFound != nil {
		t.Fatalf("GetOwnedRepositoryByURL() missing = %#v, want nil", notFound)
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

func TestClientApplicationLifecycle(t *testing.T) {
	t.Parallel()

	service := &testAPIService{testing: t}
	path, handler := genconnect.NewAPIServiceHandler(service)
	server := httptest.NewServer(withPath(path, handler))
	t.Cleanup(server.Close)

	client, err := NewClient(Options{Endpoint: server.URL, SessionCookie: testSessionCookie, HTTPClient: server.Client()})
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}
	created, err := client.CreateApplication(context.Background(), &gen.CreateApplicationRequest{
		Name: "application", RepositoryId: "repository-id", RefName: "main",
		Config: &gen.ApplicationConfig{BuildConfig: &gen.ApplicationConfig_StaticBuildpack{StaticBuildpack: &gen.BuildConfigStaticBuildpack{StaticConfig: &gen.StaticConfig{ArtifactPath: "."}}}},
	})
	if err != nil {
		t.Fatalf("CreateApplication() error = %v", err)
	}
	name := "renamed"
	if err := client.UpdateApplication(context.Background(), &gen.UpdateApplicationRequest{Id: created.GetId(), Name: &name}); err != nil {
		t.Fatalf("UpdateApplication() error = %v", err)
	}
	if err := client.SetApplicationEnvironmentVariable(context.Background(), created.GetId(), "TOKEN", "secret"); err != nil {
		t.Fatalf("SetApplicationEnvironmentVariable() error = %v", err)
	}
	variables, err := client.GetApplicationEnvironmentVariables(context.Background(), created.GetId())
	if err != nil || len(variables) != 1 || variables[0].GetKey() != "TOKEN" {
		t.Fatalf("GetApplicationEnvironmentVariables() = %#v, %v", variables, err)
	}
	if err := client.StartApplication(context.Background(), created.GetId()); err != nil {
		t.Fatalf("StartApplication() error = %v", err)
	}
	application, err := client.GetApplication(context.Background(), created.GetId())
	if err != nil || !application.GetRunning() || application.GetName() != name {
		t.Fatalf("GetApplication() = %#v, %v", application, err)
	}
	service.builds = []*gen.Build{{Id: "build-id", ApplicationId: created.GetId(), Status: gen.BuildStatus_SUCCEEDED}}
	builds, err := client.GetApplicationBuilds(context.Background(), created.GetId())
	if err != nil || len(builds) != 1 || builds[0].GetId() != "build-id" {
		t.Fatalf("GetApplicationBuilds() = %#v, %v", builds, err)
	}
	build, err := client.GetBuild(context.Background(), builds[0].GetId())
	if err != nil || build.GetStatus() != gen.BuildStatus_SUCCEEDED {
		t.Fatalf("GetBuild() = %#v, %v", build, err)
	}
	if err := client.StopApplication(context.Background(), created.GetId()); err != nil {
		t.Fatalf("StopApplication() error = %v", err)
	}
	if err := client.DeleteApplicationEnvironmentVariable(context.Background(), created.GetId(), "TOKEN"); err != nil {
		t.Fatalf("DeleteApplicationEnvironmentVariable() error = %v", err)
	}
	if err := client.DeleteApplication(context.Background(), created.GetId()); err != nil {
		t.Fatalf("DeleteApplication() error = %v", err)
	}
	_, err = client.GetApplication(context.Background(), created.GetId())
	if !IsNotFound(err) {
		t.Fatalf("GetApplication() after deletion error = %v, want not found", err)
	}
}

func withPath(path string, handler http.Handler) http.Handler {
	mux := http.NewServeMux()
	mux.Handle(path, handler)
	return mux
}
