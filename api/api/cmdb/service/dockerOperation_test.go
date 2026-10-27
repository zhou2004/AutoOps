package service

import "testing"

func TestDockerValidationAndShellQuoting(t *testing.T) {
	if got := dockerShellQuote("a'b"); got != `'a'"'"'b'` {
		t.Fatalf("unexpected shell quote: %s", got)
	}
	valid := &DockerCreateRequest{Name: "web-1", Image: "registry.local/team/web:1.2", Restart: "unless-stopped", Environment: map[string]string{"APP_ENV": "prod"}, Ports: []DockerPort{{Host: 8080, Container: 80, Protocol: "tcp"}}, Command: []string{"serve", "--port=80"}}
	if err := validateDockerCreate(valid); err != nil {
		t.Fatal(err)
	}
	for _, invalid := range []*DockerCreateRequest{
		{Name: "bad name", Image: "nginx"},
		{Name: "web", Image: "nginx;id"},
		{Name: "web", Image: "nginx", Restart: "always;id"},
		{Name: "web", Image: "nginx", Environment: map[string]string{"BAD-NAME": "x"}},
		{Name: "web", Image: "nginx", Ports: []DockerPort{{Container: 70000}}},
	} {
		if err := validateDockerCreate(invalid); err == nil {
			t.Fatalf("invalid Docker request accepted: %+v", invalid)
		}
	}
	for _, value := range []string{"web", "web_1", "0123456789ab"} {
		if err := validateDockerObject(value); err != nil {
			t.Fatalf("valid container reference rejected: %s", value)
		}
	}
	for _, value := range []string{"", "web;id", "../web", "name with space"} {
		if err := validateDockerObject(value); err == nil {
			t.Fatalf("invalid container reference accepted: %s", value)
		}
	}
}

func TestDockerJSONLines(t *testing.T) {
	items, err := parseDockerLines("{\"ID\":\"abc\",\"State\":\"running\"}\n{\"ID\":\"def\",\"State\":\"exited\"}\n")
	if err != nil || len(items) != 2 || items[0]["ID"] != "abc" {
		t.Fatalf("unexpected parse result: %+v %v", items, err)
	}
	if _, err := parseDockerLines("not-json"); err == nil {
		t.Fatal("invalid Docker output accepted")
	}
}
