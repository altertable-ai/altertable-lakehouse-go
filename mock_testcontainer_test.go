package altertable

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

// integrationMockBaseURL is set in TestMain when the mock is available
// (reachable localhost, ALTERTABLE_MOCK_BASE_URL, or testcontainers).
var integrationMockBaseURL string

func TestMain(m *testing.M) {
	os.Exit(runClientTests(m))
}

func runClientTests(m *testing.M) int {
	ctx := context.Background()
	var container testcontainers.Container

	defer func() {
		if container == nil {
			return
		}
		stopCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		if err := container.Terminate(stopCtx); err != nil {
			log.Printf("terminate altertable-mock container: %v", err)
		}
	}()

	if b := strings.TrimSpace(os.Getenv("ALTERTABLE_MOCK_BASE_URL")); b != "" {
		integrationMockBaseURL = strings.TrimSuffix(b, "/")
	} else if u := localhostMockURL(); mockReachable(u) {
		integrationMockBaseURL = u
	} else if os.Getenv("ALTERTABLE_SKIP_TESTCONTAINER") == "1" {
		log.Print("ALTERTABLE_SKIP_TESTCONTAINER=1: not starting testcontainer")
	} else {
		c, base, err := startAltertableMock(ctx)
		if err != nil {
			log.Printf("altertable-mock testcontainer: %v (integration tests will skip)", err)
		} else {
			container = c
			integrationMockBaseURL = base
		}
	}

	return m.Run()
}

func localhostMockURL() string {
	port := os.Getenv("ALTERTABLE_MOCK_PORT")
	if port == "" {
		port = "15000"
	}
	return "http://localhost:" + port
}

func startAltertableMock(ctx context.Context) (testcontainers.Container, string, error) {
	req := testcontainers.ContainerRequest{
		Image:        "ghcr.io/altertable-ai/altertable-mock:latest",
		ExposedPorts: []string{"15000/tcp"},
		Env: map[string]string{
			"ALTERTABLE_MOCK_USERS": "testuser:testpass",
		},
		WaitingFor: wait.ForListeningPort("15000/tcp"),
	}
	c, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	if err != nil {
		return nil, "", err
	}
	host, err := c.Host(ctx)
	if err != nil {
		_ = c.Terminate(ctx)
		return nil, "", err
	}
	mapped, err := c.MappedPort(ctx, "15000/tcp")
	if err != nil {
		_ = c.Terminate(ctx)
		return nil, "", err
	}
	return c, fmt.Sprintf("http://%s:%s", host, mapped.Port()), nil
}
