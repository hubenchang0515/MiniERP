package config

import (
	"bytes"
	"errors"
	"flag"
	"io"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestDefaultsAndHelp(t *testing.T) {
	var out bytes.Buffer
	c, err := Load([]string{"-c", filepath.Join(t.TempDir(), "missing")}, func(string) string { return "" }, &out)
	if err == nil {
		t.Fatal("explicit missing file accepted")
	}
	_ = c
	c, err = Load(nil, func(string) string { return "" }, &out)
	if err != nil || c.Addr != "0.0.0.0:8080" || c.JWTTTL != 168*time.Hour || !filepath.IsAbs(c.DataDir) || c.AIIdleTimeout != 2*time.Minute || c.AIMaxConcurrentRequests != 32 {
		t.Fatalf("defaults: %+v %v", c, err)
	}
	_, err = Load([]string{"--help"}, func(string) string { return "" }, &out)
	if !errors.Is(err, flag.ErrHelp) {
		t.Fatal(err)
	}
	if !bytes.Contains(out.Bytes(), []byte("jwt-secret")) {
		t.Fatal("help missing options")
	}
}

func TestAIConfiguration(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.toml")
	if err := os.WriteFile(path, []byte("[ai]\nidle_timeout='3m'\nmax_concurrent_requests=16\n"), 0600); err != nil {
		t.Fatal(err)
	}
	for _, test := range []struct {
		name string
		args []string
		env  map[string]string
		idle time.Duration
		max  int
	}{
		{"file", nil, nil, 3 * time.Minute, 16},
		{"environment", nil, map[string]string{"MINIOA_AI_IDLE_TIMEOUT": "4m", "MINIOA_AI_MAX_CONCURRENT_REQUESTS": "24"}, 4 * time.Minute, 24},
		{"arguments", []string{"--ai-idle-timeout", "5m", "--ai-max-concurrent-requests", "48"}, map[string]string{"MINIOA_AI_IDLE_TIMEOUT": "invalid", "MINIOA_AI_MAX_CONCURRENT_REQUESTS": "invalid"}, 5 * time.Minute, 48},
	} {
		t.Run(test.name, func(t *testing.T) {
			args := append([]string{"-c", path}, test.args...)
			c, err := Load(args, func(k string) string { return test.env[k] }, io.Discard)
			if err != nil || c.AIIdleTimeout != test.idle || c.AIMaxConcurrentRequests != test.max {
				t.Fatalf("config=%+v err=%v", c, err)
			}
		})
	}
}

func TestInvalidAIConfiguration(t *testing.T) {
	for _, args := range [][]string{
		{"--ai-idle-timeout", "0s"}, {"--ai-idle-timeout", "bad"},
		{"--ai-max-concurrent-requests", "0"}, {"--ai-max-concurrent-requests", "-1"},
	} {
		if _, err := Load(args, func(string) string { return "" }, io.Discard); err == nil {
			t.Fatalf("accepted %v", args)
		}
	}
	for _, text := range []string{"0", "-1", "bad"} {
		getenv := func(name string) string {
			if name == "MINIOA_AI_MAX_CONCURRENT_REQUESTS" {
				return text
			}
			return ""
		}
		if _, err := Load(nil, getenv, io.Discard); err == nil {
			t.Fatalf("accepted concurrency %q", text)
		}
	}
	path := filepath.Join(t.TempDir(), "zero.toml")
	if err := os.WriteFile(path, []byte("[ai]\nmax_concurrent_requests=0\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := Load([]string{"-c", path}, func(string) string { return "" }, io.Discard); err == nil {
		t.Fatal("accepted explicit zero concurrency in file")
	}
}
func TestPrecedence(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.toml")
	os.WriteFile(path, []byte("[server]\naddr='127.0.0.1:9000'\n[data]\ndir='./from-file'\n[auth]\njwt_secret='file-secret'\njwt_ttl='2h'"), 0600)
	env := map[string]string{"MINIOA_CONFIG_FILE": path, "MINIOA_ADDR": "127.0.0.1:9001", "MINIOA_JWT_SECRET": "env-secret", "MINIOA_JWT_TTL": "3h"}
	get := func(k string) string { return env[k] }
	var out bytes.Buffer
	c, err := Load([]string{"--addr", "127.0.0.1:9002", "-t", "4h", "-s", "cli-secret"}, get, &out)
	if err != nil || c.Addr != "127.0.0.1:9002" || c.JWTSecret != "cli-secret" || c.JWTTTL != 4*time.Hour || c.SecretSource != "命令行" {
		t.Fatalf("%+v %v", c, err)
	}
	c, err = Load(nil, get, &out)
	if err != nil || c.JWTSecret != "env-secret" || c.JWTTTL != 3*time.Hour {
		t.Fatalf("%+v %v", c, err)
	}
	c, err = Load([]string{"-c", path}, func(string) string { return "" }, &out)
	if err != nil || c.JWTSecret != "file-secret" || c.SecretSource != "TOML 文件" {
		t.Fatalf("%+v %v", c, err)
	}
}
func TestInvalidConfiguration(t *testing.T) {
	for _, args := range [][]string{{"--jwt-ttl", "0h"}, {"-t", "-1h"}, {"-t", "bad"}, {"-a", "invalid"}, {"-a", ":abc"}, {"-a", ":65536"}, {"-d", ""}, {"-s", ""}, {"--unknown"}, {"unexpected"}} {
		t.Run(args[0], func(t *testing.T) {
			_, err := Load(args, func(string) string { return "" }, &bytes.Buffer{})
			if err == nil {
				t.Fatal("accepted invalid configuration")
			}
		})
	}
}
func TestMalformedTOML(t *testing.T) {
	path := filepath.Join(t.TempDir(), "bad.toml")
	os.WriteFile(path, []byte("[broken"), 0600)
	if _, err := Load([]string{"-c", path}, func(string) string { return "" }, &bytes.Buffer{}); err == nil {
		t.Fatal("invalid TOML accepted")
	}
}
