// Copyright 2020-2021 The Datafuse Authors.
//
// SPDX-License-Identifier: Apache-2.0.
package retry

import (
	"fmt"
	"testing"
	"time"
)

type Config struct {
	err      error
	timeout  time.Duration
	interval time.Duration
	coverage int32
}

func NewConfig() *Config {
	return &Config{}
}

func (c *Config) SetTimeOut(timeout time.Duration) *Config {
	c.timeout = timeout
	return c
}

func (c *Config) SetInterval(interval time.Duration) *Config {
	c.interval = interval
	return c
}

func (c *Config) SetCoverage(cov int32) *Config {
	c.coverage = cov
	return c
}

type RetriableFunc func() (result interface{}, completed bool, err error)

// timeout: total time for the retry period
// interval: time timeval between two tests
// coverage: should success coverage times for completion
// err: latest err
var defaultConfig = Config{
	timeout:  30 * time.Second,
	interval: 3 * time.Second,
	coverage: 0,
	err:      nil,
}

// Do function will retry given func util timeout
func Do(fn RetriableFunc, arg Config) (result interface{}, err error) {
	cfg := arg
	to := time.After(cfg.timeout)
	successes := 0
	total := 0
	var latesterr error = nil
	for {
		select {
		case <-to:
			return nil, fmt.Errorf("timeout while waiting after %d attempts (last error: %v)", total, latesterr)
		default:
		}
		result, completed, err := fn()
		total++
		if completed {
			if err != nil {
				successes = 0
			} else {
				successes++
			}
			if successes >= int(cfg.coverage) {
				return result, err
			}
			continue
		} else {
			successes = 0
		}

		if err != nil {
			latesterr = err
		}

		select {
		case <-to:
			convergeStr := ""
			if cfg.coverage > 1 {
				convergeStr = fmt.Sprintf(", %d/%d successes", successes, cfg.coverage)
			}
			return nil, fmt.Errorf("timeout while waiting after %d attempts%s (last error: %v)", total, convergeStr, latesterr)
		case <-time.After(cfg.interval):
		}
	}
}

func UntilSuccess(fn func() error, arg Config) error {
	_, e := Do(func() (interface{}, bool, error) {
		err := fn()
		if err != nil {
			return nil, false, err
		}

		return nil, true, nil
	}, arg)
	return e
}

func UntilSuccessOrFail(t *testing.T, fn func() error, arg Config) {
	err := UntilSuccess(fn, arg)
	if err != nil {
		t.Fatalf("retry function failed")
	}
}
