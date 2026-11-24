package httpapp

import (
	"fmt"
	"net"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	Addr                          string
	UpstreamURL                   string
	CommandTimeout                time.Duration
	UpstreamTimeout               time.Duration
	UpstreamDialTimeout           time.Duration
	UpstreamResponseHeaderTimeout time.Duration
	UpstreamMaxBody               int64
	ReadHeaderTimeout             time.Duration
	IdleTimeout                   time.Duration
	MaxHeaderBytes                int
	ShutdownTimeout               time.Duration
}

func LoadConfig() (Config, error) {
	cfg := Config{
		Addr:        env("ADDR", "127.0.0.1:8080"),
		UpstreamURL: env("UPSTREAM_URL", "http://127.0.0.1:8081"),
	}
	host, port, err := net.SplitHostPort(cfg.Addr)
	if err != nil || strings.ContainsAny(host, "/ \t\r\n") {
		return Config{}, fmt.Errorf("SKILLSWAP_ADDR must be a host:port address")
	}
	if !validPort(port) {
		return Config{}, fmt.Errorf("SKILLSWAP_ADDR port must be between 1 and 65535")
	}
	upstream, err := url.Parse(cfg.UpstreamURL)
	if err != nil || (upstream.Scheme != "http" && upstream.Scheme != "https") ||
		upstream.Hostname() == "" || upstream.User != nil || upstream.RawQuery != "" ||
		upstream.ForceQuery || strings.Contains(cfg.UpstreamURL, "#") {
		return Config{}, fmt.Errorf("SKILLSWAP_UPSTREAM_URL must be an absolute HTTP(S) base URL without credentials, query, or fragment")
	}

	port = upstream.Port()
	if strings.HasSuffix(upstream.Host, ":") || (port != "" && !validPort(port)) {
		return Config{}, fmt.Errorf("SKILLSWAP_UPSTREAM_URL port must be between 1 and 65535")
	}

	durations := []struct {
		key, fallback string
		target        *time.Duration
	}{
		{"COMMAND_TIMEOUT", "2s", &cfg.CommandTimeout},
		{"UPSTREAM_TIMEOUT", "3s", &cfg.UpstreamTimeout},
		{"UPSTREAM_DIAL_TIMEOUT", "500ms", &cfg.UpstreamDialTimeout},
		{"UPSTREAM_RESPONSE_HEADER_TIMEOUT", "3s", &cfg.UpstreamResponseHeaderTimeout},
		{"READ_HEADER_TIMEOUT", "2s", &cfg.ReadHeaderTimeout},
		{"IDLE_TIMEOUT", "30s", &cfg.IdleTimeout},
		{"SHUTDOWN_TIMEOUT", "5s", &cfg.ShutdownTimeout},
	}
	for _, setting := range durations {
		value, err := time.ParseDuration(env(setting.key, setting.fallback))
		if err != nil || value <= 0 {
			return Config{}, fmt.Errorf("SKILLSWAP_%s must be a positive duration", setting.key)
		}
		*setting.target = value
	}
	// Leave space for the client's limit+1 byte overflow check.
	cfg.UpstreamMaxBody, err = strconv.ParseInt(env("UPSTREAM_MAX_BODY", "65536"), 10, 63)
	if err != nil || cfg.UpstreamMaxBody <= 0 {
		return Config{}, fmt.Errorf("SKILLSWAP_UPSTREAM_MAX_BODY must be a positive byte limit below 2^62")
	}
	headerBytes, err := strconv.ParseInt(env("MAX_HEADER_BYTES", "16384"), 10, 32)
	if err != nil || headerBytes <= 0 {
		return Config{}, fmt.Errorf("SKILLSWAP_MAX_HEADER_BYTES must be a positive byte limit below 2^31")
	}
	cfg.MaxHeaderBytes = int(headerBytes)
	return cfg, nil
}

func env(key, fallback string) string {
	if value, ok := os.LookupEnv("SKILLSWAP_" + key); ok {
		return value
	}
	return fallback
}

func validPort(port string) bool {
	number, err := strconv.ParseUint(port, 10, 16)
	return err == nil && number > 0
}
