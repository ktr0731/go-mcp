package mcp

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"

	"github.com/ktr0731/go-mcp/protocol"
)

// ListResourcesResult represents the response for resources list.
// ListResourcesResult is a PaginatedResult that contains a list of resources the server offers.
type ListResourcesResult struct {
	// NextCursor is an opaque token representing the current pagination position.
	// If provided, the server should return results starting after this cursor.
	NextCursor string `json:"nextCursor,omitzero"`
	// Resources is a list of resources the server offers.
	Resources []Resource `json:"resources"`
}

// listResourceTemplatesResult represents the response for resource templates list.
// listResourceTemplatesResult is the server's response to a resources/templates/list request from the client.
type listResourceTemplatesResult struct {
	NextCursor        string             `json:"nextCursor,omitzero"`
	ResourceTemplates []ResourceTemplate `json:"resourceTemplates"`
}

// ServerResourceHandler is the interface for a server that can handle resource-related requests.
type ServerResourceHandler interface {
	// HandleResourcesList handles a resources/list request.
	HandleResourcesList(ctx context.Context) (*ListResourcesResult, error)
	// HandleResourcesRead handles a resources/read request.
	HandleResourcesRead(ctx context.Context, req *ReadResourceRequest) (*ReadResourceResult, error)
}

// ReadResourceRequest represents a request to read a specific resource.
// ReadResourceRequest is sent from the client to the server, to read a specific resource URI.
type ReadResourceRequest struct {
	protocol.Request

	// URI is the URI of the resource to read. The URI can use any protocol; it is up to the server how to interpret it.
	URI string `json:"uri"`
}

// ReadResourceResult represents the response for a resource read operation.
// ReadResourceResult is the server's response to a resources/read request from the client.
type ReadResourceResult struct {
	// Contents is a list of contents of the resource.
	Contents []ResourceContent `json:"contents"`
}

// Resource represents a resource handled by the server.
// Resource is a known resource that the server is capable of reading.
type Resource struct {
	// URI is the URI of this resource.
	URI string `json:"uri"` // URI (e.g. file://...)
	// Name is a human-readable name for this resource.
	// This can be used by clients to populate UI elements.
	Name string `json:"name"`
	// Description is a description of what this resource represents.
	// This can be used by clients to improve the LLM's understanding of available resources.
	// It can be thought of like a "hint" to the model.
	Description string `json:"description,omitzero"`
	// MimeType is the MIME type of this resource, if known.
	MimeType string `json:"mimeType,omitzero"`
	// Size is the size of the raw resource content, if known.
	// This can be used by Hosts to display file sizes and estimate context window usage.
	Size int64 `json:"size,omitzero"`

	// Annotations are optional annotations for the client.
	Annotations *Annotations `json:"annotations,omitzero"`
}

// ResourceTemplate represents a resource template definition.
// ResourceTemplate is a template description for resources available on the server.
type ResourceTemplate struct {
	// URITemplate is a URI template (according to RFC 6570) that can be used to construct resource URIs.
	URITemplate string `json:"uriTemplate"`
	// Name is a human-readable name for the type of resource this template refers to.
	// This can be used by clients to populate UI elements.
	Name string `json:"name"`
	// Description is a description of what this template is for.
	// This can be used by clients to improve the LLM's understanding of available resources.
	// It can be thought of like a "hint" to the model.
	Description string `json:"description,omitzero"`
	// MimeType is the MIME type for all resources that match this template. This should only be included
	// if all resources matching this template have the same type.
	MimeType string `json:"mimeType,omitzero"`

	// Annotations are optional annotations for the client.
	Annotations *Annotations `json:"annotations,omitzero"`
}

// ResourceContent is the interface for contents of a specific resource or sub-resource.
type ResourceContent interface {
	isResourceContent()
}

// TextResourceContent represents textual resource content.
type TextResourceContent struct {
	// URI is the URI of this resource.
	URI string
	// MimeType is the MIME type of this resource, if known.
	MimeType string
	// Text is the text of the item. This must only be set if the item can actually be represented as text (not binary data).
	Text string
}

func (t TextResourceContent) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		URI      string `json:"uri"`
		MimeType string `json:"mimeType,omitzero"`
		Text     string `json:"text,omitzero"`
	}{
		URI:      t.URI,
		MimeType: t.MimeType,
		Text:     t.Text,
	})
}

func (t TextResourceContent) isResourceContent() {}

// BlobResourceContent represents binary resource content.
type BlobResourceContent struct {
	// URI is the URI of this resource.
	URI string
	// MimeType is the MIME type of this resource, if known.
	MimeType string
	// Blob is the binary data of the item.
	Blob io.Reader
}

func (b BlobResourceContent) MarshalJSON() ([]byte, error) {
	var buf bytes.Buffer
	encoder := base64.NewEncoder(base64.StdEncoding, &buf)
	_, err := io.Copy(encoder, b.Blob)
	if err != nil {
		return nil, fmt.Errorf("failed to encode blob: %w", err)
	}

	return json.Marshal(struct {
		URI      string `json:"uri"`
		MimeType string `json:"mimeType,omitzero"`
		Data     string `json:"data"`
	}{
		URI:      b.URI,
		MimeType: b.MimeType,
		Data:     buf.String(),
	})
}

func (b BlobResourceContent) isResourceContent() {}

// listPromptsResult represents the response for prompts list.
// listPromptsResult is the server's response to a prompts/list request from the client.
type listPromptsResult struct {
	NextCursor string            `json:"nextCursor,omitzero"`
	Prompts    []protocol.Prompt `json:"prompts"`
}

// GetPromptResult represents the server's response to a prompts/get request from the client.
type GetPromptResult struct {
	// Description is an optional description for the prompt.
	Description string `json:"description,omitzero"`
	// Arguments is a list of arguments to use for templating the prompt.
	Messages []PromptMessage `json:"messages"`
}

// Role represents the sender or recipient of messages and data in a conversation.
type Role string

const (
	RoleUser      Role = "user"
	RoleAssistant Role = "assistant"
)

// PromptMessage describes a message returned as part of a prompt.
// PromptMessage is similar to SamplingMessage, but also supports the embedding of
// resources from the MCP server.
type PromptMessage struct {
	// Role represents the role of the message sender/recipient.
	Role Role `json:"role"`
	// Content represents the content of the message.
	// TextContent, ImageContent, AudioContent, or EmbeddedResource.
	Content PromptMessageContent `json:"content"`
}

// TextContent represents text data.
type TextContent struct {
	// Text is the text content of the message.
	Text string `json:"text"`

	// Annotations are optional annotations for the client.
	Annotations *Annotations `json:"annotations,omitzero"`
}

func (t TextContent) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		Type        string       `json:"type"`
		Text        string       `json:"text"`
		Annotations *Annotations `json:"annotations,omitzero"`
	}{
		Type:        "text",
		Text:        t.Text,
		Annotations: t.Annotations,
	})
}

func (t TextContent) isCallToolContent()      {}
func (t TextContent) isPromptMessageContent() {}

// ImageContent represents image data.
type ImageContent struct {
	// Data is the image data.
	Data io.Reader
	// MimeType is the MIME type of the image. Different providers may support different image types.
	MimeType string

	// Annotations are optional annotations for the client.
	Annotations *Annotations
}

func (i ImageContent) MarshalJSON() ([]byte, error) {
	var buf bytes.Buffer
	encoder := base64.NewEncoder(base64.StdEncoding, &buf)
	_, err := io.Copy(encoder, i.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to encode image: %w", err)
	}

	return json.Marshal(struct {
		Type        string       `json:"type"`
		MimeType    string       `json:"mimeType"`
		Data        string       `json:"data"`
		Annotations *Annotations `json:"annotations,omitzero"`
	}{
		Type:        "image",
		MimeType:    i.MimeType,
		Data:        buf.String(),
		Annotations: i.Annotations,
	})
}

func (i ImageContent) isPromptMessageContent() {}

// AudioContent represents audio data.
type AudioContent struct {
	// Data is the audio data.
	Data io.Reader
	// MimeType is the MIME type of the audio. Different providers may support different audio types.
	MimeType string

	// Annotations are optional annotations for the client.
	Annotations *Annotations
}

func (a AudioContent) MarshalJSON() ([]byte, error) {
	var buf bytes.Buffer
	encoder := base64.NewEncoder(base64.StdEncoding, &buf)
	_, err := io.Copy(encoder, a.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to encode audio: %w", err)
	}

	return json.Marshal(struct {
		Type        string       `json:"type"`
		MimeType    string       `json:"mimeType"`
		Data        string       `json:"data"`
		Annotations *Annotations `json:"annotations,omitzero"`
	}{
		Type:        "audio",
		MimeType:    a.MimeType,
		Data:        buf.String(),
		Annotations: a.Annotations,
	})
}

func (a AudioContent) isPromptMessageContent() {}

// EmbeddedResource represents the contents of a resource, embedded into a prompt or tool call result.
// EmbeddedResource is rendered by the client in a way that best serves the benefit of the LLM and/or the user.
type EmbeddedResource struct {
	// Resource is the resource content to embed.
	Resource ResourceContent `json:"resource"`

	// Annotations are optional annotations for the client.
	Annotations *Annotations `json:"annotations,omitzero"`
}

func (e EmbeddedResource) isCallToolContent()      {}
func (e EmbeddedResource) isPromptMessageContent() {}

// listToolsResult represents the server's response to a tools/list request from the client.
type listToolsResult struct {
	NextCursor string          `json:"nextCursor,omitzero"`
	Tools      []protocol.Tool `json:"tools"`
}

// CallToolContent is the interface for content that can be returned by a tool call.
// TextContent and EmbeddedResource are the only valid types.
type CallToolContent interface {
	isCallToolContent()
}

// PromptMessageContent is the interface for content that can be included in a prompt message.
// TextContent, ImageContent, AudioContent, or EmbeddedResource.
type PromptMessageContent interface {
	isPromptMessageContent()
}

// CallToolResult represents the server's response to a tool call.
// Any errors that originate from the tool SHOULD be reported inside the result
// object, with IsError set to true, NOT as an MCP protocol-level error
// response. Otherwise, the LLM would not be able to see that an error occurred
// and self-correct.
type CallToolResult struct {
	// Content is the content of the tool call.
	// TextContent and EmbeddedResource are the only valid types.
	Content []CallToolContent `json:"content"`
	// IsError indicates whether the tool call ended in an error.
	// If not set, this is assumed to be false (the call was successful).
	IsError bool `json:"isError,omitzero"`
}

// Annotations represents optional annotations for the client.
// Annotations are used by the client to inform how objects are used or displayed.
type Annotations struct {
	// Audience describes who the intended customer of this object or data is.
	// It can include multiple entries to indicate content useful for multiple audiences (e.g., ["user", "assistant"]).
	Audience []Role `json:"audience,omitzero"`
	// Priority describes how important this data is for operating the server.
	// A value of 1 means "most important," and indicates that the data is
	// effectively required, while 0 means "least important," and indicates that
	// the data is entirely optional.
	Priority *float64 `json:"priority,omitzero"` // 0: optional, 1: required
}

// subscribeResourceRequest represents the request to subscribe to a resource.
// subscribeResourceRequest is sent from the client to request resources/updated notifications from the server whenever a particular resource changes.
type subscribeResourceRequest struct {
	protocol.Request

	// URI is the URI of the resource to subscribe to. The URI can use any protocol; it is up to the server how to interpret it.
	URI string `json:"uri"`
}

// unsubscribeResourceRequest represents the request to unsubscribe from a resource.
// unsubscribeResourceRequest is sent from the client to request cancellation of resources/updated notifications from the server.
// This should follow a previous resources/subscribe request.
type unsubscribeResourceRequest struct {
	protocol.Request

	// URI is the URI of the resource to unsubscribe from.
	URI string `json:"uri"`
}

// CompleteResult represents the completion options for argument autocompletion.
type CompleteResult struct {
	// Values is an array of completion values. Must not exceed 100 items.
	Values []string `json:"values"`
	// Total is the total number of completion options available. This can exceed the number of values actually sent in the response.
	Total int `json:"total,omitzero"`
	// HasMore indicates whether there are additional completion options beyond those provided in the current response,
	// even if the exact total is unknown.
	HasMore bool `json:"hasMore,omitzero"`
}

// ServerCompletionHandler is the interface for a server that can handle completion requests.
type ServerCompletionHandler interface {
	// HandleComplete handles a completion (completion/complete) request.
	HandleComplete(ctx context.Context, req *protocol.CompleteRequestParams) (*CompleteResult, error)
}
