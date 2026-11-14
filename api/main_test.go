package main

import (
	"context"
	"flag"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestAPIRejectsOccupiedPortBeforeStartingServices(t *testing.T) {
	const configEnv = "AUTOOPS_TEST_OCCUPIED_PORT_CONFIG"
	if configPath := os.Getenv(configEnv); configPath != "" {
		flag.CommandLine = flag.NewFlagSet("autoops", flag.ExitOnError)
		os.Args = []string{"autoops", "-c", configPath, "-database-worker", "external"}
		main()
		return
	}

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()
	configPath := filepath.Join(t.TempDir(), "config.yaml")
	// Deliberately omit database/Redis settings: the bind check must happen first.
	configData := "server:\n  address: " + listener.Addr().String() + "\n  model: debug\n"
	if err := os.WriteFile(configPath, []byte(configData), 0600); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, os.Args[0], "-test.run=^TestAPIRejectsOccupiedPortBeforeStartingServices$")
	cmd.Env = append(os.Environ(), configEnv+"="+configPath)
	output, err := cmd.CombinedOutput()
	if ctx.Err() != nil {
		t.Fatalf("API kept running after failing to bind: %s", output)
	}
	if err == nil {
		t.Fatalf("API must exit unsuccessfully on an occupied port: %s", output)
	}
	if !strings.Contains(string(output), "API 启动失败：无法监听 "+listener.Addr().String()) {
		t.Fatalf("expected an actionable listen error before service initialization: %s", output)
	}
}
