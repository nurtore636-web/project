package internal

import "time"

type Book struct {
    ID        int
    Title     string
    Author    string
    Available bool
}

type Reader struct {
    ID       int
    FullName string
    Contacts string
}

type Loan struct {
    ID         int
    BookID     int
    ReaderID   int
    DueAt      time.Time
    ReturnedAt *time.Time
}
