// Copyright 2023 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package languageserver

import (
	protocol "github.com/tliron/glsp/protocol_3_16"
)

type DocumentStore struct {
	documents map[string]*Document
}
type Document struct {
	URI               protocol.DocumentUri
	RunningDiagnostic bool
	Content           string
}

func newDocumentStore() *DocumentStore {
	return &DocumentStore{
		documents: map[string]*Document{},
	}
}
func (s *DocumentStore) Add(uri string, content string) *Document {
	doc := &Document{
		URI:     uri,
		Content: content,
	}
	s.documents[uri] = doc
	return doc
}
func (s *DocumentStore) Get(uri string) (*Document, bool) {
	d, ok := s.documents[uri]
	return d, ok
}
func (s *DocumentStore) Remove(uri string) {
	delete(s.documents, uri)
}
