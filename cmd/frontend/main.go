//go:build js && wasm
// +build js,wasm

package main

import (
	"context" // Added context
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"syscall/js"

	// For sqlite3.Conn type
	sqlite3driver "github.com/ncruces/go-sqlite3/driver" // Named import for driver.Conn interface
	_ "github.com/ncruces/go-sqlite3/embed"              // Embeds the SQLite WASM for go-sqlite3
	serdes "github.com/ncruces/go-sqlite3/ext/serdes"    // Named import for serdes
	"github.com/ncruces/go-sqlite3/vfs/memdb"
)

var (
	db                *sql.DB
	currentDbName     = "knowledge_graph.db" // Default DB name for memdb
	sqliteBusyTimeout = 5000                 // Default busy timeout
)

// --- Database Initialization and Schema ---

func initializeSchemaInternal() error {
	if db == nil {
		return fmt.Errorf("database not initialized")
	}

	schema := `
        CREATE TABLE IF NOT EXISTS entities (
            name TEXT PRIMARY KEY,
            entity_type TEXT NOT NULL
        );
        CREATE TABLE IF NOT EXISTS observations (
            id INTEGER PRIMARY KEY AUTOINCREMENT,
            entity_name TEXT NOT NULL,
            content TEXT NOT NULL,
            FOREIGN KEY(entity_name) REFERENCES entities(name) ON DELETE CASCADE
        );
        CREATE TABLE IF NOT EXISTS relations (
            id INTEGER PRIMARY KEY AUTOINCREMENT,
            from_entity TEXT NOT NULL,
            to_entity TEXT NOT NULL,
            relation_type TEXT NOT NULL,
            FOREIGN KEY(from_entity) REFERENCES entities(name) ON DELETE CASCADE,
            FOREIGN KEY(to_entity) REFERENCES entities(name) ON DELETE CASCADE
        );
    `
	_, err := db.Exec(schema)
	if err != nil {
		return fmt.Errorf("failed to create/check schema: %w", err)
	}
	fmt.Println("Go: Knowledge graph schema checked/created.")
	return nil
}

//export initDB
func initDB(this js.Value, args []js.Value) any {
	jsDbKeyName := args[0]
	jsDbBytesHex := args[1]

	if !jsDbKeyName.IsNull() && !jsDbKeyName.IsUndefined() && jsDbKeyName.String() != "" {
		currentDbName = jsDbKeyName.String()
	}

	var err error
	var dbBytes []byte

	if !jsDbBytesHex.IsNull() && !jsDbBytesHex.IsUndefined() && jsDbBytesHex.String() != "" {
		dbBytes, err = hex.DecodeString(jsDbBytesHex.String())
		if err != nil {
			return makeResult(nil, fmt.Errorf("Failed to decode dbBytesHex: %w", err))
		}
		fmt.Printf("Go: Initializing with %d bytes for DB %s\n", len(dbBytes), currentDbName)
	} else {
		fmt.Printf("Go: Initializing new/empty DB %s\n", currentDbName)
	}

	memdb.Create(currentDbName, dbBytes)

	dsn := fmt.Sprintf("file:/%s?vfs=memdb&_pragma=foreign_keys(1)&_pragma=busy_timeout(%d)", currentDbName, sqliteBusyTimeout)
	if db != nil {
		db.Close()
	}
	db, err = sql.Open("sqlite3", dsn)
	if err != nil {
		return makeResult(nil, fmt.Errorf("Failed to open database with memdb: %w", err))
	}

	err = db.Ping()
	if err != nil {
		return makeResult(nil, fmt.Errorf("Failed to ping database: %w", err))
	}

	err = initializeSchemaInternal()
	if err != nil {
		db.Close()
		return makeResult(nil, fmt.Errorf("Schema initialization failed: %w", err))
	}

	fmt.Println("Go: initDB successful for", currentDbName)
	return makeResult(map[string]any{"dbKeyName": currentDbName}, nil)
}

// --- CRUD Operations & Others ---

// Helper to return results or errors to JavaScript as a JSON string
func makeResult(data any, err error, originalPayloadJS ...js.Value) js.Value {
	var resultValue map[string]any
	if err != nil {
		fmt.Println("Go Error:", err)
		resultValue = map[string]any{"error": err.Error()}
		if len(originalPayloadJS) > 0 && originalPayloadJS[0].Type() == js.TypeString {
			resultValue["originalPayloadString"] = originalPayloadJS[0].String()
		}
	} else {
		resultValue = map[string]any{"success": true}
		if data != nil {
			resultValue["data"] = data
		}
	}

	jsonBytes, jsonErr := json.Marshal(resultValue)
	if jsonErr != nil {
		fallbackError := map[string]any{"error": "Critical: Failed to marshal final result to JSON: " + jsonErr.Error()}
		fallbackJsonBytes, _ := json.Marshal(fallbackError)
		return js.ValueOf(string(fallbackJsonBytes))
	}
	return js.ValueOf(string(jsonBytes))
}

type CreateEntityPayload struct {
	Name         string   `json:"name"`
	Type         string   `json:"type"`
	Observations []string `json:"observations"`
}

//export createEntity
func createEntity(this js.Value, args []js.Value) any {
	payloadStr := args[0].String()
	var payload CreateEntityPayload
	err := json.Unmarshal([]byte(payloadStr), &payload)
	if err != nil {
		return makeResult(nil, fmt.Errorf("invalid JSON payload for createEntity: %w", err), args[0])
	}

	if payload.Name == "" || payload.Type == "" {
		return makeResult(nil, fmt.Errorf("entity name and type are required"), args[0])
	}

	tx, err := db.Begin()
	if err != nil {
		return makeResult(nil, fmt.Errorf("failed to begin transaction: %w", err), args[0])
	}
	defer tx.Rollback()

	var exists int
	err = tx.QueryRow("SELECT 1 FROM entities WHERE name = ?", payload.Name).Scan(&exists)
	if err != nil && err != sql.ErrNoRows {
		return makeResult(nil, fmt.Errorf("failed to check if entity exists: %w", err), args[0])
	}
	if exists == 1 {
		return makeResult(nil, fmt.Errorf("entity '%s' already exists. Entity names must be unique", payload.Name), args[0])
	}

	_, err = tx.Exec("INSERT INTO entities (name, entity_type) VALUES (?, ?)", payload.Name, payload.Type)
	if err != nil {
		return makeResult(nil, fmt.Errorf("failed to insert entity: %w", err), args[0])
	}

	if payload.Observations != nil {
		for _, obs := range payload.Observations {
			if strings.TrimSpace(obs) != "" {
				_, err = tx.Exec("INSERT INTO observations (entity_name, content) VALUES (?, ?)", payload.Name, strings.TrimSpace(obs))
				if err != nil {
					return makeResult(nil, fmt.Errorf("failed to insert observation for entity '%s': %w", payload.Name, err), args[0])
				}
			}
		}
	}

	err = tx.Commit()
	if err != nil {
		return makeResult(nil, fmt.Errorf("failed to commit transaction for createEntity: %w", err), args[0])
	}
	return makeResult(payload, nil)
}

type CreateRelationPayload struct {
	FromEntity   string `json:"from_entity"`
	ToEntity     string `json:"to_entity"`
	RelationType string `json:"relation_type"`
}

//export createRelation
func createRelation(this js.Value, args []js.Value) any {
	payloadStr := args[0].String()
	var payload CreateRelationPayload
	err := json.Unmarshal([]byte(payloadStr), &payload)
	if err != nil {
		return makeResult(nil, fmt.Errorf("invalid JSON payload for createRelation: %w", err), args[0])
	}

	tx, err := db.Begin()
	if err != nil {
		return makeResult(nil, err, args[0])
	}
	defer tx.Rollback()

	var fromExists, toExists int
	err = tx.QueryRow("SELECT 1 FROM entities WHERE name = ?", payload.FromEntity).Scan(&fromExists)
	if err != nil && err != sql.ErrNoRows {
		return makeResult(nil, fmt.Errorf("error checking 'from' entity: %w", err), args[0])
	}
	if fromExists == 0 {
		return makeResult(nil, fmt.Errorf("'From' entity '%s' does not exist", payload.FromEntity), args[0])
	}

	err = tx.QueryRow("SELECT 1 FROM entities WHERE name = ?", payload.ToEntity).Scan(&toExists)
	if err != nil && err != sql.ErrNoRows {
		return makeResult(nil, fmt.Errorf("error checking 'to' entity: %w", err), args[0])
	}
	if toExists == 0 {
		return makeResult(nil, fmt.Errorf("'To' entity '%s' does not exist", payload.ToEntity), args[0])
	}

	_, err = tx.Exec("INSERT INTO relations (from_entity, to_entity, relation_type) VALUES (?, ?, ?)",
		payload.FromEntity, payload.ToEntity, payload.RelationType)
	if err != nil {
		return makeResult(nil, fmt.Errorf("failed to insert relation: %w", err), args[0])
	}

	err = tx.Commit()
	if err != nil {
		return makeResult(nil, fmt.Errorf("failed to commit transaction for createRelation: %w", err), args[0])
	}
	return makeResult(payload, nil)
}

type AddObservationPayload struct {
	EntityName string `json:"entity_name"`
	Content    string `json:"content"`
}

//export addObservation
func addObservation(this js.Value, args []js.Value) any {
	payloadStr := args[0].String()
	var payload AddObservationPayload
	err := json.Unmarshal([]byte(payloadStr), &payload)
	if err != nil {
		return makeResult(nil, fmt.Errorf("invalid JSON payload for addObservation: %w", err), args[0])
	}

	var entityExists int
	err = db.QueryRow("SELECT 1 FROM entities WHERE name = ?", payload.EntityName).Scan(&entityExists)
	if err != nil && err != sql.ErrNoRows {
		return makeResult(nil, fmt.Errorf("error checking entity for observation: %w", err), args[0])
	}
	if entityExists == 0 {
		return makeResult(nil, fmt.Errorf("entity '%s' does not exist for observation", payload.EntityName), args[0])
	}

	_, err = db.Exec("INSERT INTO observations (entity_name, content) VALUES (?, ?)", payload.EntityName, payload.Content)
	if err != nil {
		return makeResult(nil, fmt.Errorf("failed to insert observation: %w", err), args[0])
	}
	return makeResult(payload, nil)
}

type Entity struct {
	Name string `json:"name"`
	Type string `json:"entity_type"`
}
type Relation struct {
	ID           float64 `json:"id"` // Changed from int64
	FromEntity   string  `json:"from_entity"`
	ToEntity     string  `json:"to_entity"`
	RelationType string  `json:"relation_type"`
}
type Observation struct {
	ID         float64 `json:"id"` // Changed from int64
	EntityName string  `json:"entity_name"`
	Content    string  `json:"content"`
}
type GraphData struct {
	Entities     []Entity      `json:"entities"`
	Relations    []Relation    `json:"relations"`
	Observations []Observation `json:"observations"`
}

//export getGraphData
func getGraphData(this js.Value, args []js.Value) any {
	var graph GraphData

	rows, err := db.Query("SELECT name, entity_type FROM entities ORDER BY name")
	if err != nil {
		return makeResult(nil, fmt.Errorf("failed to query entities: %w", err))
	}

	for rows.Next() {
		var e Entity
		if err := rows.Scan(&e.Name, &e.Type); err != nil {
			rows.Close()
			return makeResult(nil, fmt.Errorf("failed to scan entity: %w", err))
		}
		graph.Entities = append(graph.Entities, e)
	}
	rows.Close()

	rows, err = db.Query("SELECT id, from_entity, to_entity, relation_type FROM relations ORDER BY id")
	if err != nil {
		return makeResult(nil, fmt.Errorf("failed to query relations: %w", err))
	}

	for rows.Next() {
		var r Relation
		if err := rows.Scan(&r.ID, &r.FromEntity, &r.ToEntity, &r.RelationType); err != nil {
			rows.Close()
			return makeResult(nil, fmt.Errorf("failed to scan relation: %w", err))
		}
		graph.Relations = append(graph.Relations, r)
	}
	rows.Close()

	rows, err = db.Query("SELECT id, entity_name, content FROM observations ORDER BY entity_name, id")
	if err != nil {
		return makeResult(nil, fmt.Errorf("failed to query observations: %w", err))
	}
	defer rows.Close()
	for rows.Next() {
		var o Observation
		if err := rows.Scan(&o.ID, &o.EntityName, &o.Content); err != nil {
			return makeResult(nil, fmt.Errorf("failed to scan observation: %w", err))
		}
		graph.Observations = append(graph.Observations, o)
	}
	return makeResult(map[string]any{"graphData": graph}, nil)
}

type SearchNodesPayload struct {
	Query string `json:"query"`
}

//export searchNodes
func searchNodes(this js.Value, args []js.Value) any {
	payloadStr := args[0].String()
	var payload SearchNodesPayload
	err := json.Unmarshal([]byte(payloadStr), &payload)
	if err != nil {
		return makeResult(nil, fmt.Errorf("invalid JSON payload for searchNodes: %w", err), args[0])
	}

	searchPattern := "%" + strings.ToLower(payload.Query) + "%"
	var entities []Entity

	rows, err := db.Query(`
        SELECT DISTINCT e.name, e.entity_type
        FROM entities e
        LEFT JOIN observations o ON e.name = o.entity_name
        WHERE LOWER(e.name) LIKE ?
           OR LOWER(e.entity_type) LIKE ?
           OR LOWER(o.content) LIKE ?
        ORDER BY e.name
    `, searchPattern, searchPattern, searchPattern)
	if err != nil {
		return makeResult(nil, fmt.Errorf("failed to search entities: %w", err), args[0])
	}

	for rows.Next() {
		var e Entity
		if errScan := rows.Scan(&e.Name, &e.Type); errScan != nil {
			rows.Close()
			return makeResult(nil, fmt.Errorf("failed to scan searched entity: %w", errScan), args[0])
		}
		entities = append(entities, e)
	}
	rows.Close()

	var relations []Relation
	if len(entities) > 0 {
		entityNames := make([]any, len(entities))
		for i, e := range entities {
			entityNames[i] = e.Name
		}
		placeholders := strings.Repeat("?,", len(entityNames)-1) + "?"

		allArgs := append(entityNames, entityNames...)
		query := fmt.Sprintf(`
            SELECT id, from_entity, to_entity, relation_type
            FROM relations
            WHERE from_entity IN (%s) OR to_entity IN (%s)
            ORDER BY id
        `, placeholders, placeholders)

		relRows, errRel := db.Query(query, allArgs...)
		if errRel != nil {
			return makeResult(nil, fmt.Errorf("failed to search relations: %w", errRel), args[0])
		}

		for relRows.Next() {
			var r Relation
			if errScan := relRows.Scan(&r.ID, &r.FromEntity, &r.ToEntity, &r.RelationType); errScan != nil {
				relRows.Close()
				return makeResult(nil, fmt.Errorf("failed to scan searched relation: %w", errScan), args[0])
			}
			relations = append(relations, r)
		}
		relRows.Close()
	}
	return makeResult(map[string]any{"graphData": GraphData{Entities: entities, Relations: relations, Observations: []Observation{}}}, nil)
}

type OpenNodesPayload struct {
	Names []string `json:"names"`
}

//export openNodes
func openNodes(this js.Value, args []js.Value) any {
	payloadStr := args[0].String()
	var payload OpenNodesPayload
	err := json.Unmarshal([]byte(payloadStr), &payload)
	if err != nil {
		return makeResult(nil, fmt.Errorf("invalid JSON payload for openNodes: %w", err), args[0])
	}

	if len(payload.Names) == 0 {
		return makeResult(map[string]any{"graphData": GraphData{Entities: []Entity{}, Relations: []Relation{}, Observations: []Observation{}}}, nil)
	}

	var qMarks []string
	var interfaceSlice []any
	for _, name := range payload.Names {
		qMarks = append(qMarks, "?")
		interfaceSlice = append(interfaceSlice, name)
	}
	placeholders := strings.Join(qMarks, ",")

	var entities []Entity
	queryEntities := fmt.Sprintf("SELECT name, entity_type FROM entities WHERE name IN (%s) ORDER BY name", placeholders)
	rows, err := db.Query(queryEntities, interfaceSlice...)
	if err != nil {
		return makeResult(nil, fmt.Errorf("failed to query entities for openNodes: %w", err), args[0])
	}

	for rows.Next() {
		var e Entity
		if errScan := rows.Scan(&e.Name, &e.Type); errScan != nil {
			rows.Close()
			return makeResult(nil, fmt.Errorf("failed to scan entity for openNodes: %w", errScan), args[0])
		}
		entities = append(entities, e)
	}
	rows.Close()

	var relations []Relation
	if len(entities) > 0 {
		entityNamesFound := make([]any, len(entities))
		for i, e := range entities {
			entityNamesFound[i] = e.Name
		}
		relPlaceholders := strings.Repeat("?,", len(entityNamesFound)-1) + "?"
		allArgs := append(entityNamesFound, entityNamesFound...)

		queryRelations := fmt.Sprintf(`
            SELECT id, from_entity, to_entity, relation_type
            FROM relations
            WHERE from_entity IN (%s) OR to_entity IN (%s)
            ORDER BY id
        `, relPlaceholders, relPlaceholders)
		relRows, errRel := db.Query(queryRelations, allArgs...)
		if errRel != nil {
			return makeResult(nil, fmt.Errorf("failed to query relations for openNodes: %w", errRel), args[0])
		}

		for relRows.Next() {
			var r Relation
			if errScan := relRows.Scan(&r.ID, &r.FromEntity, &r.ToEntity, &r.RelationType); errScan != nil {
				relRows.Close()
				return makeResult(nil, fmt.Errorf("failed to scan relation for openNodes: %w", errScan), args[0])
			}
			relations = append(relations, r)
		}
		relRows.Close()
	}
	return makeResult(map[string]any{"graphData": GraphData{Entities: entities, Relations: relations, Observations: []Observation{}}}, nil)
}

type DeleteEntitiesPayload struct {
	EntityNames []string `json:"entityNames"`
}

//export deleteEntities
func deleteEntities(this js.Value, args []js.Value) any {
	payloadStr := args[0].String()
	var payload DeleteEntitiesPayload
	err := json.Unmarshal([]byte(payloadStr), &payload)
	if err != nil {
		return makeResult(nil, fmt.Errorf("invalid JSON payload for deleteEntities: %w", err), args[0])
	}

	if len(payload.EntityNames) == 0 {
		return makeResult(map[string]any{"count": 0}, nil)
	}

	tx, err := db.Begin()
	if err != nil {
		return makeResult(nil, fmt.Errorf("failed to begin transaction for deleteEntities: %w", err), args[0])
	}
	defer tx.Rollback()

	qMarks := strings.Repeat("?,", len(payload.EntityNames)-1) + "?"
	var interfaceSlice []any
	for _, name := range payload.EntityNames {
		interfaceSlice = append(interfaceSlice, name)
	}

	allArgsRelations := append(interfaceSlice, interfaceSlice...)
	queryRels := fmt.Sprintf("DELETE FROM relations WHERE from_entity IN (%s) OR to_entity IN (%s)", qMarks, qMarks)
	_, err = tx.Exec(queryRels, allArgsRelations...)
	if err != nil {
		return makeResult(nil, fmt.Errorf("failed to delete relations for entities: %w", err), args[0])
	}

	queryObs := fmt.Sprintf("DELETE FROM observations WHERE entity_name IN (%s)", qMarks)
	_, err = tx.Exec(queryObs, interfaceSlice...)
	if err != nil {
		return makeResult(nil, fmt.Errorf("failed to delete observations for entities: %w", err), args[0])
	}

	queryEnt := fmt.Sprintf("DELETE FROM entities WHERE name IN (%s)", qMarks)
	_, err = tx.Exec(queryEnt, interfaceSlice...)
	if err != nil {
		return makeResult(nil, fmt.Errorf("failed to delete entities: %w", err), args[0])
	}

	err = tx.Commit()
	if err != nil {
		return makeResult(nil, fmt.Errorf("failed to commit deleteEntities: %w", err), args[0])
	}
	return makeResult(map[string]any{"count": len(payload.EntityNames)}, nil)
}

type RelationToDelete struct {
	From         string `json:"from"`
	To           string `json:"to"`
	RelationType string `json:"relationType"`
}
type DeleteRelationsPayload struct {
	Relations []RelationToDelete `json:"relations"`
}

//export deleteRelations
func deleteRelations(this js.Value, args []js.Value) any {
	payloadStr := args[0].String()
	var payload DeleteRelationsPayload
	err := json.Unmarshal([]byte(payloadStr), &payload)
	if err != nil {
		return makeResult(nil, fmt.Errorf("invalid JSON for deleteRelations: %w", err), args[0])
	}

	if len(payload.Relations) == 0 {
		return makeResult(map[string]any{"count": 0}, nil)
	}

	tx, err := db.Begin()
	if err != nil {
		return makeResult(nil, fmt.Errorf("failed to begin transaction for deleteRelations: %w", err), args[0])
	}
	defer tx.Rollback()

	deletedCount := 0
	for _, rel := range payload.Relations {
		_, err = tx.Exec("DELETE FROM relations WHERE from_entity = ? AND to_entity = ? AND relation_type = ?",
			rel.From, rel.To, rel.RelationType)
		if err != nil {
			return makeResult(nil, fmt.Errorf("failed to delete relation (%s-%s->%s): %w", rel.From, rel.RelationType, rel.To, err), args[0])
		}
		deletedCount++
	}
	err = tx.Commit()
	if err != nil {
		return makeResult(nil, fmt.Errorf("failed to commit deleteRelations: %w", err), args[0])
	}
	return makeResult(map[string]any{"count": deletedCount}, nil)
}

type ObsDeletion struct {
	EntityName   string   `json:"entityName"`
	Observations []string `json:"observations"`
}
type DeleteObservationsPayload struct {
	Deletions []ObsDeletion `json:"deletions"`
}

//export deleteObservations
func deleteObservations(this js.Value, args []js.Value) any {
	payloadStr := args[0].String()
	var payload DeleteObservationsPayload
	err := json.Unmarshal([]byte(payloadStr), &payload)
	if err != nil {
		return makeResult(nil, fmt.Errorf("invalid JSON for deleteObservations: %w", err), args[0])
	}

	if len(payload.Deletions) == 0 {
		return makeResult(map[string]any{"entityName": ""}, nil)
	}

	tx, err := db.Begin()
	if err != nil {
		return makeResult(nil, fmt.Errorf("failed to begin transaction for deleteObservations: %w", err), args[0])
	}
	defer tx.Rollback()

	var firstEntityName string
	for i, del := range payload.Deletions {
		if i == 0 {
			firstEntityName = del.EntityName
		}
		if len(del.Observations) > 0 {
			qMarks := strings.Repeat("?,", len(del.Observations)-1) + "?"
			argsForExec := make([]any, 0, len(del.Observations)+1)
			argsForExec = append(argsForExec, del.EntityName)
			for _, obsContent := range del.Observations {
				argsForExec = append(argsForExec, obsContent)
			}
			query := fmt.Sprintf("DELETE FROM observations WHERE entity_name = ? AND content IN (%s)", qMarks)
			_, err = tx.Exec(query, argsForExec...)
			if err != nil {
				return makeResult(nil, fmt.Errorf("failed to delete observations for '%s': %w", del.EntityName, err), args[0])
			}
		}
	}
	err = tx.Commit()
	if err != nil {
		return makeResult(nil, fmt.Errorf("failed to commit deleteObservations: %w", err), args[0])
	}
	return makeResult(map[string]any{"entityName": firstEntityName}, nil)
}

//export exportDB
func exportDB(this js.Value, args []js.Value) any {
	if db == nil {
		return makeResult(nil, fmt.Errorf("database not initialized for export"))
	}

	conn, err := db.Conn(context.Background())
	if err != nil {
		return makeResult(nil, fmt.Errorf("failed to get raw connection for export: %w", err))
	}
	defer conn.Close()

	var dbBytes []byte
	err = conn.Raw(func(driverConn any) error {
		// Type assert to the driver.Conn interface from the imported driver package
		sqliteDriverConn, ok := driverConn.(sqlite3driver.Conn)
		if !ok {
			return fmt.Errorf("driver connection (type %T) does not implement sqlite3driver.Conn interface", driverConn)
		}

		// Call the Raw() method on the driver.Conn interface to get *sqlite3.Conn
		sConn := sqliteDriverConn.Raw()
		if sConn == nil {
			return fmt.Errorf("failed to obtain *sqlite3.Conn via sqlite3driver.Conn.Raw()")
		}

		var serErr error
		dbBytes, serErr = serdes.Serialize(sConn, "main") // "main" is the default schema
		return serErr
	})

	if err != nil {
		return makeResult(nil, fmt.Errorf("exportDB failed: %w", err))
	}

	if len(dbBytes) == 0 {
		fmt.Println("Go: Warning - exported DB bytes are empty.")
	}

	// Data to be JSON marshalled by makeResult
	exportData := map[string]any{
		"dbBytesHex": hex.EncodeToString(dbBytes), // Send as hex string
		"dbKeyName":  currentDbName,
	}
	return makeResult(exportData, nil)
}

//export importDB
func importDB(this js.Value, args []js.Value) any {
	jsDbBytesHex := args[0]

	if jsDbBytesHex.IsNull() || jsDbBytesHex.IsUndefined() || jsDbBytesHex.String() == "" {
		return makeResult(nil, fmt.Errorf("importDB failed: No data provided or data is empty"))
	}
	dbBytes, err := hex.DecodeString(jsDbBytesHex.String())
	if err != nil {
		return makeResult(nil, fmt.Errorf("importDB failed: Could not decode hex dbBytes: %w", err))
	}
	if len(dbBytes) == 0 {
		return makeResult(nil, fmt.Errorf("importDB failed: Decoded dbBytes is empty"))
	}

	fmt.Printf("Go: importDB received %d bytes (after hex decode) for DB %s\n", len(dbBytes), currentDbName)

	if db != nil {
		errClose := db.Close()
		if errClose != nil {
			fmt.Printf("Go: Error closing old DB during import: %s\n", errClose.Error())
		}
		db = nil
	}

	memdb.Delete(currentDbName)
	memdb.Create(currentDbName, dbBytes)

	dsn := fmt.Sprintf("file:/%s?vfs=memdb&_pragma=foreign_keys(1)&_pragma=busy_timeout(%d)", currentDbName, sqliteBusyTimeout)
	db, err = sql.Open("sqlite3", dsn)
	if err != nil {
		return makeResult(nil, fmt.Errorf("failed to open imported database with memdb: %w", err))
	}
	err = db.Ping()
	if err != nil {
		return makeResult(nil, fmt.Errorf("failed to ping imported database: %w", err))
	}

	fmt.Println("Go: importDB successful.")
	return makeResult(map[string]any{"dbKeyName": currentDbName}, nil)
}

type BackendDataPayload struct {
	Entities     []Entity      `json:"entities"`
	Relations    []Relation    `json:"relations"`
	Observations []Observation `json:"observations"`
}
type SyncFromBackendDataPayload struct {
	BackendData BackendDataPayload `json:"backendData"`
}

//export syncFromBackendData
func syncFromBackendData(this js.Value, args []js.Value) any {
	payloadStr := args[0].String()
	var payload SyncFromBackendDataPayload
	err := json.Unmarshal([]byte(payloadStr), &payload)
	if err != nil {
		return makeResult(nil, fmt.Errorf("invalid JSON for syncFromBackendData: %w", err), args[0])
	}

	tx, err := db.Begin()
	if err != nil {
		return makeResult(nil, fmt.Errorf("failed to begin transaction for sync: %w", err), args[0])
	}
	defer tx.Rollback()

	queries := []string{"DELETE FROM observations", "DELETE FROM relations", "DELETE FROM entities"}
	for _, q := range queries {
		if _, errExec := tx.Exec(q); errExec != nil {
			return makeResult(nil, fmt.Errorf("failed to clear table during sync (%s): %w", q, errExec), args[0])
		}
	}

	for _, entity := range payload.BackendData.Entities {
		_, err = tx.Exec("INSERT INTO entities (name, entity_type) VALUES (?, ?)", entity.Name, entity.Type)
		if err != nil {
			return makeResult(nil, fmt.Errorf("failed to insert entity during sync: %w", err), args[0])
		}
	}
	for _, relation := range payload.BackendData.Relations {
		_, err = tx.Exec("INSERT INTO relations (from_entity, to_entity, relation_type) VALUES (?, ?, ?)",
			relation.FromEntity, relation.ToEntity, relation.RelationType)
		if err != nil {
			return makeResult(nil, fmt.Errorf("failed to insert relation during sync: %w", err), args[0])
		}
	}
	for _, observation := range payload.BackendData.Observations {
		_, err = tx.Exec("INSERT INTO observations (entity_name, content) VALUES (?, ?)",
			observation.EntityName, observation.Content)
		if err != nil {
			return makeResult(nil, fmt.Errorf("failed to insert observation during sync: %w", err), args[0])
		}
	}

	err = tx.Commit()
	if err != nil {
		return makeResult(nil, fmt.Errorf("failed to commit sync: %w", err), args[0])
	}
	return makeResult(nil, nil)
}

//export prepareSyncToServer
func prepareSyncToServer(this js.Value, args []js.Value) any {
	return exportDB(this, args)
}

//export completeSyncFromServer
func completeSyncFromServer(this js.Value, args []js.Value) any {
	return importDB(this, args)
}

// --- Main ---
func main() {
	c := make(chan struct{}, 0)
	fmt.Println("Go WASM Initialized (Knowledge Graph)")

	js.Global().Set("goInitDB", js.FuncOf(initDB))
	js.Global().Set("goCreateEntity", js.FuncOf(createEntity))
	js.Global().Set("goCreateRelation", js.FuncOf(createRelation))
	js.Global().Set("goAddObservation", js.FuncOf(addObservation))
	js.Global().Set("goGetGraphData", js.FuncOf(getGraphData))
	js.Global().Set("goSearchNodes", js.FuncOf(searchNodes))
	js.Global().Set("goOpenNodes", js.FuncOf(openNodes))
	js.Global().Set("goDeleteEntities", js.FuncOf(deleteEntities))
	js.Global().Set("goDeleteRelations", js.FuncOf(deleteRelations))
	js.Global().Set("goDeleteObservations", js.FuncOf(deleteObservations))
	js.Global().Set("goExportDB", js.FuncOf(exportDB))
	js.Global().Set("goImportDB", js.FuncOf(importDB))
	js.Global().Set("goSyncFromBackendData", js.FuncOf(syncFromBackendData))
	js.Global().Set("goPrepareSyncToServer", js.FuncOf(prepareSyncToServer))
	js.Global().Set("goCompleteSyncFromServer", js.FuncOf(completeSyncFromServer))

	<-c // Keep Go WASM alive
}
