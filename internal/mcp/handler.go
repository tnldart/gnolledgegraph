package mcp

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"gnolledgegraph/internal/db"
)

// JSON-RPC 2.0 message types
type JSONRPCRequest struct {
	JSONRPC string      `json:"jsonrpc"`
	ID      interface{} `json:"id,omitempty"`
	Method  string      `json:"method"`
	Params  interface{} `json:"params,omitempty"`
}

type JSONRPCResponse struct {
	JSONRPC string        `json:"jsonrpc"`
	ID      interface{}   `json:"id"` // Removed omitempty
	Result  interface{}   `json:"result,omitempty"`
	Error   *JSONRPCError `json:"error,omitempty"`
}

type JSONRPCError struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

type JSONRPCNotification struct {
	JSONRPC string      `json:"jsonrpc"`
	Method  string      `json:"method"`
	Params  interface{} `json:"params,omitempty"`
}

// MCP protocol types
type InitializeRequest struct {
	ProtocolVersion string             `json:"protocolVersion"`
	Capabilities    ClientCapabilities `json:"capabilities"`
	ClientInfo      ClientInfo         `json:"clientInfo"`
}

type ClientCapabilities struct {
	Sampling *struct{} `json:"sampling,omitempty"`
}

type ClientInfo struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

type InitializeResult struct {
	ProtocolVersion string             `json:"protocolVersion"`
	Capabilities    ServerCapabilities `json:"capabilities"`
	ServerInfo      ServerInfo         `json:"serverInfo"`
}

type ServerCapabilities struct {
	Tools *struct{} `json:"tools,omitempty"`
}

type ServerInfo struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

type ToolsListResult struct {
	Tools []Tool `json:"tools"`
}

type Tool struct {
	Name        string      `json:"name"`
	Description string      `json:"description"`
	InputSchema InputSchema `json:"inputSchema"`
}

type InputSchema struct {
	Type       string              `json:"type"`
	Properties map[string]Property `json:"properties"`
	Required   []string            `json:"required"`
}

type Property struct {
	Type        string `json:"type"`
	Description string `json:"description"`
}

type ToolCallRequest struct {
	Name      string                 `json:"name"`
	Arguments map[string]interface{} `json:"arguments"`
}

type ToolCallResult struct {
	Content []ToolContent `json:"content"`
	IsError bool          `json:"isError,omitempty"`
}

type ToolContent struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

// MCP SSE Session represents an active MCP session over SSE
type MCPSession struct {
	sessionID   string
	writer      http.ResponseWriter
	flusher     http.Flusher
	messageChan chan JSONRPCRequest
	done        chan bool
	initialized bool
}

type MCPSessionManager struct {
	sessions map[string]*MCPSession
	mu       sync.RWMutex
}

var sessionManager = &MCPSessionManager{
	sessions: make(map[string]*MCPSession),
}

// NewMCPHandler creates a new MCP handler that supports both GET (SSE) and POST (messages)
func NewMCPHandler(database *sql.DB) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Security: validate origin to prevent DNS rebinding attacks
		origin := r.Header.Get("Origin")
		if origin != "" && !strings.HasPrefix(origin, "http://localhost") && !strings.HasPrefix(origin, "http://127.0.0.1") {
			http.Error(w, "invalid origin", http.StatusForbidden)
			return
		}

		// Route based on path and method
		switch {
		case r.URL.Path == "/sse" && r.Method == http.MethodGet:
			handleSSEConnection(database, w, r)
		case r.URL.Path == "/messages" && r.Method == http.MethodPost:
			handleJSONRPCMessage(database, w, r)
		case r.URL.Path == "/mcp" && r.Method == http.MethodGet:
			// Legacy SSE endpoint
			handleSSEConnection(database, w, r)
		case r.URL.Path == "/mcp" && r.Method == http.MethodPost:
			// Legacy messages endpoint
			handleJSONRPCMessage(database, w, r)
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	})
}

// Legacy handler for backward compatibility
func NewHandler(database *sql.DB) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Security: validate origin to prevent DNS rebinding attacks
		origin := r.Header.Get("Origin")
		if origin != "" && !strings.HasPrefix(origin, "http://localhost") && !strings.HasPrefix(origin, "http://127.0.0.1") {
			http.Error(w, "invalid origin", http.StatusForbidden)
			return
		}

		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var req JSONRPCRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "bad JSON", http.StatusBadRequest)
			return
		}

		// Validate JSON-RPC 2.0
		if req.JSONRPC != "2.0" {
			http.Error(w, "invalid JSON-RPC version", http.StatusBadRequest)
			return
		}

		response := HandleJSONRPCMethod(database, req)

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	})
}

// handleSSEConnection handles GET requests to establish SSE connection
func handleSSEConnection(database *sql.DB, w http.ResponseWriter, r *http.Request) {
	// Set SSE headers
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Last-Event-ID")

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming not supported", http.StatusInternalServerError)
		return
	}

	// Generate session ID
	sessionID := fmt.Sprintf("session_%d", time.Now().UnixNano())

	// Create MCP session
	session := &MCPSession{
		sessionID:   sessionID,
		writer:      w,
		flusher:     flusher,
		messageChan: make(chan JSONRPCRequest, 10),
		done:        make(chan bool),
		initialized: false,
	}

	// Add to session manager
	sessionManager.mu.Lock()
	sessionManager.sessions[sessionID] = session
	sessionManager.mu.Unlock()

	// Clean up on disconnect
	defer func() {
		sessionManager.mu.Lock()
		delete(sessionManager.sessions, sessionID)
		sessionManager.mu.Unlock()
		close(session.messageChan)
	}()

	// Send session establishment event with session ID
	sessionData := map[string]string{
		"sessionId": sessionID,
	}
	sendSSEEvent(session, "session", sessionData)

	// Send endpoint event with POST message URL (MCP spec requirement)
	endpointData := map[string]string{
		"uri": "/messages",
	}
	sendSSEEvent(session, "endpoint", endpointData)

	// Process messages and handle lifecycle
	for {
		select {
		case msg := <-session.messageChan:
			// A request is a notification if its ID is nil (absent or explicitly null).
			// JSON-RPC 2.0 spec: Server MUST NOT reply to a Notification.
			if msg.ID != nil {
				response := HandleJSONRPCMethod(database, msg)
				err := sendSSEEvent(session, "message", response)
				if err != nil {
					// Log error sending SSE event, e.g., client disconnected
					// log.Printf("Error sending SSE event for session %s: %v", session.sessionID, err)
					// Consider closing session.done here or handling client disconnect
				}
			} else {
				// It's a notification. Process it (it might have side effects)
				// but do not send a response back to the client.
				_ = HandleJSONRPCMethod(database, msg)
				// log.Printf("Processed notification for session %s, method: %s. No response sent.", session.sessionID, msg.Method)
			}

		case <-session.done:
			return

		case <-r.Context().Done():
			return

		case <-time.After(30 * time.Second):
			// Send keepalive
			sendSSEEvent(session, "ping", map[string]string{"timestamp": fmt.Sprintf("%d", time.Now().Unix())})
		}
	}
}

// handleJSONRPCMessage handles POST requests with JSON-RPC messages
func handleJSONRPCMessage(database *sql.DB, w http.ResponseWriter, r *http.Request) {
	sessionID := r.Header.Get("X-Session-ID")
	if sessionID == "" {
		http.Error(w, "missing session ID", http.StatusBadRequest)
		return
	}

	var req JSONRPCRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "bad JSON", http.StatusBadRequest)
		return
	}

	// Validate JSON-RPC 2.0
	if req.JSONRPC != "2.0" {
		http.Error(w, "invalid JSON-RPC version", http.StatusBadRequest)
		return
	}

	// Find session and send message
	sessionManager.mu.RLock()
	session, exists := sessionManager.sessions[sessionID]
	sessionManager.mu.RUnlock()

	if !exists {
		http.Error(w, "session not found", http.StatusNotFound)
		return
	}

	// Send message to session for processing
	select {
	case session.messageChan <- req:
		w.WriteHeader(http.StatusAccepted)
	case <-time.After(5 * time.Second):
		http.Error(w, "session busy", http.StatusServiceUnavailable)
	}
}

// sendSSEEvent sends an SSE event with proper formatting
func sendSSEEvent(session *MCPSession, eventType string, data interface{}) error {
	jsonData, err := json.Marshal(data)
	if err != nil {
		return err
	}

	eventID := fmt.Sprintf("%d", time.Now().UnixNano())

	// Write SSE event format
	_, err = fmt.Fprintf(session.writer, "id: %s\nevent: %s\ndata: %s\n\n", eventID, eventType, jsonData)
	if err != nil {
		return err
	}

	session.flusher.Flush()
	return nil
}

func HandleJSONRPCMethod(database *sql.DB, req JSONRPCRequest) JSONRPCResponse {
	switch req.Method {
	case "initialize":
		return handleInitialize(req)
	case "tools/list":
		return handleToolsList(req)
	case "tools/call":
		return handleToolCall(database, req)
	default:
		return JSONRPCResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Error: &JSONRPCError{
				Code:    -32601,
				Message: "Method not found",
			},
		}
	}
}

func handleInitialize(req JSONRPCRequest) JSONRPCResponse {
	result := InitializeResult{
		ProtocolVersion: "2024-11-05",
		Capabilities: ServerCapabilities{
			Tools: &struct{}{},
		},
		ServerInfo: ServerInfo{
			Name:    "knowledge-graph-mcp",
			Version: "1.0.0",
		},
	}

	return JSONRPCResponse{
		JSONRPC: "2.0",
		ID:      req.ID,
		Result:  result,
	}
}

func handleToolsList(req JSONRPCRequest) JSONRPCResponse {
	tools := []Tool{
		{
			Name:        "read_graph",
			Description: "Read the entire knowledge graph including entities, relations, and observations",
			InputSchema: InputSchema{
				Type:       "object",
				Properties: map[string]Property{},
				Required:   []string{},
			},
		},
		{
			Name:        "create_entities",
			Description: "Create multiple new entities in the knowledge graph",
			InputSchema: InputSchema{
				Type: "object",
				Properties: map[string]Property{
					"entities": {
						Type:        "array",
						Description: "Array of entity objects with name, entityType, and observations",
					},
				},
				Required: []string{"entities"},
			},
		},
		{
			Name:        "create_relations",
			Description: "Create multiple new relations between entities",
			InputSchema: InputSchema{
				Type: "object",
				Properties: map[string]Property{
					"relations": {
						Type:        "array",
						Description: "Array of relation objects with from, to, and relationType",
					},
				},
				Required: []string{"relations"},
			},
		},
		{
			Name:        "add_observations",
			Description: "Add new observations to existing entities",
			InputSchema: InputSchema{
				Type: "object",
				Properties: map[string]Property{
					"observations": {
						Type:        "array",
						Description: "Array of observation objects with entityName and contents",
					},
				},
				Required: []string{"observations"},
			},
		},
		{
			Name:        "delete_entities",
			Description: "Remove entities and their associated relations",
			InputSchema: InputSchema{
				Type: "object",
				Properties: map[string]Property{
					"entityNames": {
						Type:        "array",
						Description: "Array of entity names to delete",
					},
				},
				Required: []string{"entityNames"},
			},
		},
		{
			Name:        "delete_observations",
			Description: "Remove specific observations from entities",
			InputSchema: InputSchema{
				Type: "object",
				Properties: map[string]Property{
					"deletions": {
						Type:        "array",
						Description: "Array of deletion objects with entityName and observations",
					},
				},
				Required: []string{"deletions"},
			},
		},
		{
			Name:        "delete_relations",
			Description: "Remove specific relations from the graph",
			InputSchema: InputSchema{
				Type: "object",
				Properties: map[string]Property{
					"relations": {
						Type:        "array",
						Description: "Array of relation objects with from, to, and relationType",
					},
				},
				Required: []string{"relations"},
			},
		},
		{
			Name:        "search_nodes",
			Description: "Search nodes based on query",
			InputSchema: InputSchema{
				Type: "object",
				Properties: map[string]Property{
					"query": {
						Type:        "string",
						Description: "Search string to match against entity names, types, and observation content",
					},
				},
				Required: []string{"query"},
			},
		},
		{
			Name:        "open_nodes",
			Description: "Retrieve specific nodes by name",
			InputSchema: InputSchema{
				Type: "object",
				Properties: map[string]Property{
					"names": {
						Type:        "array",
						Description: "Array of node names to retrieve",
					},
				},
				Required: []string{"names"},
			},
		},
	}

	result := ToolsListResult{Tools: tools}

	return JSONRPCResponse{
		JSONRPC: "2.0",
		ID:      req.ID,
		Result:  result,
	}
}

func handleToolCall(database *sql.DB, req JSONRPCRequest) JSONRPCResponse {
	params, ok := req.Params.(map[string]interface{})
	if !ok {
		return JSONRPCResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Error: &JSONRPCError{
				Code:    -32602,
				Message: "Invalid params",
			},
		}
	}

	name, nameOk := params["name"].(string)
	arguments, argsOk := params["arguments"].(map[string]interface{})

	if !nameOk || !argsOk {
		return JSONRPCResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Error: &JSONRPCError{
				Code:    -32602,
				Message: "Missing name or arguments",
			},
		}
	}

	var result ToolCallResult
	var err error

	switch name {
	case "read_graph":
		result, err = handleReadGraphTool(database)
	case "create_entities":
		result, err = handleCreateEntitiesToolMCP(database, arguments)
	case "create_relations":
		result, err = handleCreateRelationsToolMCP(database, arguments)
	case "add_observations":
		result, err = handleAddObservationsToolMCP(database, arguments)
	case "delete_entities":
		result, err = handleDeleteEntitiesToolMCP(database, arguments)
	case "delete_observations":
		result, err = handleDeleteObservationsToolMCP(database, arguments)
	case "delete_relations":
		result, err = handleDeleteRelationsToolMCP(database, arguments)
	case "search_nodes":
		result, err = handleSearchNodesToolMCP(database, arguments)
	case "open_nodes":
		result, err = handleOpenNodesToolMCP(database, arguments)
	// Legacy support for old endpoint names
	case "create_entity":
		result, err = handleCreateEntityTool(database, arguments)
	case "create_relation":
		result, err = handleCreateRelationTool(database, arguments)
	case "create_observation":
		result, err = handleCreateObservationTool(database, arguments)
	default:
		return JSONRPCResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Error: &JSONRPCError{
				Code:    -32601,
				Message: "Unknown tool: " + name,
			},
		}
	}

	if err != nil {
		result = ToolCallResult{
			Content: []ToolContent{{
				Type: "text",
				Text: fmt.Sprintf("Error: %s", err.Error()),
			}},
			IsError: true,
		}
	}

	return JSONRPCResponse{
		JSONRPC: "2.0",
		ID:      req.ID,
		Result:  result,
	}
}

func handleReadGraphTool(database *sql.DB) (ToolCallResult, error) {
	entities, relations, observations, err := db.ReadGraph(database)
	if err != nil {
		return ToolCallResult{}, err
	}

	result := map[string]interface{}{
		"entities":     entities,
		"relations":    relations,
		"observations": observations,
	}

	jsonData, err := json.Marshal(result)
	if err != nil {
		return ToolCallResult{}, err
	}

	return ToolCallResult{
		Content: []ToolContent{{
			Type: "text",
			Text: string(jsonData),
		}},
	}, nil
}

func handleCreateEntityTool(database *sql.DB, arguments map[string]interface{}) (ToolCallResult, error) {
	name, nameOk := arguments["name"].(string)
	entityType, typeOk := arguments["entity_type"].(string)

	if !nameOk || !typeOk {
		return ToolCallResult{}, fmt.Errorf("missing required parameters: name, entity_type")
	}

	err := db.CreateEntity(database, name, entityType)
	if err != nil {
		return ToolCallResult{}, err
	}

	return ToolCallResult{
		Content: []ToolContent{{
			Type: "text",
			Text: fmt.Sprintf("Successfully created entity '%s' of type '%s'", name, entityType),
		}},
	}, nil
}

func handleCreateRelationTool(database *sql.DB, arguments map[string]interface{}) (ToolCallResult, error) {
	from, fromOk := arguments["from_entity"].(string)
	to, toOk := arguments["to_entity"].(string)
	relationType, typeOk := arguments["relation_type"].(string)

	if !fromOk || !toOk || !typeOk {
		return ToolCallResult{}, fmt.Errorf("missing required parameters: from_entity, to_entity, relation_type")
	}

	id, err := db.CreateRelation(database, from, to, relationType)
	if err != nil {
		return ToolCallResult{}, err
	}

	return ToolCallResult{
		Content: []ToolContent{{
			Type: "text",
			Text: fmt.Sprintf("Successfully created relation (ID: %d) from '%s' to '%s' with type '%s'", id, from, to, relationType),
		}},
	}, nil
}

func handleCreateObservationTool(database *sql.DB, arguments map[string]interface{}) (ToolCallResult, error) {
	entityName, nameOk := arguments["entity_name"].(string)
	content, contentOk := arguments["content"].(string)

	if !nameOk || !contentOk {
		return ToolCallResult{}, fmt.Errorf("missing required parameters: entity_name, content")
	}

	id, err := db.CreateObservation(database, entityName, content)
	if err != nil {
		return ToolCallResult{}, err
	}

	return ToolCallResult{
		Content: []ToolContent{{
			Type: "text",
			Text: fmt.Sprintf("Successfully created observation (ID: %d) for entity '%s'", id, entityName),
		}},
	}, nil
}

func handleCreateEntitiesToolMCP(database *sql.DB, arguments map[string]interface{}) (ToolCallResult, error) {
	entitiesInterface, ok := arguments["entities"].([]interface{})
	if !ok {
		return ToolCallResult{}, fmt.Errorf("missing or invalid entities parameter")
	}

	var createdEntities []string
	for _, entityInterface := range entitiesInterface {
		entityMap, ok := entityInterface.(map[string]interface{})
		if !ok {
			continue
		}

		name, nameOk := entityMap["name"].(string)
		entityType, typeOk := entityMap["entityType"].(string)

		if !nameOk || !typeOk {
			continue
		}

		err := db.CreateEntity(database, name, entityType)
		if err != nil {
			// Continue with other entities even if one fails (spec says to ignore existing entities)
			continue
		}

		createdEntities = append(createdEntities, name)

		// Handle observations if provided
		if observationsInterface, obsOk := entityMap["observations"].([]interface{}); obsOk {
			for _, obsInterface := range observationsInterface {
				if obsStr, strOk := obsInterface.(string); strOk {
					db.CreateObservation(database, name, obsStr)
				}
			}
		}
	}

	return ToolCallResult{
		Content: []ToolContent{{
			Type: "text",
			Text: fmt.Sprintf("Successfully created %d entities: %v", len(createdEntities), createdEntities),
		}},
	}, nil
}

func handleCreateRelationsToolMCP(database *sql.DB, arguments map[string]interface{}) (ToolCallResult, error) {
	relationsInterface, ok := arguments["relations"].([]interface{})
	if !ok {
		return ToolCallResult{}, fmt.Errorf("missing or invalid relations parameter")
	}

	var createdIDs []int64
	for _, relationInterface := range relationsInterface {
		relationMap, ok := relationInterface.(map[string]interface{})
		if !ok {
			continue
		}

		from, fromOk := relationMap["from"].(string)
		to, toOk := relationMap["to"].(string)
		relationType, typeOk := relationMap["relationType"].(string)

		if !fromOk || !toOk || !typeOk {
			continue
		}

		id, err := db.CreateRelation(database, from, to, relationType)
		if err != nil {
			// Skip duplicate relations as per spec
			continue
		}

		createdIDs = append(createdIDs, id)
	}

	return ToolCallResult{
		Content: []ToolContent{{
			Type: "text",
			Text: fmt.Sprintf("Successfully created %d relations with IDs: %v", len(createdIDs), createdIDs),
		}},
	}, nil
}

func handleAddObservationsToolMCP(database *sql.DB, arguments map[string]interface{}) (ToolCallResult, error) {
	observationsInterface, ok := arguments["observations"].([]interface{})
	if !ok {
		return ToolCallResult{}, fmt.Errorf("missing or invalid observations parameter")
	}

	var observations []struct {
		EntityName string `json:"entityName"`
		Contents   string `json:"contents"`
	}

	for _, obsInterface := range observationsInterface {
		obsMap, ok := obsInterface.(map[string]interface{})
		if !ok {
			continue
		}

		entityName, nameOk := obsMap["entityName"].(string)
		contents, contentsOk := obsMap["contents"].(string)

		if !nameOk || !contentsOk {
			continue
		}

		observations = append(observations, struct {
			EntityName string `json:"entityName"`
			Contents   string `json:"contents"`
		}{EntityName: entityName, Contents: contents})
	}

	added, err := db.AddObservations(database, observations)
	if err != nil {
		return ToolCallResult{}, err
	}

	jsonData, err := json.Marshal(added)
	if err != nil {
		return ToolCallResult{}, err
	}

	return ToolCallResult{
		Content: []ToolContent{{
			Type: "text",
			Text: string(jsonData),
		}},
	}, nil
}

func handleDeleteEntitiesToolMCP(database *sql.DB, arguments map[string]interface{}) (ToolCallResult, error) {
	entityNamesInterface, ok := arguments["entityNames"].([]interface{})
	if !ok {
		return ToolCallResult{}, fmt.Errorf("missing or invalid entityNames parameter")
	}

	var entityNames []string
	for _, nameInterface := range entityNamesInterface {
		if name, ok := nameInterface.(string); ok {
			entityNames = append(entityNames, name)
		}
	}

	err := db.DeleteEntities(database, entityNames)
	if err != nil {
		return ToolCallResult{}, err
	}

	return ToolCallResult{
		Content: []ToolContent{{
			Type: "text",
			Text: fmt.Sprintf("Successfully deleted %d entities", len(entityNames)),
		}},
	}, nil
}

func handleDeleteObservationsToolMCP(database *sql.DB, arguments map[string]interface{}) (ToolCallResult, error) {
	deletionsInterface, ok := arguments["deletions"].([]interface{})
	if !ok {
		return ToolCallResult{}, fmt.Errorf("missing or invalid deletions parameter")
	}

	var deletions []struct {
		EntityName   string   `json:"entityName"`
		Observations []string `json:"observations"`
	}

	for _, deletionInterface := range deletionsInterface {
		deletionMap, ok := deletionInterface.(map[string]interface{})
		if !ok {
			continue
		}

		entityName, nameOk := deletionMap["entityName"].(string)
		observationsInterface, obsOk := deletionMap["observations"].([]interface{})

		if !nameOk || !obsOk {
			continue
		}

		var observations []string
		for _, obsInterface := range observationsInterface {
			if obs, ok := obsInterface.(string); ok {
				observations = append(observations, obs)
			}
		}

		deletions = append(deletions, struct {
			EntityName   string   `json:"entityName"`
			Observations []string `json:"observations"`
		}{EntityName: entityName, Observations: observations})
	}

	err := db.DeleteObservations(database, deletions)
	if err != nil {
		return ToolCallResult{}, err
	}

	return ToolCallResult{
		Content: []ToolContent{{
			Type: "text",
			Text: fmt.Sprintf("Successfully processed deletion of observations for %d entities", len(deletions)),
		}},
	}, nil
}

func handleDeleteRelationsToolMCP(database *sql.DB, arguments map[string]interface{}) (ToolCallResult, error) {
	relationsInterface, ok := arguments["relations"].([]interface{})
	if !ok {
		return ToolCallResult{}, fmt.Errorf("missing or invalid relations parameter")
	}

	var relations []struct {
		From string `json:"from"`
		To   string `json:"to"`
		Type string `json:"relationType"`
	}

	for _, relationInterface := range relationsInterface {
		relationMap, ok := relationInterface.(map[string]interface{})
		if !ok {
			continue
		}

		from, fromOk := relationMap["from"].(string)
		to, toOk := relationMap["to"].(string)
		relationType, typeOk := relationMap["relationType"].(string)

		if !fromOk || !toOk || !typeOk {
			continue
		}

		relations = append(relations, struct {
			From string `json:"from"`
			To   string `json:"to"`
			Type string `json:"relationType"`
		}{From: from, To: to, Type: relationType})
	}

	err := db.DeleteRelations(database, relations)
	if err != nil {
		return ToolCallResult{}, err
	}

	return ToolCallResult{
		Content: []ToolContent{{
			Type: "text",
			Text: fmt.Sprintf("Successfully processed deletion of %d relations", len(relations)),
		}},
	}, nil
}

func handleSearchNodesToolMCP(database *sql.DB, arguments map[string]interface{}) (ToolCallResult, error) {
	query, ok := arguments["query"].(string)
	if !ok {
		return ToolCallResult{}, fmt.Errorf("missing or invalid query parameter")
	}

	entities, relations, err := db.SearchNodes(database, query)
	if err != nil {
		return ToolCallResult{}, err
	}

	result := map[string]interface{}{
		"entities":  entities,
		"relations": relations,
	}

	jsonData, err := json.Marshal(result)
	if err != nil {
		return ToolCallResult{}, err
	}

	return ToolCallResult{
		Content: []ToolContent{{
			Type: "text",
			Text: string(jsonData),
		}},
	}, nil
}

func handleOpenNodesToolMCP(database *sql.DB, arguments map[string]interface{}) (ToolCallResult, error) {
	namesInterface, ok := arguments["names"].([]interface{})
	if !ok {
		return ToolCallResult{}, fmt.Errorf("missing or invalid names parameter")
	}

	var names []string
	for _, nameInterface := range namesInterface {
		if name, ok := nameInterface.(string); ok {
			names = append(names, name)
		}
	}

	entities, relations, err := db.OpenNodes(database, names)
	if err != nil {
		return ToolCallResult{}, err
	}

	result := map[string]interface{}{
		"entities":  entities,
		"relations": relations,
	}

	jsonData, err := json.Marshal(result)
	if err != nil {
		return ToolCallResult{}, err
	}

	return ToolCallResult{
		Content: []ToolContent{{
			Type: "text",
			Text: string(jsonData),
		}},
	}, nil
}
