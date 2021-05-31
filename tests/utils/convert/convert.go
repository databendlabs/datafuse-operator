// Copyright 2020-2021 The Datafuse Authors.
//
// SPDX-License-Identifier: Apache-2.0.
package convert

func Int32ToPtr(num int32) *int32 {
	return &num
}

func StringToPtr(str string) *string {
	return &str
}
