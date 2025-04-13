package codegen

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"reflect"
	"slices"
	"strconv"
	"strings"

	"github.com/invopop/jsonschema"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
	"golang.org/x/tools/imports"
)

// ServerCapabilities represents the capabilities that a server may support.
//
// https://modelcontextprotocol.io/specification/2025-03-26/basic/lifecycle
type ServerCapabilities struct {
	// Prompts is present if the server offers any prompt templates.
	Prompts *PromptCapability `json:"prompts,omitempty"`
	// Resources is present if the server offers any resources to read.
	Resources *ResourceCapability `json:"resources,omitempty"`
	// Tools is present if the server offers any tools to call.
	Tools *ToolCapability `json:"tools,omitempty"`
	// Completions is present if the server supports argument autocompletion suggestions.
	Completions *CompletionsCapability `json:"completions,omitempty"`
	// Logging is present if the server supports sending log messages to the client.
	Logging *LoggingCapability `json:"logging,omitempty"`
}

// PromptCapability represents server capability for prompts.
type PromptCapability struct {
	// ListChanged is always false. See README.md for more details.
}

// ResourceCapability represents server capability for resources.
type ResourceCapability struct {
	// Subscribe indicates whether this server supports subscribing to resource updates.
	Subscribe bool `json:"subscribe,omitempty"`
	// ListChanged indicates whether this server supports notifications for changes to the resource list.
	ListChanged bool `json:"listChanged,omitempty"`
}

// ToolCapability represents server capability for tools.
type ToolCapability struct {
	// ListChanged is always false. See README.md for more details.
}

// LoggingCapability represents server capability for logging.
type LoggingCapability struct{}

// CompletionsCapability represents server capability for completions.
type CompletionsCapability struct{}

// Implementation describes the name and version of an MCP implementation.
type Implementation struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

// Prompt represents a prompt or prompt template that the server offers.
type Prompt struct {
	// Name is the name of the prompt or prompt template.
	Name string `json:"name"`
	// Description is an optional description of what this prompt provides.
	Description string `json:"description,omitempty"`
	// Arguments is a list of arguments to use for templating the prompt.
	Arguments []PromptArgument `json:"arguments,omitempty"`
}

// PromptArgument describes an argument that a prompt can accept.
type PromptArgument struct {
	// Name is the name of the argument.
	Name string `json:"name"`
	// Description is a human-readable description of the argument.
	Description string `json:"description,omitempty"`
	// Required indicates whether this argument must be provided.
	Required bool `json:"required,omitempty"`
}

// Tool represents a definition for a tool the client can call.
type Tool struct {
	// Name is the name of the tool.
	Name string `json:"name"`
	// Description is a human-readable description of the tool.
	// This can be used by clients to improve the LLM's understanding of available tools.
	// It can be thought of like a "hint" to the model.
	Description string `json:"description,omitempty"`
	// InputSchema is a Go struct that represents the input schema of the tool.
	// The struct fields can specify JSON tags supported by https://github.com/invopop/jsonschema.
	// See README.md or examples directory for more details.
	InputSchema any `json:"inputSchema"`
}

// ResourceTemplate represents a template description for resources available on the server.
type ResourceTemplate struct {
	// URITemplate is a URI template (according to RFC 6570) that can be used to construct resource URIs.
	URITemplate string `json:"uriTemplate"`
	// Name is a human-readable name for the type of resource this template refers to.
	// This can be used by clients to populate UI elements.
	Name string `json:"name"`
	// Description is a description of what this template is for.
	// This can be used by clients to improve the LLM's understanding of available resources.
	// It can be thought of like a "hint" to the model.
	Description string `json:"description,omitempty"`
	// MimeType is the MIME type for all resources that match this template. This should only be included
	// if all resources matching this template have the same type.
	MimeType string `json:"mimeType,omitempty"`
}

// ServerDefinition represents the definition of an MCP server.
type ServerDefinition struct {
	// Capabilities defines the capabilities that this server supports.
	Capabilities ServerCapabilities
	// Implementation contains information about this server implementation.
	Implementation Implementation

	// Prompts is the list of prompts offered by this server.
	Prompts []Prompt
	// ResourceTemplates is the list of resource templates offered by this server.
	ResourceTemplates []ResourceTemplate
	// Tools is the list of tools offered by this server.
	Tools []Tool
}

// Generate generates the server code from the server definition.
// See README.md or examples directory for more details.
func Generate(w io.Writer, def *ServerDefinition, pkgName string) error {
	if w == nil {
		w = os.Stdout
	}
	if pkgName == "" {
		pkgName = "mcpgen"
	}

	return (&generator{
		def: def,
		pkg: pkgName,
	}).generate(w)
}

type generator struct {
	buf strings.Builder
	def *ServerDefinition

	pkg string
}

func (g *generator) generate(w io.Writer) error {
	g.println("// Code generated by mcp-codegen. DO NOT EDIT.")
	g.println("package " + g.pkg)

	g.println("import (")
	g.println(`	"context"`)
	g.println(`	"encoding/json"`)
	g.println(`	"fmt"`)
	g.println(`	"slices"`)
	g.println(`	"strconv"`)
	g.println(`	mcp "github.com/ktr0731/go-mcp"`)
	g.println(`	"github.com/ktr0731/go-mcp/protocol"`)
	g.println(")")

	// Generate prompt handlers and input types
	g.generatePromptHandlers()

	// Resource list
	g.generateResourceTemplateList()

	// Tool handlers and input types
	g.generateToolHandlers()

	// Prompt list
	g.generatePromptList()

	// Tool list
	g.generateToolList()

	// NewHandler
	g.generateNewHandler()

	out := []byte(g.buf.String())

	b, err := imports.Process("", out, &imports.Options{
		AllErrors: true,
		Comments:  true,
		TabIndent: true,
		TabWidth:  8,
	})
	if err != nil {
		return err
	}

	if _, err := w.Write(b); err != nil {
		return err
	}

	return nil
}

// generatePromptHandlers generates prompt handlers and input types.
func (g *generator) generatePromptHandlers() {
	g.println("// ServerPromptHandler is the interface for prompt handlers.")
	g.println("type ServerPromptHandler interface {")
	for _, prompt := range g.def.Prompts {
		promptName := pascalCase(prompt.Name)
		g.println("	HandlePrompt" + promptName + "(ctx context.Context, req *Prompt" + promptName + "Request) (*mcp.GetPromptResult, error)")
	}
	g.println("}")
	g.println("")

	for _, prompt := range g.def.Prompts {
		promptName := pascalCase(prompt.Name)
		g.println("// Prompt" + promptName + "Request contains input parameters for the " + prompt.Name + " prompt.")
		g.println("type Prompt" + promptName + "Request struct {")
		for _, arg := range prompt.Arguments {
			argName := pascalCase(arg.Name)
			g.println("	" + argName + " string `json:\"" + arg.Name + "\"`")
		}
		g.println("}")
		g.println("")
	}
}

// getEnumFields extracts enum fields from a tool's input schema
func (g *generator) getEnumFields(tool Tool) map[string][]any {
	reflector := jsonschema.Reflector{}
	schema := reflector.Reflect(tool.InputSchema)
	schemaJSON, err := schema.MarshalJSON()
	if err != nil {
		panic(err)
	}

	var schemaMap map[string]any
	if err := json.Unmarshal(schemaJSON, &schemaMap); err != nil {
		panic(err)
	}

	// Track fields with enum values to generate custom types
	enumFields := make(map[string][]any)

	// Check for enum values in properties
	if props, ok := schemaMap["properties"].(map[string]any); ok {
		for propName, propDef := range props {
			if propMap, ok := propDef.(map[string]any); ok {
				if enumValues, ok := propMap["enum"].([]any); ok && len(enumValues) > 0 {
					enumFields[propName] = enumValues
				}
			}
		}
	}

	return enumFields
}

// getEnumType determines the appropriate type for an enum based on its values
func (g *generator) getEnumType(enumValues []any) string {
	// Default to string
	enumType := "string"

	// Check if all values are integers
	allInts := true
	for _, val := range enumValues {
		_, isFloat := val.(float64)
		if !isFloat {
			allInts = false
			break
		}
		// In JSON, all numbers are float64, but we need to check if they're integers
		floatVal := val.(float64)
		if floatVal != float64(int(floatVal)) {
			allInts = false
			break
		}
	}

	if allInts {
		enumType = "int"
	}

	return enumType
}

// generateToolHandlers generates tool handlers and input types.
func (g *generator) generateToolHandlers() {
	if len(g.def.Tools) == 0 {
		return
	}

	g.println("// ServerToolHandler is the interface for tool handlers.")
	g.println("type ServerToolHandler interface {")
	for _, tool := range g.def.Tools {
		toolName := pascalCase(tool.Name)
		g.println("	HandleTool" + toolName + "(ctx context.Context, req *Tool" + toolName + "Request) (*mcp.CallToolResult, error)")
	}
	g.println("}")
	g.println("")

	for _, tool := range g.def.Tools {
		toolName := pascalCase(tool.Name)

		// Extract enum fields
		enumFields := g.getEnumFields(tool)

		// Sort field names to ensure consistent generation order
		fieldNames := make([]string, 0, len(enumFields))
		for fieldName := range enumFields {
			fieldNames = append(fieldNames, fieldName)
		}
		slices.Sort(fieldNames)

		// Generate custom type for each enum field
		for _, fieldName := range fieldNames {
			enumValues := enumFields[fieldName]
			enumTypeName := toolName + pascalCase(fieldName) + "Type"
			enumType := g.getEnumType(enumValues)

			// Generate type definition
			g.println("// " + enumTypeName + " represents possible values for " + fieldName)
			g.println("type " + enumTypeName + " " + enumType)
			g.println("")

			// Generate constants
			g.println("const (")

			// Sort enum values for consistent generation order
			sortedEnumValues := make([]any, len(enumValues))
			copy(sortedEnumValues, enumValues)
			if enumType == "int" {
				slices.SortFunc(sortedEnumValues, func(a, b any) int {
					aVal := int(a.(float64))
					bVal := int(b.(float64))
					return aVal - bVal
				})
			} else {
				slices.SortFunc(sortedEnumValues, func(a, b any) int {
					aStr := fmt.Sprintf("%v", a)
					bStr := fmt.Sprintf("%v", b)
					return strings.Compare(aStr, bStr)
				})
			}

			for _, val := range sortedEnumValues {
				strVal := fmt.Sprintf("%v", val)
				constName := pascalCase(strVal)

				if enumType == "int" {
					// For integer enums, don't quote the value
					intVal := int(val.(float64))
					g.println("	" + enumTypeName + constName + " " + enumTypeName + " = " + strconv.Itoa(intVal))
				} else {
					// For string enums, quote the value
					g.println("	" + enumTypeName + constName + " " + enumTypeName + " = \"" + strVal + "\"")
				}
			}
			g.println(")")
			g.println("")
		}

		g.println("// Tool" + toolName + "Request contains input parameters for the " + tool.Name + " tool.")
		g.println("type Tool" + toolName + "Request struct {")

		rt := reflect.TypeOf(tool.InputSchema)
		// Generate fields from JSONSchema
		for i := 0; i < rt.NumField(); i++ {
			field := rt.Field(i)
			fieldName := field.Name
			fieldType := field.Type.String()
			jsonTag := field.Tag.Get("json")

			// If this field has enum values, use the custom type
			if _, hasEnum := enumFields[jsonTag]; hasEnum {
				enumTypeName := toolName + pascalCase(jsonTag) + "Type"
				g.println("	" + fieldName + " " + enumTypeName + " `json:\"" + jsonTag + "\"`")
			} else {
				g.println("	" + fieldName + " " + fieldType + " `json:\"" + jsonTag + "\"`")
			}
		}

		g.println("}")
		g.println("")
	}
}

// generatePromptList generates the list of available prompts.
func (g *generator) generatePromptList() {
	g.println("// PromptList contains all available prompts.")
	g.println("var PromptList = []protocol.Prompt{")
	for _, prompt := range g.def.Prompts {
		g.println("	{")
		g.println("		Name: \"" + prompt.Name + "\",")
		g.println("		Description: \"" + prompt.Description + "\",")
		g.println("		Arguments: []protocol.PromptArgument{")
		for _, arg := range prompt.Arguments {
			g.println("			{")
			g.println("				Name: \"" + arg.Name + "\",")
			g.println("				Description: \"" + arg.Description + "\",")
			if arg.Required {
				g.println("				Required: true,")
			}
			g.println("			},")
		}
		g.println("		},")
		g.println("	},")
	}
	g.println("}")
	g.println("")
}

// generateToolList generates the list of available tools.
func (g *generator) generateToolList() {
	if len(g.def.Tools) == 0 {
		return
	}

	reflector := jsonschema.Reflector{}
	g.println("// JSON Schema type definitions generated from inputSchema")
	g.println("var (")
	for _, tool := range g.def.Tools {
		schema := reflector.Reflect(tool.InputSchema)
		b, err := schema.MarshalJSON()
		if err != nil {
			panic(err)
		}
		g.println("	Tool" + pascalCase(tool.Name) + "InputSchema = json.RawMessage(`" + string(b) + "`)")
	}
	g.println(")")

	g.println("// ToolList contains all available tools.")
	g.println("var ToolList = []protocol.Tool{")
	for _, tool := range g.def.Tools {
		g.println("	{")
		g.printf("		Name: %q,\n", tool.Name)
		g.printf("		Description: %q,\n", tool.Description)
		g.printf("		InputSchema: Tool%sInputSchema,\n", pascalCase(tool.Name))
		g.println("	},")
	}
	g.println("}")
	g.println("")
}

// generateResourceTemplateList generates the list of available ResourceTemplates.
func (g *generator) generateResourceTemplateList() {
	if len(g.def.ResourceTemplates) == 0 {
		return
	}

	g.println("// ResourceTemplateList contains all available ResourceTemplates.")
	g.println("var ResourceTemplateList = []mcp.ResourceTemplate{")
	for _, resourceTemplate := range g.def.ResourceTemplates {
		g.println("	{")
		g.println("		URITemplate: \"" + resourceTemplate.URITemplate + "\",")
		g.println("		Name: \"" + resourceTemplate.Name + "\",")
		g.println("		Description: \"" + resourceTemplate.Description + "\",")
		if resourceTemplate.MimeType != "" {
			g.println("		MimeType: \"" + resourceTemplate.MimeType + "\",")
		}
		g.println("	},")
	}
	g.println("}")
	g.println("")
}

// generateNewHandler generates the NewHandler function.
func (g *generator) generateNewHandler() {
	g.println("// NewHandler creates a new MCP handler.")

	var handlerParams []string
	// Generate handler parameters
	if g.def.Capabilities.Prompts != nil {
		handlerParams = append(handlerParams, "promptHandler ServerPromptHandler")
	}
	if g.def.Capabilities.Resources != nil {
		handlerParams = append(handlerParams, "resourceHandler mcp.ServerResourceHandler")
	}
	if g.def.Capabilities.Tools != nil {
		handlerParams = append(handlerParams, "toolHandler ServerToolHandler")
	}
	if g.def.Capabilities.Completions != nil {
		handlerParams = append(handlerParams, "completionHandler mcp.ServerCompletionHandler")
	}

	g.println("func NewHandler(" + strings.Join(handlerParams, ", ") + ") *mcp.Handler {")
	g.println("	h := &mcp.Handler{}")
	g.println("	h.Capabilities = protocol.ServerCapabilities{")
	if g.def.Capabilities.Prompts != nil {
		g.println("		Prompts: &protocol.PromptCapability{},")
	}
	if g.def.Capabilities.Resources != nil {
		g.println("		Resources: &protocol.ResourceCapability{")
		g.println("			Subscribe: " + strconv.FormatBool(g.def.Capabilities.Resources.Subscribe) + ",")
		g.println("			ListChanged: " + strconv.FormatBool(g.def.Capabilities.Resources.ListChanged) + ",")
		g.println("		},")
	}
	if g.def.Capabilities.Tools != nil {
		g.println("		Tools: &protocol.ToolCapability{},")
	}
	if g.def.Capabilities.Completions != nil {
		g.println("		Completions: &protocol.CompletionsCapability{},")
	}
	if g.def.Capabilities.Logging != nil {
		g.println("		Logging: &protocol.LoggingCapability{},")
	}
	g.println("	}")
	g.println("	h.Implementation = protocol.Implementation{")
	g.println("		Name: \"" + g.def.Implementation.Name + "\",")
	g.println("		Version: \"" + g.def.Implementation.Version + "\",")
	g.println("	}")

	// Set prompt handler
	if g.def.Capabilities.Prompts != nil {
		g.println("	h.Prompts = PromptList")
		g.println("	h.PromptHandler = protocol.ServerHandlerFunc[protocol.GetPromptRequestParams](func(ctx context.Context, method string, req protocol.GetPromptRequestParams) (any, error) {")
		g.println("		switch method {")
		g.println("		case \"prompts/get\":")
		g.println("			switch req.Name {")
		for _, prompt := range g.def.Prompts {
			promptName := pascalCase(prompt.Name)
			g.println("			case \"" + prompt.Name + "\":")
			g.println("				var in Prompt" + promptName + "Request")
			g.println("				if err := json.Unmarshal(req.Arguments, &in); err != nil {")
			g.println("					return nil, err")
			g.println("				}")
			g.println("				return promptHandler.HandlePrompt" + promptName + "(ctx, &in)")
		}
		g.println("			default:")
		g.println("				return nil, fmt.Errorf(\"prompt not found: %s\", req.Name)")
		g.println("			}")
		g.println("		default:")
		g.println("			return nil, fmt.Errorf(\"method %s not found\", method)")
		g.println("		}")
		g.println("	})")
	}

	// Set resource handler
	if g.def.Capabilities.Resources != nil {
		g.println("	h.ResourceHandler = resourceHandler")
	}
	// Set resource templates
	if g.def.Capabilities.Resources != nil {
		g.println("	h.ResourceTemplates = ResourceTemplateList")
	}

	// Set tool handler
	if g.def.Capabilities.Tools != nil {
		g.println("	h.Tools = ToolList")
		g.println("	h.ToolHandler = protocol.ServerHandlerFunc[protocol.CallToolRequestParams](func(ctx context.Context, method string, req protocol.CallToolRequestParams) (any, error) {")
		g.println("		idx := slices.IndexFunc(ToolList, func(t protocol.Tool) bool {")
		g.println("			return t.Name == req.Name")
		g.println("		})")
		g.println("		if idx == -1 {")
		g.println("			return nil, fmt.Errorf(\"tool not found: %s\", req.Name)")
		g.println("		}")
		g.println("		switch method {")
		g.println("		case \"tools/call\":")
		g.println("			switch req.Name {")
		for _, tool := range g.def.Tools {
			toolName := pascalCase(tool.Name)
			g.println("			case \"" + tool.Name + "\":")
			g.println("				var in Tool" + toolName + "Request")
			g.println("				if err := json.Unmarshal(req.Arguments, &in); err != nil {")
			g.println("					return nil, err")
			g.println("				}")

			// Keep the schema validation for all fields
			g.println("				inputSchema, _ := ToolList[idx].InputSchema.(json.RawMessage)")
			g.println("				if err := protocol.ValidateByJSONSchema(string(inputSchema), in); err != nil {")
			g.println("					return nil, err")
			g.println("				}")
			g.println("				return toolHandler.HandleTool" + toolName + "(ctx, &in)")
		}
		g.println("			default:")
		g.println("				return nil, fmt.Errorf(\"tool not found: %s\", req.Name)")
		g.println("			}")
		g.println("		default:")
		g.println("			return nil, fmt.Errorf(\"method %s not found\", method)")
		g.println("		}")
		g.println("	})")
	}

	// Set completion handler
	if g.def.Capabilities.Completions != nil {
		g.println("	h.CompletionHandler = completionHandler")
	}

	g.println("	return h")
	g.println("}")
}

// pascalCase converts prompt.Name to PascalCase
// e.g. "prompt_name" -> "PromptName"
func pascalCase(name string) string {
	name = strings.NewReplacer(";", "_", " ", "").Replace(name)
	words := strings.Split(name, "_")
	title := cases.Title(language.Und)
	for i, word := range words {
		words[i] = title.String(word)
	}
	return strings.Join(words, "")
}

func (g *generator) println(s string) error {
	_, err := fmt.Fprintln(&g.buf, s)
	return err
}

func (g *generator) printf(format string, args ...any) error {
	_, err := fmt.Fprintf(&g.buf, format, args...)
	return err
}
