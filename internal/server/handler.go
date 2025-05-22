package server

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
)

const (
	defaultRandomCount = 3
	maxRandomCount     = 10
)

type Handler struct {
	DB *sql.DB
}

type Book struct {
	ID            int64  `json:"id"`
	ISBN          string `json:"isbn"`
	Title         string `json:"title"`
	Subtitle      string `json:"subtitle"`
	Authors       string `json:"authors"`
	Publisher     string `json:"publisher"`
	PublishedDate string `json:"publishedDate"`
	Description   string `json:"description"`
	BookURL       string `json:"bookUrl"`
	ImageURL      string `json:"imageUrl"`
}

func NewHandler(db *sql.DB) *Handler {
	return &Handler{DB: db}
}

func (h *Handler) RandomBooks(w http.ResponseWriter, r *http.Request) {
	count := defaultRandomCount
	if q := r.URL.Query().Get("count"); q != "" {
		n, err := strconv.Atoi(q)
		if err != nil || n < 1 || n > maxRandomCount {
			writeJSONError(w, http.StatusBadRequest,
				fmt.Sprintf("invalid count parameter: must be integer between 1 and %d", maxRandomCount))
			return
		}
		count = n
	}

	const query = `
		SELECT
			id,
			isbn,
			title,
			subtitle,
			authors,
			publisher,
			published_date,
			description,
			book_url,
			image_url
		FROM books
		ORDER BY RANDOM()
		LIMIT $1
	`
	rows, err := h.DB.Query(query, count)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, err.Error())
		return
	}
	defer rows.Close()

	books := make([]Book, 0, count)
	for rows.Next() {
		var b Book
		err := rows.Scan(
			&b.ID,
			&b.ISBN,
			&b.Title,
			&b.Subtitle,
			&b.Authors,
			&b.Publisher,
			&b.PublishedDate,
			&b.Description,
			&b.BookURL,
			&b.ImageURL,
		)
		if err != nil {
			writeJSONError(w, http.StatusInternalServerError, err.Error())
			return
		}
		books = append(books, b)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(books)
}
