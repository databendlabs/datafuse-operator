// Copyright 2020-2021 The Datafuse Authors.
//
// SPDX-License-Identifier: Apache-2.0.

package controller

const (
	ControllerAgentName = "datafuse-operator"
	// SuccessSynced is used as part of the Event 'reason' when a Tenant is synced
	SuccessSynced = "Synced"
	// ErrResourceExists is used as part of the Event 'reason' when a Tenant fails
	// to sync due to a StatefulSet of the same name already existing.
	ErrResourceExists = "ErrResourceExists"

	// MessageResourceSynced is the message used for an Event fired when a Operator
	// is synced successfully
	MessageResourceSynced              = "datafuse operator synced successfully"
	queueTokenRefillRate               = 50
	queueTokenBucketSize               = 500
	OperatorLabel                      = "datafuse-operator"
	ComputeGroupLabel                  = "datafuse-computegroup"
	ComputeGroupRoleLabel              = "datafuse-computegrouprole"
	ComputeGroupRoleLeader             = "leader"
	ComputeGroupRoleFollower           = "follower"
	InstanceContainerName              = "fusequery"
	FUSE_QUERY_NUM_CPUS                = "FUSE_QUERY_NUM_CPUS"
	FUSE_QUERY_MYSQL_HANDLER_HOST      = "FUSE_QUERY_MYSQL_HANDLER_HOST"
	FUSE_QUERY_MYSQL_HANDLER_PORT      = "FUSE_QUERY_MYSQL_HANDLER_PORT"
	FUSE_QUERY_CLICKHOUSE_HANDLER_HOST = "FUSE_QUERY_CLICKHOUSE_HANDLER_HOST"
	FUSE_QUERY_CLICKHOUSE_HANDLER_PORT = "FUSE_QUERY_CLICKHOUSE_HANDLER_PORT"
	FUSE_QUERY_RPC_API_ADDRESS         = "FUSE_QUERY_FLIGHT_API_ADDRESS"
	FUSE_QUERY_HTTP_API_ADDRESS        = "FUSE_QUERY_HTTP_API_ADDRESS"
	FUSE_QUERY_METRIC_API_ADDRESS      = "FUSE_QUERY_METRIC_API_ADDRESS"
	FUSE_QUERY_PRIORITY                = "FUSE_QUERY_PRIORITY"
	ContainerHTTPPort                  = "http"
	ContainerMetricsPort               = "metrics"
	ContainerRPCPort                   = "rpc"
	ContainerMysqlPort                 = "mysql"
	ContainerClickhousePort            = "clickhouse"
)
