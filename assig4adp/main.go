package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"sync"
)

type Book struct {
	BookID    string `json:"bookId"`
	BookTitle string `json:"bookTitle"`
	Author    string `json:"author"`
}

type LoanRecord struct {
	FullName   string `json:"fullName"`
	Email      string `json:"email"`
	Phone      string `json:"phone"`
	BookTitle  string `json:"bookTitle"`
	Author     string `json:"author"`
	BookID     string `json:"bookId"`
	LoanDate   string `json:"loanDate"`
	ReturnDate string `json:"returnDate"`
}

var (
	dbFile = "data.json"
	loans  []LoanRecord
	// Бастапқы 20 кітап каталогы
	inventory = []Book{
		{"1", "Abai Joly", "Mukhtar Auezov"},
		{"2", "Koshpendiler", "Iliyas Esenberlin"},
		{"3", "Kand men Ter", "Abdi-Jamil Nurpeisov"},
		{"4", "Ushbakyt", "Mukagali Makataev"},
		{"5", "Botagoz", "Sabit Mukanov"},
		{"6", "Olganin Orni Bolganin", "Gabit Musirepov"},
		{"7", "Akbilek", "Jusipbek Aimautov"},
		{"8", "Kiz Jibek", "Folklor"},
		{"9", "Meni Atim Kojah", "Berdibek Sokpakbaev"},
		{"10", "Sari Arka", "Saken Seifullin"},
		{"11", "Kala men Dala", "Olzhas Suleimenov"},
		{"12", "Zhibek Joly", "Anuar Alimzhanov"},
		{"13", "Alash", "Alikhan Bokeikhanov"},
		{"14", "Bakytsiz Jamal", "Mirzhakip Dulatov"},
		{"15", "Oyan Qazaq", "Mirzhakip Dulatov"},
		{"16", "Turkistan", "Magzhan Zhumabaev"},
		{"17", "Kokserke", "Mukhtar Auezov"},
		{"18", "Ak Kemey", "Chingiz Aitmatov"},
		{"19", "Jamilya", "Chingiz Aitmatov"},
		{"20", "Birinshi Mugalim", "Chingiz Aitmatov"},
	}
	mu sync.Mutex
)

func loadData() {
	file, err := os.ReadFile(dbFile)
	if err == nil {
		json.Unmarshal(file, &loans)
	}
}

func saveData() {
	data, _ := json.MarshalIndent(loans, "", "  ")
	os.WriteFile(dbFile, data, 0644)
}

func main() {
	loadData()

	fs := http.FileServer(http.Dir("./static"))
	http.Handle("/", fs)

	// 1. Тек кітаптар тізімін беру (Тізімде көрінуі үшін)
	http.HandleFunc("/api/books", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(inventory)
	})

	// 2. Жаңа жазбаны сақтау (Формадан келген деректер)
	http.HandleFunc("/api/save", func(w http.ResponseWriter, r *http.Request) {
		var newLoan LoanRecord
		if err := json.NewDecoder(r.Body).Decode(&newLoan); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		mu.Lock()
		loans = append(loans, newLoan)
		saveData()
		mu.Unlock()
		w.WriteHeader(http.StatusCreated)
	})

	// 3. Кесте үшін барлық жазбаларды алу
	http.HandleFunc("/api/data", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(loans)
	})

	fmt.Println("Сервер қосылды: http://localhost:8080")
	http.ListenAndServe(":8080", nil)
}
