package api

import (
	"memory-parttwo/internal/db"
)

// Python-compatible data models
type PythonEntity struct {
	Name         string   `json:"name"`
	EntityType   string   `json:"entityType"`
	Observations []string `json:"observations"`
}

type PythonRelation struct {
	From         string `json:"from"`
	To           string `json:"to"`
	RelationType string `json:"relationType"`
}

type PythonKnowledgeGraph struct {
	Entities  []PythonEntity   `json:"entities"`
	Relations []PythonRelation `json:"relations"`
}

// TransformToPython converts Go database models to Python-compatible format
func TransformToPython(entities []db.Entity, relations []db.Relation, observations []db.Observation) PythonKnowledgeGraph {
	// Create map of entity name to observations for efficient lookup
	obsMap := make(map[string][]string)
	for _, obs := range observations {
		obsMap[obs.EntityName] = append(obsMap[obs.EntityName], obs.Content)
	}

	// Transform entities with embedded observations
	pythonEntities := make([]PythonEntity, len(entities))
	for i, entity := range entities {
		pythonEntities[i] = PythonEntity{
			Name:         entity.Name,
			EntityType:   entity.Type, // entity_type -> entityType
			Observations: obsMap[entity.Name],
		}
		// Ensure observations is never nil
		if pythonEntities[i].Observations == nil {
			pythonEntities[i].Observations = []string{}
		}
	}

	// Transform relations with field name mapping
	pythonRelations := make([]PythonRelation, len(relations))
	for i, relation := range relations {
		pythonRelations[i] = PythonRelation{
			From:         relation.From, // from_entity -> from
			To:           relation.To,   // to_entity -> to
			RelationType: relation.Type, // relation_type -> relationType
		}
	}

	return PythonKnowledgeGraph{
		Entities:  pythonEntities,
		Relations: pythonRelations,
	}
}

// TransformFromPython converts Python format to Go database models
func TransformFromPython(pythonGraph PythonKnowledgeGraph) ([]db.Entity, []db.Relation, []db.Observation) {
	// Transform entities
	entities := make([]db.Entity, len(pythonGraph.Entities))
	var observations []db.Observation

	for i, pythonEntity := range pythonGraph.Entities {
		entities[i] = db.Entity{
			Name: pythonEntity.Name,
			Type: pythonEntity.EntityType, // entityType -> entity_type
		}

		// Extract observations from embedded format
		for _, obsContent := range pythonEntity.Observations {
			observations = append(observations, db.Observation{
				EntityName: pythonEntity.Name,
				Content:    obsContent,
			})
		}
	}

	// Transform relations with field name mapping
	relations := make([]db.Relation, len(pythonGraph.Relations))
	for i, pythonRelation := range pythonGraph.Relations {
		relations[i] = db.Relation{
			From: pythonRelation.From,         // from -> from_entity
			To:   pythonRelation.To,           // to -> to_entity
			Type: pythonRelation.RelationType, // relationType -> relation_type
		}
	}

	return entities, relations, observations
}

// EmbedObservationsInEntities combines entities with their observations
func EmbedObservationsInEntities(entities []db.Entity, observations []db.Observation) []PythonEntity {
	// Create map of entity name to observations
	obsMap := make(map[string][]string)
	for _, obs := range observations {
		obsMap[obs.EntityName] = append(obsMap[obs.EntityName], obs.Content)
	}

	// Create Python entities with embedded observations
	pythonEntities := make([]PythonEntity, len(entities))
	for i, entity := range entities {
		pythonEntities[i] = PythonEntity{
			Name:         entity.Name,
			EntityType:   entity.Type,
			Observations: obsMap[entity.Name],
		}
		// Ensure observations is never nil
		if pythonEntities[i].Observations == nil {
			pythonEntities[i].Observations = []string{}
		}
	}

	return pythonEntities
}

// ExtractObservationsFromEntities separates entities from their embedded observations
func ExtractObservationsFromEntities(pythonEntities []PythonEntity) ([]db.Entity, []db.Observation) {
	entities := make([]db.Entity, len(pythonEntities))
	var observations []db.Observation

	for i, pythonEntity := range pythonEntities {
		entities[i] = db.Entity{
			Name: pythonEntity.Name,
			Type: pythonEntity.EntityType,
		}

		// Extract observations
		for _, obsContent := range pythonEntity.Observations {
			observations = append(observations, db.Observation{
				EntityName: pythonEntity.Name,
				Content:    obsContent,
			})
		}
	}

	return entities, observations
}

// TransformRelationsToPython converts Go relations to Python format
func TransformRelationsToPython(relations []db.Relation) []PythonRelation {
	pythonRelations := make([]PythonRelation, len(relations))
	for i, relation := range relations {
		pythonRelations[i] = PythonRelation{
			From:         relation.From,
			To:           relation.To,
			RelationType: relation.Type,
		}
	}
	return pythonRelations
}

// TransformRelationsFromPython converts Python relations to Go format
func TransformRelationsFromPython(pythonRelations []PythonRelation) []db.Relation {
	relations := make([]db.Relation, len(pythonRelations))
	for i, pythonRelation := range pythonRelations {
		relations[i] = db.Relation{
			From: pythonRelation.From,
			To:   pythonRelation.To,
			Type: pythonRelation.RelationType,
		}
	}
	return relations
}
