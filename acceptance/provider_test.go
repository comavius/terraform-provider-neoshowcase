//go:build acceptance

package acceptance

import (
	"context"
	"fmt"
	"os"
	"regexp"
	"strings"
	"testing"
	"time"

	frameworkproviderserver "github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"

	"github.com/traP-jp/terraform-provider-neoshowcase/internal/neoshowcase"
	neoshowcaseprovider "github.com/traP-jp/terraform-provider-neoshowcase/internal/provider"
)

func TestAccRealRepositoryLifecycle(t *testing.T) {
	environment := requireRealAcceptanceEnvironment(t)
	owner := environment.runID + "-owner"
	resourceName := "neoshowcase_repository.test"

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: realProviderFactories(),
		CheckDestroy:             checkRealRepositoriesDestroyed(environment.endpoint, owner),
		Steps: []resource.TestStep{
			{
				Config: realPublicRepositoryConfig(environment.endpoint, owner, environment.additionalOwnerID, environment.publicURLOne, environment.runID+"-public-one", true),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "name", environment.runID+"-public-one"),
					resource.TestCheckResourceAttr(resourceName, "url", environment.publicURLOne),
					resource.TestCheckResourceAttr(resourceName, "auth.method", "none"),
					resource.TestCheckResourceAttr(resourceName, "additional_owner_ids.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "effective_owner_ids.#", "2"),
					resource.TestCheckResourceAttrPair(resourceName, "authoritative_owner_id", "data.neoshowcase_current_user.owner", "id"),
					resource.TestCheckTypeSetElemAttr(resourceName, "additional_owner_ids.*", environment.additionalOwnerID),
					resource.TestCheckResourceAttr("data.neoshowcase_current_user.owner", "name", owner),
					resource.TestCheckResourceAttrSet("data.neoshowcase_system_info.test", "public_key"),
					resource.TestCheckResourceAttrSet("data.neoshowcase_system_info.test", "version"),
				),
			},
			{
				Config: realPublicRepositoryConfig(environment.endpoint, owner, environment.additionalOwnerID, environment.publicURLTwo, environment.runID+"-public-two", false),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "name", environment.runID+"-public-two"),
					resource.TestCheckResourceAttr(resourceName, "url", environment.publicURLTwo),
					resource.TestCheckResourceAttr(resourceName, "additional_owner_ids.#", "0"),
					resource.TestCheckResourceAttr(resourceName, "effective_owner_ids.#", "1"),
				),
			},
			{
				ResourceName:            resourceName,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"auth.username", "auth.password_wo_version"},
			},
		},
	})
}

func TestAccRealRepositoryAuthentication(t *testing.T) {
	environment := requireRealAcceptanceEnvironment(t)
	owner := environment.runID + "-basic-owner"
	resourceName := "neoshowcase_repository.test"

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: realProviderFactories(),
		CheckDestroy:             checkRealRepositoriesDestroyed(environment.endpoint, owner),
		Steps: []resource.TestStep{
			{
				Config: realBasicRepositoryConfig(environment, owner, environment.basicUserOne, environment.basicPasswordOne, 1),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "auth.method", "basic"),
					resource.TestCheckResourceAttr(resourceName, "auth.username", environment.basicUserOne),
					resource.TestCheckResourceAttr(resourceName, "auth.password_wo_version", "1"),
					resource.TestCheckNoResourceAttr(resourceName, "auth.password_wo"),
				),
			},
			{
				Config: realBasicRepositoryConfig(environment, owner, environment.basicUserTwo, environment.basicPasswordTwo, 2),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "auth.username", environment.basicUserTwo),
					resource.TestCheckResourceAttr(resourceName, "auth.password_wo_version", "2"),
					resource.TestCheckNoResourceAttr(resourceName, "auth.password_wo"),
				),
			},
			{
				Config: realSSHRepositoryConfig(environment, owner),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "auth.method", "ssh"),
					resource.TestCheckNoResourceAttr(resourceName, "auth.username"),
					resource.TestCheckResourceAttr(resourceName, "url", environment.privateSSHURL),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccRealApplicationWithEnvironmentVariables(t *testing.T) {
	environment := requireRealAcceptanceEnvironment(t)
	owner := environment.runID + "-app-owner"
	resourceName := "neoshowcase_application.test"

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: realProviderFactories(),
		CheckDestroy:             checkRealRepositoriesDestroyed(environment.endpoint, owner),
		Steps: []resource.TestStep{
			{
				Config: realApplicationConfig(environment, owner, environment.runID+"-app-one", "TOKEN", "first-value", 1, false),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "name", environment.runID+"-app-one"),
					resource.TestCheckResourceAttr(resourceName, "ref_name", "main"),
					resource.TestCheckResourceAttr(resourceName, "build.type", "static_buildpack"),
					resource.TestCheckResourceAttr(resourceName, "build.artifact_path", "."),
					resource.TestCheckResourceAttr(resourceName, "running", "false"),
					resource.TestCheckResourceAttr(resourceName, "environment_variables.%", "1"),
					resource.TestCheckResourceAttr(resourceName, "environment_variables.TOKEN.value_wo_version", "1"),
					resource.TestCheckNoResourceAttr(resourceName, "environment_variables.TOKEN.value_wo"),
					resource.TestCheckResourceAttr(resourceName, "effective_owner_ids.#", "1"),
				),
			},
			{
				Config: realApplicationConfig(environment, owner, environment.runID+"-app-two", "FEATURE_FLAG", "enabled", 2, true),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "name", environment.runID+"-app-two"),
					resource.TestCheckResourceAttr(resourceName, "running", "true"),
					resource.TestCheckResourceAttr(resourceName, "environment_variables.%", "1"),
					resource.TestCheckResourceAttr(resourceName, "environment_variables.FEATURE_FLAG.value_wo_version", "2"),
					resource.TestCheckNoResourceAttr(resourceName, "environment_variables.TOKEN"),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateVerifyIgnore: []string{
					"environment_variables",
					"commit",
					"container_state",
					"container_message",
					"current_build_id",
					"latest_build_status",
					"updated_at",
				},
			},
		},
	})
}

func TestAccRealRepositoryRejectsMissingGitRemote(t *testing.T) {
	environment := requireRealAcceptanceEnvironment(t)
	owner := environment.runID + "-negative-owner"

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: realProviderFactories(),
		CheckDestroy:             checkRealRepositoriesDestroyed(environment.endpoint, owner),
		Steps: []resource.TestStep{
			{
				Config:      realMissingRepositoryConfig(environment.endpoint, owner, environment.runID),
				ExpectError: regexp.MustCompile(`cannot fetch repository|Unable to create NeoShowcase repository`),
			},
		},
	})
}

type realAcceptanceEnvironment struct {
	endpoint          string
	runID             string
	publicURLOne      string
	publicURLTwo      string
	privateURL        string
	privateSSHURL     string
	additionalOwnerID string
	basicUserOne      string
	basicPasswordOne  string
	basicUserTwo      string
	basicPasswordTwo  string
}

func requireRealAcceptanceEnvironment(t *testing.T) realAcceptanceEnvironment {
	t.Helper()
	if os.Getenv("NEOSHOWCASE_ACC") != "1" {
		t.Skip("set NEOSHOWCASE_ACC=1 and start the real NeoShowcase integration environment")
	}
	t.Setenv("TF_ACC_PROVIDER_NAMESPACE", "comavius")

	environment := realAcceptanceEnvironment{
		endpoint:          os.Getenv("NEOSHOWCASE_TEST_ENDPOINT"),
		runID:             os.Getenv("NEOSHOWCASE_TEST_RUN_ID"),
		publicURLOne:      os.Getenv("NEOSHOWCASE_TEST_PUBLIC_URL_ONE"),
		publicURLTwo:      os.Getenv("NEOSHOWCASE_TEST_PUBLIC_URL_TWO"),
		privateURL:        os.Getenv("NEOSHOWCASE_TEST_PRIVATE_URL"),
		privateSSHURL:     os.Getenv("NEOSHOWCASE_TEST_PRIVATE_SSH_URL"),
		additionalOwnerID: os.Getenv("NEOSHOWCASE_TEST_ADDITIONAL_OWNER_ID"),
		basicUserOne:      os.Getenv("NEOSHOWCASE_TEST_BASIC_USER_ONE"),
		basicPasswordOne:  os.Getenv("NEOSHOWCASE_TEST_BASIC_PASSWORD_ONE"),
		basicUserTwo:      os.Getenv("NEOSHOWCASE_TEST_BASIC_USER_TWO"),
		basicPasswordTwo:  os.Getenv("NEOSHOWCASE_TEST_BASIC_PASSWORD_TWO"),
	}
	for name, value := range map[string]string{
		"NEOSHOWCASE_TEST_ENDPOINT":            environment.endpoint,
		"NEOSHOWCASE_TEST_RUN_ID":              environment.runID,
		"NEOSHOWCASE_TEST_PUBLIC_URL_ONE":      environment.publicURLOne,
		"NEOSHOWCASE_TEST_PUBLIC_URL_TWO":      environment.publicURLTwo,
		"NEOSHOWCASE_TEST_PRIVATE_URL":         environment.privateURL,
		"NEOSHOWCASE_TEST_PRIVATE_SSH_URL":     environment.privateSSHURL,
		"NEOSHOWCASE_TEST_ADDITIONAL_OWNER_ID": environment.additionalOwnerID,
		"NEOSHOWCASE_TEST_BASIC_USER_ONE":      environment.basicUserOne,
		"NEOSHOWCASE_TEST_BASIC_PASSWORD_ONE":  environment.basicPasswordOne,
		"NEOSHOWCASE_TEST_BASIC_USER_TWO":      environment.basicUserTwo,
		"NEOSHOWCASE_TEST_BASIC_PASSWORD_TWO":  environment.basicPasswordTwo,
	} {
		if strings.TrimSpace(value) == "" {
			t.Fatalf("%s must be set", name)
		}
	}
	return environment
}

func realProviderFactories() map[string]func() (tfprotov6.ProviderServer, error) {
	return map[string]func() (tfprotov6.ProviderServer, error){
		"neoshowcase": frameworkproviderserver.NewProtocol6WithError(neoshowcaseprovider.New("acceptance")()),
	}
}

func checkRealRepositoriesDestroyed(endpoint, user string) func(*terraform.State) error {
	return func(state *terraform.State) error {
		client, err := neoshowcase.NewClient(neoshowcase.Options{Endpoint: endpoint, User: user})
		if err != nil {
			return err
		}
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		for _, terraformResource := range state.RootModule().Resources {
			if terraformResource.Type != "neoshowcase_repository" {
				continue
			}
			_, err := client.GetRepository(ctx, terraformResource.Primary.ID)
			if err == nil {
				return fmt.Errorf("NeoShowcase repository %q still exists", terraformResource.Primary.ID)
			}
			if !neoshowcase.IsNotFound(err) {
				return err
			}
		}
		return nil
	}
}

func realPublicRepositoryConfig(endpoint, owner, additionalOwnerID, repositoryURL, name string, includeAdditionalOwner bool) string {
	additionalOwnerIDs := "[]"
	if includeAdditionalOwner {
		additionalOwnerIDs = fmt.Sprintf("[%q]", additionalOwnerID)
	}
	return fmt.Sprintf(`
provider "neoshowcase" {
  endpoint = %q
  user     = %q
}

data "neoshowcase_current_user" "owner" {}

data "neoshowcase_system_info" "test" {}

resource "neoshowcase_repository" "test" {
  name = %q
  url  = %q
  auth = { method = "none" }
  additional_owner_ids = %s
}
`, endpoint, owner, name, repositoryURL, additionalOwnerIDs)
}

func realBasicRepositoryConfig(environment realAcceptanceEnvironment, owner, username, password string, passwordVersion int) string {
	return fmt.Sprintf(`
provider "neoshowcase" {
  endpoint = %q
  user     = %q
}

resource "neoshowcase_repository" "test" {
  name = %q
  url  = %q
  auth = {
    method              = "basic"
    username            = %q
    password_wo         = %q
    password_wo_version = %d
  }
  additional_owner_ids = []
}
`, environment.endpoint, owner, environment.runID+"-private", environment.privateURL, username, password, passwordVersion)
}

func realApplicationConfig(environment realAcceptanceEnvironment, owner, name, environmentKey, environmentValue string, environmentVersion int, running bool) string {
	return fmt.Sprintf(`
provider "neoshowcase" {
  endpoint = %q
  user     = %q
}

resource "neoshowcase_repository" "test" {
  name = %q
  url  = %q
  auth = { method = "none" }
}

resource "neoshowcase_application" "test" {
  name          = %q
  repository_id = neoshowcase_repository.test.id
  ref_name      = "main"
  running       = %t

  build = {
    type          = "static_buildpack"
    artifact_path = "."
  }

  environment_variables = {
    %s = {
      value_wo         = %q
      value_wo_version = %d
    }
  }
}
`, environment.endpoint, owner, environment.runID+"-app-repository", environment.publicURLOne, name, running, environmentKey, environmentValue, environmentVersion)
}

func realSSHRepositoryConfig(environment realAcceptanceEnvironment, owner string) string {
	return fmt.Sprintf(`
provider "neoshowcase" {
  endpoint = %q
  user     = %q
}

resource "neoshowcase_repository" "test" {
  name = %q
  url  = %q
  auth = { method = "ssh" }
  additional_owner_ids = []
}
`, environment.endpoint, owner, environment.runID+"-private-ssh", environment.privateSSHURL)
}

func realMissingRepositoryConfig(endpoint, owner, runID string) string {
	return fmt.Sprintf(`
provider "neoshowcase" {
  endpoint = %q
  user     = %q
}

resource "neoshowcase_repository" "test" {
  name = %q
  url  = "http://gitea:3000/integration-admin/does-not-exist.git"
  auth = { method = "none" }
  additional_owner_ids = []
}
`, endpoint, owner, runID+"-missing")
}
