package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os/exec"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show infrastructure and database status",
	Long: `Check the health and status of all services required by aqe:

  - Docling (local Python - document parsing)
  - Weaviate (vector search)
  - Ollama (embeddings)
  - Claude CLI (LLM scoring)
  - SQLite database`,
	RunE: runStatus,
}

func init() {
	rootCmd.AddCommand(statusCmd)
}

type serviceStatus struct {
	name    string
	url     string
	status  string
	details string
	latency time.Duration
}

func runStatus(cmd *cobra.Command, args []string) error {
	fmt.Println("AQE Infrastructure Status")
	fmt.Println(strings.Repeat("=", 60))
	fmt.Println()

	ctx := context.Background()

	// Check all services
	services := checkServices(ctx)
	for _, svc := range services {
		icon := "OK"
		if svc.status == "DOWN" {
			icon = "FAIL"
		} else if svc.status == "WARN" {
			icon = "WARN"
		}

		fmt.Printf("  %-12s [%s] %s", svc.name, icon, svc.url)
		if svc.latency > 0 {
			fmt.Printf("  (%s)", svc.latency.Round(time.Millisecond))
		}
		fmt.Println()
		if svc.details != "" {
			fmt.Printf("               %s\n", svc.details)
		}
	}

	// Docker container status
	fmt.Println()
	fmt.Println("Docker Containers")
	fmt.Println(strings.Repeat("-", 60))
	checkDocker()

	// Database stats
	fmt.Println()
	fmt.Println("Database")
	fmt.Println(strings.Repeat("-", 60))
	checkDatabase()

	// Weaviate stats
	fmt.Println()
	fmt.Println("Weaviate Index")
	fmt.Println(strings.Repeat("-", 60))
	checkWeaviateStats(ctx)

	fmt.Println()
	return nil
}

func checkServices(ctx context.Context) []serviceStatus {
	var results []serviceStatus

	// Docling (local Python)
	results = append(results, checkLocalDocling())

	// Weaviate
	results = append(results, checkHTTPService("Weaviate", "http://localhost:8080/v1/.well-known/ready", func(body []byte) string {
		// Get version info from meta endpoint
		metaResp, err := http.Get("http://localhost:8080/v1/meta")
		if err != nil {
			return ""
		}
		defer metaResp.Body.Close()
		metaBody, _ := io.ReadAll(metaResp.Body)
		var meta struct {
			Version string `json:"version"`
		}
		if json.Unmarshal(metaBody, &meta) == nil {
			return fmt.Sprintf("v%s", meta.Version)
		}
		return ""
	}))

	// Ollama
	results = append(results, checkHTTPService("Ollama", "http://localhost:11434/api/version", func(body []byte) string {
		var resp struct {
			Version string `json:"version"`
		}
		if json.Unmarshal(body, &resp) == nil {
			return fmt.Sprintf("v%s", resp.Version)
		}
		return ""
	}))

	// Ollama model check
	results = append(results, checkOllamaModel())

	// Claude CLI
	results = append(results, checkClaudeCLI())

	return results
}

func checkLocalDocling() serviceStatus {
	svc := serviceStatus{
		name: "Docling",
		url:  "local Python",
	}

	// Check Python
	pythonPath, err := exec.LookPath("python3")
	if err != nil {
		pythonPath, err = exec.LookPath("python")
		if err != nil {
			svc.status = "DOWN"
			svc.details = "Python 3 not found in PATH"
			return svc
		}
	}

	// Check Docling packages
	start := time.Now()
	out, err := exec.Command(pythonPath, "-c",
		"import docling; import docling_core; import transformers; from importlib.metadata import version; print(version('docling'))").CombinedOutput()
	svc.latency = time.Since(start)

	if err != nil {
		svc.status = "DOWN"
		errMsg := strings.TrimSpace(string(out))
		// Extract the last meaningful line from the traceback
		if errMsg != "" {
			lines := strings.Split(errMsg, "\n")
			lastLine := lines[len(lines)-1]
			if strings.Contains(lastLine, "ModuleNotFoundError") || strings.Contains(lastLine, "ImportError") {
				svc.details = lastLine + "\nFix: pip install -r scripts/requirements.txt"
			} else {
				svc.details = lastLine
			}
		} else {
			svc.details = "Missing packages. Run: pip install -r scripts/requirements.txt"
		}
		return svc
	}

	svc.status = "OK"
	version := strings.TrimSpace(string(out))
	svc.details = fmt.Sprintf("v%s (via %s)", version, pythonPath)
	return svc
}

func checkHTTPService(name, url string, parseDetails func([]byte) string) serviceStatus {
	svc := serviceStatus{name: name, url: url}

	client := &http.Client{Timeout: 5 * time.Second}
	start := time.Now()
	resp, err := client.Get(url)
	svc.latency = time.Since(start)

	if err != nil {
		svc.status = "DOWN"
		svc.details = err.Error()
		return svc
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		svc.status = "OK"
		if parseDetails != nil {
			svc.details = parseDetails(body)
		}
	} else {
		svc.status = "DOWN"
		svc.details = fmt.Sprintf("HTTP %d: %s", resp.StatusCode, string(body))
	}

	return svc
}

func checkOllamaModel() serviceStatus {
	svc := serviceStatus{
		name: "Embeddings",
		url:  "nomic-embed-text",
	}

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get("http://localhost:11434/api/tags")
	if err != nil {
		svc.status = "DOWN"
		svc.details = "Ollama unreachable"
		return svc
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	var tags struct {
		Models []struct {
			Name    string `json:"name"`
			Size    int64  `json:"size"`
			Details struct {
				ParameterSize string `json:"parameter_size"`
			} `json:"details"`
		} `json:"models"`
	}

	if json.Unmarshal(body, &tags) != nil {
		svc.status = "WARN"
		svc.details = "Could not parse model list"
		return svc
	}

	for _, m := range tags.Models {
		if strings.HasPrefix(m.Name, "nomic-embed-text") {
			svc.status = "OK"
			svc.details = fmt.Sprintf("model loaded (%s, %.0fMB)", m.Details.ParameterSize, float64(m.Size)/1024/1024)
			return svc
		}
	}

	svc.status = "FAIL"
	svc.details = "Model not found. Run: docker exec -it ollama ollama pull nomic-embed-text"
	return svc
}

func checkClaudeCLI() serviceStatus {
	svc := serviceStatus{
		name: "Claude CLI",
		url:  "claude",
	}

	path, err := exec.LookPath("claude")
	if err != nil {
		svc.status = "DOWN"
		svc.details = "Not found in PATH"
		return svc
	}

	// Get version
	out, err := exec.Command(path, "--version").CombinedOutput()
	if err != nil {
		svc.status = "WARN"
		svc.details = fmt.Sprintf("Found at %s but version check failed", path)
		return svc
	}

	svc.status = "OK"
	svc.details = strings.TrimSpace(string(out))
	return svc
}

func checkDocker() {
	out, err := exec.Command("docker", "ps",
		"--filter", "name=weaviate",
		"--filter", "name=ollama",
		"--format", "  {{.Names}}\t{{.Status}}\t{{.Ports}}",
	).CombinedOutput()

	if err != nil {
		fmt.Printf("  Could not query Docker: %v\n", err)
		return
	}

	output := strings.TrimSpace(string(out))
	if output == "" {
		fmt.Println("  No AQE containers running")
		fmt.Println("  Start with: docker-compose up -d")
	} else {
		fmt.Println(output)
	}
}

func checkDatabase() {
	st := GetStore()
	if st == nil {
		fmt.Printf("  Database: not initialized\n")
		return
	}

	fmt.Printf("  Path:        %s\n", dbPath)

	// Documents with chunk counts
	docs, err := st.ListDocumentsWithChunkCounts()
	if err != nil {
		fmt.Printf("  Documents:   error: %v\n", err)
	} else {
		fmt.Printf("  Documents:   %d\n", len(docs))
		totalChunks := 0
		for _, doc := range docs {
			totalChunks += doc.ChunkCount
			title := "(untitled)"
			if doc.Title != nil {
				title = *doc.Title
			}
			// Truncate long titles
			if len(title) > 50 {
				title = title[:47] + "..."
			}
			authors := "unknown"
			if len(doc.Authors) > 0 {
				authors = strings.Join(doc.Authors, "; ")
			}
			year := "n.d."
			if doc.Year != nil {
				year = fmt.Sprintf("%d", *doc.Year)
			}
			fmt.Printf("    #%-3d %-50s %s (%s) [%d chunks]\n", doc.ID, title, authors, year, doc.ChunkCount)
		}
		fmt.Printf("  Chunks:      %d\n", totalChunks)
	}

	// Extractions and quotes
	extractions, err := st.ListExtractions()
	if err != nil {
		fmt.Printf("  Extractions: error: %v\n", err)
	} else {
		totalQuotes := 0
		for _, ext := range extractions {
			totalQuotes += ext.QuoteCount
		}
		fmt.Printf("  Extractions: %d\n", len(extractions))
		fmt.Printf("  Quotes:      %d\n", totalQuotes)
		for _, ext := range extractions {
			fmt.Printf("    #%-3d %-50s [%d quotes]\n", ext.ID, ext.Topic, ext.QuoteCount)
		}
	}
}

func checkWeaviateStats(ctx context.Context) {
	client := &http.Client{Timeout: 5 * time.Second}

	// Get node info with shard details
	resp, err := client.Get("http://localhost:8080/v1/nodes?output=verbose")
	if err != nil {
		fmt.Printf("  Unreachable: %v\n", err)
		return
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	var nodes struct {
		Nodes []struct {
			Name   string `json:"name"`
			Status string `json:"status"`
			Shards []struct {
				Class       string `json:"class"`
				ObjectCount int    `json:"objectCount"`
			} `json:"shards"`
		} `json:"nodes"`
	}

	if json.Unmarshal(body, &nodes) != nil {
		fmt.Println("  Could not parse node info")
		return
	}

	for _, node := range nodes.Nodes {
		fmt.Printf("  Node:   %s (%s)\n", node.Name, node.Status)
		if len(node.Shards) == 0 {
			fmt.Println("  Shards: none (no data indexed)")
		}
		for _, shard := range node.Shards {
			fmt.Printf("  Class:  %s (%d objects)\n", shard.Class, shard.ObjectCount)
		}
	}
}
