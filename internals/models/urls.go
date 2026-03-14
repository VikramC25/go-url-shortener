package models

import (
	"database/sql"
	"errors"
)

var ErrNoRecord = errors.New("model: no matching record found")

type ShortenerData struct {
	OriginalURL, ShortenedURL 	string
	Clicks						int
}

type ShortenerDataModel struct {
	DB *sql.DB
}

func (m *ShortenerDataModel) Latest() ([]*ShortenerData, error) {
	stmt := `SELECT original_url, shortened_url, clicks FROM urls`
	rows, err := m.DB.Query(stmt)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	urls := []*ShortenerData{}
	for rows.Next() {
		url := &ShortenerData{}
		err := rows.Scan(&url.OriginalURL, &url.ShortenedURL, &url.Clicks)
		if err != nil {
			return nil, err
		}
		urls = append(urls, url)
	}
	if err = rows.Err(); err != nil {
		return nil, err
	}

	return urls, nil
}

func (m *ShortenerDataModel) Insert(original string, shortened string, clicks int) (int, error) {
	stmt := `INSERT INTO urls (original_url, shortened_url, clicks) VALUES($1, $2, $3)`
	result, err := m.DB.Exec(stmt, original, shortened, clicks)
	if err != nil {
		return 0, err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return 0, err
	}

	return int(rowsAffected), nil
}

func (m *ShortenerDataModel) Get(shortened string) (string, error) {
	stmt := `SELECT original_url FROM urls WHERE shortened_url = $1`
	var originalURL string
	row := m.DB.QueryRow(stmt, shortened)
	err := row.Scan(&originalURL)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", ErrNoRecord
		} else {
			return "", err
		}
	}
	return originalURL, nil
}

func (m *ShortenerDataModel) IncrementClicks(shortened string) error {
	stmt := `UPDATE urls SET clicks = clicks + 1, updated = CURRENT_TIMESTAMP WHERE shortened_url = $1`
	_, err := m.DB.Exec(stmt, shortened)
	if err != nil {
		return err
	}

	return nil
}