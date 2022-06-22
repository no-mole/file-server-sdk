package ossClient

import (
	"time"
)

type ClientConfig struct {
	endpoint     string
	retryTimeout time.Duration
	timeout      time.Duration
	retryBackoff RetryBackoff
}

type RetryBackoff struct {
	waitBetween    time.Duration
	jitterFraction float64
}

func getDefaultClientConfig() *ClientConfig {
	config := new(ClientConfig)
	config.retryTimeout = time.Second * 2
	config.timeout = time.Second * 60
	config.retryBackoff.waitBetween = time.Second
	config.retryBackoff.jitterFraction = 0.2
	return config
}
