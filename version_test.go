package altertable

import "testing"

func TestDefaultUserAgentUsesCurrentVersion(t *testing.T) {
	client, err := NewClient(Config{BasicAuthToken: "dGVzdDp0ZXN0"})
	if err != nil {
		t.Fatalf("NewClient returned error: %v", err)
	}

	want := "altertable-lakehouse-go/" + Version
	if client.userAgent != want {
		t.Fatalf("unexpected user agent: got %q want %q", client.userAgent, want)
	}
}
