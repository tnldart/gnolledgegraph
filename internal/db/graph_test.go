package db

import (
	"testing"
)

func TestCreateEntity(t *testing.T) {
	db := setupTestDB(t)

	tests := []struct {
		name       string
		entityName string
		entityType string
		wantErr    bool
	}{
		{"valid entity", "Alice", "person", false},
		{"another valid entity", "Company", "organization", false},
		{"duplicate entity", "Alice", "person", false}, // INSERT OR IGNORE should not error
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := CreateEntity(db, tt.entityName, tt.entityType)
			if (err != nil) != tt.wantErr {
				t.Errorf("CreateEntity() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}

	// Verify entities were created
	entities, _, _, err := ReadGraph(db)
	if err != nil {
		t.Fatalf("ReadGraph() failed: %v", err)
	}
	if len(entities) != 2 { // Alice should only appear once due to OR IGNORE
		t.Errorf("Expected 2 entities, got %d", len(entities))
	}
}

func TestCreateRelation(t *testing.T) {
	db := setupTestDB(t)

	// Create entities first
	CreateEntity(db, "Alice", "person")
	CreateEntity(db, "Company", "organization")

	tests := []struct {
		name         string
		from         string
		to           string
		relationType string
		wantErr      bool
	}{
		{"valid relation", "Alice", "Company", "works_at", false},
		{"invalid from entity", "Bob", "Company", "works_at", true}, // Foreign key constraint
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			id, err := CreateRelation(db, tt.from, tt.to, tt.relationType)
			if (err != nil) != tt.wantErr {
				t.Errorf("CreateRelation() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !tt.wantErr && id <= 0 {
				t.Errorf("CreateRelation() should return valid ID, got %d", id)
			}
		})
	}
}

func TestCreateObservation(t *testing.T) {
	db := setupTestDB(t)

	// Create entity first
	CreateEntity(db, "Alice", "person")

	tests := []struct {
		name       string
		entityName string
		content    string
		wantErr    bool
	}{
		{"valid observation", "Alice", "Alice is a software engineer", false},
		{"invalid entity", "Bob", "Bob is unknown", true}, // Foreign key constraint
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			id, err := CreateObservation(db, tt.entityName, tt.content)
			if (err != nil) != tt.wantErr {
				t.Errorf("CreateObservation() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !tt.wantErr && id <= 0 {
				t.Errorf("CreateObservation() should return valid ID, got %d", id)
			}
		})
	}
}

func TestReadGraph(t *testing.T) {
	db := setupTestDB(t)

	// Create test data
	CreateEntity(db, "Alice", "person")
	CreateEntity(db, "Company", "organization")
	relID, _ := CreateRelation(db, "Alice", "Company", "works_at")
	obsID, _ := CreateObservation(db, "Alice", "Alice is a software engineer")

	// Read the graph
	entities, relations, observations, err := ReadGraph(db)
	if err != nil {
		t.Fatalf("ReadGraph() failed: %v", err)
	}

	// Verify entities
	if len(entities) != 2 {
		t.Errorf("Expected 2 entities, got %d", len(entities))
	}
	found := false
	for _, e := range entities {
		if e.Name == "Alice" && e.Type == "person" {
			found = true
			break
		}
	}
	if !found {
		t.Error("Alice entity not found")
	}

	// Verify relations
	if len(relations) != 1 {
		t.Errorf("Expected 1 relation, got %d", len(relations))
	}
	if relations[0].ID != relID {
		t.Errorf("Expected relation ID %d, got %d", relID, relations[0].ID)
	}
	if relations[0].From != "Alice" || relations[0].To != "Company" || relations[0].Type != "works_at" {
		t.Error("Relation data mismatch")
	}

	// Verify observations
	if len(observations) != 1 {
		t.Errorf("Expected 1 observation, got %d", len(observations))
	}
	if observations[0].ID != obsID {
		t.Errorf("Expected observation ID %d, got %d", obsID, observations[0].ID)
	}
	if observations[0].EntityName != "Alice" || observations[0].Content != "Alice is a software engineer" {
		t.Error("Observation data mismatch")
	}
}

func TestReadGraphEmpty(t *testing.T) {
	db := setupTestDB(t)

	entities, relations, observations, err := ReadGraph(db)
	if err != nil {
		t.Fatalf("ReadGraph() failed: %v", err)
	}

	if len(entities) != 0 || len(relations) != 0 || len(observations) != 0 {
		t.Error("Empty database should return empty slices")
	}
}