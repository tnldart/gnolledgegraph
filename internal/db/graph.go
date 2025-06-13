package db

import (
    "database/sql"
    "fmt"
    "strings"
)

// Domain models
type Entity struct {
    Name string `json:"name"`
    Type string `json:"entity_type"`
}

type Relation struct {
    ID       int64  `json:"id"`
    From     string `json:"from_entity"`
    To       string `json:"to_entity"`
    Type     string `json:"relation_type"`
}

type Observation struct {
    ID         int64  `json:"id"`
    EntityName string `json:"entity_name"`
    Content    string `json:"content"`
}

// ReadGraph loads all entities, relations and observations
func ReadGraph(db *sql.DB) (
    []Entity,
    []Relation,
    []Observation,
    error,
) {
    // 1) entities
    ents := []Entity{}
    rows, err := db.Query(`SELECT name, entity_type FROM entities`)
    if err != nil {
        return nil, nil, nil, err
    }
    defer rows.Close()
    for rows.Next() {
        var e Entity
        if err := rows.Scan(&e.Name, &e.Type); err != nil {
            return nil, nil, nil, err
        }
        ents = append(ents, e)
    }

    // 2) relations
    rels := []Relation{}
    rows, err = db.Query(`SELECT id, from_entity, to_entity, relation_type FROM relations`)
    if err != nil {
        return nil, nil, nil, err
    }
    defer rows.Close()
    for rows.Next() {
        var r Relation
        if err := rows.Scan(&r.ID, &r.From, &r.To, &r.Type); err != nil {
            return nil, nil, nil, err
        }
        rels = append(rels, r)
    }

    // 3) observations
    obs := []Observation{}
    rows, err = db.Query(`SELECT id, entity_name, content FROM observations`)
    if err != nil {
        return nil, nil, nil, err
    }
    defer rows.Close()
    for rows.Next() {
        var o Observation
        if err := rows.Scan(&o.ID, &o.EntityName, &o.Content); err != nil {
            return nil, nil, nil, err
        }
        obs = append(obs, o)
    }

    return ents, rels, obs, nil
}

// CreateEntity inserts a new entity
func CreateEntity(db *sql.DB, name, entityType string) error {
    _, err := db.Exec(
        `INSERT OR IGNORE INTO entities(name, entity_type) VALUES(?, ?)`,
        name, entityType,
    )
    return err
}

// CreateRelation inserts a new relation and returns its new ID
func CreateRelation(db *sql.DB, from, to, relationType string) (int64, error) {
    res, err := db.Exec(
        `INSERT INTO relations(from_entity, to_entity, relation_type) VALUES(?, ?, ?)`,
        from, to, relationType,
    )
    if err != nil {
        return 0, err
    }
    return res.LastInsertId()
}

// CreateObservation inserts a new observation and returns its new ID
func CreateObservation(db *sql.DB, entityName, content string) (int64, error) {
    res, err := db.Exec(
        `INSERT INTO observations(entity_name, content) VALUES(?, ?)`,
        entityName, content,
    )
    if err != nil {
        return 0, err
    }
    return res.LastInsertId()
}

// AddObservations adds multiple observations to existing entities
func AddObservations(db *sql.DB, observations []struct {
    EntityName string `json:"entityName"`
    Contents   string `json:"contents"`
}) ([]Observation, error) {
    var added []Observation
    
    for _, obs := range observations {
        // Check if entity exists
        var exists bool
        err := db.QueryRow(`SELECT EXISTS(SELECT 1 FROM entities WHERE name = ?)`, obs.EntityName).Scan(&exists)
        if err != nil {
            return nil, err
        }
        if !exists {
            return nil, fmt.Errorf("entity '%s' does not exist", obs.EntityName)
        }
        
        // Add observation
        id, err := CreateObservation(db, obs.EntityName, obs.Contents)
        if err != nil {
            return nil, err
        }
        
        added = append(added, Observation{
            ID:         id,
            EntityName: obs.EntityName,
            Content:    obs.Contents,
        })
    }
    
    return added, nil
}

// DeleteEntities removes entities and their associated relations
func DeleteEntities(db *sql.DB, entityNames []string) error {
    if len(entityNames) == 0 {
        return nil
    }
    
    tx, err := db.Begin()
    if err != nil {
        return err
    }
    defer tx.Rollback()
    
    placeholders := strings.Repeat("?,", len(entityNames))
    placeholders = placeholders[:len(placeholders)-1] // Remove trailing comma
    
    args := make([]interface{}, len(entityNames))
    for i, name := range entityNames {
        args[i] = name
    }
    
    // Delete relations involving these entities
    _, err = tx.Exec(fmt.Sprintf(`DELETE FROM relations WHERE from_entity IN (%s) OR to_entity IN (%s)`, 
        placeholders, placeholders), append(args, args...)...)
    if err != nil {
        return err
    }
    
    // Delete observations for these entities
    _, err = tx.Exec(fmt.Sprintf(`DELETE FROM observations WHERE entity_name IN (%s)`, placeholders), args...)
    if err != nil {
        return err
    }
    
    // Delete entities
    _, err = tx.Exec(fmt.Sprintf(`DELETE FROM entities WHERE name IN (%s)`, placeholders), args...)
    if err != nil {
        return err
    }
    
    return tx.Commit()
}

// DeleteObservations removes specific observations from entities
func DeleteObservations(db *sql.DB, deletions []struct {
    EntityName   string   `json:"entityName"`
    Observations []string `json:"observations"`
}) error {
    if len(deletions) == 0 {
        return nil
    }
    
    for _, deletion := range deletions {
        if len(deletion.Observations) == 0 {
            continue
        }
        
        placeholders := strings.Repeat("?,", len(deletion.Observations))
        placeholders = placeholders[:len(placeholders)-1]
        
        args := make([]interface{}, 0, len(deletion.Observations)+1)
        args = append(args, deletion.EntityName)
        for _, obs := range deletion.Observations {
            args = append(args, obs)
        }
        
        _, err := db.Exec(fmt.Sprintf(`DELETE FROM observations WHERE entity_name = ? AND content IN (%s)`, 
            placeholders), args...)
        if err != nil {
            return err
        }
    }
    
    return nil
}

// DeleteRelations removes specific relations from the graph
func DeleteRelations(db *sql.DB, relations []struct {
    From string `json:"from"`
    To   string `json:"to"`
    Type string `json:"relationType"`
}) error {
    if len(relations) == 0 {
        return nil
    }
    
    for _, rel := range relations {
        _, err := db.Exec(`DELETE FROM relations WHERE from_entity = ? AND to_entity = ? AND relation_type = ?`,
            rel.From, rel.To, rel.Type)
        if err != nil {
            return err
        }
    }
    
    return nil
}

// SearchNodes searches entities based on query string
func SearchNodes(db *sql.DB, query string) ([]Entity, []Relation, error) {
    searchPattern := "%" + strings.ToLower(query) + "%"
    
    // Search entities by name, type, or observation content
    entityQuery := `
        SELECT DISTINCT e.name, e.entity_type 
        FROM entities e
        LEFT JOIN observations o ON e.name = o.entity_name
        WHERE LOWER(e.name) LIKE ? 
           OR LOWER(e.entity_type) LIKE ?
           OR LOWER(o.content) LIKE ?
    `
    
    var entities []Entity
    rows, err := db.Query(entityQuery, searchPattern, searchPattern, searchPattern)
    if err != nil {
        return nil, nil, err
    }
    defer rows.Close()
    
    for rows.Next() {
        var e Entity
        if err := rows.Scan(&e.Name, &e.Type); err != nil {
            return nil, nil, err
        }
        entities = append(entities, e)
    }
    
    // Get all relations involving the found entities
    if len(entities) == 0 {
        return entities, nil, nil
    }
    
    entityNames := make([]string, len(entities))
    for i, e := range entities {
        entityNames[i] = e.Name
    }
    
    placeholders := strings.Repeat("?,", len(entityNames))
    placeholders = placeholders[:len(placeholders)-1]
    
    args := make([]interface{}, len(entityNames)*2)
    for i, name := range entityNames {
        args[i] = name
        args[i+len(entityNames)] = name
    }
    
    relationQuery := fmt.Sprintf(`
        SELECT id, from_entity, to_entity, relation_type 
        FROM relations 
        WHERE from_entity IN (%s) OR to_entity IN (%s)
    `, placeholders, placeholders)
    
    var relations []Relation
    rows, err = db.Query(relationQuery, args...)
    if err != nil {
        return nil, nil, err
    }
    defer rows.Close()
    
    for rows.Next() {
        var r Relation
        if err := rows.Scan(&r.ID, &r.From, &r.To, &r.Type); err != nil {
            return nil, nil, err
        }
        relations = append(relations, r)
    }
    
    return entities, relations, nil
}

// OpenNodes retrieves specific nodes by name
func OpenNodes(db *sql.DB, nodeNames []string) ([]Entity, []Relation, error) {
    if len(nodeNames) == 0 {
        return nil, nil, nil
    }
    
    placeholders := strings.Repeat("?,", len(nodeNames))
    placeholders = placeholders[:len(placeholders)-1]
    
    args := make([]interface{}, len(nodeNames))
    for i, name := range nodeNames {
        args[i] = name
    }
    
    // Get requested entities
    entityQuery := fmt.Sprintf(`SELECT name, entity_type FROM entities WHERE name IN (%s)`, placeholders)
    
    var entities []Entity
    rows, err := db.Query(entityQuery, args...)
    if err != nil {
        return nil, nil, err
    }
    defer rows.Close()
    
    for rows.Next() {
        var e Entity
        if err := rows.Scan(&e.Name, &e.Type); err != nil {
            return nil, nil, err
        }
        entities = append(entities, e)
    }
    
    // Get all relations involving these entities
    if len(entities) == 0 {
        return entities, nil, nil
    }
    
    relationQuery := fmt.Sprintf(`
        SELECT id, from_entity, to_entity, relation_type 
        FROM relations 
        WHERE from_entity IN (%s) OR to_entity IN (%s)
    `, placeholders, placeholders)
    
    doubleArgs := make([]interface{}, len(args)*2)
    copy(doubleArgs, args)
    copy(doubleArgs[len(args):], args)
    
    var relations []Relation
    rows, err = db.Query(relationQuery, doubleArgs...)
    if err != nil {
        return nil, nil, err
    }
    defer rows.Close()
    
    for rows.Next() {
        var r Relation
        if err := rows.Scan(&r.ID, &r.From, &r.To, &r.Type); err != nil {
            return nil, nil, err
        }
        relations = append(relations, r)
    }
    
    return entities, relations, nil
}
