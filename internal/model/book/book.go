package book

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/lib/pq"
)

type Book struct {
	ISBN          string
	Title         string
	Subtitle      string
	Authors       []string
	Publisher     string
	PublishedDate string
	Description   string
	InfoLink      string
	ImageURL      string
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

func NewBook(isbn, title, subtitle string, authors []string, publisher, publishedDate, description, infoLink, imageURL string) *Book {
	return &Book{
		ISBN:          isbn,
		Title:         title,
		Subtitle:      subtitle,
		Authors:       authors,
		Publisher:     publisher,
		PublishedDate: publishedDate,
		Description:   description,
		InfoLink:      infoLink,
		ImageURL:      imageURL,
	}
}

func DeleteBefore(ctx context.Context, db *sql.DB, cutoff time.Time) (int64, error) {
	query := `DELETE FROM books WHERE updated_at < $1`

	res, err := db.ExecContext(ctx, query, cutoff)
	if err != nil {
		return 0, fmt.Errorf("DeleteBefore ExecContext エラー: %w", err)
	}

	n, err := res.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("DeleteBefore RowsAffected エラー: %w", err)
	}
	return n, nil
}

func (b *Book) Insert(ctx context.Context, db *sql.DB) error {
	authors := strings.Join(b.Authors, ", ")

	query := `
		INSERT INTO books
			(isbn, title, subtitle, authors, publisher, published_date, description, book_url, image_url, created_at, updated_at)
		VALUES
			($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		ON CONFLICT (isbn) DO UPDATE
        	SET updated_at = EXCLUDED.updated_at
	`

	now := time.Now()
	if b.CreatedAt.IsZero() {
		b.CreatedAt = now
	}
	b.UpdatedAt = now

	_, err := db.ExecContext(ctx, query,
		b.ISBN,
		b.Title,
		b.Subtitle,
		authors,
		b.Publisher,
		b.PublishedDate,
		b.Description,
		b.InfoLink,
		b.ImageURL,
		b.CreatedAt,
		b.UpdatedAt,
	)

	if err != nil {
		return fmt.Errorf("Book.Insert ExecContext エラー: %w", err)
	}

	return nil
}

func BulkInsert(ctx context.Context, tx *sql.Tx, books []*Book) (int, error) {
	stmt, err := tx.PrepareContext(ctx, pq.CopyIn(
		"books",
		"isbn",
		"title",
		"subtitle",
		"authors",
		"publisher",
		"published_date",
		"description",
		"book_url",
		"image_url",
		"created_at",
		"updated_at"))
	if err != nil {
		return 0, fmt.Errorf("COPY文準備エラー: %w", err)
	}
	defer stmt.Close()

	now := time.Now()

	for _, b := range books {
		if b.CreatedAt.IsZero() {
			b.CreatedAt = now
		}
		b.UpdatedAt = now

		authors := strings.Join(b.Authors, ", ")

		_, err = stmt.ExecContext(ctx,
			b.ISBN,
			b.Title,
			b.Subtitle,
			authors,
			b.Publisher,
			b.PublishedDate,
			b.Description,
			b.InfoLink,
			b.ImageURL,
			b.CreatedAt,
			b.UpdatedAt,
		)
		if err != nil {
			return 0, fmt.Errorf("COPY挿入エラー (ISBN: %s): %w", b.ISBN, err)
		}
	}

	res, err := stmt.ExecContext(ctx)
	if err != nil {
		return 0, fmt.Errorf("COPY完了処理エラー: %w", err)
	}

	count64, err := res.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("挿入件数取得エラー: %w", err)
	}

	return int(count64), nil
}
