package config

import (
    "errors"
    "os"
    "strconv"
)

type CircuitBreakerConfig struct {
    MaxFailures uint32
    IntervalSec int
    TimeoutSec  int
}

type Config struct {
    Port             string
    JWTPublicKeyPath string
    RateLimitPerMin  int
    CircuitBreaker   CircuitBreakerConfig
    // static route mapping env e.g. SERVICES_JSON='{"auth":"http://auth:8000","user":"http://user:8000"}'
    ServicesJSON string
    ConsulAddr   string // optional
}

func LoadFromEnv() (*Config, error) {
    port := os.Getenv("GATEWAY_PORT")
    if port == "" {
        port = "8080"
    }
    jwtPath := os.Getenv("JWT_PUBLIC_KEY_PATH")
    if jwtPath == "" {
        return nil, errors.New("JWT_PUBLIC_KEY_PATH is required")
    }
    rlStr := os.Getenv("RATE_LIMIT_PER_MIN")
    rl := 60
    if rlStr != "" {
        if v, err := strconv.Atoi(rlStr); err == nil {
            rl = v
        }
    }
    maxFail := uint32(5)
    if s := os.Getenv("CB_MAX_FAILURES"); s != "" {
        if v, err := strconv.Atoi(s); err == nil {
            maxFail = uint32(v)
        }
    }
    interval := 60
    if s := os.Getenv("CB_INTERVAL_SEC"); s != "" {
        if v, err := strconv.Atoi(s); err == nil {
            interval = v
        }
    }
    timeout := 30
    if s := os.Getenv("CB_TIMEOUT_SEC"); s != "" {
        if v, err := strconv.Atoi(s); err == nil {
            timeout = v
        }
    }

    cfg := &Config{
        Port:             port,
        JWTPublicKeyPath: jwtPath,
        RateLimitPerMin:  rl,
        CircuitBreaker: CircuitBreakerConfig{
            MaxFailures: maxFail,
            IntervalSec: interval,
            TimeoutSec:  timeout,
        },
        ServicesJSON: os.Getenv("SERVICES_JSON"),
        ConsulAddr:   os.Getenv("CONSUL_ADDR"),
    }
    return cfg, nil
}
