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
			"description": "API for managing and querying a knowledge graph.\n    *   Embeds observations directly within entity structures for request and response bodies.\n    *   Provides direct data models in responses for some operations, rather than status wrappers.",
		},
		"servers": []map[string]interface{}{
			{
				"url":         "http://localhost:8080",
				"description": "Local dev server",
			},
		},
		"tags": []map[string]interface{}{
			{
				"name":        "Root-path API",
				"description": "Endpoints at the root path offering embedded observation models and modern conventions.",
			},
		},
		"paths": map[string]interface{}{
			// Client Compatibility API Endpoints
			"/read_graph": map[string]interface{}{
				"get": map[string]interface{}{
					"operationId": "compat_read_graph",
					"summary":     "Read the complete knowledge graph",
					"responses": map[string]interface{}{
						"200": map[string]interface{}{
							"description": "Graph data for the entire knowledge graph",
							"content": map[string]interface{}{
								"application/json": map[string]interface{}{
									"schema": map[string]interface{}{
										"$ref": "#/components/schemas/CompatibleKnowledgeGraph",
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
					"operationId": "compat_create_entities",
					"summary":     "Create new entities with observations",
					"requestBody": map[string]interface{}{
						"required": true,
						"content": map[string]interface{}{
							"application/json": map[string]interface{}{
								"schema": map[string]interface{}{
									"type": "object",
									"properties": map[string]interface{}{
										"entities": map[string]interface{}{
											"type":  "array",
											"items": map[string]interface{}{"$ref": "#/components/schemas/CompatibleEntity"},
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
					"operationId": "compat_create_relations",
					"summary":     "Create new relations",
					"requestBody": map[string]interface{}{
						"required": true,
						"content": map[string]interface{}{
							"application/json": map[string]interface{}{
								"schema": map[string]interface{}{
									"type": "object",
									"properties": map[string]interface{}{
										"relations": map[string]interface{}{
											"type":  "array",
											"items": map[string]interface{}{"$ref": "#/components/schemas/CompatibleRelation"},
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
										"items": map[string]interface{}{"$ref": "#/components/schemas/CompatibleRelation"},
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
					"operationId": "compat_add_observations",
					"summary":     "Add observations to entities",
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
					"operationId": "compat_search_nodes",
					"summary":     "Search nodes",
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
							"description": "Search results in client-compatible API format",
							"content": map[string]interface{}{
								"application/json": map[string]interface{}{
									"schema": map[string]interface{}{"$ref": "#/components/schemas/CompatibleKnowledgeGraph"},
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
					"operationId": "compat_open_nodes",
					"summary":     "Retrieve nodes by name",
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
							"description": "Requested entities and relations",
							"content": map[string]interface{}{
								"application/json": map[string]interface{}{
									"schema": map[string]interface{}{"$ref": "#/components/schemas/CompatibleKnowledgeGraph"},
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
					"operationId": "compat_delete_entities",
					"summary":     "Delete entities",
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
					"operationId": "compat_delete_observations",
					"summary":     "Delete observations",
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
					"operationId": "compat_delete_relations",
					"summary":     "Delete relations",
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
		},
		"components": map[string]interface{}{
			"schemas": map[string]interface{}{

				// Python Compatibility API Schemas
				"PythonEntity": map[string]interface{}{
					"type":        "object",
					"description": "Represents an entity in the knowledge graph.",
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
				"CompatibleRelation": map[string]interface{}{
					"type":        "object",
					"description": "Represents a relation between entities.",
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
				"CompatibleKnowledgeGraph": map[string]interface{}{
					"type":        "object",
					"description": "The full knowledge graph with entities and relations.",
					"properties": map[string]interface{}{
						"entities": map[string]interface{}{
							"type":  "array",
							"items": map[string]interface{}{"$ref": "#/components/schemas/CompatibleEntity"},
						},
						"relations": map[string]interface{}{
							"type":  "array",
							"items": map[string]interface{}{"$ref": "#/components/schemas/CompatibleRelation"},
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
