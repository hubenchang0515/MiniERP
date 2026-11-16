package config

import (
	"flag"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/pelletier/go-toml/v2"
)

type Config struct {
	Addr, DataDir, JWTSecret, SecretSource string
	JWTTTL                                 time.Duration
}
type fileConfig struct {
	Server struct {
		Addr string `toml:"addr"`
	} `toml:"server"`
	Data struct {
		Dir string `toml:"dir"`
	} `toml:"data"`
	Auth struct {
		Secret string `toml:"jwt_secret"`
		TTL    string `toml:"jwt_ttl"`
	} `toml:"auth"`
}

func Load(args []string, getenv func(string) string, out io.Writer) (Config, error) {
	c := Config{Addr: "0.0.0.0:8080", DataDir: "./data", JWTSecret: "minierp-insecure-default-secret-change-me", JWTTTL: 168 * time.Hour, SecretSource: "内置默认值"}
	fs := flag.NewFlagSet("minierp", flag.ContinueOnError)
	fs.SetOutput(out)
	path, addr, dir, secret, ttl := "", "", "", "", ""
	for _, n := range []string{"c", "config"} {
		fs.StringVar(&path, n, "", "TOML 配置文件（默认 ./config.toml）")
	}
	for _, n := range []string{"a", "addr"} {
		fs.StringVar(&addr, n, "", "监听地址（默认 0.0.0.0:8080）")
	}
	for _, n := range []string{"d", "data-dir"} {
		fs.StringVar(&dir, n, "", "数据目录（默认 ./data）")
	}
	for _, n := range []string{"s", "jwt-secret"} {
		fs.StringVar(&secret, n, "", "JWT 密钥（建议通过环境变量设置）")
	}
	for _, n := range []string{"t", "jwt-ttl"} {
		fs.StringVar(&ttl, n, "", "JWT 有效期（默认 168h）")
	}
	if err := fs.Parse(args); err != nil {
		return c, err
	}
	if fs.NArg() != 0 {
		return c, fmt.Errorf("不接受位置参数")
	}
	visited := map[string]bool{}
	fs.Visit(func(f *flag.Flag) { visited[f.Name] = true })
	explicit := visited["c"] || visited["config"]
	if !explicit {
		path = getenv("MINIERP_CONFIG_FILE")
		explicit = path != ""
	}
	if path == "" {
		if explicit {
			return c, fmt.Errorf("配置路径不能为空")
		}
		path = "./config.toml"
	}
	b, err := os.ReadFile(path)
	if err != nil && (explicit || !os.IsNotExist(err)) {
		return c, err
	}
	ttlValue := "168h"
	if err == nil {
		var f fileConfig
		if err = toml.Unmarshal(b, &f); err != nil {
			return c, err
		}
		if f.Server.Addr != "" {
			c.Addr = f.Server.Addr
		}
		if f.Data.Dir != "" {
			c.DataDir = f.Data.Dir
		}
		if f.Auth.Secret != "" {
			c.JWTSecret = f.Auth.Secret
			c.SecretSource = "TOML 文件"
		}
		if f.Auth.TTL != "" {
			ttlValue = f.Auth.TTL
		}
	}
	if v := getenv("MINIERP_ADDR"); v != "" {
		c.Addr = v
	}
	if v := getenv("MINIERP_DATA_DIR"); v != "" {
		c.DataDir = v
	}
	if v := getenv("MINIERP_JWT_SECRET"); v != "" {
		c.JWTSecret = v
		c.SecretSource = "环境变量"
	}
	if v := getenv("MINIERP_JWT_TTL"); v != "" {
		ttlValue = v
	}
	if visited["a"] || visited["addr"] {
		c.Addr = addr
	}
	if visited["d"] || visited["data-dir"] {
		c.DataDir = dir
	}
	if visited["s"] || visited["jwt-secret"] {
		c.JWTSecret = secret
		c.SecretSource = "命令行"
	}
	if visited["t"] || visited["jwt-ttl"] {
		ttlValue = ttl
	}
	_, port, err := net.SplitHostPort(c.Addr)
	if err != nil {
		return c, fmt.Errorf("监听地址无效: %w", err)
	}
	portNumber, err := strconv.Atoi(port)
	if err != nil || portNumber < 1 || portNumber > 65535 {
		return c, fmt.Errorf("监听端口必须为 1–65535")
	}
	if c.DataDir == "" || c.JWTSecret == "" {
		return c, fmt.Errorf("数据目录和密钥不能为空")
	}
	c.JWTTTL, err = time.ParseDuration(ttlValue)
	if err != nil || c.JWTTTL <= 0 {
		return c, fmt.Errorf("JWT 有效期必须是正时间长度")
	}
	c.DataDir, err = filepath.Abs(c.DataDir)
	return c, err
}
