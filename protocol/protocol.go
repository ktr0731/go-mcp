package protocol

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/xeipuuv/gojsonschema"
)

const (
	ProtocolVersion20250326 = "2025-03-26"
	ProtocolVersion20241105 = "2024-11-05"

	LatestProtocolVersion = ProtocolVersion20250326
)

const (
	MethodPing = "ping"

	MethodInitialize = "initialize"

	MethodPromptsList = "prompts/list"
	MethodPromptsGet  = "prompts/get"

	MethodToolsList = "tools/list"
	MethodToolsCall = "tools/call"

	MethodResourcesList         = "resources/list"
	MethodResourcesRead         = "resources/read"
	MethodResourceTemplatesList = "resources/templates/list"
	MethodResourcesSubscribe    = "resources/subscribe"
	MethodResourcesUnsubscribe  = "resources/unsubscribe"

	MethodNotificationsInitialized          = "notifications/initialized"
	MethodNotificationsResourcesListChanged = "notifications/resources/list_changed"
	MethodNotificationsResourcesUpdated     = "notifications/resources/updated"
	MethodNotificationsMessage              = "notifications/message"
	MethodNotificationsCancelled            = "notifications/cancelled"

	MethodCompletionComplete = "completion/complete"

	MethodLoggingSetLevel = "logging/setLevel"
)

const (
	LevelDebug     LogLevel = -4
	LevelInfo      LogLevel = 0
	LevelNotice    LogLevel = 1
	LevelWarning   LogLevel = 4
	LevelError     LogLevel = 8
	LevelCritical  LogLevel = 9
	LevelAlert     LogLevel = 10
	LevelEmergency LogLevel = 11
)

const (
	// CompletionReferenceTypePrompt identifies a prompt.
	CompletionReferenceTypePrompt CompletionReferenceType = "ref/prompt"
	// CompletionReferenceTypeResource is a reference to a resource or resource template definition.
	CompletionReferenceTypeResource CompletionReferenceType = "ref/resource"
)

var AvailableProtocolVersions = map[string]struct{}{
	ProtocolVersion20250326: {},
	ProtocolVersion20241105: {},
}

// Implementation describes the name and version of an MCP implementation.
type Implementation struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

// Request represents a common request structure.
type Request struct {
	Method string          `json:"method"`
	Params json.RawMessage `json:"params,omitzero"`
}

// PaginationParams represents pagination parameters.
type PaginationParams struct {
	// Cursor is an opaque token representing the current pagination position.
	// If provided, the server should return results starting after this cursor.
	Cursor string `json:"cursor,omitzero"`
}

// ServerHandlerFunc is an adapter to allow the use of functions as serverHandler implementations.
type ServerHandlerFunc[Req any] func(ctx context.Context, method string, req Req) (any, error)

func (f ServerHandlerFunc[Req]) Handle(ctx context.Context, method string, req Req) (any, error) {
	return f(ctx, method, req)
}

// ValidateByJSONSchema validates a document against a JSON schema.
func ValidateByJSONSchema(schema string, document any) error {
	schemaLoader := gojsonschema.NewStringLoader(schema)
	documentLoader := gojsonschema.NewGoLoader(document)
	result, err := gojsonschema.Validate(schemaLoader, documentLoader)
	if err != nil {
		return fmt.Errorf("failed to validate by JSON schema: %w", err)
	}
	if !result.Valid() {
		errs := make([]error, len(result.Errors()))
		for i := range result.Errors() {
			errs[i] = errors.New(result.Errors()[i].String())
		}
		return fmt.Errorf("invalid tool arguments: %w", errors.Join(errs...))
	}
	return nil
}

//
// Client-related Types
//

// InitializeRequestParams is sent from the client to the server when it first connects, asking it to begin initialization.
type InitializeRequestParams struct {
	// ProtocolVersion is the latest version of the Model Context Protocol that the client supports.
	// The client MAY decide to support older versions as well.
	ProtocolVersion string             `json:"protocolVersion"`
	Capabilities    ClientCapabilities `json:"capabilities"`
	ClientInfo      Implementation     `json:"clientInfo"`
}

// ClientCapabilities is a set of capabilities a client may support. Known capabilities are defined here, in this schema,
// but this is not a closed set: any client can define its own, additional capabilities.
type ClientCapabilities struct {
	// Experimental contains non-standard capabilities that the client supports.
	Experimental map[string]any `json:"experimental,omitzero"`
	// Roots is present if the client supports listing roots.
	Roots *RootsCapability `json:"roots,omitzero"`
}

// RootsCapability represents the client's capability to support roots features.
type RootsCapability struct {
	// ListChanged indicates whether the client supports notifications for changes to the roots list.
	ListChanged bool `json:"listChanged,omitzero"`
}

// CallToolRequestParams is used by the client to invoke a tool provided by the server.
type CallToolRequestParams struct {
	// Name is the name of the tool.
	Name string `json:"name"`
	// Arguments contains the arguments to use for the tool.
	Arguments json.RawMessage `json:"arguments,omitempty"`
}

// GetPromptRequestParams is used by the client to get a prompt provided by the server.
type GetPromptRequestParams struct {
	// Name is the name of the prompt or prompt template.
	Name string `json:"name"`
	// Arguments contains the arguments to use for templating the prompt.
	Arguments json.RawMessage `json:"arguments,omitempty"`
}

// NotificationsCancelledRequestParams is sent by either side to indicate that it is cancelling a previously-issued request.
type NotificationsCancelledRequestParams struct {
	// RequestID is the ID of the request to cancel.
	// This MUST correspond to the ID of a request previously issued in the same direction.
	RequestID string `json:"requestId"`
	// Reason is an optional string describing the reason for the cancellation. This MAY be logged or presented to the user.
	Reason string `json:"reason"`
}

// LoggingSetLevelRequestParams is a request from the client to the server, to enable or adjust logging.
type LoggingSetLevelRequestParams struct {
	// Level is the level of logging that the client wants to receive from the server.
	// The server should send all logs at this level and higher (i.e., more severe)
	// to the client as notifications/message.
	Level LogLevel `json:"level"`
}

// CompleteRequestParams is a request from the client to the server, to ask for completion options.
type CompleteRequestParams struct {
	// Ref is a reference to a prompt or resource
	Ref Reference `json:"ref"`
	// Argument contains the argument's information
	Argument CompletionArgument `json:"argument"`
}

//
// Server-related Types
//

// ServerCapabilities is a set of capabilities defined here, but this is not a closed set:
// any server can define its own, additional capabilities.
type ServerCapabilities struct {
	// Prompts is present if the server offers any prompt templates.
	Prompts *PromptCapability `json:"prompts,omitzero"`
	// Resources is present if the server offers any resources to read.
	Resources *ResourceCapability `json:"resources,omitzero"`
	// Tools is present if the server offers any tools to call.
	Tools *ToolCapability `json:"tools,omitzero"`
	// Experimental contains non-standard capabilities that the server supports.
	Experimental map[string]any `json:"experimental,omitzero"`
	// Logging is present if the server supports sending log messages to the client.
	Logging *LoggingCapability `json:"logging,omitzero"`
	// Completions is present if the server supports argument autocompletion suggestions.
	Completions *CompletionsCapability `json:"completions,omitzero"`
}

// InitializeResult is sent from the server after receiving an initialize request from the client.
type InitializeResult struct {
	// ProtocolVersion is the version of the Model Context Protocol that the server wants to use.
	// This may not match the version that the client requested.
	// If the client cannot support this version, it MUST disconnect.
	ProtocolVersion string             `json:"protocolVersion"`
	Capabilities    ServerCapabilities `json:"capabilities"`
	ServerInfo      Implementation     `json:"serverInfo"`
	// Instructions describe how to use the server and its features.
	Instructions string `json:"instructions,omitempty"`
}

// PromptCapability represents server capabilities for prompts.
type PromptCapability struct {
	// ListChanged indicates whether this server supports notifications for changes to the prompt list.
	ListChanged bool `json:"listChanged,omitzero"`
}

// ResourceCapability represents server capabilities for resources.
type ResourceCapability struct {
	// Subscribe indicates whether this server supports subscribing to resource updates.
	Subscribe bool `json:"subscribe,omitzero"`
	// ListChanged indicates whether this server supports notifications for changes to the resource list.
	ListChanged bool `json:"listChanged,omitzero"`
}

// ToolCapability represents server capabilities for tools.
type ToolCapability struct {
	// ListChanged indicates whether this server supports notifications for changes to the tool list.
	ListChanged bool `json:"listChanged,omitzero"`
}

// LoggingCapability represents server capability for logging.
type LoggingCapability struct{}

// CompletionsCapability represents server capability for completions.
type CompletionsCapability struct{}

//
// Feature-specific Types
//

// Logging Types

// LogLevel is the severity of a log message.
// These map to syslog message severities, as specified in RFC-5424.
type LogLevel int

// UnmarshalJSON implements json.Unmarshaler for LogLevel.
func (l *LogLevel) UnmarshalJSON(b []byte) error {
	switch string(b) {
	case `"debug"`:
		*l = LevelDebug
	case `"info"`:
		*l = LevelInfo
	case `"notice"`:
		*l = LevelNotice
	case `"warning"`:
		*l = LevelWarning
	case `"error"`:
		*l = LevelError
	case `"critical"`:
		*l = LevelCritical
	case `"alert"`:
		*l = LevelAlert
	case `"emergency"`:
		*l = LevelEmergency
	default:
		return fmt.Errorf("invalid log level: %s", string(b))
	}
	return nil
}

// Completion Types

// Reference represents a reference to a completion item.
type Reference struct {
	Type CompletionReferenceType `json:"type"`
	// Name is the name of the prompt or URI of the resource
	Name string `json:"name"`
}

// CompletionReferenceType represents the type of a completion reference.
type CompletionReferenceType string

// CompletionArgument represents an argument for completion.
type CompletionArgument struct {
	// Name is the name of the argument
	Name string `json:"name"`
	// Value is the value of the argument to use for completion matching.
	Value string `json:"value"`
}

// Tool Types

// Tool represents a definition for a tool the client can call.
type Tool struct {
	// Name is the name of the tool.
	Name string `json:"name"`
	// Description is a human-readable description of the tool.
	Description string `json:"description,omitzero"`
	// InputSchema is a JSON Schema object defining the expected parameters for the tool.
	InputSchema any `json:"inputSchema"`

	// Annotations contains optional additional tool information.
	Annotations *ToolAnnotations `json:"annotations,omitzero"`
}

// ToolAnnotations represents additional properties describing a Tool to clients.
// NOTE: all properties in ToolAnnotations are **hints**.
// They are not guaranteed to provide a faithful description of
// tool behavior (including descriptive properties like `title`).
type ToolAnnotations struct {
	// Title is a human-readable title for the tool.
	Title string `json:"title,omitzero"`
	// ReadOnlyHint indicates if the tool does not modify its environment.
	// Default: false
	ReadOnlyHint bool `json:"readOnlyHint,omitzero"`
	// DestructiveHint indicates if the tool may perform destructive updates to its environment.
	// If false, the tool performs only additive updates.
	// (This property is meaningful only when `readOnlyHint == false`)
	// Default: true
	DestructiveHint bool `json:"destructiveHint,omitzero"`
	// IdempotentHint indicates if calling the tool repeatedly with the same arguments
	// will have no additional effect on the its environment.
	// (This property is meaningful only when `readOnlyHint == false`)
	// Default: false
	IdempotentHint bool `json:"idempotentHint,omitzero"`
	// OpenWorldHint indicates if this tool may interact with an "open world" of external
	// entities. If false, the tool's domain of interaction is closed.
	// For example, the world of a web search tool is open, whereas that
	// of a memory tool is not.
	// Default: true
	OpenWorldHint bool `json:"openWorldHint,omitzero"`
}

// Prompt Types

// Prompt is a prompt or prompt template that the server offers.
type Prompt struct {
	// Name is the name of the prompt or prompt template.
	Name string `json:"name"`
	// Description is an optional description of what this prompt provides
	Description string `json:"description,omitzero"`
	// Arguments is a list of arguments to use for templating the prompt.
	Arguments []PromptArgument `json:"arguments,omitzero"`
}

// PromptArgument describes an argument that a prompt can accept.
type PromptArgument struct {
	// Name is the name of the argument.
	Name string `json:"name"`
	// Description is a human-readable description of the argument.
	Description string `json:"description,omitzero"`
	// Required indicates whether this argument must be provided.
	Required bool `json:"required,omitzero"`
}
