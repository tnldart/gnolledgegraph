package api

import (
	"database/sql"
	"encoding/json"
	"io"
	"net/http"
	"os"

	"gnolledgegraph/internal/db"
)

// now captures the on-disk sqlite file path
func NewHandler(database *sql.DB, dbPath string) http.Handler {
	mux := http.NewServeMux()

	// POST /api/import_db  ←  upload new DB blob
	mux.HandleFunc("/api/import_db", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		data, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "Cannot read body: "+err.Error(), http.StatusBadRequest)
			return
		}
		if err := os.WriteFile(dbPath, data, 0o644); err != nil {
			http.Error(w, "Cannot write DB file: "+err.Error(), http.StatusInternalServerError)
			return
		}
		// Optionally you could re-open the database here
		w.WriteHeader(http.StatusNoContent)
	})

	// GET /api/export_db  ←  download current DB blob
	mux.HandleFunc("/api/export_db", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		w.Header().Set("Content-Type", "application/octet-stream")
		http.ServeFile(w, r, dbPath)
	})
	mux.HandleFunc("/api/read_graph", func(w http.ResponseWriter, r *http.Request) {
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

	mux.HandleFunc("/api/create_entities", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var req struct {
			Entities []struct {
				Name string `json:"name"`
				Type string `json:"entity_type"`
			} `json:"entities"`
		}

		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid JSON: "+err.Error(), http.StatusBadRequest)
			return
		}

		for _, entity := range req.Entities {
			if err := db.CreateEntity(database, entity.Name, entity.Type); err != nil {
				http.Error(w, "Failed to create entity: "+err.Error(), http.StatusInternalServerError)
				return
			}
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]string{"status": "success"})
	})

	mux.HandleFunc("/api/create_relations", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var req struct {
			Relations []struct {
				From string `json:"from_entity"`
				To   string `json:"to_entity"`
				Type string `json:"relation_type"`
			} `json:"relations"`
		}

		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid JSON: "+err.Error(), http.StatusBadRequest)
			return
		}

		var createdIDs []int64
		for _, relation := range req.Relations {
			id, err := db.CreateRelation(database, relation.From, relation.To, relation.Type)
			if err != nil {
				http.Error(w, "Failed to create relation: "+err.Error(), http.StatusInternalServerError)
				return
			}
			createdIDs = append(createdIDs, id)
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status": "success",
			"ids":    createdIDs,
		})
	})

	mux.HandleFunc("/api/add_observations", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var req struct {
			Observations []struct {
				EntityName string `json:"entityName"`
				Contents   string `json:"contents"`
			} `json:"observations"`
		}

		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid JSON: "+err.Error(), http.StatusBadRequest)
			return
		}

		added, err := db.AddObservations(database, req.Observations)
		if err != nil {
			http.Error(w, "Failed to add observations: "+err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status": "success",
			"added":  added,
		})
	})

	mux.HandleFunc("/api/delete_entities", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
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

	mux.HandleFunc("/api/delete_observations", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
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

	mux.HandleFunc("/api/delete_relations", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var req struct {
			Relations []struct {
				From string `json:"from"`
				To   string `json:"to"`
				Type string `json:"relationType"`
			} `json:"relations"`
		}

		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid JSON: "+err.Error(), http.StatusBadRequest)
			return
		}

		err := db.DeleteRelations(database, req.Relations)
		if err != nil {
			http.Error(w, "Failed to delete relations: "+err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": "success"})
	})

	mux.HandleFunc("/api/search_nodes", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		query := r.URL.Query().Get("query")
		if query == "" {
			http.Error(w, "Missing query parameter", http.StatusBadRequest)
			return
		}

		entities, relations, err := db.SearchNodes(database, query)
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

	mux.HandleFunc("/api/open_nodes", func(w http.ResponseWriter, r *http.Request) {
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

	return mux
}
