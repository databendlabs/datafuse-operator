// Copyright 2020-2021 The Datafuse Authors.
//
// SPDX-License-Identifier: Apache-2.0.
package ssa

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

const testYaml = `
apiVersion: apps/v1
kind: Deployment
metadata:
  name: nginx-test
  namespace: default
spec:
  selector:
    matchLabels:
      app: nginx
  template:
    metadata:
      labels:
        app: nginx
    spec:
      containers:
      - name: nginx
        image: nginx:latest
`

func TestServerSideApply(t *testing.T) {
	err := ServerSideApply(testYaml)
	assert.NoError(t, err)
}
