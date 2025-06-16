package grpcserver

import (
	"context"
	"database/sql"
	"fmt"

	pb "github.com/taiki-umetsu/ndc007-bookpicker/api/v1"
)

const (
	defaultRandomCount = 3
	maxRandomCount     = 10
)

type BookServiceServer struct {
	pb.UnimplementedBookServiceServer
	DB *sql.DB
}

func NewBookServiceServer(db *sql.DB) *BookServiceServer {
	return &BookServiceServer{DB: db}
}

func (s *BookServiceServer) GetRandomBooks(ctx context.Context, req *pb.RandomBooksRequest) (*pb.RandomBooksResponse, error) {
	count := int(req.Count)
	if count <= 0 {
		count = defaultRandomCount
	}
	if count > maxRandomCount {
		return nil, fmt.Errorf("count must be between 1 and %d", maxRandomCount)
	}

	const query = `
		SELECT id, isbn, title, subtitle, authors, publisher,
		       published_date, description, book_url, image_url
		FROM books
		ORDER BY RANDOM()
		LIMIT $1
	`

	rows, err := s.DB.QueryContext(ctx, query, count)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var books []*pb.Book
	for rows.Next() {
		var b pb.Book
		if err := rows.Scan(
			&b.Id, &b.Isbn, &b.Title, &b.Subtitle,
			&b.Authors, &b.Publisher, &b.PublishedDate,
			&b.Description, &b.BookUrl, &b.ImageUrl,
		); err != nil {
			return nil, err
		}
		books = append(books, &b)
	}

	return &pb.RandomBooksResponse{Books: books}, nil
}
