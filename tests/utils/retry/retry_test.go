// Copyright 2020-2021 The Datafuse Authors.
//
// SPDX-License-Identifier: Apache-2.0.
package retry

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestUntilSuccess(t *testing.T) {
	cfg := Config{
		interval: 1 * time.Millisecond,
		timeout:  10 * time.Millisecond,
		coverage: 2,
	}
	t.Run("success", func(t *testing.T) {
		success := 0
		retryFunc := func() error {
			if success < 2 {
				success++
				return fmt.Errorf("not success")
			}
			return nil
		}
		err := UntilSuccess(retryFunc, cfg)
		assert.Equal(t, nil, err)
		assert.Equal(t, 2, success)
	})
	t.Run("failed", func(t *testing.T) {
		success := 0
		retryFunc := func() error {
			if success < 100 {
				success++
				return fmt.Errorf("not success")
			}
			return nil
		}
		err := UntilSuccess(retryFunc, cfg)
		assert.Contains(t, err.Error(), "timeout")
		assert.Equal(t, true, success <= 10)
	})
}
