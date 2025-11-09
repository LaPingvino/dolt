// Copyright 2024 Dolthub, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package bundle

import (
	"compress/gzip"
	"context"
	stdsql "database/sql"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/dolthub/go-mysql-server/sql"
	_ "modernc.org/sqlite"

	"github.com/dolthub/dolt/go/libraries/doltcore/row"
	"github.com/dolthub/dolt/go/libraries/doltcore/schema"
)

const (
	BundleFormatVersion = "1.0"
	DefaultBundleExt    = ".bundle"
)

// Bundle SQLite schema
const createBundleSchema = `
CREATE TABLE IF NOT EXISTS bundle_metadata (
    key TEXT PRIMARY KEY,
    value TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS dolt_data (
    path TEXT PRIMARY KEY,
    data BLOB NOT NULL,
    compressed INTEGER NOT NULL DEFAULT 1
);

CREATE TABLE IF NOT EXISTS table_schemas (
    table_name TEXT PRIMARY KEY,
    schema_json TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS table_data (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    table_name TEXT NOT NULL,
    row_data TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_table_data_table ON table_data(table_name);
`

// BundleInfo contains metadata about a bundle
type BundleInfo struct {
	FormatVersion string
	CreatedAt     time.Time
	Creator       string
	Description   string
	RepoRoot      string
	Branch        string
	CommitHash    string
}

// BundleReader reads data from a Dolt bundle file
type BundleReader struct {
	db           *stdsql.DB
	bundlePath   string
	info         *BundleInfo
	tables       []string
	currentTable string
	currentRows  *stdsql.Rows
	sch          schema.Schema
}

// BundleWriter writes data to a Dolt bundle file
type BundleWriter struct {
	db         *stdsql.DB
	bundlePath string
	info       *BundleInfo
	tx         *stdsql.Tx
	sch        schema.Schema
	tableName  string
	stmt       *stdsql.Stmt
}

// OpenBundleReader opens a bundle file for reading
func OpenBundleReader(bundlePath string) (*BundleReader, error) {
	if _, err := os.Stat(bundlePath); os.IsNotExist(err) {
		return nil, fmt.Errorf("bundle file does not exist: %s", bundlePath)
	}

	db, err := stdsql.Open("sqlite", bundlePath+"?mode=ro")
	if err != nil {
		return nil, fmt.Errorf("failed to open bundle database: %v", err)
	}

	reader := &BundleReader{
		db:         db,
		bundlePath: bundlePath,
	}

	// Load bundle metadata
	if err := reader.loadMetadata(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to load bundle metadata: %v", err)
	}

	// Load table list
	if err := reader.loadTables(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to load table list: %v", err)
	}

	return reader, nil
}

// CreateBundle creates a new bundle from a Dolt repository
func CreateBundle(ctx context.Context, bundlePath, repoRoot string, info *BundleInfo) error {
	// Remove existing bundle file
	os.Remove(bundlePath)

	db, err := stdsql.Open("sqlite", bundlePath)
	if err != nil {
		return fmt.Errorf("failed to create bundle database: %v", err)
	}
	defer db.Close()

	// Create schema
	if _, err := db.Exec(createBundleSchema); err != nil {
		return fmt.Errorf("failed to create bundle schema: %v", err)
	}

	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %v", err)
	}
	defer tx.Rollback()

	writer := &BundleWriter{
		db:         db,
		bundlePath: bundlePath,
		info:       info,
		tx:         tx,
	}

	// Write metadata
	if err := writer.writeMetadata(); err != nil {
		return fmt.Errorf("failed to write metadata: %v", err)
	}

	// Archive and store .dolt directory
	if err := writer.storeDoltData(repoRoot); err != nil {
		return fmt.Errorf("failed to store dolt data: %v", err)
	}

	return tx.Commit()
}

// ExtractBundle extracts a bundle to create a new Dolt repository
func ExtractBundle(ctx context.Context, bundlePath, targetDir string) error {
	reader, err := OpenBundleReader(bundlePath)
	if err != nil {
		return fmt.Errorf("failed to open bundle: %v", err)
	}
	defer reader.Close(ctx)

	// Create target directory
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		return fmt.Errorf("failed to create target directory: %v", err)
	}

	// Extract .dolt directory
	if err := reader.extractDoltData(targetDir); err != nil {
		return fmt.Errorf("failed to extract dolt data: %v", err)
	}

	return nil
}

func (r *BundleReader) loadMetadata() error {
	r.info = &BundleInfo{}

	rows, err := r.db.Query("SELECT key, value FROM bundle_metadata")
	if err != nil {
		return err
	}
	defer rows.Close()

	metadata := make(map[string]string)
	for rows.Next() {
		var key, value string
		if err := rows.Scan(&key, &value); err != nil {
			return err
		}
		metadata[key] = value
	}

	r.info.FormatVersion = metadata["format_version"]
	r.info.Creator = metadata["creator"]
	r.info.Description = metadata["description"]
	r.info.RepoRoot = metadata["repo_root"]
	r.info.Branch = metadata["branch"]
	r.info.CommitHash = metadata["commit_hash"]

	if createdAtStr := metadata["created_at"]; createdAtStr != "" {
		if createdAt, err := time.Parse(time.RFC3339, createdAtStr); err == nil {
			r.info.CreatedAt = createdAt
		}
	}

	return nil
}

func (r *BundleReader) loadTables() error {
	rows, err := r.db.Query("SELECT table_name FROM table_schemas ORDER BY table_name")
	if err != nil {
		return err
	}
	defer rows.Close()

	r.tables = []string{}
	for rows.Next() {
		var tableName string
		if err := rows.Scan(&tableName); err != nil {
			return err
		}
		r.tables = append(r.tables, tableName)
	}

	return nil
}

func (r *BundleReader) extractDoltData(targetDir string) error {
	rows, err := r.db.Query("SELECT path, data, compressed FROM dolt_data ORDER BY path")
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var path string
		var data []byte
		var compressed int

		if err := rows.Scan(&path, &data, &compressed); err != nil {
			return err
		}

		targetPath := filepath.Join(targetDir, path)
		if err := os.MkdirAll(filepath.Dir(targetPath), 0755); err != nil {
			return err
		}

		if compressed == 1 {
			// Decompress the data
			data, err = decompressData(data)
			if err != nil {
				return fmt.Errorf("failed to decompress data for %s: %v", path, err)
			}
		}

		if err := os.WriteFile(targetPath, data, 0644); err != nil {
			return fmt.Errorf("failed to write file %s: %v", targetPath, err)
		}
	}

	return nil
}

func (w *BundleWriter) writeMetadata() error {
	metadata := map[string]string{
		"format_version": BundleFormatVersion,
		"created_at":     w.info.CreatedAt.Format(time.RFC3339),
		"creator":        w.info.Creator,
		"description":    w.info.Description,
		"repo_root":      w.info.RepoRoot,
		"branch":         w.info.Branch,
		"commit_hash":    w.info.CommitHash,
	}

	stmt, err := w.tx.Prepare("INSERT INTO bundle_metadata (key, value) VALUES (?, ?)")
	if err != nil {
		return err
	}
	defer stmt.Close()

	for key, value := range metadata {
		if _, err := stmt.Exec(key, value); err != nil {
			return err
		}
	}

	return nil
}

func (w *BundleWriter) storeDoltData(repoRoot string) error {
	doltDir := filepath.Join(repoRoot, ".dolt")
	if _, err := os.Stat(doltDir); os.IsNotExist(err) {
		return fmt.Errorf(".dolt directory not found in %s", repoRoot)
	}

	stmt, err := w.tx.Prepare("INSERT INTO dolt_data (path, data, compressed) VALUES (?, ?, ?)")
	if err != nil {
		return err
	}
	defer stmt.Close()

	return filepath.Walk(doltDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		// Get relative path from repo root
		relPath, err := filepath.Rel(repoRoot, path)
		if err != nil {
			return err
		}

		// Read file
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}

		// Compress data
		compressedData, err := compressData(data)
		if err != nil {
			return fmt.Errorf("failed to compress %s: %v", relPath, err)
		}

		// Store in database
		_, err = stmt.Exec(relPath, compressedData, 1)
		return err
	})
}

// Implement table.SqlRowReader interface
func (r *BundleReader) ReadSqlRow(ctx context.Context) (sql.Row, error) {
	if r.currentRows == nil {
		return nil, io.EOF
	}

	if !r.currentRows.Next() {
		r.currentRows.Close()
		r.currentRows = nil
		return nil, io.EOF
	}

	var tableName, rowData string
	if err := r.currentRows.Scan(&tableName, &rowData); err != nil {
		return nil, err
	}

	// Parse row data (simplified - in reality would need proper deserialization)
	parts := strings.Split(rowData, "|")
	row := make(sql.Row, len(parts))
	for i, part := range parts {
		row[i] = part
	}

	return row, nil
}

func (r *BundleReader) ReadRow(ctx context.Context) (row.Row, error) {
	// This would need to implement the proper row.Row interface
	// For now, return an error indicating this is not implemented
	return nil, fmt.Errorf("ReadRow not implemented for BundleReader")
}

func (r *BundleReader) GetSchema() schema.Schema {
	return r.sch
}

func (r *BundleReader) Close(ctx context.Context) error {
	if r.currentRows != nil {
		r.currentRows.Close()
	}
	if r.db != nil {
		return r.db.Close()
	}
	return nil
}

// Implement table.SqlRowWriter interface
func (w *BundleWriter) WriteSqlRow(ctx *sql.Context, row sql.Row) error {
	if w.stmt == nil {
		return fmt.Errorf("bundle writer not properly initialized")
	}

	// Serialize row data (simplified - in reality would need proper serialization)
	parts := make([]string, len(row))
	for i, val := range row {
		if val == nil {
			parts[i] = ""
		} else {
			parts[i] = fmt.Sprintf("%v", val)
		}
	}
	rowData := strings.Join(parts, "|")

	_, err := w.stmt.Exec(w.tableName, rowData)
	return err
}

func (w *BundleWriter) Close(ctx context.Context) error {
	if w.stmt != nil {
		w.stmt.Close()
	}
	if w.tx != nil {
		w.tx.Rollback()
	}
	if w.db != nil {
		return w.db.Close()
	}
	return nil
}

// Helper functions for compression
func compressData(data []byte) ([]byte, error) {
	var compressed strings.Builder
	writer := gzip.NewWriter(&compressed)

	if _, err := writer.Write(data); err != nil {
		return nil, err
	}

	if err := writer.Close(); err != nil {
		return nil, err
	}

	return []byte(compressed.String()), nil
}

func decompressData(data []byte) ([]byte, error) {
	reader, err := gzip.NewReader(strings.NewReader(string(data)))
	if err != nil {
		return nil, err
	}
	defer reader.Close()

	return io.ReadAll(reader)
}

// GetBundleInfo returns information about the bundle
func (r *BundleReader) GetBundleInfo() *BundleInfo {
	return r.info
}

// GetTables returns the list of tables in the bundle
func (r *BundleReader) GetTables() []string {
	return r.tables
}

// SetCurrentTable sets which table to read from
func (r *BundleReader) SetCurrentTable(tableName string) error {
	if r.currentRows != nil {
		r.currentRows.Close()
		r.currentRows = nil
	}

	rows, err := r.db.Query("SELECT table_name, row_data FROM table_data WHERE table_name = ? ORDER BY id", tableName)
	if err != nil {
		return err
	}

	r.currentTable = tableName
	r.currentRows = rows
	return nil
}

// NewBundleWriter creates a new bundle writer for a specific table
func NewBundleWriter(db *stdsql.DB, tableName string, sch schema.Schema) (*BundleWriter, error) {
	tx, err := db.Begin()
	if err != nil {
		return nil, err
	}

	stmt, err := tx.Prepare("INSERT INTO table_data (table_name, row_data) VALUES (?, ?)")
	if err != nil {
		tx.Rollback()
		return nil, err
	}

	return &BundleWriter{
		db:        db,
		tx:        tx,
		stmt:      stmt,
		tableName: tableName,
		sch:       sch,
	}, nil
}
