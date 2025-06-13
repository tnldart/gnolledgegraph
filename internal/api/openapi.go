package api

import (
	"encoding/json"
)

// OpenAPISpec generates the OpenAPI 3.1 specification for the API
func OpenAPISpec() map[string]interface{} {
	return map[string]interface{}{
		"openapi":           "3.1.0",
		"jsonSchemaDialect": "https://json-schema.org/draft/2020-12/schema",
		"info": map[string]interface{}{
			"title":       "Knowledge Graph API",
			"version":     "0.1.0",
			"description": "API for managing and querying a knowledge graph. This specification details two API formats:\n\n1.  **Original Go API** (prefixed with `/api`):\n    *   Uses Go-centric conventions (e.g., snake_case field names, separate observation endpoints).\n\n2.  **Python FastAPI Compatibility API** (root paths, e.g., `/read_graph`):\n    *   Designed for seamless integration with Python FastAPI clients.\n    *   Features Pythonic conventions (e.g., camelCase field names like `entityType`).\n    *   Embeds observations directly within entity structures for request and response bodies.\n    *   Adapts certain request methods (e.g., `POST` for `/search_nodes` with a JSON body) for Python client expectations.\n    *   Provides responses more aligned with typical FastAPI patterns (e.g., direct data models instead of status wrappers for some operations).\n\nThis dual API approach ensures backward compatibility for existing Go clients while offering a familiar and convenient interface for Python developers.",
		},
		"servers": []map[string]interface{}{
			{
				"url":         "http://localhost:8080",
				"description": "Local dev server",
			},
		},
		"tags": []map[string]interface{}{
			{
				"name":        "Go API",
				"description": "Endpoints for the original Go API.",
			},
			{
				"name":        "Python Compatibility API",
				"description": "Endpoints compatible with Python FastAPI clients.",
			},
			{
				"name":        "MCP",
				"description": "Model Context Protocol endpoints.",
			},
		},
		"paths": map[string]interface{}{
			// Python Compatibility API Endpoints
			"/read_graph": map[string]interface{}{
				"get": map[string]interface{}{
					"tags":        []string{"Python Compatibility API"},
					"operationId": "python_read_graph",
					"summary":     "Read the complete knowledge graph (Python compatible)",
					"responses": map[string]interface{}{
						"200": map[string]interface{}{
							"description": "Graph data in Python-compatible format",
							"content": map[string]interface{}{
								"application/json": map[string]interface{}{
									"schema": map[string]interface{}{
										"$ref": "#/components/schemas/PythonKnowledgeGraph",
									},
								},
							},
						},
						"500": map[string]interface{}{
							"description": "Internal server error",
						},
					},
				},
			},
			"/create_entities": map[string]interface{}{
				"post": map[string]interface{}{
					"tags":        []string{"Python Compatibility API"},
					"operationId": "python_create_entities",
					"summary":     "Create new entities with embedded observations (Python compatible)",
					"requestBody": map[string]interface{}{
						"required": true,
						"content": map[string]interface{}{
							"application/json": map[string]interface{}{
								"schema": map[string]interface{}{
									"type": "object",
									"properties": map[string]interface{}{
										"entities": map[string]interface{}{
											"type":  "array",
											"items": map[string]interface{}{"$ref": "#/components/schemas/PythonEntity"},
										},
									},
									"required": []string{"entities"},
								},
								"examples": map[string]interface{}{
									"example1": map[string]interface{}{
										"value": map[string]interface{}{
											"entities": []map[string]interface{}{
												{
													"name":         "Python",
													"entityType":   "Language",
													"observations": []string{"High-level", "Interpreted"},
												},
											},
										},
									},
								},
							},
						},
					},
					"responses": map[string]interface{}{
						"201": map[string]interface{}{
							"description": "Entities created successfully",
							"content": map[string]interface{}{
								"application/json": map[string]interface{}{
									"schema": map[string]interface{}{
										"type":  "array",
										"items": map[string]interface{}{"$ref": "#/components/schemas/PythonEntity"},
									},
								},
							},
						},
						"400": map[string]interface{}{"description": "Invalid request body"},
						"409": map[string]interface{}{
							"description": "Conflict, one or more entities already exist",
							"content": map[string]interface{}{
								"application/json": map[string]interface{}{
									"schema": map[string]interface{}{
										"type": "object",
										"properties": map[string]interface{}{
											"error":                map[string]interface{}{"type": "string"},
											"conflicting_entities": map[string]interface{}{"type": "array", "items": map[string]interface{}{"type": "string"}},
										},
									},
								},
							},
						},
						"500": map[string]interface{}{"description": "Internal server error"},
					},
				},
			},
			"/create_relations": map[string]interface{}{
				"post": map[string]interface{}{
					"tags":        []string{"Python Compatibility API"},
					"operationId": "python_create_relations",
					"summary":     "Create new relations (Python compatible)",
					"requestBody": map[string]interface{}{
						"required": true,
						"content": map[string]interface{}{
							"application/json": map[string]interface{}{
								"schema": map[string]interface{}{
									"type": "object",
									"properties": map[string]interface{}{
										"relations": map[string]interface{}{
											"type":  "array",
											"items": map[string]interface{}{"$ref": "#/components/schemas/PythonRelation"},
										},
									},
									"required": []string{"relations"},
								},
								"examples": map[string]interface{}{
									"example1": map[string]interface{}{
										"value": map[string]interface{}{
											"relations": []map[string]interface{}{
												{
													"from":         "Python",
													"to":           "Django",
													"relationType": "hasFramework",
												},
											},
										},
									},
								},
							},
						},
					},
					"responses": map[string]interface{}{
						"201": map[string]interface{}{
							"description": "Relations created successfully",
							"content": map[string]interface{}{
								"application/json": map[string]interface{}{
									"schema": map[string]interface{}{
										"type":  "array",
										"items": map[string]interface{}{"$ref": "#/components/schemas/PythonRelation"},
									},
								},
							},
						},
						"400": map[string]interface{}{"description": "Invalid request body or referenced entity does not exist"},
						"500": map[string]interface{}{"description": "Internal server error"},
					},
				},
			},
			"/add_observations": map[string]interface{}{
				"post": map[string]interface{}{
					"tags":        []string{"Python Compatibility API"},
					"operationId": "python_add_observations",
					"summary":     "Add observations to entities (Python compatible)",
					"requestBody": map[string]interface{}{
						"required": true,
						"content": map[string]interface{}{
							"application/json": map[string]interface{}{
								"schema": map[string]interface{}{
									"type": "object",
									"properties": map[string]interface{}{
										"observations": map[string]interface{}{
											"type": "array",
											"items": map[string]interface{}{
												"type": "object",
												"properties": map[string]interface{}{
													"entityName": map[string]interface{}{"type": "string"},
													"contents":   map[string]interface{}{"type": "array", "items": map[string]interface{}{"type": "string"}},
												},
												"required": []string{"entityName", "contents"},
											},
										},
									},
									"required": []string{"observations"},
								},
								"examples": map[string]interface{}{
									"example1": map[string]interface{}{
										"value": map[string]interface{}{
											"observations": []map[string]interface{}{
												{
													"entityName": "Python",
													"contents":   []string{"observation1", "observation2"},
												},
											},
										},
									},
								},
							},
						},
					},
					"responses": map[string]interface{}{
						"201": map[string]interface{}{
							"description": "Observations added successfully",
							"content": map[string]interface{}{
								"application/json": map[string]interface{}{
									"schema": map[string]interface{}{ // Schema matches request for simplicity in this example
										"type": "object",
										"properties": map[string]interface{}{
											"observations": map[string]interface{}{
												"type": "array",
												"items": map[string]interface{}{
													"type": "object",
													"properties": map[string]interface{}{
														"entityName": map[string]interface{}{"type": "string"},
														"contents":   map[string]interface{}{"type": "array", "items": map[string]interface{}{"type": "string"}},
													},
												},
											},
										},
									},
								},
							},
						},
						"400": map[string]interface{}{"description": "Invalid request body"},
						"500": map[string]interface{}{"description": "Internal server error"},
					},
				},
			},
			"/search_nodes": map[string]interface{}{ // Note: This is POST for Python API
				"post": map[string]interface{}{
					"tags":        []string{"Python Compatibility API"},
					"operationId": "python_search_nodes",
					"summary":     "Search nodes (Python compatible, POST with JSON body)",
					"requestBody": map[string]interface{}{
						"required": true,
						"content": map[string]interface{}{
							"application/json": map[string]interface{}{
								"schema": map[string]interface{}{
									"type":       "object",
									"properties": map[string]interface{}{"query": map[string]interface{}{"type": "string"}},
									"required":   []string{"query"},
								},
								"examples": map[string]interface{}{
									"example1": map[string]interface{}{
										"value": map[string]interface{}{"query": "programming"},
									},
								},
							},
						},
					},
					"responses": map[string]interface{}{
						"200": map[string]interface{}{
							"description": "Search results in Python-compatible format",
							"content": map[string]interface{}{
								"application/json": map[string]interface{}{
									"schema": map[string]interface{}{"$ref": "#/components/schemas/PythonKnowledgeGraph"},
								},
							},
						},
						"400": map[string]interface{}{"description": "Invalid request body"},
						"500": map[string]interface{}{"description": "Internal server error"},
					},
				},
			},
			"/open_nodes": map[string]interface{}{ // Note: This is POST for Python API
				"post": map[string]interface{}{
					"tags":        []string{"Python Compatibility API"},
					"operationId": "python_open_nodes",
					"summary":     "Retrieve specific nodes by name (Python compatible)",
					"requestBody": map[string]interface{}{
						"required": true,
						"content": map[string]interface{}{
							"application/json": map[string]interface{}{
								"schema": map[string]interface{}{
									"type":       "object",
									"properties": map[string]interface{}{"names": map[string]interface{}{"type": "array", "items": map[string]interface{}{"type": "string"}}},
									"required":   []string{"names"},
								},
								"examples": map[string]interface{}{
									"example1": map[string]interface{}{
										"value": map[string]interface{}{"names": []string{"Python", "Django"}},
									},
								},
							},
						},
					},
					"responses": map[string]interface{}{
						"200": map[string]interface{}{
							"description": "Requested entities and relations in Python-compatible format",
							"content": map[string]interface{}{
								"application/json": map[string]interface{}{
									"schema": map[string]interface{}{"$ref": "#/components/schemas/PythonKnowledgeGraph"},
								},
							},
						},
						"400": map[string]interface{}{"description": "Invalid request body"},
						"500": map[string]interface{}{"description": "Internal server error"},
					},
				},
			},
			"/delete_entities": map[string]interface{}{
				"post": map[string]interface{}{ // Changed from DELETE to POST for consistency with other Python endpoints if desired, or keep as DELETE
					"tags":        []string{"Python Compatibility API"},
					"operationId": "python_delete_entities",
					"summary":     "Delete entities (Python compatible)",
					"requestBody": map[string]interface{}{
						"required": true,
						"content": map[string]interface{}{
							"application/json": map[string]interface{}{
								"schema": map[string]interface{}{
									"type":       "object",
									"properties": map[string]interface{}{"entityNames": map[string]interface{}{"type": "array", "items": map[string]interface{}{"type": "string"}}},
									"required":   []string{"entityNames"},
								},
								"examples": map[string]interface{}{
									"example1": map[string]interface{}{
										"value": map[string]interface{}{"entityNames": []string{"OldEntity"}},
									},
								},
							},
						},
					},
					"responses": map[string]interface{}{
						"200": map[string]interface{}{
							"description": "Entities deletion process initiated",
							"content": map[string]interface{}{
								"application/json": map[string]interface{}{
									"schema": map[string]interface{}{
										"type": "object",
										"properties": map[string]interface{}{
											"status":  map[string]interface{}{"type": "string", "example": "success"},
											"deleted": map[string]interface{}{"type": "integer", "description": "Number of entities requested for deletion"},
										},
									},
								},
							},
						},
						"400": map[string]interface{}{"description": "Invalid request body"},
						"500": map[string]interface{}{"description": "Internal server error"},
					},
				},
			},
			"/delete_observations": map[string]interface{}{
				"post": map[string]interface{}{
					"tags":        []string{"Python Compatibility API"},
					"operationId": "python_delete_observations",
					"summary":     "Delete observations (Python compatible)",
					"requestBody": map[string]interface{}{
						"required": true,
						"content": map[string]interface{}{
							"application/json": map[string]interface{}{
								"schema": map[string]interface{}{
									"type": "object",
									"properties": map[string]interface{}{
										"deletions": map[string]interface{}{
											"type": "array",
											"items": map[string]interface{}{
												"type": "object",
												"properties": map[string]interface{}{
													"entityName":   map[string]interface{}{"type": "string"},
													"observations": map[string]interface{}{"type": "array", "items": map[string]interface{}{"type": "string"}},
												},
												"required": []string{"entityName", "observations"},
											},
										},
									},
									"required": []string{"deletions"},
								},
								"examples": map[string]interface{}{
									"example1": map[string]interface{}{
										"value": map[string]interface{}{
											"deletions": []map[string]interface{}{
												{
													"entityName":   "Python",
													"observations": []string{"outdated_obs"},
												},
											},
										},
									},
								},
							},
						},
					},
					"responses": map[string]interface{}{
						"200": map[string]interface{}{
							"description": "Observations deletion process initiated",
							"content": map[string]interface{}{
								"application/json": map[string]interface{}{
									"schema": map[string]interface{}{
										"type":       "object",
										"properties": map[string]interface{}{"status": map[string]interface{}{"type": "string", "example": "success"}},
									},
								},
							},
						},
						"400": map[string]interface{}{"description": "Invalid request body"},
						"500": map[string]interface{}{"description": "Internal server error"},
					},
				},
			},
			"/delete_relations": map[string]interface{}{
				"post": map[string]interface{}{
					"tags":        []string{"Python Compatibility API"},
					"operationId": "python_delete_relations",
					"summary":     "Delete relations (Python compatible)",
					"requestBody": map[string]interface{}{
						"required": true,
						"content": map[string]interface{}{
							"application/json": map[string]interface{}{
								"schema": map[string]interface{}{
									"type": "object",
									"properties": map[string]interface{}{
										"relations": map[string]interface{}{
											"type":  "array",
											"items": map[string]interface{}{"$ref": "#/components/schemas/PythonRelation"},
										},
									},
									"required": []string{"relations"},
								},
								"examples": map[string]interface{}{
									"example1": map[string]interface{}{
										"value": map[string]interface{}{
											"relations": []map[string]interface{}{
												{
													"from":         "OldApp",
													"to":           "OldDB",
													"relationType": "uses",
												},
											},
										},
									},
								},
							},
						},
					},
					"responses": map[string]interface{}{
						"200": map[string]interface{}{
							"description": "Relations deletion process initiated",
							"content": map[string]interface{}{
								"application/json": map[string]interface{}{
									"schema": map[string]interface{}{
										"type":       "object",
										"properties": map[string]interface{}{"status": map[string]interface{}{"type": "string", "example": "success"}},
									},
								},
							},
						},
						"400": map[string]interface{}{"description": "Invalid request body"},
						"500": map[string]interface{}{"description": "Internal server error"},
					},
				},
			},

			// Original Go API Endpoints
			"/api/read_graph": map[string]interface{}{
				"get": map[string]interface{}{
					"tags":        []string{"Go API"},
					"operationId": "go_read_graph",
					"summary":     "Read the complete knowledge graph (Go API)",
					"responses": map[string]interface{}{
						"200": map[string]interface{}{
							"description": "Graph data containing entities, relations, and observations",
							"content": map[string]interface{}{
								"application/json": map[string]interface{}{
									"schema": map[string]interface{}{
										"type": "object",
										"properties": map[string]interface{}{
											"entities": map[string]interface{}{
												"type": "array",
												"items": map[string]interface{}{
													"$ref": "#/components/schemas/Entity",
												},
											},
											"relations": map[string]interface{}{
												"type": "array",
												"items": map[string]interface{}{
													"$ref": "#/components/schemas/Relation",
												},
											},
											"observations": map[string]interface{}{
												"type": "array",
												"items": map[string]interface{}{
													"$ref": "#/components/schemas/Observation",
												},
											},
										},
									},
								},
							},
						},
						"500": map[string]interface{}{
							"description": "Internal server error",
						},
					},
				},
			},
			"/api/create_entities": map[string]interface{}{
				"post": map[string]interface{}{
					"operationId": "go_create_entities", // Renamed
					"summary":     "Create new entities in the knowledge graph (Go API)",
					"tags":        []string{"Go API"},
					"requestBody": map[string]interface{}{
						"required": true,
						"content": map[string]interface{}{
							"application/json": map[string]interface{}{
								"schema": map[string]interface{}{
									"type": "object",
									"properties": map[string]interface{}{
										"entities": map[string]interface{}{
											"type": "array",
											"items": map[string]interface{}{
												"type": "object",
												"properties": map[string]interface{}{
													"name": map[string]interface{}{
														"type": "string",
													},
													"entity_type": map[string]interface{}{
														"type": "string",
													},
												},
												"required": []string{"name", "entity_type"},
											},
										},
									},
									"required": []string{"entities"},
								},
							},
						},
					},
					"responses": map[string]interface{}{
						"201": map[string]interface{}{
							"description": "Entities created successfully",
							"content": map[string]interface{}{
								"application/json": map[string]interface{}{
									"schema": map[string]interface{}{
										"type": "object",
										"properties": map[string]interface{}{
											"status": map[string]interface{}{
												"type": "string",
											},
										},
									},
								},
							},
						},
						"400": map[string]interface{}{
							"description": "Invalid request body",
						},
						"500": map[string]interface{}{
							"description": "Internal server error",
						},
					},
				},
			},
			"/api/create_relations": map[string]interface{}{
				"post": map[string]interface{}{
					"operationId": "go_create_relations", // Renamed
					"summary":     "Create new relations between entities (Go API)",
					"tags":        []string{"Go API"},
					"requestBody": map[string]interface{}{
						"required": true,
						"content": map[string]interface{}{
							"application/json": map[string]interface{}{
								"schema": map[string]interface{}{
									"type": "object",
									"properties": map[string]interface{}{
										"relations": map[string]interface{}{
											"type": "array",
											"items": map[string]interface{}{
												"type": "object",
												"properties": map[string]interface{}{
													"from_entity": map[string]interface{}{
														"type": "string",
													},
													"to_entity": map[string]interface{}{
														"type": "string",
													},
													"relation_type": map[string]interface{}{
														"type": "string",
													},
												},
												"required": []string{"from_entity", "to_entity", "relation_type"},
											},
										},
									},
									"required": []string{"relations"},
								},
							},
						},
					},
					"responses": map[string]interface{}{
						"201": map[string]interface{}{
							"description": "Relations created successfully",
							"content": map[string]interface{}{
								"application/json": map[string]interface{}{
									"schema": map[string]interface{}{
										"type": "object",
										"properties": map[string]interface{}{
											"status": map[string]interface{}{
												"type": "string",
											},
											"ids": map[string]interface{}{
												"type": "array",
												"items": map[string]interface{}{
													"type":   "integer",
													"format": "int64",
												},
											},
										},
									},
								},
							},
						},
						"400": map[string]interface{}{
							"description": "Invalid request body",
						},
						"500": map[string]interface{}{
							"description": "Internal server error",
						},
					},
				},
			},
			"/api/add_observations": map[string]interface{}{
				"post": map[string]interface{}{
					"operationId": "go_add_observations", // Renamed
					"summary":     "Add new observations to existing entities (Go API)",
					"tags":        []string{"Go API"},
					"requestBody": map[string]interface{}{
						"required": true,
						"content": map[string]interface{}{
							"application/json": map[string]interface{}{
								"schema": map[string]interface{}{
									"type": "object",
									"properties": map[string]interface{}{
										"observations": map[string]interface{}{
											"type": "array",
											"items": map[string]interface{}{
												"type": "object",
												"properties": map[string]interface{}{
													"entityName": map[string]interface{}{
														"type": "string",
													},
													"contents": map[string]interface{}{
														"type": "string",
													},
												},
												"required": []string{"entityName", "contents"},
											},
										},
									},
									"required": []string{"observations"},
								},
							},
						},
					},
					"responses": map[string]interface{}{
						"201": map[string]interface{}{
							"description": "Observations added successfully",
							"content": map[string]interface{}{
								"application/json": map[string]interface{}{
									"schema": map[string]interface{}{
										"type": "object",
										"properties": map[string]interface{}{
											"status": map[string]interface{}{
												"type": "string",
											},
											"added": map[string]interface{}{
												"type": "array",
												"items": map[string]interface{}{
													"$ref": "#/components/schemas/Observation",
												},
											},
										},
									},
								},
							},
						},
						"400": map[string]interface{}{
							"description": "Invalid request body",
						},
						"500": map[string]interface{}{
							"description": "Internal server error",
						},
					},
				},
			},
			"/api/delete_entities": map[string]interface{}{
				"delete": map[string]interface{}{
					"operationId": "go_delete_entities", // Renamed
					"summary":     "Remove entities and their associated relations (Go API)",
					"tags":        []string{"Go API"},
					"requestBody": map[string]interface{}{
						"required": true,
						"content": map[string]interface{}{
							"application/json": map[string]interface{}{
								"schema": map[string]interface{}{
									"type": "object",
									"properties": map[string]interface{}{
										"entityNames": map[string]interface{}{
											"type": "array",
											"items": map[string]interface{}{
												"type": "string",
											},
										},
									},
									"required": []string{"entityNames"},
								},
							},
						},
					},
					"responses": map[string]interface{}{
						"200": map[string]interface{}{
							"description": "Entities deleted successfully",
							"content": map[string]interface{}{
								"application/json": map[string]interface{}{
									"schema": map[string]interface{}{
										"type": "object",
										"properties": map[string]interface{}{
											"status": map[string]interface{}{
												"type": "string",
											},
											"deleted": map[string]interface{}{
												"type": "integer",
											},
										},
									},
								},
							},
						},
						"400": map[string]interface{}{
							"description": "Invalid request body",
						},
						"500": map[string]interface{}{
							"description": "Internal server error",
						},
					},
				},
			},
			"/api/delete_observations": map[string]interface{}{
				"delete": map[string]interface{}{
					"operationId": "go_delete_observations", // Renamed
					"summary":     "Remove specific observations from entities (Go API)",
					"tags":        []string{"Go API"},
					"requestBody": map[string]interface{}{
						"required": true,
						"content": map[string]interface{}{
							"application/json": map[string]interface{}{
								"schema": map[string]interface{}{
									"type": "object",
									"properties": map[string]interface{}{
										"deletions": map[string]interface{}{
											"type": "array",
											"items": map[string]interface{}{
												"type": "object",
												"properties": map[string]interface{}{
													"entityName": map[string]interface{}{
														"type": "string",
													},
													"observations": map[string]interface{}{
														"type": "array",
														"items": map[string]interface{}{
															"type": "string",
														},
													},
												},
												"required": []string{"entityName", "observations"},
											},
										},
									},
									"required": []string{"deletions"},
								},
							},
						},
					},
					"responses": map[string]interface{}{
						"200": map[string]interface{}{
							"description": "Observations deleted successfully",
							"content": map[string]interface{}{
								"application/json": map[string]interface{}{
									"schema": map[string]interface{}{
										"type": "object",
										"properties": map[string]interface{}{
											"status": map[string]interface{}{
												"type": "string",
											},
										},
									},
								},
							},
						},
						"400": map[string]interface{}{
							"description": "Invalid request body",
						},
						"500": map[string]interface{}{
							"description": "Internal server error",
						},
					},
				},
			},
			"/api/delete_relations": map[string]interface{}{
				"delete": map[string]interface{}{
					"operationId": "go_delete_relations", // Renamed
					"summary":     "Remove specific relations from the graph (Go API)",
					"tags":        []string{"Go API"},
					"requestBody": map[string]interface{}{
						"required": true,
						"content": map[string]interface{}{
							"application/json": map[string]interface{}{
								"schema": map[string]interface{}{
									"type": "object",
									"properties": map[string]interface{}{
										"relations": map[string]interface{}{
											"type": "array",
											"items": map[string]interface{}{
												"type": "object",
												"properties": map[string]interface{}{
													"from": map[string]interface{}{
														"type": "string",
													},
													"to": map[string]interface{}{
														"type": "string",
													},
													"relationType": map[string]interface{}{
														"type": "string",
													},
												},
												"required": []string{"from", "to", "relationType"},
											},
										},
									},
									"required": []string{"relations"},
								},
							},
						},
					},
					"responses": map[string]interface{}{
						"200": map[string]interface{}{
							"description": "Relations deleted successfully",
							"content": map[string]interface{}{
								"application/json": map[string]interface{}{
									"schema": map[string]interface{}{
										"type": "object",
										"properties": map[string]interface{}{
											"status": map[string]interface{}{
												"type": "string",
											},
										},
									},
								},
							},
						},
						"400": map[string]interface{}{
							"description": "Invalid request body",
						},
						"500": map[string]interface{}{
							"description": "Internal server error",
						},
					},
				},
			},
			"/api/search_nodes": map[string]interface{}{
				"get": map[string]interface{}{
					"operationId": "go_search_nodes", // Renamed
					"summary":     "Search nodes based on query (Go API)",
					"tags":        []string{"Go API"},
					"parameters": []map[string]interface{}{
						{
							"name":     "query",
							"in":       "query",
							"required": true,
							"schema": map[string]interface{}{
								"type": "string",
							},
							"description": "Search string to match against entity names, types, and observation content",
						},
					},
					"responses": map[string]interface{}{
						"200": map[string]interface{}{
							"description": "Search results containing matching entities and relations",
							"content": map[string]interface{}{
								"application/json": map[string]interface{}{
									"schema": map[string]interface{}{
										"type": "object",
										"properties": map[string]interface{}{
											"entities": map[string]interface{}{
												"type": "array",
												"items": map[string]interface{}{
													"$ref": "#/components/schemas/Entity",
												},
											},
											"relations": map[string]interface{}{
												"type": "array",
												"items": map[string]interface{}{
													"$ref": "#/components/schemas/Relation",
												},
											},
										},
									},
								},
							},
						},
						"400": map[string]interface{}{
							"description": "Missing or invalid query parameter",
						},
						"500": map[string]interface{}{
							"description": "Internal server error",
						},
					},
				},
			},
			"/api/open_nodes": map[string]interface{}{
				"post": map[string]interface{}{
					"operationId": "go_open_nodes", // Renamed
					"summary":     "Retrieve specific nodes by name (Go API)",
					"tags":        []string{"Go API"},
					"requestBody": map[string]interface{}{
						"required": true,
						"content": map[string]interface{}{
							"application/json": map[string]interface{}{
								"schema": map[string]interface{}{
									"type": "object",
									"properties": map[string]interface{}{
										"names": map[string]interface{}{
											"type": "array",
											"items": map[string]interface{}{
												"type": "string",
											},
										},
									},
									"required": []string{"names"},
								},
							},
						},
					},
					"responses": map[string]interface{}{
						"200": map[string]interface{}{
							"description": "Requested entities and their relations",
							"content": map[string]interface{}{
								"application/json": map[string]interface{}{
									"schema": map[string]interface{}{
										"type": "object",
										"properties": map[string]interface{}{
											"entities": map[string]interface{}{
												"type": "array",
												"items": map[string]interface{}{
													"$ref": "#/components/schemas/Entity",
												},
											},
											"relations": map[string]interface{}{
												"type": "array",
												"items": map[string]interface{}{
													"$ref": "#/components/schemas/Relation",
												},
											},
										},
									},
								},
							},
						},
						"400": map[string]interface{}{
							"description": "Invalid request body",
						},
						"500": map[string]interface{}{
							"description": "Internal server error",
						},
					},
				},
			},
			"/mcp": map[string]interface{}{
				"post": map[string]interface{}{
					"operationId": "mcp_post", // Renamed
					"summary":     "MCP (Model Context Protocol) endpoint",
					"description": "JSON-RPC style endpoint for LLM integration",
					"tags":        []string{"MCP"},
					"requestBody": map[string]interface{}{
						"required": true,
						"content": map[string]interface{}{
							"application/json": map[string]interface{}{
								"schema": map[string]interface{}{
									"type": "object",
									"properties": map[string]interface{}{
										"method": map[string]interface{}{
											"type": "string",
										},
									},
								},
							},
						},
					},
					"responses": map[string]interface{}{
						"200": map[string]interface{}{
							"description": "MCP response",
							"content": map[string]interface{}{
								"application/json": map[string]interface{}{
									"schema": map[string]interface{}{
										"type": "object",
									},
								},
							},
						},
						"400": map[string]interface{}{
							"description": "Invalid request body",
						},
						"405": map[string]interface{}{
							"description": "Method not allowed",
						},
					},
				},
			},
		},
		"components": map[string]interface{}{
			"schemas": map[string]interface{}{
				// Go API Schemas
				"Entity": map[string]interface{}{
					"type":        "object",
					"description": "Represents an entity in the Go API format.",
					"properties": map[string]interface{}{
						"name": map[string]interface{}{
							"type": "string",
						},
						"entity_type": map[string]interface{}{ // Snake case
							"type": "string",
						},
					},
					"required": []string{"name", "entity_type"},
				},
				"Relation": map[string]interface{}{
					"type":        "object",
					"description": "Represents a relation in the Go API format.",
					"properties": map[string]interface{}{
						"id": map[string]interface{}{
							"type":   "integer",
							"format": "int64",
						},
						"from_entity": map[string]interface{}{ // Snake case
							"type": "string",
						},
						"to_entity": map[string]interface{}{ // Snake case
							"type": "string",
						},
						"relation_type": map[string]interface{}{ // Snake case
							"type": "string",
						},
					},
					"required": []string{"id", "from_entity", "to_entity", "relation_type"},
				},
				"Observation": map[string]interface{}{
					"type":        "object",
					"description": "Represents an observation in the Go API format.",
					"properties": map[string]interface{}{
						"id": map[string]interface{}{
							"type":   "integer",
							"format": "int64",
						},
						"entity_name": map[string]interface{}{ // Snake case
							"type": "string",
						},
						"content": map[string]interface{}{
							"type": "string",
						},
					},
					"required": []string{"id", "entity_name", "content"},
				},

				// Python Compatibility API Schemas
				"PythonEntity": map[string]interface{}{
					"type":        "object",
					"description": "Represents an entity in the Python-compatible API format.",
					"properties": map[string]interface{}{
						"name": map[string]interface{}{
							"type": "string",
						},
						"entityType": map[string]interface{}{ // Camel case
							"type": "string",
						},
						"observations": map[string]interface{}{
							"type":  "array",
							"items": map[string]interface{}{"type": "string"},
						},
					},
					"required": []string{"name", "entityType"},
				},
				"PythonRelation": map[string]interface{}{
					"type":        "object",
					"description": "Represents a relation in the Python-compatible API format.",
					"properties": map[string]interface{}{
						"from": map[string]interface{}{ // Camel case (matches Python client)
							"type": "string",
						},
						"to": map[string]interface{}{ // Camel case
							"type": "string",
						},
						"relationType": map[string]interface{}{ // Camel case
							"type": "string",
						},
					},
					"required": []string{"from", "to", "relationType"},
				},
				"PythonKnowledgeGraph": map[string]interface{}{
					"type":        "object",
					"description": "Represents the entire knowledge graph in Python-compatible API format.",
					"properties": map[string]interface{}{
						"entities": map[string]interface{}{
							"type":  "array",
							"items": map[string]interface{}{"$ref": "#/components/schemas/PythonEntity"},
						},
						"relations": map[string]interface{}{
							"type":  "array",
							"items": map[string]interface{}{"$ref": "#/components/schemas/PythonRelation"},
						},
					},
				},
			},
		},
	}
}

// GenerateOpenAPIJSON returns the OpenAPI spec as JSON bytes
func GenerateOpenAPIJSON() ([]byte, error) {
	spec := OpenAPISpec()
	return json.MarshalIndent(spec, "", "  ")
}
