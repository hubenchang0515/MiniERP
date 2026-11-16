package config

import (
	"bytes"
	"errors"
	"flag"
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
	if err != nil || c.Addr != "0.0.0.0:8080" || c.JWTTTL != 168*time.Hour || !filepath.IsAbs(c.DataDir) {
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
func TestPrecedence(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.toml")
	os.WriteFile(path, []byte("[server]\naddr='127.0.0.1:9000'\n[data]\ndir='./from-file'\n[auth]\njwt_secret='file-secret'\njwt_ttl='2h'"), 0600)
	env := map[string]string{"MINIERP_CONFIG_FILE": path, "MINIERP_ADDR": "127.0.0.1:9001", "MINIERP_JWT_SECRET": "env-secret", "MINIERP_JWT_TTL": "3h"}
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
