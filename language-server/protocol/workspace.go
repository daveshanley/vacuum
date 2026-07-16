// Copyright 2026 Princess Beef Heavy Industries, LLC / Dave Shanley
// SPDX-License-Identifier: MIT

package protocol

const (
	// MethodWorkspaceDidChangeConfiguration reports changed client configuration.
	MethodWorkspaceDidChangeConfiguration = Method("workspace/didChangeConfiguration")
	// MethodWorkspaceDidChangeWorkspaceFolders reports workspace-folder changes.
	MethodWorkspaceDidChangeWorkspaceFolders = Method("workspace/didChangeWorkspaceFolders")
	// MethodWorkspaceExecuteCommand requests command execution.
	MethodWorkspaceExecuteCommand = Method("workspace/executeCommand")
	// ServerClientRegisterCapability dynamically registers a client capability.
	ServerClientRegisterCapability = Method("client/registerCapability")
	// ServerWorkspaceConfiguration requests scoped client configuration.
	ServerWorkspaceConfiguration = Method("workspace/configuration")
)

// WorkspaceFolder identifies a named workspace root.
type WorkspaceFolder struct {
	URI  DocumentUri `json:"uri"`
	Name string      `json:"name"`
}

// WorkspaceFoldersChangeEvent contains added and removed workspace folders.
type WorkspaceFoldersChangeEvent struct {
	Added   []WorkspaceFolder `json:"added"`
	Removed []WorkspaceFolder `json:"removed"`
}

// DidChangeWorkspaceFoldersParams contains a workspace-folder change.
type DidChangeWorkspaceFoldersParams struct {
	Event WorkspaceFoldersChangeEvent `json:"event"`
}

// DidChangeConfigurationParams contains client configuration settings.
type DidChangeConfigurationParams struct {
	Settings any `json:"settings"`
}

// Registration describes a dynamically registered capability.
type Registration struct {
	ID              string `json:"id"`
	Method          string `json:"method"`
	RegisterOptions any    `json:"registerOptions,omitempty"`
}

// RegistrationParams contains dynamic capability registrations.
type RegistrationParams struct {
	Registrations []Registration `json:"registrations"`
}

// ConfigurationItem requests configuration scoped to a URI and section.
type ConfigurationItem struct {
	ScopeURI *DocumentUri `json:"scopeUri,omitempty"`
	Section  *string      `json:"section,omitempty"`
}

// ConfigurationParams contains configuration items requested from the client.
type ConfigurationParams struct {
	Items []ConfigurationItem `json:"items"`
}

// ExecuteCommandParams identifies a command and its arguments.
type ExecuteCommandParams struct {
	Command   string `json:"command"`
	Arguments []any  `json:"arguments,omitempty"`
}
