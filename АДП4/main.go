package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
)

type Reader struct {
	ID    int    `json:"id"`
	Name  string `json:"name"`
	Phone string `json:"phone"`
}

type Loan struct {
	ReaderName string `json:"reader_name"`
	BookTitle  string `json:"book_title"`
	LoanDate   string `json:"loan_date"`
	ReturnDate string `json:"return_date"`
}

var (
	readers []Reader
	loans   []Loan
	mu      sync.Mutex
	nextRID = 1
)

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_ = json.NewEncoder(w).Encode(v)
}

func main() {
	// статиканы беру (index.html, script.js, style.css)
	fs := http.FileServer(http.Dir("./static"))
	http.Handle("/", fs)

	// Reader қосу
	http.HandleFunc("/add-reader", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Only POST allowed", http.StatusMethodNotAllowed)
			return
		}

		var req struct {
			Name  string `json:"name"`
			Phone string `json:"phone"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Bad JSON", http.StatusBadRequest)
			return
		}
		if req.Name == "" || req.Phone == "" {
			http.Error(w, "name and phone required", http.StatusBadRequest)
			return
		}

		mu.Lock()
		reader := Reader{ID: nextRID, Name: req.Name, Phone: req.Phone}
		nextRID++
		readers = append(readers, reader)
		mu.Unlock()

		writeJSON(w, reader)
	})

	// Loan қосу
	http.HandleFunc("/add-loan", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Only POST allowed", http.StatusMethodNotAllowed)
			return
		}

		var req Loan
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Bad JSON", http.StatusBadRequest)
			return
		}
		if req.ReaderName == "" || req.BookTitle == "" || req.LoanDate == "" || req.ReturnDate == "" {
			http.Error(w, "All fields required", http.StatusBadRequest)
			return
		}

		mu.Lock()
		loans = append(loans, req)
		mu.Unlock()

		writeJSON(w, req)
	})

	// Loan тізімі
	http.HandleFunc("/get-loans", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Only GET allowed", http.StatusMethodNotAllowed)
			return
		}
		mu.Lock()
		defer mu.Unlock()
		writeJSON(w, loans)
	})

	fmt.Println("Сервер қосылды: http://localhost:8080")
	_ = http.ListenAndServe(":8080", nil)
}
