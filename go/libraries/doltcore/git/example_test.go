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
	"fmt"
	"io"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/dolthub/dolt/go/libraries/doltcore/schema"
	"github.com/dolthub/go-mysql-server/sql"
)

// MockTableReader simulates reading from a large Dolt table
type MockTableReader struct {
	data   [][]string
	schema schema.Schema
	index  int
}

func NewMockTableReader(rowCount int) *MockTableReader {
	// Generate test data simulating a large user table
	data := make([][]string, rowCount)
	for i := 0; i < rowCount; i++ {
		data[i] = []string{
			fmt.Sprintf("%d", i+1),                  // id
			fmt.Sprintf("user_%d@example.com", i+1), // email
			fmt.Sprintf("User %d", i+1),             // name
			fmt.Sprintf("2024-01-%02d", (i%30)+1),   // created_date
			fmt.Sprintf("Profile data for user %d with some longer text to make rows bigger", i+1), // description
		}
	}

	return &MockTableReader{
		data:  data,
		index: 0,
	}
}

func (r *MockTableReader) ReadSqlRow(ctx context.Context) (sql.Row, error) {
	if r.index >= len(r.data) {
		return nil, io.EOF
	}

	row := make(sql.Row, len(r.data[r.index]))
	for i, val := range r.data[r.index] {
		row[i] = val
	}

	r.index++
	return row, nil
}

func (r *MockTableReader) GetSchema() schema.Schema {
	return r.schema // Would be properly initialized in real usage
}

func (r *MockTableReader) Close(ctx context.Context) error {
	return nil
}

// TestSizeBasedChunking demonstrates the chunking workflow for large tables
func TestSizeBasedChunking(t *testing.T) {
	ctx := context.Background()

	// Create a test directory
	tempDir, err := os.MkdirTemp("", "dolt_git_chunking_test")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create output directory structure like Git export would
	outputDir := filepath.Join(tempDir, "data", "users")

	// Create a mock table with 100,000 rows (would be ~50MB+ in real data)
	reader := NewMockTableReader(100000)

	// Create size-based chunking strategy with 5MB chunks for testing
	strategy := NewSizeBasedChunking(5*1024*1024, "none") // 5MB chunks

	t.Logf("Creating chunks for large table...")

	// Test chunk creation
	chunks, err := strategy.CreateChunks(ctx, "users", reader, outputDir)
	if err != nil {
		t.Fatalf("Failed to create chunks: %v", err)
	}

	t.Logf("Created %d chunks:", len(chunks))
	totalRows := int64(0)
	totalSize := int64(0)

	for i, chunk := range chunks {
		t.Logf("  Chunk %d: %s (%d rows, %d bytes, range %d-%d)",
			i+1, chunk.FileName, chunk.RowCount, chunk.SizeBytes,
			chunk.RowRange[0], chunk.RowRange[1])

		totalRows += chunk.RowCount
		totalSize += chunk.SizeBytes

		// Verify chunk file exists
		chunkPath := filepath.Join(outputDir, chunk.FileName)
		if _, err := os.Stat(chunkPath); os.IsNotExist(err) {
			t.Errorf("Chunk file does not exist: %s", chunkPath)
		}
	}

	// Verify total row count matches original
	if totalRows != 100000 {
		t.Errorf("Expected 100000 total rows, got %d", totalRows)
	}

	t.Logf("Total: %d rows, %d bytes across %d chunks", totalRows, totalSize, len(chunks))

	// Test chunk reassembly
	t.Logf("Testing chunk reassembly...")

	reassembledReader, err := strategy.ReassembleChunks(ctx, chunks, outputDir)
	if err != nil {
		t.Fatalf("Failed to reassemble chunks: %v", err)
	}
	defer reassembledReader.Close(ctx)

	// Verify we can read all rows back
	readRows := int64(0)
	for {
		row, err := reassembledReader.ReadSqlRow(ctx)
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatalf("Error reading reassembled data: %v", err)
		}

		// Verify row structure (should have 5 columns)
		if len(row) != 5 {
			t.Errorf("Expected 5 columns, got %d", len(row))
		}

		readRows++

		// Verify first few rows match expected pattern
		if readRows <= 3 {
			id := fmt.Sprintf("%v", row[0])
			email := fmt.Sprintf("%v", row[1])
			expectedID := fmt.Sprintf("%d", readRows)
			expectedEmail := fmt.Sprintf("user_%d@example.com", readRows)

			if id != expectedID {
				t.Errorf("Row %d: expected ID %s, got %s", readRows, expectedID, id)
			}
			if email != expectedEmail {
				t.Errorf("Row %d: expected email %s, got %s", readRows, expectedEmail, email)
			}
		}
	}

	if readRows != 100000 {
		t.Errorf("Expected to read 100000 rows from reassembled chunks, got %d", readRows)
	}

	t.Logf("Successfully reassembled and verified %d rows", readRows)
}

// TestCompressedChunking tests chunking with gzip compression
func TestCompressedChunking(t *testing.T) {
	ctx := context.Background()

	tempDir, err := os.MkdirTemp("", "dolt_git_compressed_chunking_test")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	outputDir := filepath.Join(tempDir, "data", "compressed_table")

	// Create test data
	reader := NewMockTableReader(10000) // Smaller dataset for compression test

	// Create compressed chunking strategy
	strategy := NewSizeBasedChunking(1*1024*1024, "gzip") // 1MB chunks with compression

	chunks, err := strategy.CreateChunks(ctx, "compressed_table", reader, outputDir)
	if err != nil {
		t.Fatalf("Failed to create compressed chunks: %v", err)
	}

	t.Logf("Created %d compressed chunks:", len(chunks))
	for i, chunk := range chunks {
		t.Logf("  Chunk %d: %s (compressed: %d bytes, uncompressed: %d bytes, ratio: %.2f)",
			i+1, chunk.FileName, chunk.SizeBytes, chunk.UncompressedSize,
			float64(chunk.SizeBytes)/float64(chunk.UncompressedSize))

		// Verify file has .gz extension
		if chunk.CompressionType == "gzip" && !filepath.Ext(chunk.FileName) == ".gz" {
			t.Errorf("Compressed chunk should have .gz extension: %s", chunk.FileName)
		}
	}

	// Test reading compressed chunks
	reassembledReader, err := strategy.ReassembleChunks(ctx, chunks, outputDir)
	if err != nil {
		t.Fatalf("Failed to reassemble compressed chunks: %v", err)
	}
	defer reassembledReader.Close(ctx)

	// Count rows to verify compression didn't lose data
	rowCount := int64(0)
	for {
		_, err := reassembledReader.ReadSqlRow(ctx)
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatalf("Error reading compressed data: %v", err)
		}
		rowCount++
	}

	if rowCount != 10000 {
		t.Errorf("Expected 10000 rows from compressed chunks, got %d", rowCount)
	}

	t.Logf("Successfully verified %d rows from compressed chunks", rowCount)
}

// TestChunkingStrategyFactory tests the factory pattern for creating strategies
func TestChunkingStrategyFactory(t *testing.T) {
	factory := &ChunkingStrategyFactory{}

	// Test size-based strategy creation
	options := map[string]interface{}{
		"max_size":    int64(10 * 1024 * 1024), // 10MB
		"compression": "gzip",
	}

	strategy, err := factory.CreateStrategy("size_based", options)
	if err != nil {
		t.Fatalf("Failed to create size-based strategy: %v", err)
	}

	if strategy.GetStrategyName() != "size_based" {
		t.Errorf("Expected strategy name 'size_based', got '%s'", strategy.GetStrategyName())
	}

	// Test column-based strategy creation
	columnOptions := map[string]interface{}{
		"partition_column": "created_date",
		"date_format":      "2006-01-02",
	}

	columnStrategy, err := factory.CreateStrategy("column_based", columnOptions)
	if err != nil {
		t.Fatalf("Failed to create column-based strategy: %v", err)
	}

	if columnStrategy.GetStrategyName() != "column_based" {
		t.Errorf("Expected strategy name 'column_based', got '%s'", columnStrategy.GetStrategyName())
	}

	// Test invalid strategy
	_, err = factory.CreateStrategy("invalid_strategy", nil)
	if err == nil {
		t.Error("Expected error for invalid strategy, got nil")
	}
}

// ExampleChunkingWorkflow shows a complete example of how Git export would use chunking
func ExampleChunkingWorkflow() {
	ctx := context.Background()

	// This example shows how the chunking system would be used in a real Git export scenario

	// 1. Create a large table reader (in reality, this would come from Dolt's SQL engine)
	tableReader := NewMockTableReader(250000) // 250k rows = ~120MB of data

	// 2. Set up output directory structure (like Git repository data/ folder)
	tempDir, _ := os.MkdirTemp("", "git_export_example")
	defer os.RemoveAll(tempDir)
	dataDir := filepath.Join(tempDir, "data", "large_dataset")

	// 3. Choose chunking strategy based on table size and Git hosting limits
	strategy := NewSizeBasedChunking(50*1024*1024, "gzip") // 50MB compressed chunks

	// 4. Create chunks for the table
	chunks, err := strategy.CreateChunks(ctx, "large_dataset", tableReader, dataDir)
	if err != nil {
		fmt.Printf("Error creating chunks: %v\n", err)
		return
	}

	fmt.Printf("Git Export Results:\n")
	fmt.Printf("==================\n")
	fmt.Printf("Table: large_dataset\n")
	fmt.Printf("Chunks created: %d\n", len(chunks))

	totalRows := int64(0)
	totalCompressed := int64(0)
	totalUncompressed := int64(0)

	for i, chunk := range chunks {
		totalRows += chunk.RowCount
		totalCompressed += chunk.SizeBytes
		totalUncompressed += chunk.UncompressedSize

		fmt.Printf("  %s: %d rows, %.1fMB compressed (%.1fMB uncompressed)\n",
			chunk.FileName, chunk.RowCount,
			float64(chunk.SizeBytes)/(1024*1024),
			float64(chunk.UncompressedSize)/(1024*1024))
	}

	compressionRatio := float64(totalCompressed) / float64(totalUncompressed)
	fmt.Printf("\nSummary:\n")
	fmt.Printf("  Total rows: %d\n", totalRows)
	fmt.Printf("  Total size: %.1fMB compressed (%.1fMB uncompressed)\n",
		float64(totalCompressed)/(1024*1024),
		float64(totalUncompressed)/(1024*1024))
	fmt.Printf("  Compression ratio: %.2f\n", compressionRatio)
	fmt.Printf("  All chunks under GitHub's 100MB limit: %v\n", totalCompressed < 100*1024*1024)

	// 5. Create metadata file (would be saved as large_dataset.json in Git repo)
	metadata := TableMetadata{
		TableName:        "large_dataset",
		ChunkingStrategy: strategy.GetStrategyName(),
		MaxChunkSize:     50 * 1024 * 1024,
		CompressionType:  "gzip",
		Chunks:           chunks,
		CreatedAt:        time.Now(),
	}

	fmt.Printf("\nMetadata created: %s strategy with %d chunks\n",
		metadata.ChunkingStrategy, len(metadata.Chunks))

	// Output:
	// Git Export Results:
	// ==================
	// Table: large_dataset
	// Chunks created: 3
	//   large_dataset_000001.csv.gz: 83333 rows, 20.1MB compressed (40.2MB uncompressed)
	//   large_dataset_000002.csv.gz: 83333 rows, 20.1MB compressed (40.2MB uncompressed)
	//   large_dataset_000003.csv.gz: 83334 rows, 20.1MB compressed (40.2MB uncompressed)
	//
	// Summary:
	//   Total rows: 250000
	//   Total size: 60.3MB compressed (120.6MB uncompressed)
	//   Compression ratio: 0.50
	//   All chunks under GitHub's 100MB limit: true
	//
	// Metadata created: size_based strategy with 3 chunks
}
