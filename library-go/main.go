package main

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"golang.org/x/crypto/bcrypt"
)

type User struct {
	ID           string `json:"id"`
	FullName     string `json:"fullName"`
	Email        string `json:"email"`
	Phone        string `json:"phone"`
	PasswordHash string `json:"passwordHash"`
	Role         string `json:"role"`
	CreatedAt    string `json:"createdAt"`
}

type Book struct {
	ID           string  `json:"id"`
	BookCode     string  `json:"bookCode"`
	Title        string  `json:"title"`
	Author       string  `json:"author"`
	Price        float64 `json:"price"`
	TotalQty     int     `json:"totalQty"`
	AvailableQty int     `json:"availableQty"`
	CreatedAt    string  `json:"createdAt"`
}

type Loan struct {
	ID         string  `json:"id"`
	ReaderID   string  `json:"readerId"`
	BookID     string  `json:"bookId"`
	LoanDate   string  `json:"loanDate"`
	DueDate    string  `json:"dueDate"`
	ReturnDate string  `json:"returnDate"`
	Status     string  `json:"status"`
	FineAmount float64 `json:"fineAmount"`
}

type Database struct {
	Users []User `json:"users"`
	Books []Book `json:"books"`
	Loans []Loan `json:"loans"`
}

var (
	dbFile = "data.json"
	db     Database
	mu     sync.Mutex

	tokenStore = map[string]string{}
)

func nowDate() string {
	return time.Now().Format("2006-01-02")
}

func jsonWrite(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func jsonRead(r *http.Request, dst any) error {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		return err
	}
	if len(body) == 0 {
		return errors.New("empty body")
	}
	return json.Unmarshal(body, dst)
}

func loadDB() {
	f, err := os.Open(dbFile)
	if err != nil {
		return
	}
	defer f.Close()
	_ = json.NewDecoder(f).Decode(&db)
}

func saveDB() {
	tmp := dbFile + ".tmp"
	f, err := os.Create(tmp)
	if err != nil {
		return
	}
	defer f.Close()
	_ = json.NewEncoder(f).Encode(db)
	_ = os.Rename(tmp, dbFile)
}

func genID(prefix string, n int) string {
	return fmt.Sprintf("%s%d", prefix, n)
}

func genToken() string {
	b := make([]byte, 32)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

func findUserByEmail(email string) (*User, int) {
	for i := range db.Users {
		if strings.EqualFold(db.Users[i].Email, email) {
			return &db.Users[i], i
		}
	}
	return nil, -1
}

func findUserByID(id string) (*User, int) {
	for i := range db.Users {
		if db.Users[i].ID == id {
			return &db.Users[i], i
		}
	}
	return nil, -1
}

func findBookByID(id string) (*Book, int) {
	for i := range db.Books {
		if db.Books[i].ID == id {
			return &db.Books[i], i
		}
	}
	return nil, -1
}

func findLoanByID(id string) (*Loan, int) {
	for i := range db.Loans {
		if db.Loans[i].ID == id {
			return &db.Loans[i], i
		}
	}
	return nil, -1
}

func anyAdminExists() bool {
	for _, u := range db.Users {
		if u.Role == "admin" {
			return true
		}
	}
	return false
}

func authUser(r *http.Request) (*User, error) {
	h := r.Header.Get("Authorization")
	if !strings.HasPrefix(h, "Bearer ") {
		return nil, errors.New("no token")
	}
	token := strings.TrimSpace(strings.TrimPrefix(h, "Bearer "))
	mu.Lock()
	uid, ok := tokenStore[token]
	mu.Unlock()
	if !ok {
		return nil, errors.New("invalid token")
	}

	mu.Lock()
	defer mu.Unlock()
	u, _ := findUserByID(uid)
	if u == nil {
		return nil, errors.New("user not found")
	}
	return u, nil
}

func requireAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		_, err := authUser(r)
		if err != nil {
			jsonWrite(w, 401, map[string]any{"error": "Unauthorized"})
			return
		}
		next(w, r)
	}
}

func requireAdmin(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		u, err := authUser(r)
		if err != nil {
			jsonWrite(w, 401, map[string]any{"error": "Unauthorized"})
			return
		}
		if u.Role != "admin" {
			jsonWrite(w, 403, map[string]any{"error": "Admin only"})
			return
		}
		next(w, r)
	}
}

func apiMeta(w http.ResponseWriter, r *http.Request) {
	mu.Lock()
	defer mu.Unlock()
	jsonWrite(w, 200, map[string]any{
		"needBootstrapAdmin": !anyAdminExists(),
	})
}

func apiRegister(w http.ResponseWriter, r *http.Request) {
	type Req struct {
		FullName string `json:"fullName"`
		Email    string `json:"email"`
		Phone    string `json:"phone"`
		Password string `json:"password"`
		Role     string `json:"role"`
	}
	var req Req
	if err := jsonRead(r, &req); err != nil {
		jsonWrite(w, 400, map[string]any{"error": "Bad JSON"})
		return
	}
	req.Email = strings.ToLower(strings.TrimSpace(req.Email))
	req.Role = strings.ToLower(strings.TrimSpace(req.Role))
	if req.Role != "admin" {
		req.Role = "reader"
	}
	if req.FullName == "" || req.Email == "" || req.Phone == "" || req.Password == "" {
		jsonWrite(w, 400, map[string]any{"error": "Missing fields"})
		return
	}

	mu.Lock()
	defer mu.Unlock()

	if u, _ := findUserByEmail(req.Email); u != nil {
		jsonWrite(w, 409, map[string]any{"error": "Email already exists"})
		return
	}

	if req.Role == "admin" && anyAdminExists() {
		creator, err := authUser(r)
		if err != nil || creator.Role != "admin" {
			jsonWrite(w, 403, map[string]any{"error": "Only admin can create another admin"})
			return
		}
	}

	hash, _ := bcrypt.GenerateFromPassword([]byte(req.Password), 10)

	user := User{
		ID:           genID("U", len(db.Users)+1),
		FullName:     req.FullName,
		Email:        req.Email,
		Phone:        req.Phone,
		PasswordHash: string(hash),
		Role:         req.Role,
		CreatedAt:    nowDate(),
	}
	db.Users = append(db.Users, user)
	saveDB()

	jsonWrite(w, 200, map[string]any{
		"ok":   true,
		"user": map[string]any{"id": user.ID, "fullName": user.FullName, "email": user.Email, "phone": user.Phone, "role": user.Role},
	})
}

func apiLogin(w http.ResponseWriter, r *http.Request) {
	type Req struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	var req Req
	if err := jsonRead(r, &req); err != nil {
		jsonWrite(w, 400, map[string]any{"error": "Bad JSON"})
		return
	}
	req.Email = strings.ToLower(strings.TrimSpace(req.Email))

	mu.Lock()
	u, _ := findUserByEmail(req.Email)
	mu.Unlock()

	if u == nil {
		jsonWrite(w, 401, map[string]any{"error": "Invalid email/password"})
		return
	}
	if bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(req.Password)) != nil {
		jsonWrite(w, 401, map[string]any{"error": "Invalid email/password"})
		return
	}

	token := genToken()
	mu.Lock()
	tokenStore[token] = u.ID
	mu.Unlock()

	jsonWrite(w, 200, map[string]any{
		"ok":    true,
		"token": token,
		"user":  map[string]any{"id": u.ID, "fullName": u.FullName, "email": u.Email, "phone": u.Phone, "role": u.Role},
	})
}

func apiMe(w http.ResponseWriter, r *http.Request) {
	u, err := authUser(r)
	if err != nil {
		jsonWrite(w, 401, map[string]any{"error": "Unauthorized"})
		return
	}
	jsonWrite(w, 200, map[string]any{
		"id": u.ID, "fullName": u.FullName, "email": u.Email, "phone": u.Phone, "role": u.Role,
	})
}

func apiListUsers(w http.ResponseWriter, r *http.Request) {
	role := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("role")))
	mu.Lock()
	defer mu.Unlock()

	out := []any{}
	for _, u := range db.Users {
		if role != "" && u.Role != role {
			continue
		}
		out = append(out, map[string]any{
			"id": u.ID, "fullName": u.FullName, "email": u.Email, "phone": u.Phone, "role": u.Role, "createdAt": u.CreatedAt,
		})
	}
	jsonWrite(w, 200, out)
}

func apiGetBooks(w http.ResponseWriter, r *http.Request) {
	mu.Lock()
	defer mu.Unlock()
	jsonWrite(w, 200, db.Books)
}

func apiCreateBook(w http.ResponseWriter, r *http.Request) {
	type Req struct {
		BookCode string  `json:"bookCode"`
		Title    string  `json:"title"`
		Author   string  `json:"author"`
		Price    float64 `json:"price"`
		TotalQty int     `json:"totalQty"`
	}
	var req Req
	if err := jsonRead(r, &req); err != nil {
		jsonWrite(w, 400, map[string]any{"error": "Bad JSON"})
		return
	}
	req.BookCode = strings.TrimSpace(req.BookCode)
	if req.BookCode == "" || req.Title == "" || req.Author == "" || req.TotalQty <= 0 {
		jsonWrite(w, 400, map[string]any{"error": "Missing/invalid fields"})
		return
	}

	mu.Lock()
	defer mu.Unlock()

	for _, b := range db.Books {
		if strings.EqualFold(b.BookCode, req.BookCode) {
			jsonWrite(w, 409, map[string]any{"error": "bookCode already exists"})
			return
		}
	}

	book := Book{
		ID:           genID("B", len(db.Books)+1),
		BookCode:     req.BookCode,
		Title:        req.Title,
		Author:       req.Author,
		Price:        req.Price,
		TotalQty:     req.TotalQty,
		AvailableQty: req.TotalQty,
		CreatedAt:    nowDate(),
	}
	db.Books = append(db.Books, book)
	saveDB()
	jsonWrite(w, 200, book)
}

func apiUpdateBook(w http.ResponseWriter, r *http.Request) {

	id := strings.TrimPrefix(r.URL.Path, "/api/books/")
	if id == "" {
		jsonWrite(w, 400, map[string]any{"error": "Missing id"})
		return
	}

	type Req struct {
		Title    *string  `json:"title"`
		Author   *string  `json:"author"`
		Price    *float64 `json:"price"`
		TotalQty *int     `json:"totalQty"`
	}
	var req Req
	if err := jsonRead(r, &req); err != nil {
		jsonWrite(w, 400, map[string]any{"error": "Bad JSON"})
		return
	}

	mu.Lock()
	defer mu.Unlock()

	book, idx := findBookByID(id)
	if book == nil {
		jsonWrite(w, 404, map[string]any{"error": "Book not found"})
		return
	}

	if req.TotalQty != nil {
		newTotal := *req.TotalQty
		if newTotal < 0 {
			jsonWrite(w, 400, map[string]any{"error": "totalQty invalid"})
			return
		}
		diff := newTotal - db.Books[idx].TotalQty
		db.Books[idx].TotalQty = newTotal
		db.Books[idx].AvailableQty += diff
		if db.Books[idx].AvailableQty < 0 {
			db.Books[idx].AvailableQty = 0
		}
	}
	if req.Title != nil {
		db.Books[idx].Title = *req.Title
	}
	if req.Author != nil {
		db.Books[idx].Author = *req.Author
	}
	if req.Price != nil {
		db.Books[idx].Price = *req.Price
	}

	saveDB()
	jsonWrite(w, 200, db.Books[idx])
}

func apiDeleteBook(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, "/api/books/")
	if id == "" {
		jsonWrite(w, 400, map[string]any{"error": "Missing id"})
		return
	}

	mu.Lock()
	defer mu.Unlock()

	_, idx := findBookByID(id)
	if idx == -1 {
		jsonWrite(w, 404, map[string]any{"error": "Book not found"})
		return
	}

	for _, l := range db.Loans {
		if l.BookID == id && l.Status == "borrowed" {
			jsonWrite(w, 400, map[string]any{"error": "Cannot delete: book is borrowed now"})
			return
		}
	}

	db.Books = append(db.Books[:idx], db.Books[idx+1:]...)
	saveDB()
	jsonWrite(w, 200, map[string]any{"ok": true})
}

func apiBorrow(w http.ResponseWriter, r *http.Request) {
	type Req struct {
		ReaderID string `json:"readerId"`
		BookID   string `json:"bookId"`
		DueDate  string `json:"dueDate"`
	}
	var req Req
	if err := jsonRead(r, &req); err != nil {
		jsonWrite(w, 400, map[string]any{"error": "Bad JSON"})
		return
	}
	if req.ReaderID == "" || req.BookID == "" {
		jsonWrite(w, 400, map[string]any{"error": "Missing readerId/bookId"})
		return
	}

	mu.Lock()
	defer mu.Unlock()

	reader, _ := findUserByID(req.ReaderID)
	if reader == nil || reader.Role != "reader" {
		jsonWrite(w, 404, map[string]any{"error": "Reader not found"})
		return
	}
	book, bidx := findBookByID(req.BookID)
	if book == nil {
		jsonWrite(w, 404, map[string]any{"error": "Book not found"})
		return
	}
	if db.Books[bidx].AvailableQty <= 0 {
		jsonWrite(w, 400, map[string]any{"error": "Book out of stock"})
		return
	}

	due := req.DueDate
	if due == "" {
		due = time.Now().Add(7 * 24 * time.Hour).Format("2006-01-02")
	}

	db.Books[bidx].AvailableQty--

	loan := Loan{
		ID:         genID("L", len(db.Loans)+1),
		ReaderID:   req.ReaderID,
		BookID:     req.BookID,
		LoanDate:   nowDate(),
		DueDate:    due,
		ReturnDate: "",
		Status:     "borrowed",
		FineAmount: 0,
	}
	db.Loans = append(db.Loans, loan)
	saveDB()
	jsonWrite(w, 200, loan)
}

func apiReturn(w http.ResponseWriter, r *http.Request) {
	type Req struct {
		LoanID string `json:"loanId"`
	}
	var req Req
	if err := jsonRead(r, &req); err != nil {
		jsonWrite(w, 400, map[string]any{"error": "Bad JSON"})
		return
	}

	mu.Lock()
	defer mu.Unlock()

	loan, lidx := findLoanByID(req.LoanID)
	if loan == nil {
		jsonWrite(w, 404, map[string]any{"error": "Loan not found"})
		return
	}
	if db.Loans[lidx].Status != "borrowed" {
		jsonWrite(w, 400, map[string]any{"error": "Loan is not active"})
		return
	}

	db.Loans[lidx].Status = "returned"
	db.Loans[lidx].ReturnDate = nowDate()

	_, bidx := findBookByID(db.Loans[lidx].BookID)
	if bidx != -1 {
		db.Books[bidx].AvailableQty++
	}

	saveDB()
	jsonWrite(w, 200, db.Loans[lidx])
}

func apiLost(w http.ResponseWriter, r *http.Request) {
	type Req struct {
		LoanID     string   `json:"loanId"`
		FineAmount *float64 `json:"fineAmount"`
	}
	var req Req
	if err := jsonRead(r, &req); err != nil {
		jsonWrite(w, 400, map[string]any{"error": "Bad JSON"})
		return
	}

	mu.Lock()
	defer mu.Unlock()

	loan, lidx := findLoanByID(req.LoanID)
	if loan == nil {
		jsonWrite(w, 404, map[string]any{"error": "Loan not found"})
		return
	}
	if db.Loans[lidx].Status != "borrowed" {
		jsonWrite(w, 400, map[string]any{"error": "Loan is not active"})
		return
	}

	fine := 0.0
	book, bidx := findBookByID(db.Loans[lidx].BookID)
	if req.FineAmount != nil {
		fine = *req.FineAmount
	} else if book != nil {
		fine = book.Price
	}

	db.Loans[lidx].Status = "lost"
	db.Loans[lidx].ReturnDate = nowDate()
	db.Loans[lidx].FineAmount = fine

	if bidx != -1 && db.Books[bidx].TotalQty > 0 {
		db.Books[bidx].TotalQty--

		if db.Books[bidx].AvailableQty > db.Books[bidx].TotalQty {
			db.Books[bidx].AvailableQty = db.Books[bidx].TotalQty
		}
	}

	saveDB()
	jsonWrite(w, 200, db.Loans[lidx])
}

func apiListLoans(w http.ResponseWriter, r *http.Request) {
	mu.Lock()
	defer mu.Unlock()

	type LoanView struct {
		Loan
		ReaderName  string `json:"readerName"`
		ReaderPhone string `json:"readerPhone"`
		ReaderEmail string `json:"readerEmail"`
		BookCode    string `json:"bookCode"`
		BookTitle   string `json:"bookTitle"`
		BookAuthor  string `json:"bookAuthor"`
	}

	out := []LoanView{}
	for _, l := range db.Loans {
		reader, _ := findUserByID(l.ReaderID)
		book, _ := findBookByID(l.BookID)
		v := LoanView{Loan: l}
		if reader != nil {
			v.ReaderName = reader.FullName
			v.ReaderPhone = reader.Phone
			v.ReaderEmail = reader.Email
		}
		if book != nil {
			v.BookCode = book.BookCode
			v.BookTitle = book.Title
			v.BookAuthor = book.Author
		}
		out = append(out, v)
	}

	jsonWrite(w, 200, out)
}

func apiMyLoans(w http.ResponseWriter, r *http.Request) {
	u, err := authUser(r)
	if err != nil {
		jsonWrite(w, 401, map[string]any{"error": "Unauthorized"})
		return
	}
	if u.Role != "reader" {
		jsonWrite(w, 403, map[string]any{"error": "Reader only"})
		return
	}

	mu.Lock()
	defer mu.Unlock()

	out := []any{}
	for _, l := range db.Loans {
		if l.ReaderID != u.ID {
			continue
		}
		book, _ := findBookByID(l.BookID)
		out = append(out, map[string]any{
			"id": l.ID, "status": l.Status, "loanDate": l.LoanDate, "dueDate": l.DueDate, "returnDate": l.ReturnDate, "fineAmount": l.FineAmount,
			"book": book,
		})
	}
	jsonWrite(w, 200, out)
}

func method(w http.ResponseWriter, r *http.Request, m string) bool {
	if r.Method != m {
		jsonWrite(w, 405, map[string]any{"error": "Method not allowed"})
		return false
	}
	return true
}

func booksHandler(w http.ResponseWriter, r *http.Request) {

	if r.URL.Path == "/api/books" {
		if r.Method == "GET" {
			requireAuth(apiGetBooks)(w, r)
			return
		}
		if r.Method == "POST" {
			requireAdmin(apiCreateBook)(w, r)
			return
		}
		jsonWrite(w, 405, map[string]any{"error": "Method not allowed"})
		return
	}

	if strings.HasPrefix(r.URL.Path, "/api/books/") {
		if r.Method == "PATCH" {
			requireAdmin(apiUpdateBook)(w, r)
			return
		}
		if r.Method == "DELETE" {
			requireAdmin(apiDeleteBook)(w, r)
			return
		}
		jsonWrite(w, 405, map[string]any{"error": "Method not allowed"})
		return
	}

	jsonWrite(w, 404, map[string]any{"error": "Not found"})
}

func main() {
	loadDB()

	publicDir := filepath.Join(".", "public")
	fs := http.FileServer(http.Dir(publicDir))
	http.Handle("/", fs)

	http.HandleFunc("/api/meta", func(w http.ResponseWriter, r *http.Request) {
		if !method(w, r, "GET") {
			return
		}
		apiMeta(w, r)
	})

	http.HandleFunc("/api/auth/register", func(w http.ResponseWriter, r *http.Request) {
		if !method(w, r, "POST") {
			return
		}
		apiRegister(w, r)
	})
	http.HandleFunc("/api/auth/login", func(w http.ResponseWriter, r *http.Request) {
		if !method(w, r, "POST") {
			return
		}
		apiLogin(w, r)
	})
	http.HandleFunc("/api/me", requireAuth(apiMe))

	http.HandleFunc("/api/users", requireAdmin(func(w http.ResponseWriter, r *http.Request) {
		if !method(w, r, "GET") {
			return
		}
		apiListUsers(w, r)
	}))

	http.HandleFunc("/api/books", booksHandler)
	http.HandleFunc("/api/books/", booksHandler)

	http.HandleFunc("/api/loans", requireAdmin(func(w http.ResponseWriter, r *http.Request) {
		if !method(w, r, "GET") {
			return
		}
		apiListLoans(w, r)
	}))
	http.HandleFunc("/api/loans/borrow", requireAdmin(func(w http.ResponseWriter, r *http.Request) {
		if !method(w, r, "POST") {
			return
		}
		apiBorrow(w, r)
	}))
	http.HandleFunc("/api/loans/return", requireAdmin(func(w http.ResponseWriter, r *http.Request) {
		if !method(w, r, "POST") {
			return
		}
		apiReturn(w, r)
	}))
	http.HandleFunc("/api/loans/lost", requireAdmin(func(w http.ResponseWriter, r *http.Request) {
		if !method(w, r, "POST") {
			return
		}
		apiLost(w, r)
	}))
	http.HandleFunc("/api/myloans", requireAuth(func(w http.ResponseWriter, r *http.Request) {
		if !method(w, r, "GET") {
			return
		}
		apiMyLoans(w, r)
	}))

	port := 8080
	if p := os.Getenv("PORT"); p != "" {
		if n, err := strconv.Atoi(p); err == nil {
			port = n
		}
	}

	fmt.Printf("✅ Library app running: http://localhost:%d\n", port)
	_ = http.ListenAndServe(fmt.Sprintf(":%d", port), nil)
}
