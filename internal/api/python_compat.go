package api

import (
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"path/filepath"

	"gnolledgegraph/internal/db"
)

// StaticFS, if non-nil, is used to serve embedded static frontend assets.
// If nil, assets are served from disk relative to the executable.
var StaticFS http.FileSystem

// NewPythonCompatHandler creates a new HTTP handler for Python FastAPI compatibility
func NewPythonCompatHandler(database *sql.DB) http.Handler {
	mux := http.NewServeMux()

	// Add CORS headers for web client compatibility
	addCORSHeaders := func(w http.ResponseWriter) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
	}

	// Handle preflight requests for all routes
	handleWithCORS := func(pattern string, handler func(http.ResponseWriter, *http.Request)) {
		mux.HandleFunc(pattern, func(w http.ResponseWriter, r *http.Request) {
			addCORSHeaders(w)
			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusOK)
				return
			}
			handler(w, r)
		})
	}

	// 1. GET /read_graph - Read entire knowledge graph
	handleWithCORS("/read_graph", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		entities, relations, observations, err := db.ReadGraph(database)
		if err != nil {
			http.Error(w, "Failed to read graph: "+err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(struct {
			Entities     []db.Entity      `json:"entities"`
			Relations    []db.Relation    `json:"relations"`
			Observations []db.Observation `json:"observations"`
		}{
			Entities:     entities,
			Relations:    relations,
			Observations: observations,
		})
	})

	// 2. POST /create_entities - Create entities with embedded observations
	mux.HandleFunc("/create_entities", func(w http.ResponseWriter, r *http.Request) {
		addCORSHeaders(w)
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var req struct {
			Entities []db.Entity `json:"entities"`
		}

		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid JSON: "+err.Error(), http.StatusBadRequest)
			return
		}

		var createdEntities []db.Entity
		var conflictingEntityNames []string

		// First, check for existing entities to handle conflicts gracefully
		for _, entity := range req.Entities {
			var exists bool
			err := database.QueryRow(`SELECT EXISTS(SELECT 1 FROM entities WHERE name = ?)`, entity.Name).Scan(&exists)
			if err != nil {
				http.Error(w, "Database error checking entity existence: "+err.Error(), http.StatusInternalServerError)
				return
			}
			if exists {
				conflictingEntityNames = append(conflictingEntityNames, entity.Name)
			}
		}

		if len(conflictingEntityNames) > 0 {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusConflict) // 409 Conflict
			json.NewEncoder(w).Encode(map[string]interface{}{
				"error":                "entities already exist",
				"conflicting_entities": conflictingEntityNames,
			})
			return
		}

		// If no conflicts, proceed to create entities and their observations
		for _, entity := range req.Entities {
			// Create entity (db.CreateEntity uses INSERT OR IGNORE, so no error on duplicate here,
			// but we've already checked above for explicit conflict reporting)
			if err := db.CreateEntity(database, entity.Name, entity.Type); err != nil {
				// This error would be for issues other than duplicates, e.g., DB connection
				http.Error(w, "Failed to create entity '"+entity.Name+"': "+err.Error(), http.StatusInternalServerError)
				return
			}

			// Create observations
			for _, obsContent := range entity.Observations {
				if _, err := db.CreateObservation(database, entity.Name, obsContent); err != nil {
					http.Error(w, "Failed to create observation for '"+entity.Name+"': "+err.Error(), http.StatusInternalServerError)
					return
				}
			}
			createdEntities = append(createdEntities, entity)
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(createdEntities)
	})

	// 3. POST /create_relations - Create relations with Python field names
	mux.HandleFunc("/create_relations", func(w http.ResponseWriter, r *http.Request) {
		addCORSHeaders(w)
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var req struct {
			Relations []db.Relation `json:"relations"`
		}

		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid JSON: "+err.Error(), http.StatusBadRequest)
			return
		}

		var createdRelations []db.Relation

		for _, relation := range req.Relations {
			// Validate that referenced entities exist
			var fromExists, toExists bool
			err := database.QueryRow(`SELECT EXISTS(SELECT 1 FROM entities WHERE name = ?)`, relation.From).Scan(&fromExists)
			if err != nil {
				http.Error(w, "Database error: "+err.Error(), http.StatusInternalServerError)
				return
			}
			err = database.QueryRow(`SELECT EXISTS(SELECT 1 FROM entities WHERE name = ?)`, relation.To).Scan(&toExists)
			if err != nil {
				http.Error(w, "Database error: "+err.Error(), http.StatusInternalServerError)
				return
			}

			if !fromExists {
				http.Error(w, "Entity '"+relation.From+"' does not exist", http.StatusBadRequest)
				return
			}
			if !toExists {
				http.Error(w, "Entity '"+relation.To+"' does not exist", http.StatusBadRequest)
				return
			}

			// Create relation
			if _, err := db.CreateRelation(database, relation.From, relation.To, relation.Type); err != nil {
				http.Error(w, "Failed to create relation: "+err.Error(), http.StatusInternalServerError)
				return
			}

			createdRelations = append(createdRelations, relation)
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(createdRelations)
	})

	// 4. POST /add_observations - Add observations with Python format
	mux.HandleFunc("/add_observations", func(w http.ResponseWriter, r *http.Request) {
		addCORSHeaders(w)
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var req struct {
			Observations []struct {
				EntityName string   `json:"entityName"`
				Contents   []string `json:"contents"`
			} `json:"observations"`
		}

		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid JSON: "+err.Error(), http.StatusBadRequest)
			return
		}

		// Transform to single-content observations for database
		var dbObservations []struct {
			EntityName string `json:"entityName"`
			Contents   string `json:"contents"`
		}

		for _, obs := range req.Observations {
			for _, content := range obs.Contents {
				dbObservations = append(dbObservations, struct {
					EntityName string `json:"entityName"`
					Contents   string `json:"contents"`
				}{
					EntityName: obs.EntityName,
					Contents:   content,
				})
			}
		}

		added, err := db.AddObservations(database, dbObservations)
		if err != nil {
			http.Error(w, "Failed to add observations: "+err.Error(), http.StatusInternalServerError)
			return
		}

		// Transform response back to Python format
		responseMap := make(map[string][]string)
		for _, obs := range added {
			responseMap[obs.EntityName] = append(responseMap[obs.EntityName], obs.Content)
		}

		var response []struct {
			EntityName string   `json:"entityName"`
			Contents   []string `json:"contents"`
		}

		for entityName, contents := range responseMap {
			response = append(response, struct {
				EntityName string   `json:"entityName"`
				Contents   []string `json:"contents"`
			}{
				EntityName: entityName,
				Contents:   contents,
			})
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(response)
	})

	// 5. POST /search_nodes - Search nodes with POST method and JSON body
	mux.HandleFunc("/search_nodes", func(w http.ResponseWriter, r *http.Request) {
		addCORSHeaders(w)
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var req struct {
			Query string `json:"query"`
		}

		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid JSON: "+err.Error(), http.StatusBadRequest)
			return
		}

		if req.Query == "" {
			http.Error(w, "Missing query field", http.StatusBadRequest)
			return
		}

		entities, relations, err := db.SearchNodes(database, req.Query)
		if err != nil {
			http.Error(w, "Failed to search nodes: "+err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(struct {
			Entities  []db.Entity   `json:"entities"`
			Relations []db.Relation `json:"relations"`
		}{
			Entities:  entities,
			Relations: relations,
		})
	})

	// 6. POST /open_nodes - Open specific nodes
	mux.HandleFunc("/open_nodes", func(w http.ResponseWriter, r *http.Request) {
		addCORSHeaders(w)
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var req struct {
			Names []string `json:"names"`
		}

		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid JSON: "+err.Error(), http.StatusBadRequest)
			return
		}

		entities, relations, err := db.OpenNodes(database, req.Names)
		if err != nil {
			http.Error(w, "Failed to open nodes: "+err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(struct {
			Entities  []db.Entity   `json:"entities"`
			Relations []db.Relation `json:"relations"`
		}{
			Entities:  entities,
			Relations: relations,
		})
	})

	// 7. POST /delete_entities - Delete entities with Python format
	mux.HandleFunc("/delete_entities", func(w http.ResponseWriter, r *http.Request) {
		addCORSHeaders(w)
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var req struct {
			EntityNames []string `json:"entityNames"`
		}

		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid JSON: "+err.Error(), http.StatusBadRequest)
			return
		}

		err := db.DeleteEntities(database, req.EntityNames)
		if err != nil {
			http.Error(w, "Failed to delete entities: "+err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":  "success",
			"deleted": len(req.EntityNames),
		})
	})

	// 8. POST /delete_observations - Delete observations with Python format
	mux.HandleFunc("/delete_observations", func(w http.ResponseWriter, r *http.Request) {
		addCORSHeaders(w)
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var req struct {
			Deletions []struct {
				EntityName   string   `json:"entityName"`
				Observations []string `json:"observations"`
			} `json:"deletions"`
		}

		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid JSON: "+err.Error(), http.StatusBadRequest)
			return
		}

		err := db.DeleteObservations(database, req.Deletions)
		if err != nil {
			http.Error(w, "Failed to delete observations: "+err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": "success"})
	})

	// 9. POST /delete_relations - Delete relations with Python format
	mux.HandleFunc("/delete_relations", func(w http.ResponseWriter, r *http.Request) {
		addCORSHeaders(w)
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var req struct {
			Relations []db.Relation `json:"relations"`
		}

		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid JSON: "+err.Error(), http.StatusBadRequest)
			return
		}

		// Convert to the format expected by db.DeleteRelations
		var dbRelations []struct {
			From string `json:"from"`
			To   string `json:"to"`
			Type string `json:"relationType"`
		}

		for _, rel := range req.Relations {
			dbRelations = append(dbRelations, struct {
				From string `json:"from"`
				To   string `json:"to"`
				Type string `json:"relationType"`
			}{
				From: rel.From,
				To:   rel.To,
				Type: rel.Type,
			})
		}

		err := db.DeleteRelations(database, dbRelations)
		if err != nil {
			http.Error(w, "Failed to delete relations: "+err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": "success"})
	})

	// Serve static frontend assets from embedded FS or disk as fallback.
	var fileServer http.Handler
	if StaticFS != nil {
		fileServer = http.FileServer(StaticFS)
	} else {
		staticFileDir := "cmd/knowledge-graph/web"
		if exePath, err := os.Executable(); err == nil {
			exeDir := filepath.Dir(exePath)
			staticFileDir = filepath.Join(exeDir, staticFileDir)
		} else {
			log.Printf("api: failed to get executable path, serving static from working directory %q: %v", staticFileDir, err)
		}
		fileServer = http.FileServer(http.Dir(staticFileDir))
	}
	mux.Handle("/", fileServer)
	return mux
}
