package api

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"memory-parttwo/internal/db"

	_ "github.com/mattn/go-sqlite3"
)

func setupTestDB(t *testing.T) *sql.DB {
	database, err := db.Init(":memory:")
	if err != nil {
		t.Fatalf("Failed to initialize database: %v", err)
	}

	return database
}

func TestPythonReadGraph(t *testing.T) {
	database := setupTestDB(t)
	defer database.Close()

	// Add test data
	db.CreateEntity(database, "Python", "Language")
	db.CreateEntity(database, "Django", "Framework")
	db.CreateRelation(database, "Python", "Django", "hasFramework")
	db.CreateObservation(database, "Python", "High-level")
	db.CreateObservation(database, "Python", "Interpreted")

	handler := NewPythonCompatHandler(database)

	req := httptest.NewRequest("GET", "/read_graph", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var response PythonKnowledgeGraph
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	// Check entities
	if len(response.Entities) != 2 {
		t.Errorf("Expected 2 entities, got %d", len(response.Entities))
	}

	// Find Python entity
	var pythonEntity *PythonEntity
	for _, entity := range response.Entities {
		if entity.Name == "Python" {
			pythonEntity = &entity
			break
		}
	}

	if pythonEntity == nil {
		t.Fatal("Python entity not found")
	}

	if pythonEntity.EntityType != "Language" {
		t.Errorf("Expected entity type 'Language', got '%s'", pythonEntity.EntityType)
	}

	if len(pythonEntity.Observations) != 2 {
		t.Errorf("Expected 2 observations, got %d", len(pythonEntity.Observations))
	}

	// Check relations
	if len(response.Relations) != 1 {
		t.Errorf("Expected 1 relation, got %d", len(response.Relations))
	}

	relation := response.Relations[0]
	if relation.From != "Python" || relation.To != "Django" || relation.RelationType != "hasFramework" {
		t.Errorf("Unexpected relation: %+v", relation)
	}
}

func TestPythonCreateEntities(t *testing.T) {
	database := setupTestDB(t)
	defer database.Close()

	handler := NewPythonCompatHandler(database)

	requestBody := struct {
		Entities []PythonEntity `json:"entities"`
	}{
		Entities: []PythonEntity{
			{
				Name:         "Python",
				EntityType:   "Language",
				Observations: []string{"High-level", "Interpreted"},
			},
			{
				Name:         "Django",
				EntityType:   "Framework",
				Observations: []string{"Web framework"},
			},
		},
	}

	body, _ := json.Marshal(requestBody)
	req := httptest.NewRequest("POST", "/create_entities", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("Expected status 201, got %d", w.Code)
	}

	var response []PythonEntity
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if len(response) != 2 {
		t.Errorf("Expected 2 entities in response, got %d", len(response))
	}

	// Verify entities were created in database
	entities, _, observations, err := db.ReadGraph(database)
	if err != nil {
		t.Fatalf("Failed to read graph: %v", err)
	}

	if len(entities) != 2 {
		t.Errorf("Expected 2 entities in database, got %d", len(entities))
	}

	if len(observations) != 3 {
		t.Errorf("Expected 3 observations in database, got %d", len(observations))
	}
}
func TestPythonCreateEntitiesConflict(t *testing.T) {
	database := setupTestDB(t)
	defer database.Close()

	handler := NewPythonCompatHandler(database)

	// First, create an entity
	initialEntity := PythonEntity{
		Name:         "ConflictEntity",
		EntityType:   "TestType",
		Observations: []string{"Initial observation"},
	}
	err := db.CreateEntity(database, initialEntity.Name, initialEntity.EntityType)
	if err != nil {
		t.Fatalf("Failed to create initial entity for conflict test: %v", err)
	}
	for _, obs := range initialEntity.Observations {
		_, err := db.CreateObservation(database, initialEntity.Name, obs)
		if err != nil {
			t.Fatalf("Failed to create initial observation for conflict test: %v", err)
		}
	}

	// Now, attempt to create it again along with a new one
	requestBody := struct {
		Entities []PythonEntity `json:"entities"`
	}{
		Entities: []PythonEntity{
			initialEntity, // This one should cause a conflict
			{
				Name:         "NewEntity",
				EntityType:   "AnotherType",
				Observations: []string{"New observation"},
			},
		},
	}

	body, _ := json.Marshal(requestBody)
	req := httptest.NewRequest("POST", "/create_entities", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusConflict {
		t.Errorf("Expected status %d, got %d. Body: %s", http.StatusConflict, w.Code, w.Body.String())
	}

	var response map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode conflict response: %v", err)
	}

	if errMsg, ok := response["error"].(string); !ok || errMsg != "entities already exist" {
		t.Errorf("Expected error message 'entities already exist', got '%v'", response["error"])
	}

	conflicting, ok := response["conflicting_entities"].([]interface{})
	if !ok {
		t.Fatalf("Expected 'conflicting_entities' to be an array, got %T", response["conflicting_entities"])
	}
	if len(conflicting) != 1 {
		t.Errorf("Expected 1 conflicting entity, got %d", len(conflicting))
	}
	if conflicting[0].(string) != "ConflictEntity" {
		t.Errorf("Expected conflicting entity 'ConflictEntity', got '%s'", conflicting[0])
	}

	// Verify that "NewEntity" was not created due to the conflict
	entities, _, _, err := db.ReadGraph(database)
	if err != nil {
		t.Fatalf("Failed to read graph: %v", err)
	}
	foundNewEntity := false
	for _, e := range entities {
		if e.Name == "NewEntity" {
			foundNewEntity = true
			break
		}
	}
	if foundNewEntity {
		t.Error("NewEntity should not have been created when a conflict occurred in the batch")
	}
	if len(entities) != 1 { // Should only contain ConflictEntity
		t.Errorf("Expected only 1 entity in the database after conflict, found %d", len(entities))
	}
}

func TestPythonCreateRelations(t *testing.T) {
	database := setupTestDB(t)
	defer database.Close()

	// Create entities first
	db.CreateEntity(database, "Python", "Language")
	db.CreateEntity(database, "Django", "Framework")

	handler := NewPythonCompatHandler(database)

	requestBody := struct {
		Relations []PythonRelation `json:"relations"`
	}{
		Relations: []PythonRelation{
			{
				From:         "Python",
				To:           "Django",
				RelationType: "hasFramework",
			},
		},
	}

	body, _ := json.Marshal(requestBody)
	req := httptest.NewRequest("POST", "/create_relations", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("Expected status 201, got %d", w.Code)
	}

	var response []PythonRelation
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if len(response) != 1 {
		t.Errorf("Expected 1 relation in response, got %d", len(response))
	}

	// Verify relation was created in database
	_, relations, _, err := db.ReadGraph(database)
	if err != nil {
		t.Fatalf("Failed to read graph: %v", err)
	}

	if len(relations) != 1 {
		t.Errorf("Expected 1 relation in database, got %d", len(relations))
	}
}

func TestPythonCreateRelationsNonExistentEntity(t *testing.T) {
	database := setupTestDB(t)
	defer database.Close()

	handler := NewPythonCompatHandler(database)

	requestBody := struct {
		Relations []PythonRelation `json:"relations"`
	}{
		Relations: []PythonRelation{
			{
				From:         "NonExistent",
				To:           "AlsoNonExistent",
				RelationType: "hasFramework",
			},
		},
	}

	body, _ := json.Marshal(requestBody)
	req := httptest.NewRequest("POST", "/create_relations", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", w.Code)
	}
}

func TestPythonAddObservations(t *testing.T) {
	database := setupTestDB(t)
	defer database.Close()

	// Create entity first
	db.CreateEntity(database, "Python", "Language")

	handler := NewPythonCompatHandler(database)

	requestBody := struct {
		Observations []struct {
			EntityName string   `json:"entityName"`
			Contents   []string `json:"contents"`
		} `json:"observations"`
	}{
		Observations: []struct {
			EntityName string   `json:"entityName"`
			Contents   []string `json:"contents"`
		}{
			{
				EntityName: "Python",
				Contents:   []string{"High-level", "Interpreted"},
			},
		},
	}

	body, _ := json.Marshal(requestBody)
	req := httptest.NewRequest("POST", "/add_observations", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("Expected status 201, got %d", w.Code)
	}

	// Verify observations were added
	_, _, observations, err := db.ReadGraph(database)
	if err != nil {
		t.Fatalf("Failed to read graph: %v", err)
	}

	if len(observations) != 2 {
		t.Errorf("Expected 2 observations in database, got %d", len(observations))
	}
}

func TestPythonSearchNodes(t *testing.T) {
	database := setupTestDB(t)
	defer database.Close()

	// Add test data
	db.CreateEntity(database, "Python", "Language")
	db.CreateEntity(database, "Django", "Framework")
	db.CreateObservation(database, "Python", "programming language")

	handler := NewPythonCompatHandler(database)

	requestBody := struct {
		Query string `json:"query"`
	}{
		Query: "programming",
	}

	body, _ := json.Marshal(requestBody)
	req := httptest.NewRequest("POST", "/search_nodes", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var response PythonKnowledgeGraph
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	// Should find Python entity due to observation content
	if len(response.Entities) == 0 {
		t.Error("Expected to find entities, got none")
	}

	found := false
	for _, entity := range response.Entities {
		if entity.Name == "Python" {
			found = true
			break
		}
	}

	if !found {
		t.Error("Expected to find Python entity in search results")
	}
}

func TestPythonOpenNodes(t *testing.T) {
	database := setupTestDB(t)
	defer database.Close()

	// Add test data
	db.CreateEntity(database, "Python", "Language")
	db.CreateEntity(database, "Django", "Framework")
	db.CreateObservation(database, "Python", "High-level")

	handler := NewPythonCompatHandler(database)

	requestBody := struct {
		Names []string `json:"names"`
	}{
		Names: []string{"Python", "Django"},
	}

	body, _ := json.Marshal(requestBody)
	req := httptest.NewRequest("POST", "/open_nodes", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var response PythonKnowledgeGraph
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if len(response.Entities) != 2 {
		t.Errorf("Expected 2 entities, got %d", len(response.Entities))
	}
}

func TestPythonDeleteEntities(t *testing.T) {
	database := setupTestDB(t)
	defer database.Close()

	// Add test data
	db.CreateEntity(database, "Python", "Language")
	db.CreateEntity(database, "Django", "Framework")

	handler := NewPythonCompatHandler(database)

	requestBody := struct {
		EntityNames []string `json:"entityNames"`
	}{
		EntityNames: []string{"Python"},
	}

	body, _ := json.Marshal(requestBody)
	req := httptest.NewRequest("POST", "/delete_entities", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	// Verify entity was deleted
	entities, _, _, err := db.ReadGraph(database)
	if err != nil {
		t.Fatalf("Failed to read graph: %v", err)
	}

	if len(entities) != 1 {
		t.Errorf("Expected 1 entity remaining, got %d", len(entities))
	}

	if entities[0].Name == "Python" {
		t.Error("Python entity should have been deleted")
	}
}

func TestPythonDeleteObservations(t *testing.T) {
	database := setupTestDB(t)
	defer database.Close()

	// Add test data
	db.CreateEntity(database, "Python", "Language")
	db.CreateObservation(database, "Python", "High-level")
	db.CreateObservation(database, "Python", "Interpreted")

	handler := NewPythonCompatHandler(database)

	requestBody := struct {
		Deletions []struct {
			EntityName   string   `json:"entityName"`
			Observations []string `json:"observations"`
		} `json:"deletions"`
	}{
		Deletions: []struct {
			EntityName   string   `json:"entityName"`
			Observations []string `json:"observations"`
		}{
			{
				EntityName:   "Python",
				Observations: []string{"High-level"},
			},
		},
	}

	body, _ := json.Marshal(requestBody)
	req := httptest.NewRequest("POST", "/delete_observations", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	// Verify observation was deleted
	_, _, observations, err := db.ReadGraph(database)
	if err != nil {
		t.Fatalf("Failed to read graph: %v", err)
	}

	if len(observations) != 1 {
		t.Errorf("Expected 1 observation remaining, got %d", len(observations))
	}

	// Check that the correct observation remains
	if observations[0].Content == "High-level" {
		t.Error("High-level observation should have been deleted")
	}
}

func TestPythonDeleteRelations(t *testing.T) {
	database := setupTestDB(t)
	defer database.Close()

	// Add test data
	db.CreateEntity(database, "Python", "Language")
	db.CreateEntity(database, "Django", "Framework")
	db.CreateRelation(database, "Python", "Django", "hasFramework")

	handler := NewPythonCompatHandler(database)

	requestBody := struct {
		Relations []PythonRelation `json:"relations"`
	}{
		Relations: []PythonRelation{
			{
				From:         "Python",
				To:           "Django",
				RelationType: "hasFramework",
			},
		},
	}

	body, _ := json.Marshal(requestBody)
	req := httptest.NewRequest("POST", "/delete_relations", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	// Verify relation was deleted
	_, relations, _, err := db.ReadGraph(database)
	if err != nil {
		t.Fatalf("Failed to read graph: %v", err)
	}

	if len(relations) != 0 {
		t.Errorf("Expected 0 relations remaining, got %d", len(relations))
	}
}

func TestPythonCORSHeaders(t *testing.T) {
	database := setupTestDB(t)
	defer database.Close()

	handler := NewPythonCompatHandler(database)

	req := httptest.NewRequest("OPTIONS", "/read_graph", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200 for OPTIONS, got %d", w.Code)
	}

	// Check CORS headers
	if w.Header().Get("Access-Control-Allow-Origin") != "*" {
		t.Error("Missing or incorrect CORS origin header")
	}

	if w.Header().Get("Access-Control-Allow-Methods") == "" {
		t.Error("Missing CORS methods header")
	}
}

func TestPythonMethodNotAllowed(t *testing.T) {
	database := setupTestDB(t)
	defer database.Close()

	handler := NewPythonCompatHandler(database)

	// Test wrong method on read_graph (should be GET)
	req := httptest.NewRequest("POST", "/read_graph", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("Expected status 405, got %d", w.Code)
	}
}

func TestPythonInvalidJSON(t *testing.T) {
	database := setupTestDB(t)
	defer database.Close()

	handler := NewPythonCompatHandler(database)

	req := httptest.NewRequest("POST", "/create_entities", bytes.NewReader([]byte("invalid json")))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", w.Code)
	}
}
