package api

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/MdSaifAliMolla/GoGoL/internal/index"
)

type Server struct {
	Index *index.Indexer
}

func NewServer(idx *index.Indexer) *Server {
	return &Server{Index: idx}
}

func (s *Server) Start(port string) error {
	http.Handle("/", http.FileServer(http.Dir("./web")))
	http.HandleFunc("/search", s.handleSearch)
	http.HandleFunc("/stats", s.handleStats)
	fmt.Printf("Server listening on port %s...\n", port)
	return http.ListenAndServe(":"+port, nil)
}

func (s *Server) handleStats(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(s.Index.Stats())
}

func (s *Server) handleSearch(w http.ResponseWriter, r *http.Request) {
	// CORS
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Content-Type", "application/json")

	query := r.URL.Query().Get("q")
	if query == "" {
		http.Error(w, `{"error": "Missing query parameter 'q'"}`, http.StatusBadRequest)
		return
	}
	results := s.Index.Search(query)
	
	json.NewEncoder(w).Encode(results)
}
