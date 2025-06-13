package api

import (
	"reflect"
	"testing"

	"memory-parttwo/internal/db"
)

func TestTransformToPython(t *testing.T) {
	// Test data
	entities := []db.Entity{
		{Name: "Python", Type: "Language"},
		{Name: "Django", Type: "Framework"},
	}

	relations := []db.Relation{
		{ID: 1, From: "Python", To: "Django", Type: "hasFramework"},
	}

	observations := []db.Observation{
		{ID: 1, EntityName: "Python", Content: "High-level"},
		{ID: 2, EntityName: "Python", Content: "Interpreted"},
		{ID: 3, EntityName: "Django", Content: "Web framework"},
	}

	result := TransformToPython(entities, relations, observations)

	// Check entities
	if len(result.Entities) != 2 {
		t.Errorf("Expected 2 entities, got %d", len(result.Entities))
	}

	// Check Python entity
	pythonEntity := result.Entities[0]
	if pythonEntity.Name != "Python" {
		t.Errorf("Expected entity name 'Python', got '%s'", pythonEntity.Name)
	}
	if pythonEntity.EntityType != "Language" {
		t.Errorf("Expected entity type 'Language', got '%s'", pythonEntity.EntityType)
	}
	if len(pythonEntity.Observations) != 2 {
		t.Errorf("Expected 2 observations for Python, got %d", len(pythonEntity.Observations))
	}

	// Check relations
	if len(result.Relations) != 1 {
		t.Errorf("Expected 1 relation, got %d", len(result.Relations))
	}

	relation := result.Relations[0]
	if relation.From != "Python" {
		t.Errorf("Expected relation from 'Python', got '%s'", relation.From)
	}
	if relation.To != "Django" {
		t.Errorf("Expected relation to 'Django', got '%s'", relation.To)
	}
	if relation.RelationType != "hasFramework" {
		t.Errorf("Expected relation type 'hasFramework', got '%s'", relation.RelationType)
	}
}

func TestTransformFromPython(t *testing.T) {
	// Test data
	pythonGraph := PythonKnowledgeGraph{
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
		Relations: []PythonRelation{
			{From: "Python", To: "Django", RelationType: "hasFramework"},
		},
	}

	entities, relations, observations := TransformFromPython(pythonGraph)

	// Check entities
	if len(entities) != 2 {
		t.Errorf("Expected 2 entities, got %d", len(entities))
	}

	pythonEntity := entities[0]
	if pythonEntity.Name != "Python" {
		t.Errorf("Expected entity name 'Python', got '%s'", pythonEntity.Name)
	}
	if pythonEntity.Type != "Language" {
		t.Errorf("Expected entity type 'Language', got '%s'", pythonEntity.Type)
	}

	// Check relations
	if len(relations) != 1 {
		t.Errorf("Expected 1 relation, got %d", len(relations))
	}

	relation := relations[0]
	if relation.From != "Python" {
		t.Errorf("Expected relation from 'Python', got '%s'", relation.From)
	}
	if relation.To != "Django" {
		t.Errorf("Expected relation to 'Django', got '%s'", relation.To)
	}
	if relation.Type != "hasFramework" {
		t.Errorf("Expected relation type 'hasFramework', got '%s'", relation.Type)
	}

	// Check observations
	if len(observations) != 3 {
		t.Errorf("Expected 3 observations, got %d", len(observations))
	}

	// Check that observations are correctly extracted
	pythonObs := 0
	djangoObs := 0
	for _, obs := range observations {
		if obs.EntityName == "Python" {
			pythonObs++
		} else if obs.EntityName == "Django" {
			djangoObs++
		}
	}

	if pythonObs != 2 {
		t.Errorf("Expected 2 observations for Python, got %d", pythonObs)
	}
	if djangoObs != 1 {
		t.Errorf("Expected 1 observation for Django, got %d", djangoObs)
	}
}

func TestEmbedObservationsInEntities(t *testing.T) {
	entities := []db.Entity{
		{Name: "Python", Type: "Language"},
		{Name: "Django", Type: "Framework"},
	}

	observations := []db.Observation{
		{ID: 1, EntityName: "Python", Content: "High-level"},
		{ID: 2, EntityName: "Python", Content: "Interpreted"},
		{ID: 3, EntityName: "Django", Content: "Web framework"},
	}

	result := EmbedObservationsInEntities(entities, observations)

	if len(result) != 2 {
		t.Errorf("Expected 2 entities, got %d", len(result))
	}

	// Check Python entity
	pythonEntity := result[0]
	if len(pythonEntity.Observations) != 2 {
		t.Errorf("Expected 2 observations for Python, got %d", len(pythonEntity.Observations))
	}

	// Check Django entity
	djangoEntity := result[1]
	if len(djangoEntity.Observations) != 1 {
		t.Errorf("Expected 1 observation for Django, got %d", len(djangoEntity.Observations))
	}
}

func TestExtractObservationsFromEntities(t *testing.T) {
	pythonEntities := []PythonEntity{
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
	}

	entities, observations := ExtractObservationsFromEntities(pythonEntities)

	// Check entities
	if len(entities) != 2 {
		t.Errorf("Expected 2 entities, got %d", len(entities))
	}

	// Check observations
	if len(observations) != 3 {
		t.Errorf("Expected 3 observations, got %d", len(observations))
	}

	// Verify observation extraction
	pythonObs := 0
	for _, obs := range observations {
		if obs.EntityName == "Python" {
			pythonObs++
		}
	}

	if pythonObs != 2 {
		t.Errorf("Expected 2 observations for Python, got %d", pythonObs)
	}
}

func TestTransformRelationsToPython(t *testing.T) {
	relations := []db.Relation{
		{ID: 1, From: "Python", To: "Django", Type: "hasFramework"},
		{ID: 2, From: "Django", To: "REST", Type: "supports"},
	}

	result := TransformRelationsToPython(relations)

	if len(result) != 2 {
		t.Errorf("Expected 2 relations, got %d", len(result))
	}

	relation := result[0]
	if relation.From != "Python" {
		t.Errorf("Expected relation from 'Python', got '%s'", relation.From)
	}
	if relation.To != "Django" {
		t.Errorf("Expected relation to 'Django', got '%s'", relation.To)
	}
	if relation.RelationType != "hasFramework" {
		t.Errorf("Expected relation type 'hasFramework', got '%s'", relation.RelationType)
	}
}

func TestTransformRelationsFromPython(t *testing.T) {
	pythonRelations := []PythonRelation{
		{From: "Python", To: "Django", RelationType: "hasFramework"},
		{From: "Django", To: "REST", RelationType: "supports"},
	}

	result := TransformRelationsFromPython(pythonRelations)

	if len(result) != 2 {
		t.Errorf("Expected 2 relations, got %d", len(result))
	}

	relation := result[0]
	if relation.From != "Python" {
		t.Errorf("Expected relation from 'Python', got '%s'", relation.From)
	}
	if relation.To != "Django" {
		t.Errorf("Expected relation to 'Django', got '%s'", relation.To)
	}
	if relation.Type != "hasFramework" {
		t.Errorf("Expected relation type 'hasFramework', got '%s'", relation.Type)
	}
}

func TestEmptyObservations(t *testing.T) {
	entities := []db.Entity{
		{Name: "Python", Type: "Language"},
	}

	observations := []db.Observation{} // Empty observations

	result := EmbedObservationsInEntities(entities, observations)

	if len(result) != 1 {
		t.Errorf("Expected 1 entity, got %d", len(result))
	}

	entity := result[0]
	if entity.Observations == nil {
		t.Error("Expected empty slice, got nil")
	}
	if len(entity.Observations) != 0 {
		t.Errorf("Expected 0 observations, got %d", len(entity.Observations))
	}
}

func TestBidirectionalTransformation(t *testing.T) {
	// Test that transforming to Python and back preserves data integrity
	originalEntities := []db.Entity{
		{Name: "Python", Type: "Language"},
		{Name: "Django", Type: "Framework"},
	}

	originalRelations := []db.Relation{
		{ID: 1, From: "Python", To: "Django", Type: "hasFramework"},
	}

	originalObservations := []db.Observation{
		{ID: 1, EntityName: "Python", Content: "High-level"},
		{ID: 2, EntityName: "Django", Content: "Web framework"},
	}

	// Transform to Python format
	pythonGraph := TransformToPython(originalEntities, originalRelations, originalObservations)

	// Transform back to Go format
	entities, relations, observations := TransformFromPython(pythonGraph)

	// Check entities (ignore ID fields as they're not preserved)
	if len(entities) != len(originalEntities) {
		t.Errorf("Entity count mismatch: expected %d, got %d", len(originalEntities), len(entities))
	}

	for i, entity := range entities {
		if entity.Name != originalEntities[i].Name {
			t.Errorf("Entity name mismatch: expected %s, got %s", originalEntities[i].Name, entity.Name)
		}
		if entity.Type != originalEntities[i].Type {
			t.Errorf("Entity type mismatch: expected %s, got %s", originalEntities[i].Type, entity.Type)
		}
	}

	// Check relations (ignore ID fields)
	if len(relations) != len(originalRelations) {
		t.Errorf("Relation count mismatch: expected %d, got %d", len(originalRelations), len(relations))
	}

	for i, relation := range relations {
		if relation.From != originalRelations[i].From {
			t.Errorf("Relation from mismatch: expected %s, got %s", originalRelations[i].From, relation.From)
		}
		if relation.To != originalRelations[i].To {
			t.Errorf("Relation to mismatch: expected %s, got %s", originalRelations[i].To, relation.To)
		}
		if relation.Type != originalRelations[i].Type {
			t.Errorf("Relation type mismatch: expected %s, got %s", originalRelations[i].Type, relation.Type)
		}
	}

	// Check observations (ignore ID fields)
	if len(observations) != len(originalObservations) {
		t.Errorf("Observation count mismatch: expected %d, got %d", len(originalObservations), len(observations))
	}

	// Create maps for easier comparison
	originalObsMap := make(map[string][]string)
	for _, obs := range originalObservations {
		originalObsMap[obs.EntityName] = append(originalObsMap[obs.EntityName], obs.Content)
	}

	transformedObsMap := make(map[string][]string)
	for _, obs := range observations {
		transformedObsMap[obs.EntityName] = append(transformedObsMap[obs.EntityName], obs.Content)
	}

	if !reflect.DeepEqual(originalObsMap, transformedObsMap) {
		t.Errorf("Observation content mismatch: expected %v, got %v", originalObsMap, transformedObsMap)
	}
}

// Benchmark data
var (
	benchEntities = []db.Entity{
		{Name: "Python", Type: "Language"},
		{Name: "Django", Type: "Framework"},
		{Name: "Flask", Type: "MicroFramework"},
		{Name: "FastAPI", Type: "APIFramework"},
		{Name: "Go", Type: "Language"},
		{Name: "Gin", Type: "Framework"},
	}
	benchRelations = []db.Relation{
		{ID: 1, From: "Python", To: "Django", Type: "hasFramework"},
		{ID: 2, From: "Python", To: "Flask", Type: "hasFramework"},
		{ID: 3, From: "Python", To: "FastAPI", Type: "hasFramework"},
		{ID: 4, From: "Go", To: "Gin", Type: "hasFramework"},
	}
	benchObservations = []db.Observation{
		{ID: 1, EntityName: "Python", Content: "High-level"},
		{ID: 2, EntityName: "Python", Content: "Interpreted"},
		{ID: 3, EntityName: "Django", Content: "Web framework"},
		{ID: 4, EntityName: "Flask", Content: "Micro web framework"},
		{ID: 5, EntityName: "FastAPI", Content: "Modern, fast web framework"},
		{ID: 6, EntityName: "Go", Content: "Statically typed"},
		{ID: 7, EntityName: "Go", Content: "Compiled"},
		{ID: 8, EntityName: "Gin", Content: "HTTP web framework"},
	}
	benchPythonGraph = PythonKnowledgeGraph{
		Entities: []PythonEntity{
			{Name: "Python", EntityType: "Language", Observations: []string{"High-level", "Interpreted"}},
			{Name: "Django", EntityType: "Framework", Observations: []string{"Web framework"}},
			{Name: "Flask", EntityType: "MicroFramework", Observations: []string{"Micro web framework"}},
			{Name: "FastAPI", EntityType: "APIFramework", Observations: []string{"Modern, fast web framework"}},
			{Name: "Go", EntityType: "Language", Observations: []string{"Statically typed", "Compiled"}},
			{Name: "Gin", EntityType: "Framework", Observations: []string{"HTTP web framework"}},
		},
		Relations: []PythonRelation{
			{From: "Python", To: "Django", RelationType: "hasFramework"},
			{From: "Python", To: "Flask", RelationType: "hasFramework"},
			{From: "Python", To: "FastAPI", RelationType: "hasFramework"},
			{From: "Go", To: "Gin", RelationType: "hasFramework"},
		},
	}
)

func BenchmarkTransformToPython(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = TransformToPython(benchEntities, benchRelations, benchObservations)
	}
}

func BenchmarkTransformFromPython(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_, _, _ = TransformFromPython(benchPythonGraph)
	}
}

func BenchmarkEmbedObservationsInEntities(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = EmbedObservationsInEntities(benchEntities, benchObservations)
	}
}

func BenchmarkExtractObservationsFromEntities(b *testing.B) {
	for i := 0; i < b.N; i++ {
		// Using benchPythonGraph.Entities directly as it matches the expected input type
		_, _ = ExtractObservationsFromEntities(benchPythonGraph.Entities)
	}
}
