// Code generated by sqlc. DO NOT EDIT.
// versions:
//   sqlc v1.27.0
// source: words.sql

package database

import (
	"context"

	"github.com/jackc/pgx/v5/pgtype"
)

const createWord = `-- name: CreateWord :one
INSERT INTO words (value, created_at)
VALUES ($1, CURRENT_TIMESTAMP)
RETURNING id, value, created_at
`

type CreateWordRow struct {
	ID        int64              `json:"id"`
	Value     string             `json:"value"`
	CreatedAt pgtype.Timestamptz `json:"created_at"`
}

func (q *Queries) CreateWord(ctx context.Context, value string) (CreateWordRow, error) {
	row := q.db.QueryRow(ctx, createWord, value)
	var i CreateWordRow
	err := row.Scan(&i.ID, &i.Value, &i.CreatedAt)
	return i, err
}

const createWordsBatch = `-- name: CreateWordsBatch :one
WITH new_batch AS (
    INSERT INTO word_batches (name)
    VALUES ($1)
    RETURNING id
)

INSERT INTO words (value, batch_id)
SELECT
    word_value,
    (SELECT id FROM new_batch)
FROM UNNEST($2::text []) AS word_value
RETURNING id, value, batch_id
`

type CreateWordsBatchParams struct {
	Name    string   `json:"name"`
	Column2 []string `json:"column_2"`
}

type CreateWordsBatchRow struct {
	ID      int64       `json:"id"`
	Value   string      `json:"value"`
	BatchID pgtype.Int8 `json:"batch_id"`
}

func (q *Queries) CreateWordsBatch(ctx context.Context, arg CreateWordsBatchParams) (CreateWordsBatchRow, error) {
	row := q.db.QueryRow(ctx, createWordsBatch, arg.Name, arg.Column2)
	var i CreateWordsBatchRow
	err := row.Scan(&i.ID, &i.Value, &i.BatchID)
	return i, err
}

const listWordBatches = `-- name: ListWordBatches :many
SELECT
    id,
    name,
    created_at
FROM word_batches
WHERE deleted_at IS NULL
ORDER BY created_at ASC
LIMIT $1 OFFSET $2
`

type ListWordBatchesParams struct {
	Limit  int32 `json:"limit"`
	Offset int32 `json:"offset"`
}

type ListWordBatchesRow struct {
	ID        int64              `json:"id"`
	Name      string             `json:"name"`
	CreatedAt pgtype.Timestamptz `json:"created_at"`
}

func (q *Queries) ListWordBatches(ctx context.Context, arg ListWordBatchesParams) ([]ListWordBatchesRow, error) {
	rows, err := q.db.Query(ctx, listWordBatches, arg.Limit, arg.Offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []ListWordBatchesRow
	for rows.Next() {
		var i ListWordBatchesRow
		if err := rows.Scan(&i.ID, &i.Name, &i.CreatedAt); err != nil {
			return nil, err
		}
		items = append(items, i)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

const listWordFrequencies = `-- name: ListWordFrequencies :many
SELECT
    words.value,
    COUNT(*) AS total
FROM words
WHERE words.deleted_at IS NULL
GROUP BY words.value
ORDER BY total ASC
LIMIT $1 OFFSET $2
`

type ListWordFrequenciesParams struct {
	Limit  int32 `json:"limit"`
	Offset int32 `json:"offset"`
}

type ListWordFrequenciesRow struct {
	Value string `json:"value"`
	Total int64  `json:"total"`
}

func (q *Queries) ListWordFrequencies(ctx context.Context, arg ListWordFrequenciesParams) ([]ListWordFrequenciesRow, error) {
	rows, err := q.db.Query(ctx, listWordFrequencies, arg.Limit, arg.Offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []ListWordFrequenciesRow
	for rows.Next() {
		var i ListWordFrequenciesRow
		if err := rows.Scan(&i.Value, &i.Total); err != nil {
			return nil, err
		}
		items = append(items, i)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

const listWordRankings = `-- name: ListWordRankings :many
SELECT
    words.value,
    ROW_NUMBER() OVER (ORDER BY COUNT(*) DESC) AS ranking
FROM words
WHERE words.deleted_at IS NULL
GROUP BY words.value
ORDER BY ranking ASC
LIMIT $1 OFFSET $2
`

type ListWordRankingsParams struct {
	Limit  int32 `json:"limit"`
	Offset int32 `json:"offset"`
}

type ListWordRankingsRow struct {
	Value   string `json:"value"`
	Ranking int64  `json:"ranking"`
}

func (q *Queries) ListWordRankings(ctx context.Context, arg ListWordRankingsParams) ([]ListWordRankingsRow, error) {
	rows, err := q.db.Query(ctx, listWordRankings, arg.Limit, arg.Offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []ListWordRankingsRow
	for rows.Next() {
		var i ListWordRankingsRow
		if err := rows.Scan(&i.Value, &i.Ranking); err != nil {
			return nil, err
		}
		items = append(items, i)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

const listWords = `-- name: ListWords :many
SELECT
    id,
    value,
    created_at
FROM words
WHERE deleted_at IS NULL
ORDER BY value ASC
LIMIT $1 OFFSET $2
`

type ListWordsParams struct {
	Limit  int32 `json:"limit"`
	Offset int32 `json:"offset"`
}

type ListWordsRow struct {
	ID        int64              `json:"id"`
	Value     string             `json:"value"`
	CreatedAt pgtype.Timestamptz `json:"created_at"`
}

func (q *Queries) ListWords(ctx context.Context, arg ListWordsParams) ([]ListWordsRow, error) {
	rows, err := q.db.Query(ctx, listWords, arg.Limit, arg.Offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []ListWordsRow
	for rows.Next() {
		var i ListWordsRow
		if err := rows.Scan(&i.ID, &i.Value, &i.CreatedAt); err != nil {
			return nil, err
		}
		items = append(items, i)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

const listWordsByBatchName = `-- name: ListWordsByBatchName :many
SELECT
    wb.name AS batch_name,
    w.value AS word_value
FROM word_batches AS wb
INNER JOIN words AS w ON wb.id = w.id
WHERE wb.name = $1 AND wb.deleted_at IS NULL
ORDER BY wb.created_at DESC
`

type ListWordsByBatchNameRow struct {
	BatchName string `json:"batch_name"`
	WordValue string `json:"word_value"`
}

func (q *Queries) ListWordsByBatchName(ctx context.Context, name string) ([]ListWordsByBatchNameRow, error) {
	rows, err := q.db.Query(ctx, listWordsByBatchName, name)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []ListWordsByBatchNameRow
	for rows.Next() {
		var i ListWordsByBatchNameRow
		if err := rows.Scan(&i.BatchName, &i.WordValue); err != nil {
			return nil, err
		}
		items = append(items, i)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}
