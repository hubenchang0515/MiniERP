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
	AIIdleTimeout                          time.Duration
	AIMaxConcurrentRequests                int
}

const DefaultAIIdleTimeout = 2 * time.Minute
const DefaultAIMaxConcurrentRequests = 32

func (c Config) ProxyIdleTimeout() time.Duration {
	if c.AIIdleTimeout > 0 {
		return c.AIIdleTimeout
	}
	return DefaultAIIdleTimeout
}

func (c Config) ProxyConcurrency() int {
	if c.AIMaxConcurrentRequests > 0 {
		return c.AIMaxConcurrentRequests
	}
	return DefaultAIMaxConcurrentRequests
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
	AI struct {
		IdleTimeout           string `toml:"idle_timeout"`
		MaxConcurrentRequests *int   `toml:"max_concurrent_requests"`
	} `toml:"ai"`
}

func Load(args []string, getenv func(string) string, out io.Writer) (Config, error) {
	c := Config{Addr: "0.0.0.0:8080", DataDir: "./data", JWTSecret: "minioa-insecure-default-secret-change-me", JWTTTL: 168 * time.Hour, SecretSource: "内置默认值"}
	fs := flag.NewFlagSet("minioa", flag.ContinueOnError)
	fs.SetOutput(out)
	path, addr, dir, secret, ttl := "", "", "", "", ""
	idleTimeout := ""
	maxConcurrent := DefaultAIMaxConcurrentRequests
	fs.StringVar(&idleTimeout, "ai-idle-timeout", "", "AI 无数据传输超时（默认 2m，持续输出不会超时）")
	fs.IntVar(&maxConcurrent, "ai-max-concurrent-requests", DefaultAIMaxConcurrentRequests, "AI 最大并发转发数（默认 32）")
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
		path = getenv("MINIOA_CONFIG_FILE")
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
	idleValue := DefaultAIIdleTimeout.String()
	c.AIMaxConcurrentRequests = DefaultAIMaxConcurrentRequests
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
		if f.AI.IdleTimeout != "" {
			idleValue = f.AI.IdleTimeout
		}
		if f.AI.MaxConcurrentRequests != nil {
			c.AIMaxConcurrentRequests = *f.AI.MaxConcurrentRequests
		}
	}
	if v := getenv("MINIOA_ADDR"); v != "" {
		c.Addr = v
	}
	if v := getenv("MINIOA_DATA_DIR"); v != "" {
		c.DataDir = v
	}
	if v := getenv("MINIOA_JWT_SECRET"); v != "" {
		c.JWTSecret = v
		c.SecretSource = "环境变量"
	}
	if v := getenv("MINIOA_JWT_TTL"); v != "" {
		ttlValue = v
	}
	if v := getenv("MINIOA_AI_IDLE_TIMEOUT"); v != "" {
		idleValue = v
	}
	if v := getenv("MINIOA_AI_MAX_CONCURRENT_REQUESTS"); v != "" && !visited["ai-max-concurrent-requests"] {
		c.AIMaxConcurrentRequests, err = strconv.Atoi(v)
		if err != nil {
			return c, fmt.Errorf("AI 最大并发转发数必须是正整数")
		}
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
	if visited["ai-idle-timeout"] {
		idleValue = idleTimeout
	}
	if visited["ai-max-concurrent-requests"] {
		c.AIMaxConcurrentRequests = maxConcurrent
	}
	c.AIIdleTimeout, err = time.ParseDuration(idleValue)
	if err != nil || c.AIIdleTimeout <= 0 {
		return c, fmt.Errorf("AI 无数据传输超时必须是正时间长度")
	}
	if c.AIMaxConcurrentRequests <= 0 {
		return c, fmt.Errorf("AI 最大并发转发数必须是正整数")
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
