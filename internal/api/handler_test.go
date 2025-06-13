package api

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"memory-parttwo/internal/db"
)

func setupTestAPI(t *testing.T) (*sql.DB, http.Handler) {
	tmpfile, err := os.CreateTemp("", "test_*.db")
	if err != nil {
		t.Fatal(err)
	}
	tmpfile.Close()
	
	database, err := db.Init(tmpfile.Name())
	if err != nil {
		t.Fatal(err)
	}
	
	t.Cleanup(func() {
		database.Close()
		os.Remove(tmpfile.Name())
	})
	
	handler := NewHandler(database, tmpfile.Name())
	return database, handler
}

func TestReadGraphAPI(t *testing.T) {
	database, handler := setupTestAPI(t)
	
	// Add some test data
	db.CreateEntity(database, "Alice", "person")
	db.CreateEntity(database, "Company", "organization")
	db.CreateRelation(database, "Alice", "Company", "works_at")
	db.CreateObservation(database, "Alice", "Software engineer")

	req := httptest.NewRequest("GET", "/api/read_graph", nil)
	w := httptest.NewRecorder()
	
	handler.ServeHTTP(w, req)
	
	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
	
	if w.Header().Get("Content-Type") != "application/json" {
		t.Error("Expected Content-Type application/json")
	}
	
	var response struct {
		Entities     []db.Entity     `json:"entities"`
		Relations    []db.Relation   `json:"relations"`
		Observations []db.Observation `json:"observations"`
	}
	
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}
	
	if len(response.Entities) != 2 {
		t.Errorf("Expected 2 entities, got %d", len(response.Entities))
	}
	if len(response.Relations) != 1 {
		t.Errorf("Expected 1 relation, got %d", len(response.Relations))
	}
	if len(response.Observations) != 1 {
		t.Errorf("Expected 1 observation, got %d", len(response.Observations))
	}
}

func TestReadGraphAPIWrongMethod(t *testing.T) {
	_, handler := setupTestAPI(t)
	
	req := httptest.NewRequest("POST", "/api/read_graph", nil)
	w := httptest.NewRecorder()
	
	handler.ServeHTTP(w, req)
	
	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("Expected status 405, got %d", w.Code)
	}
}

func TestCreateEntitiesAPI(t *testing.T) {
	_, handler := setupTestAPI(t)
	
	reqBody := map[string]interface{}{
		"entities": []map[string]string{
			{"name": "Alice", "entity_type": "person"},
			{"name": "Bob", "entity_type": "person"},
		},
	}
	
	jsonData, _ := json.Marshal(reqBody)
	req := httptest.NewRequest("POST", "/api/create_entities", bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	
	handler.ServeHTTP(w, req)
	
	if w.Code != http.StatusCreated {
		t.Errorf("Expected status 201, got %d", w.Code)
	}
	
	var response map[string]string
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}
	
	if response["status"] != "success" {
		t.Errorf("Expected success status, got %s", response["status"])
	}
}

func TestCreateEntitiesAPIInvalidJSON(t *testing.T) {
	_, handler := setupTestAPI(t)
	
	req := httptest.NewRequest("POST", "/api/create_entities", bytes.NewBufferString("invalid json"))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	
	handler.ServeHTTP(w, req)
	
	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", w.Code)
	}
}

func TestCreateEntitiesAPIWrongMethod(t *testing.T) {
	_, handler := setupTestAPI(t)
	
	req := httptest.NewRequest("GET", "/api/create_entities", nil)
	w := httptest.NewRecorder()
	
	handler.ServeHTTP(w, req)
	
	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("Expected status 405, got %d", w.Code)
	}
}

func TestCreateRelationsAPI(t *testing.T) {
	database, handler := setupTestAPI(t)
	
	// Create entities first
	db.CreateEntity(database, "Alice", "person")
	db.CreateEntity(database, "Company", "organization")
	
	reqBody := map[string]interface{}{
		"relations": []map[string]string{
			{"from_entity": "Alice", "to_entity": "Company", "relation_type": "works_at"},
		},
	}
	
	jsonData, _ := json.Marshal(reqBody)
	req := httptest.NewRequest("POST", "/api/create_relations", bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	
	handler.ServeHTTP(w, req)
	
	if w.Code != http.StatusCreated {
		t.Errorf("Expected status 201, got %d", w.Code)
	}
	
	var response map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}
	
	if response["status"] != "success" {
		t.Errorf("Expected success status, got %s", response["status"])
	}
	
	ids, ok := response["ids"].([]interface{})
	if !ok || len(ids) != 1 {
		t.Error("Expected ids array with one element")
	}
}

func TestCreateRelationsAPIInvalidEntity(t *testing.T) {
	_, handler := setupTestAPI(t)
	
	reqBody := map[string]interface{}{
		"relations": []map[string]string{
			{"from_entity": "NonExistent", "to_entity": "AlsoNonExistent", "relation_type": "knows"},
		},
	}
	
	jsonData, _ := json.Marshal(reqBody)
	req := httptest.NewRequest("POST", "/api/create_relations", bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	
	handler.ServeHTTP(w, req)
	
	if w.Code != http.StatusInternalServerError {
		t.Errorf("Expected status 500, got %d", w.Code)
	}
}

func TestCreateRelationsAPIWrongMethod(t *testing.T) {
	_, handler := setupTestAPI(t)
	
	req := httptest.NewRequest("GET", "/api/create_relations", nil)
	w := httptest.NewRecorder()
	
	handler.ServeHTTP(w, req)
	
	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("Expected status 405, got %d", w.Code)
	}
}

func TestIntegrationWorkflow(t *testing.T) {
	_, handler := setupTestAPI(t)
	
	// 1. Create entities
	entitiesReq := map[string]interface{}{
		"entities": []map[string]string{
			{"name": "Alice", "entity_type": "person"},
			{"name": "TechCorp", "entity_type": "organization"},
		},
	}
	
	jsonData, _ := json.Marshal(entitiesReq)
	req := httptest.NewRequest("POST", "/api/create_entities", bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	
	if w.Code != http.StatusCreated {
		t.Fatalf("Failed to create entities: status %d", w.Code)
	}
	
	// 2. Create relations
	relationsReq := map[string]interface{}{
		"relations": []map[string]string{
			{"from_entity": "Alice", "to_entity": "TechCorp", "relation_type": "works_at"},
		},
	}
	
	jsonData, _ = json.Marshal(relationsReq)
	req = httptest.NewRequest("POST", "/api/create_relations", bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	
	if w.Code != http.StatusCreated {
		t.Fatalf("Failed to create relations: status %d", w.Code)
	}
	
	// 3. Read the graph
	req = httptest.NewRequest("GET", "/api/read_graph", nil)
	w = httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	
	if w.Code != http.StatusOK {
		t.Fatalf("Failed to read graph: status %d", w.Code)
	}
	
	var response struct {
		Entities     []db.Entity     `json:"entities"`
		Relations    []db.Relation   `json:"relations"`
		Observations []db.Observation `json:"observations"`
	}
	
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}
	
	// Verify the complete workflow
	if len(response.Entities) != 2 {
		t.Errorf("Expected 2 entities, got %d", len(response.Entities))
	}
	if len(response.Relations) != 1 {
		t.Errorf("Expected 1 relation, got %d", len(response.Relations))
	}
	if response.Relations[0].From != "Alice" || response.Relations[0].To != "TechCorp" {
		t.Error("Relation data mismatch in integration test")
	}
}