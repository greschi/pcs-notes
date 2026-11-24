package main

import (
	"encoding/json"
	"log"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

type Note struct {
	ID        string   `json:"id"`
	Title     string   `json:"title"`
	Content   string   `json:"content"`
	Tags      []string `json:"tags"`
	CreatedAt string   `json:"created_at"`
	UpdatedAt string   `json:"updated_at"`
}

type Store struct {
	mu       sync.Mutex
	filePath string
	Notes    map[string]Note `json:"notes"`
}

func NewStore(filePath string) (*Store, error) {
	s := &Store{
		filePath: filePath,
		Notes:    make(map[string]Note),
	}
	if err := s.load(); err != nil && !os.IsNotExist(err) {
		return nil, err
	}
	return s, nil
}

func (s *Store) load() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	file, err := os.Open(s.filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	decoder := json.NewDecoder(file)
	var notesList []Note
	if err := decoder.Decode(&notesList); err != nil {
		file.Seek(0, 0)
		return decoder.Decode(&s.Notes)
	}

	s.Notes = make(map[string]Note)
	for _, n := range notesList {
		s.Notes[n.ID] = n
	}
	return nil
}

func (s *Store) save() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	notesList := make([]Note, 0, len(s.Notes))
	for _, n := range s.Notes {
		notesList = append(notesList, n)
	}

	data, err := json.MarshalIndent(notesList, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(s.filePath, data, 0644)
}

func generateID() string {
	return time.Now().Format("20060102150405.000000")
}

func main() {
	dbFile := "notes.json"
	store, err := NewStore(dbFile)
	if err != nil {
		log.Fatalf("Failed to initialize store: %v", err)
	}

	mux := http.NewServeMux()

	mux.HandleFunc("/notes", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			query := r.URL.Query().Get("q")
			tagFilter := r.URL.Query().Get("tag")

			store.mu.Lock()
			var result []Note
			for _, n := range store.Notes {
				matchQuery := true
				matchTag := true

				if query != "" {
					qLower := strings.ToLower(query)
					matchQuery = strings.Contains(strings.ToLower(n.Title), qLower) ||
						strings.Contains(strings.ToLower(n.Content), qLower)
					if !matchQuery {
						for _, t := range n.Tags {
							if strings.Contains(strings.ToLower(t), qLower) {
								matchQuery = true
								break
							}
						}
					}
				}

				if tagFilter != "" {
					matchTag = false
					for _, t := range n.Tags {
						if strings.EqualFold(t, tagFilter) {
							matchTag = true
							break
						}
					}
				}

				if matchQuery && matchTag {
					result = append(result, n)
				}
			}
			store.mu.Unlock()

			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(result)

		case http.MethodPost:
			r.Body = http.MaxBytesReader(w, r.Body, 1024*1024)

			var req Note
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			if req.Title == "" {
				http.Error(w, "Title is required", http.StatusBadRequest)
				return
			}

			now := time.Now().UTC().Format(time.RFC3339)
			req.ID = generateID()
			req.CreatedAt = now
			req.UpdatedAt = now
			if req.Tags == nil {
				req.Tags = []string{}
			}

			store.mu.Lock()
			store.Notes[req.ID] = req
			store.mu.Unlock()

			if err := store.save(); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(req)

		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	})

	mux.HandleFunc("/notes/", func(w http.ResponseWriter, r *http.Request) {
		id := strings.TrimPrefix(r.URL.Path, "/notes/")
		if id == "" {
			http.Error(w, "Not found", http.StatusNotFound)
			return
		}

		switch r.Method {
		case http.MethodGet:
			store.mu.Lock()
			note, exists := store.Notes[id]
			store.mu.Unlock()

			if !exists {
				http.Error(w, "Note not found", http.StatusNotFound)
				return
			}

			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(note)

		case http.MethodPut:
			r.Body = http.MaxBytesReader(w, r.Body, 1024*1024)

			var req Note
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}

			store.mu.Lock()
			existing, exists := store.Notes[id]
			if !exists {
				store.mu.Unlock()
				http.Error(w, "Note not found", http.StatusNotFound)
				return
			}

			if req.Title != "" {
				existing.Title = req.Title
			}
			if req.Content != "" {
				existing.Content = req.Content
			}
			if req.Tags != nil {
				existing.Tags = req.Tags
			}
			existing.UpdatedAt = time.Now().UTC().Format(time.RFC3339)

			store.Notes[id] = existing
			store.mu.Unlock()

			if err := store.save(); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(existing)

		case http.MethodDelete:
			store.mu.Lock()
			_, exists := store.Notes[id]
			if !exists {
				store.mu.Unlock()
				http.Error(w, "Note not found", http.StatusNotFound)
				return
			}
			delete(store.Notes, id)
			store.mu.Unlock()

			if err := store.save(); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

			w.WriteHeader(http.StatusNoContent)

		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	})

	socketPath := "/run/notes/notes.sock"
	if err := os.MkdirAll(filepath.Dir(socketPath), 0755); err != nil {
		log.Fatalf("Failed to create socket directory: %v", err)
	}

	if err := os.Remove(socketPath); err != nil && !os.IsNotExist(err) {
		log.Fatalf("Failed to remove old socket: %v", err)
	}

	listener, err := net.Listen("unix", socketPath)
	if err != nil {
		log.Fatalf("Failed to listen on unix socket: %v", err)
	}
	defer listener.Close()

	if err := os.Chmod(socketPath, 0777); err != nil {
		log.Fatalf("Failed to set socket permissions: %v", err)
	}

	log.Printf("Notes service listening on Unix socket: %s", socketPath)
	if err := http.Serve(listener, mux); err != nil {
		log.Fatalf("HTTP server failed: %v", err)
	}
}
