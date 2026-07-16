// Copyright 2026 Princess Beef Heavy Industries, LLC / Dave Shanley
// SPDX-License-Identifier: MIT

package protocol

const (
	// MethodInitialize starts an LSP session.
	MethodInitialize = Method("initialize")
	// MethodInitialized confirms that client initialization is complete.
	MethodInitialized = Method("initialized")
	// MethodShutdown requests an orderly server shutdown.
	MethodShutdown = Method("shutdown")
	// MethodExit terminates the LSP connection.
	MethodExit = Method("exit")
)

// InitializeParams contains client capabilities and workspace initialization state.
type InitializeParams struct {
	ProcessID             *Integer           `json:"processId"`
	ClientInfo            *ClientInfo        `json:"clientInfo,omitempty"`
	Locale                *string            `json:"locale,omitempty"`
	RootPath              *string            `json:"rootPath,omitempty"`
	RootURI               *DocumentUri       `json:"rootUri"`
	InitializationOptions any                `json:"initializationOptions,omitempty"`
	Capabilities          ClientCapabilities `json:"capabilities"`
	Trace                 *TraceValue        `json:"trace,omitempty"`
	WorkspaceFolders      []WorkspaceFolder  `json:"workspaceFolders,omitempty"`
}

// ClientInfo identifies the connected language client.
type ClientInfo struct {
	Name    string  `json:"name"`
	Version *string `json:"version,omitempty"`
}

// ClientCapabilities contains the capabilities used by Vacuum.
type ClientCapabilities struct {
	Workspace *WorkspaceClientCapabilities `json:"workspace,omitempty"`
}

// WorkspaceClientCapabilities contains workspace features used by Vacuum.
type WorkspaceClientCapabilities struct {
	DidChangeConfiguration *DidChangeConfigurationClientCapabilities `json:"didChangeConfiguration,omitempty"`
	WorkspaceFolders       *bool                                     `json:"workspaceFolders,omitempty"`
	Configuration          *bool                                     `json:"configuration,omitempty"`
}

// DidChangeConfigurationClientCapabilities describes dynamic configuration registration.
type DidChangeConfigurationClientCapabilities struct {
	DynamicRegistration *bool `json:"dynamicRegistration,omitempty"`
}

// InitializeResult contains Vacuum's advertised capabilities and identity.
type InitializeResult struct {
	Capabilities ServerCapabilities          `json:"capabilities"`
	ServerInfo   *InitializeResultServerInfo `json:"serverInfo,omitempty"`
}

// InitializeResultServerInfo identifies the Vacuum language server.
type InitializeResultServerInfo struct {
	Name    string  `json:"name"`
	Version *string `json:"version,omitempty"`
}

// InitializedParams is the empty initialized notification payload.
type InitializedParams struct{}

// ServerCapabilities contains the server features advertised by Vacuum.
type ServerCapabilities struct {
	TextDocumentSync       any                          `json:"textDocumentSync,omitempty"`
	CompletionProvider     *CompletionOptions           `json:"completionProvider,omitempty"`
	CodeActionProvider     any                          `json:"codeActionProvider,omitempty"`
	ExecuteCommandProvider *ExecuteCommandOptions       `json:"executeCommandProvider,omitempty"`
	Workspace              *ServerCapabilitiesWorkspace `json:"workspace,omitempty"`
	Experimental           any                          `json:"experimental,omitempty"`
}

// CompletionOptions advertises completion support without additional options.
type CompletionOptions struct{}

// CodeActionOptions advertises supported code-action kinds.
type CodeActionOptions struct {
	CodeActionKinds []CodeActionKind `json:"codeActionKinds,omitempty"`
	ResolveProvider *bool            `json:"resolveProvider,omitempty"`
}

// ExecuteCommandOptions advertises commands handled by Vacuum.
type ExecuteCommandOptions struct {
	Commands []string `json:"commands"`
}

// ServerCapabilitiesWorkspace contains workspace-specific server features.
type ServerCapabilitiesWorkspace struct {
	WorkspaceFolders *WorkspaceFoldersServerCapabilities `json:"workspaceFolders,omitempty"`
}

// WorkspaceFoldersServerCapabilities advertises workspace-folder support.
type WorkspaceFoldersServerCapabilities struct {
	Supported           *bool         `json:"supported"`
	ChangeNotifications *BoolOrString `json:"changeNotifications,omitempty"`
}
