package main

import (
	"testing"

	"github.com/handlename/ssmwrap/v2"
	"github.com/stretchr/testify/require"
)

func TestBuildSSMWrapExportRules(t *testing.T) {
	cases := []struct {
		name     string
		paths    string
		names    string
		expected []ssmwrap.ExportRule
	}{
		{
			name:     "empty",
			expected: nil,
		},
		{
			name:  "paths with and without trailing slash",
			paths: "/shimesaba/prod/,/shimesaba/common",
			expected: []ssmwrap.ExportRule{
				{Path: "/shimesaba/prod/**/*"},
				{Path: "/shimesaba/common/**/*"},
			},
		},
		{
			name:  "names",
			names: "/shimesaba/MACKEREL_APIKEY,/shimesaba/OTHER",
			expected: []ssmwrap.ExportRule{
				{Path: "/shimesaba/MACKEREL_APIKEY"},
				{Path: "/shimesaba/OTHER"},
			},
		},
		{
			name:  "paths and names",
			paths: "/shimesaba/prod",
			names: "/shimesaba/MACKEREL_APIKEY",
			expected: []ssmwrap.ExportRule{
				{Path: "/shimesaba/prod/**/*"},
				{Path: "/shimesaba/MACKEREL_APIKEY"},
			},
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			require.Equal(t, c.expected, buildSSMWrapExportRules(c.paths, c.names))
		})
	}
}

func TestIsLambda(t *testing.T) {
	cases := []struct {
		name       string
		execEnv    string
		runtimeAPI string
		expected   bool
	}{
		{name: "local", expected: false},
		{name: "AWS_EXECUTION_ENV", execEnv: "AWS_Lambda_provided.al2023", expected: true},
		{name: "AWS_LAMBDA_RUNTIME_API", runtimeAPI: "127.0.0.1:9001", expected: true},
		{name: "other execution env", execEnv: "AWS_ECS_FARGATE", expected: false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Setenv("AWS_EXECUTION_ENV", c.execEnv)
			t.Setenv("AWS_LAMBDA_RUNTIME_API", c.runtimeAPI)
			require.Equal(t, c.expected, isLambda())
		})
	}
}
