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

package git

import (
	"context"
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/dolthub/dolt/go/libraries/doltcore/schema"
	"github.com/dolthub/go-mysql-server/sql"
)

const (
	DefaultMaxChunkSize = 50 * 1024 * 1024 // 50MB default chunk size
	ChunkFileFormat     = "%s_%06d.csv"    // table_000001.csv
)

// ChunkingStrategy defines how to split large tables for Git storage
type ChunkingStrategy interface {
	// ShouldChunk determines if a table needs chunking based on estimated size
	ShouldChunk(tableName string, estimatedSize int64) bool

	// CreateChunks splits table data into manageable chunks
	CreateChunks(ctx context.Context, tableName string, reader TableReader, outputDir string) ([]ChunkInfo, error)

	// ReassembleChunks combines chunks back into a single table reader
	ReassembleChunks(ctx context.Context, chunks []ChunkInfo, inputDir string) (TableReader, error)

	// GetStrategyName returns the name of the chunking strategy
	GetStrategyName() string
}

// ChunkInfo describes a single chunk of table data
type ChunkInfo struct {
	FileName  string            `json:"file_name"`
	RowCount  int64             `json:"row_count"`
	SizeBytes int64             `json:"size_bytes"`
	RowRange  [2]int64          `json:"row_range,omitempty"` // [start, end] for size-based
	Filter    string            `json:"filter,omitempty"`    // SQL WHERE clause for column-based
	Metadata  map[string]string `json:"metadata,omitempty"`
}

// TableMetadata contains information about a table's chunking
type TableMetadata struct {
	TableName        string      `json:"table_name"`
	ChunkingStrategy string      `json:"chunking_strategy"`
	MaxChunkSize     int64       `json:"max_chunk_size,omitempty"`
	PartitionColumn  string      `json:"partition_column,omitempty"`
	Chunks           []ChunkInfo `json:"chunks"`
	Schema           string      `json:"schema"` // Table schema as SQL DDL
	CreatedAt        time.Time   `json:"created_at"`
}

// TableReader interface for reading table data (compatible with existing Dolt readers)
type TableReader interface {
	ReadSqlRow(ctx context.Context) (sql.Row, error)
	GetSchema() schema.Schema
	Close(ctx context.Context) error
}

// SizeBasedChunking splits tables based on file size limits
type SizeBasedChunking struct {
	MaxChunkSize int64
}

// NewSizeBasedChunking creates a new size-based chunking strategy
func NewSizeBasedChunking(maxSize int64) *SizeBasedChunking {
	if maxSize <= 0 {
		maxSize = DefaultMaxChunkSize
	}
	return &SizeBasedChunking{
		MaxChunkSize: maxSize,
	}
}

func (s *SizeBasedChunking) GetStrategyName() string {
	return "size_based"
}

func (s *SizeBasedChunking) ShouldChunk(tableName string, estimatedSize int64) bool {
	return estimatedSize > s.MaxChunkSize
}

func (s *SizeBasedChunking) CreateChunks(ctx context.Context, tableName string, reader TableReader, outputDir string) ([]ChunkInfo, error) {
	var chunks []ChunkInfo
	chunkIndex := 1
	currentSize := int64(0)
	currentRows := int64(0)
	startRow := int64(1)

	// Ensure output directory exists
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create output directory: %v", err)
	}

	// Get table schema for CSV header
	sch := reader.GetSchema()
	headers := getSchemaHeaders(sch)

	// Create first chunk
	chunkWriter, chunkFile, err := s.createChunkWriter(tableName, chunkIndex, outputDir, headers)
	if err != nil {
		return nil, fmt.Errorf("failed to create first chunk writer: %v", err)
	}
	defer chunkFile.Close()

	for {
		sqlRow, err := reader.ReadSqlRow(ctx)
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("error reading row: %v", err)
		}

		// Convert SQL row to string slice
		rowData := sqlRowToStrings(sqlRow)

		// Estimate row size (approximate)
		rowSize := estimateRowSize(rowData)

		// Check if adding this row would exceed chunk size
		if currentSize+rowSize > s.MaxChunkSize && currentRows > 0 {
			// Finalize current chunk
			chunkInfo, err := s.finalizeChunk(tableName, chunkIndex, currentRows, currentSize, startRow, chunkFile)
			if err != nil {
				return nil, fmt.Errorf("failed to finalize chunk %d: %v", chunkIndex, err)
			}
			chunks = append(chunks, chunkInfo)

			// Start new chunk
			chunkIndex++
			startRow += currentRows
			currentSize = 0
			currentRows = 0

			chunkWriter, chunkFile, err = s.createChunkWriter(tableName, chunkIndex, outputDir, headers)
			if err != nil {
				return nil, fmt.Errorf("failed to create chunk writer %d: %v", chunkIndex, err)
			}
			defer chunkFile.Close()
		}

		// Write row to current chunk
		if err := chunkWriter.Write(rowData); err != nil {
			return nil, fmt.Errorf("failed to write row to chunk %d: %v", chunkIndex, err)
		}

		currentSize += rowSize
		currentRows++
	}

	// Handle final chunk if any data remains
	if currentRows > 0 {
		chunkInfo, err := s.finalizeChunk(tableName, chunkIndex, currentRows, currentSize, startRow, chunkFile)
		if err != nil {
			return nil, fmt.Errorf("failed to finalize final chunk: %v", err)
		}
		chunks = append(chunks, chunkInfo)
	}

	return chunks, nil
}

func (s *SizeBasedChunking) ReassembleChunks(ctx context.Context, chunks []ChunkInfo, inputDir string) (TableReader, error) {
	// Create a multi-chunk reader that can read from multiple CSV files
	return NewMultiChunkReader(chunks, inputDir)
}

func (s *SizeBasedChunking) createChunkWriter(tableName string, chunkIndex int, outputDir string, headers []string) (*csv.Writer, *os.File, error) {
	fileName := fmt.Sprintf(ChunkFileFormat, tableName, chunkIndex)
	filePath := filepath.Join(outputDir, fileName)
	file, err := os.Create(filePath)
	if err != nil {
		return nil, nil, err
	}

	csvWriter := csv.NewWriter(file)

	// Write CSV header
	if err := csvWriter.Write(headers); err != nil {
		file.Close()
		return nil, nil, fmt.Errorf("failed to write CSV header: %v", err)
	}
	csvWriter.Flush()

	return csvWriter, file, nil
}

func (s *SizeBasedChunking) finalizeChunk(tableName string, chunkIndex int, rowCount, sizeBytes, startRow int64, file *os.File) (ChunkInfo, error) {
	// Close and get final file size
	fileName := filepath.Base(file.Name())
	filePath := file.Name()
	file.Close()

	stat, err := os.Stat(filePath)
	if err != nil {
		return ChunkInfo{}, fmt.Errorf("failed to stat chunk file: %v", err)
	}

	chunk := ChunkInfo{
		FileName:  fileName,
		RowCount:  rowCount,
		SizeBytes: stat.Size(),
		RowRange:  [2]int64{startRow, startRow + rowCount - 1},
	}

	return chunk, nil
}

// ColumnBasedChunking splits tables based on column values (e.g., date ranges)
type ColumnBasedChunking struct {
	PartitionColumn string
	DateFormat      string
	MaxChunkSize    int64
}

// NewColumnBasedChunking creates a column-based chunking strategy
func NewColumnBasedChunking(column, dateFormat string) *ColumnBasedChunking {
	return &ColumnBasedChunking{
		PartitionColumn: column,
		DateFormat:      dateFormat,
		MaxChunkSize:    DefaultMaxChunkSize,
	}
}

func (c *ColumnBasedChunking) GetStrategyName() string {
	return "column_based"
}

func (c *ColumnBasedChunking) ShouldChunk(tableName string, estimatedSize int64) bool {
	// Always chunk if a partition column is specified
	return c.PartitionColumn != ""
}

func (c *ColumnBasedChunking) CreateChunks(ctx context.Context, tableName string, reader TableReader, outputDir string) ([]ChunkInfo, error) {
	// Column-based chunking implementation would require SQL query capabilities
	// For now, fall back to size-based chunking
	sizeChunking := NewSizeBasedChunking(c.MaxChunkSize, "none")
	return sizeChunking.CreateChunks(ctx, tableName, reader, outputDir)
}

func (c *ColumnBasedChunking) ReassembleChunks(ctx context.Context, chunks []ChunkInfo, inputDir string) (TableReader, error) {
	return NewMultiChunkReader(chunks, inputDir, "none")
}

// Helper functions

// getSchemaHeaders extracts column names from a schema for CSV header
func getSchemaHeaders(sch schema.Schema) []string {
	cols := sch.GetAllCols()
	headers := make([]string, len(cols.GetColumns()))
	for i, col := range cols.GetColumns() {
		headers[i] = col.Name
	}
	return headers
}

// sqlRowToStrings converts a SQL row to string slice for CSV writing
func sqlRowToStrings(sqlRow sql.Row) []string {
	rowData := make([]string, len(sqlRow))
	for i, val := range sqlRow {
		if val == nil {
			rowData[i] = ""
		} else {
			rowData[i] = fmt.Sprintf("%v", val)
		}
	}
	return rowData
}

// estimateRowSize estimates the size of a CSV row in bytes
func estimateRowSize(rowData []string) int64 {
	size := int64(0)
	for i, field := range rowData {
		size += int64(len(field))
		if i < len(rowData)-1 {
			size += 1 // comma separator
		}
	}
	size += 1 // newline
	return size
}

// MultiChunkReader reads from multiple CSV chunk files sequentially
type MultiChunkReader struct {
	chunks        []ChunkInfo
	inputDir      string
	currentIdx    int
	currentFile   *os.File
	currentReader *csv.Reader
	schema        schema.Schema
	headers       []string
}

// NewMultiChunkReader creates a reader that can read from multiple CSV chunks
func NewMultiChunkReader(chunks []ChunkInfo, inputDir string) (*MultiChunkReader, error) {
	if len(chunks) == 0 {
		return nil, fmt.Errorf("no chunks to read")
	}

	reader := &MultiChunkReader{
		chunks:     chunks,
		inputDir:   inputDir,
		currentIdx: 0,
	}

	// Open first chunk to read headers and determine schema
	if err := reader.openNextChunk(); err != nil {
		return nil, fmt.Errorf("failed to open first chunk: %v", err)
	}

	return reader, nil
}

func (r *MultiChunkReader) openNextChunk() error {
	// Close current file if open
	if r.currentFile != nil {
		r.currentFile.Close()
		r.currentFile = nil
		r.currentReader = nil
	}

	// Check if we have more chunks
	if r.currentIdx >= len(r.chunks) {
		return io.EOF
	}

	chunk := r.chunks[r.currentIdx]
	filePath := filepath.Join(r.inputDir, chunk.FileName)

	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("failed to open chunk file %s: %v", filePath, err)
	}

	csvReader := csv.NewReader(file)
	r.currentFile = file
	r.currentReader = csvReader

	// Read and store headers from first chunk
	if r.currentIdx == 0 {
		headers, err := csvReader.Read()
		if err != nil {
			file.Close()
			return fmt.Errorf("failed to read headers: %v", err)
		}
		r.headers = headers
	} else {
		// Skip headers in subsequent chunks
		_, err := csvReader.Read()
		if err != nil {
			file.Close()
			return fmt.Errorf("failed to skip headers in chunk %d: %v", r.currentIdx, err)
		}
	}

	r.currentIdx++
	return nil
}

func (r *MultiChunkReader) ReadSqlRow(ctx context.Context) (sql.Row, error) {
	for {
		if r.currentReader == nil {
			return nil, io.EOF
		}

		record, err := r.currentReader.Read()
		if err == io.EOF {
			// Try to open next chunk
			if err := r.openNextChunk(); err != nil {
				return nil, err // This will be io.EOF if no more chunks
			}
			continue // Try reading from new chunk
		}
		if err != nil {
			return nil, fmt.Errorf("error reading CSV record: %v", err)
		}

		// Convert string record to SQL row
		sqlRow := make(sql.Row, len(record))
		for i, field := range record {
			// For simplicity, store everything as strings
			// In a real implementation, would parse based on schema types
			if field == "" {
				sqlRow[i] = nil
			} else {
				sqlRow[i] = field
			}
		}

		return sqlRow, nil
	}
}

func (r *MultiChunkReader) GetSchema() schema.Schema {
	return r.schema
}

func (r *MultiChunkReader) Close(ctx context.Context) error {
	if r.currentFile != nil {
		return r.currentFile.Close()
	}
	return nil
}

// ChunkingStrategyFactory creates chunking strategies based on configuration
type ChunkingStrategyFactory struct{}

func (f *ChunkingStrategyFactory) CreateStrategy(strategyType string, options map[string]interface{}) (ChunkingStrategy, error) {
	switch strategyType {
	case "size_based":
		maxSize := DefaultMaxChunkSize
		if v, ok := options["max_size"].(int64); ok {
			maxSize = v
		}

		return NewSizeBasedChunking(maxSize), nil

	case "column_based":
		column, ok := options["partition_column"].(string)
		if !ok {
			return nil, fmt.Errorf("partition_column is required for column_based strategy")
		}

		dateFormat := "2006-01-02"
		if v, ok := options["date_format"].(string); ok {
			dateFormat = v
		}

		return NewColumnBasedChunking(column, dateFormat), nil

	default:
		return nil, fmt.Errorf("unknown chunking strategy: %s", strategyType)
	}
}
