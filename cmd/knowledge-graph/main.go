package main

import (
	"bufio"
	"database/sql"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"mime"
	"net/http"
	"os"

	"embed"
	"io/fs"

	"memory-parttwo/internal/api"
	"memory-parttwo/internal/db"
	"memory-parttwo/internal/mcp"
)

// The go:generate command will be executed by `go generate ./...`
// It compiles the frontend WASM and places it where it can be embedded.
//go:generate go run ../../build.go

// embeddedWebFS contains the static frontend assets (index.html, JS, WASM).
//
//go:embed web/*
var embeddedWebFS embed.FS

// corsMiddleware adds CORS headers to the response
func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*") // Allow any origin
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Requested-With")

		// Handle preflight requests
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func init() {
	// serve .wasm with the proper MIME type for instantiateStreaming()
	mime.AddExtensionType(".wasm", "application/wasm")
	// serve .js with the proper MIME type for ES modules
	mime.AddExtensionType(".js", "application/javascript")
}

// handleStdioMCP handles MCP communication over stdin/stdout
func handleStdioMCP(database *sql.DB) {
	scanner := bufio.NewScanner(os.Stdin)
	encoder := json.NewEncoder(os.Stdout)

	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}

		var req mcp.JSONRPCRequest
		if err := json.Unmarshal([]byte(line), &req); err != nil {
			log.Printf("stdio MCP: invalid JSON: %v", err)
			continue
		}

		// Process the request using existing MCP handler logic
		// req.ID will be nil if the original request JSON had no "id" or "id": null.
		// Such requests are Notifications as per JSON-RPC 2.0.

		// Only generate a response if it's not a notification.
		// However, HandleJSONRPCMethod is designed to always return a response structure.
		// The decision to send it back should be here.
		if req.ID != nil {
			response := mcp.HandleJSONRPCMethod(database, req)
			if err := encoder.Encode(response); err != nil {
				log.Printf("stdio MCP: failed to encode response: %v", err)
			}
		} else {
			// It's a notification, do not send a response.
			// Optionally, log that a notification was received and processed if needed.
			// log.Printf("stdio MCP: received notification, method: %s, no response sent", req.Method)
			// Depending on whether methods invoked by notifications are expected to do something,
			// you might still call a handler but just not send the JSONRPCResponse.
			// For now, we assume HandleJSONRPCMethod might have side effects even for notifications
			// if specific methods are designed that way, but no JSON response is sent back.
			// If methods called via notification should truly do nothing or are not expected,
			// then mcp.HandleJSONRPCMethod(database, req) could also be inside the if req.ID != nil block.
			// Let's assume for now that some processing might occur, but no response.
			// To be safe and ensure methods are still called if they are notifications:
			_ = mcp.HandleJSONRPCMethod(database, req) // Process but discard response for notifications
		}
	}

	if err := scanner.Err(); err != nil {
		log.Printf("stdio MCP: scanner error: %v", err)
	}
}

func main() {

	// flags
	port := flag.Int("port", 8080, "HTTP port")
	dbPath := flag.String("db-path", "kg.db", "path to sqlite database")
	enableStdio := flag.Bool("enable-stdio", true, "enable stdio MCP transport alongside HTTP server")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage of %s:\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "Runs the Knowledge Graph server, providing dual API access:\n")
		fmt.Fprintf(os.Stderr, "  - Original Go API: mounted at /api/\n")
		fmt.Fprintf(os.Stderr, "  - Python FastAPI Compatibility API: mounted at / (root)\n\n")
		fmt.Fprintf(os.Stderr, "Flags:\n")
		flag.PrintDefaults()
	}
	flag.Parse()

	// 1) init sqlite + schema
	sqldb, err := db.Init(*dbPath)
	if err != nil {
		log.Fatalf("db.Init: %v", err)
	}

	// setup embedded static assets for frontend
	staticFiles, err := fs.Sub(embeddedWebFS, "web")
	if err != nil {
		log.Fatalf("failed to access embedded web files: %v", err)
	}
	api.StaticFS = http.FS(staticFiles)

	// 2) Mount Python compatibility handler at root.
	// This handler is now responsible for serving static files for the root path,
	// including WASM MIME type handling, by presumably embedding web/* itself.
	http.Handle("/", api.NewPythonCompatHandler(sqldb))

	// 3) mount API under /api/, pass on-disk path so import/export can read/write it
	http.Handle("/api/", api.NewHandler(sqldb, *dbPath))

	// 4) mount MCP endpoints according to MCP specification
	mcpHandler := mcp.NewMCPHandler(sqldb)
	http.Handle("/sse", mcpHandler)      // SSE connection endpoint
	http.Handle("/messages", mcpHandler) // POST messages endpoint

	// Legacy endpoints for backward compatibility
	http.Handle("/mcp", mcpHandler) // Legacy combined endpoint
	http.Handle("/mcp/legacy", mcp.NewHandler(sqldb))

	// 5) serve generated OpenAPI JSON
	http.HandleFunc("/openapi.json", func(w http.ResponseWriter, r *http.Request) {
		data, err := api.GenerateOpenAPIJSON()
		if err != nil {
			http.Error(w, "Failed to generate OpenAPI spec", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(data)
	})

	// 6) start stdio MCP transport
	// If enableStdio is true, this will be the main blocking call if the HTTP server
	// is not started or fails.
	if *enableStdio {
		log.Printf("starting stdio MCP transport")
		// If only stdio is desired, the HTTP server part below can be skipped
		// or made conditional based on another flag.
		// For now, we'll allow both but ensure stdio can run even if HTTP fails.
		go handleStdioMCP(sqldb) // Run stdio handler in a goroutine to allow HTTP server to also start
	}

	// 7) start HTTP server
	addr := fmt.Sprintf(":%d", *port)
	log.Printf("attempting to listen on %s for HTTP server", addr)

	// Wrap DefaultServeMux with CORS middleware
	handlerWithCors := corsMiddleware(http.DefaultServeMux)

	// Start HTTP server. If --enable-stdio is the primary mode,
	// an error here (like "address already in use") shouldn't kill the stdio transport.
	err = http.ListenAndServe(addr, handlerWithCors)
	if err != nil {
		log.Printf("HTTP server ListenAndServe error: %v", err)
		// If stdio is not enabled, this is a fatal error.
		// If stdio IS enabled, the program might continue running for stdio.
		// However, if stdio was also in a goroutine, the main thread needs to block.
		// The original logic for stdio was to run it in a goroutine and then fatal on HTTP error.
		// Let's adjust: if stdio is enabled, we don't fatal here.
		// The stdio goroutine will keep the process alive.
		// If stdio is NOT enabled, then this is a fatal error.
		if !*enableStdio {
			log.Fatalf("HTTP server failed to start and stdio not enabled: %v", err)
		}
		// If stdio is enabled, we log the error and the stdio goroutine (if started)
		// will keep the application alive. If stdio was NOT started in a goroutine
		// and was intended to be the main loop, this logic needs more refinement
		// based on whether HTTP is primary or secondary.

		// For the current problem: "address already in use" when launched by Claude for stdio.
		// We want the stdio part to continue.
		// The `handleStdioMCP` is now in a goroutine.
		// If HTTP server fails, and stdio is enabled, we need main to not exit.
		// A simple way is to block indefinitely if stdio is enabled and HTTP failed.
		if *enableStdio {
			log.Println("HTTP server failed to start, but stdio MCP is enabled and running. Process will remain alive for stdio.")
			select {} // Block forever to keep stdio transport alive
		}
	}
}
