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
	testing *testing.T
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

func TestClientGetMe(t *testing.T) {
	t.Parallel()

	path, handler := genconnect.NewAPIServiceHandler(testAPIService{testing: t})
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

func withPath(path string, handler http.Handler) http.Handler {
	mux := http.NewServeMux()
	mux.Handle(path, handler)
	return mux
}
