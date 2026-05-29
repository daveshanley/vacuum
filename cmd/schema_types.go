// Copyright 2020-2026 Dave Shanley / Quobix / Princess Beef Heavy Industries, LLC
// https://quobix.com/vacuum/ | https://pb33f.io
// SPDX-License-Identifier: MIT

package cmd

import (
	"net/http"

	"github.com/daveshanley/vacuum/model"
	"github.com/pb33f/libopenapi/datamodel"
	"go.yaml.in/yaml/v4"
)

type schemaLintFlags struct {
	GlobPatterns         []string
	Includes             []string
	Excludes             []string
	Stdin                bool
	Format               string
	Output               string
	Details              bool
	Snippets             bool
	ErrorsOnly           bool
	Silent               bool
	NoStyle              bool
	NoBanner             bool
	NoMessage            bool
	AllResults           bool
	NoClip               bool
	ShowRules            bool
	FailSeverity         string
	IgnoreFile           string
	Base                 string
	Remote               bool
	Timeout              int
	LookupTimeout        int
	Ruleset              string
	Functions            string
	Time                 bool
	Debug                bool
	ExtRefs              bool
	CertFile             string
	KeyFile              string
	CAFile               string
	Insecure             bool
	AllowPrivateNetworks bool
	AllowHTTP            bool
	FetchTimeout         int
	IgnoreArrayCircleRef bool
	IgnorePolyCircleRef  bool
	ResolveAllRefs       bool
	NestedRefsDocContext bool
	OutputAbsPaths       bool
	Bundle               bool
}

type schemaBundleFlags struct {
	Stdin     bool
	Stdout    bool
	Output    string
	Format    string
	Delimiter string
	NoStyle   bool
	Base      string
	Remote    bool
	CertFile  string
	KeyFile   string
	CAFile    string
	Insecure  bool
}

type schemaInput struct {
	Path       string
	Display    string
	Bytes      []byte
	Base       string
	MirrorRoot string
	FromStdin  bool
}

type schemaLintRun struct {
	Input     schemaInput
	ResultSet *model.RuleResultSet
	SpecInfo  *datamodel.SpecInfo
	Errors    []error
}

type schemaBundleContext struct {
	rootFormat  string
	rootDefs    *yaml.Node
	delimiter   string
	cache       map[string]string
	usedKeys    map[string]bool
	warnings    []string
	httpClient  *http.Client
	allowRemote bool
}
