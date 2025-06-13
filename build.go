//go:build ignore

// This program builds the WASM frontend for the knowledge-graph application.
// It is intended to be run by `go generate`.

package main

import (
	"log"
	"os"
	"os/exec"
)

func main() {
	log.Println("Building WASM frontend...")

	// This build script builds the WASM module from `cmd/frontend` and places 
    // the output in `cmd/knowledge-graph/web/main.wasm`.
	cmd := exec.Command("go", "build", "-o", "web/main.wasm", "../frontend")

	// Set the required environment variables for the WASM build in a cross-platform way.
	cmd.Env = append(os.Environ(), "GOOS=js", "GOARCH=wasm")

	// Combine stdout and stderr to see all output.
	output, err := cmd.CombinedOutput()

	if err != nil {
		log.Fatalf("WASM build failed:\n---[BEGIN BUILD OUTPUT]---\n%s\n---[END BUILD OUTPUT]---\nError: %s\n", string(output), err)
	}

    // Print the output only if there is any.
	if len(output) > 0 {
		log.Printf("WASM build successful:\n---[BEGIN BUILD OUTPUT]---\n%s\n---[END BUILD OUTPUT]---\n", string(output))
	} else {
        log.Println("WASM build successful.")
    }
}
