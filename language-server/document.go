// Copyright 2024 Princess Beef Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT
// https://pb33f.io
//
// This code was originally written by KDanisme (https://github.com/KDanisme) and was submitted as a PR
// to the vacuum project. It then was modified by Dave Shanley to fit the needs of the vacuum project.
// The original code can be found here:
// https://github.com/KDanisme/vacuum/tree/language-server
//
// I (Dave Shanley) do not know what happened to KDasnime, or why the PR was
// closed, but I am grateful for the contribution.
//
// This feature is why I built vacuum. This is the reason for its existence.

package languageserver

import (
	"sync"

	protocol "github.com/tliron/glsp/protocol_3_16"
)

type DocumentStore struct {
	documents map[string]*Document
	mu        sync.RWMutex
}
type Document struct {
	URI               protocol.DocumentUri
	RunningDiagnostic bool
	Content           string
	mu                sync.RWMutex // Protects Content field
}

func newDocumentStore() *DocumentStore {
	return &DocumentStore{
		documents: map[string]*Document{},
	}
}
func (s *DocumentStore) Add(uri string, content string) *Document {
	s.mu.Lock()
	defer s.mu.Unlock()

	doc := &Document{
		URI:     uri,
		Content: content,
	}
	s.documents[uri] = doc
	return doc
}

func (s *DocumentStore) Get(uri string) (*Document, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	d, ok := s.documents[uri]
	return d, ok
}

func (s *DocumentStore) Remove(uri string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.documents, uri)
}
