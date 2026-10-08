// Copyright 2026 Dave Shanley / Quobix / Princess Beef Heavy Industries, LLC
// SPDX-License-Identifier: MIT

package arazzo

import (
	"context"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/pb33f/go-yaml"
	validation "github.com/pb33f/libopenapi-validator/arazzo"
	source "github.com/pb33f/libopenapi/arazzo"
)

// SourceDocument retains authored text and nodes from a resolved source.
// It is used only during validation; findings retain small snippet copies.
type SourceDocument struct {
	Root    *yaml.Node
	Content string
	lines   []string
}

// Resolver retrieves source descriptions under the caller's lookup policy.
// The validator owns deduplication and aggregate limits. A Resolver belongs to
// one validation call and is not shared across executions.
type Resolver struct {
	AllowFiles  bool
	AllowRemote bool
	LocalFS     fs.FS
	BasePath    string
	Client      *http.Client
	Sources     map[string]*SourceDocument
	Files       int
	Bytes       int64
}

// Resolve reads one bounded source document and preserves its original nodes.
func (r *Resolver) Resolve(ctx context.Context, request source.SourceRequest) (*source.ResolvedSource, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	u, err := url.Parse(request.URL)
	if err != nil {
		return nil, err
	}
	retrievalURI := request.URL
	var body io.ReadCloser
	switch u.Scheme {
	case "http", "https":
		if !r.AllowRemote {
			return nil, source.ErrUnresolvedSourceDesc
		}
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
		if err != nil {
			return nil, err
		}
		client := r.Client
		if client == nil {
			client = http.DefaultClient
		}
		response, err := client.Do(req)
		if err != nil {
			return nil, err
		}
		if response.Request != nil && response.Request.URL != nil {
			retrievalURI = response.Request.URL.String()
		}
		body = response.Body
		defer body.Close()
		if response.StatusCode < 200 || response.StatusCode >= 300 {
			return nil, fmt.Errorf("source %s returned HTTP %d", u.Redacted(), response.StatusCode)
		}
	case "", "file":
		if !r.AllowFiles {
			return nil, source.ErrUnresolvedSourceDesc
		}
		if u.Host != "" && u.Host != "localhost" {
			return nil, fmt.Errorf("unsupported file host %q", u.Host)
		}
		path := filepath.FromSlash(u.Path)
		var file fs.File
		if r.LocalFS != nil {
			if filepath.IsAbs(path) {
				path, err = filepath.Rel(r.BasePath, path)
				if err != nil {
					return nil, err
				}
			}
			file, err = r.LocalFS.Open(filepath.ToSlash(path))
		} else {
			info, statErr := os.Stat(path)
			if statErr != nil {
				return nil, statErr
			}
			if !info.Mode().IsRegular() {
				return nil, fmt.Errorf("source %q is not a regular file", path)
			}
			file, err = os.Open(path)
		}
		if err != nil {
			return nil, err
		}
		body = file
		defer body.Close()
		info, err := file.Stat()
		if err != nil {
			return nil, err
		}
		if !info.Mode().IsRegular() {
			return nil, fmt.Errorf("source %q is not a regular file", path)
		}
	default:
		return nil, fmt.Errorf("unsupported source URL scheme %q", u.Scheme)
	}
	limit := int64(validation.DefaultLimits().MaxBytes)
	data, err := io.ReadAll(io.LimitReader(body, limit+1))
	if err != nil {
		return nil, err
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if int64(len(data)) > limit {
		return nil, fmt.Errorf("source %q exceeds the %d byte limit", request.URL, limit)
	}
	root, err := parse(data)
	if err != nil {
		return nil, fmt.Errorf("source %q: %w", request.URL, err)
	}
	if r.Sources == nil {
		r.Sources = make(map[string]*SourceDocument)
	}
	document := &SourceDocument{Root: root, Content: string(data)}
	r.Sources[request.URL] = document
	r.Sources[retrievalURI] = document
	r.Files++
	r.Bytes += int64(len(data))
	return &source.ResolvedSource{Name: request.Name, RetrievalURI: retrievalURI, RootNode: root, SourceBytes: data}, nil
}

// DocumentURI resolves a CLI filename against its explicit base. File names
// become absolute file URIs so nested Arazzo sources have an unambiguous base.
func DocumentURI(filename, base string) (string, error) {
	if strings.Contains(filename, "://") {
		if base == "" {
			return filename, nil
		}
		u, err := url.Parse(filename)
		if err != nil {
			return "", err
		}
		filename = filepath.Base(u.Path)
	}
	if strings.HasPrefix(base, "http://") || strings.HasPrefix(base, "https://") {
		u, err := url.Parse(strings.TrimRight(base, "/") + "/")
		if err != nil {
			return "", err
		}
		return u.ResolveReference(&url.URL{Path: filepath.Base(filename)}).String(), nil
	}
	if filename == "" {
		filename = "stdin.yaml"
	}
	if base != "" {
		filename = filepath.Join(base, filepath.Base(filename))
	}
	path, err := filepath.Abs(filename)
	if err != nil {
		return "", err
	}
	return (&url.URL{Scheme: "file", Path: filepath.ToSlash(path)}).String(), nil
}
