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

package zipcsv

import (
	"archive/zip"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/dolthub/go-mysql-server/sql"

	"github.com/dolthub/dolt/go/libraries/doltcore/row"
	"github.com/dolthub/dolt/go/libraries/doltcore/schema"
	"github.com/dolthub/dolt/go/libraries/doltcore/table"
	"github.com/dolthub/dolt/go/libraries/doltcore/table/untyped/csv"
	"github.com/dolthub/dolt/go/libraries/utils/filesys"
	"github.com/dolthub/dolt/go/store/types"
)

// GTFSRequiredFiles are the files that must be present for a valid GTFS feed
var GTFSRequiredFiles = []string{
	"agency.txt",
	"stops.txt",
	"routes.txt",
	"trips.txt",
	"stop_times.txt",
}

// ZipCsvReader reads CSV files from within a ZIP archive
type ZipCsvReader struct {
	zipReader     *zip.Reader
	zipFile       *os.File
	csvFiles      []*zip.File
	currentIdx    int
	currentReader table.SqlRowReader
	nbf           *types.NomsBinFormat
	csvInfo       *csv.CSVFileInfo
	isGTFS        bool
}

// ZipCsvWriter writes CSV files to a ZIP archive
type ZipCsvWriter struct {
	zipWriter  *zip.Writer
	csvWriter  table.SqlRowWriter
	fileWriter io.Writer
	closer     io.Closer
	sch        schema.Schema
}

// OpenZipCsvReader opens a ZIP file and creates a reader for CSV files within it
func OpenZipCsvReader(nbf *types.NomsBinFormat, zipPath string, fs filesys.ReadableFS, csvInfo *csv.CSVFileInfo) (*ZipCsvReader, error) {
	// We need to open the file with os.Open to get ReadAt support
	// First get the absolute path through the filesystem
	absPath, err := fs.Abs(zipPath)
	if err != nil {
		return nil, fmt.Errorf("failed to get absolute path: %v", err)
	}

	// Open the ZIP file
	file, err := os.Open(absPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open ZIP file: %v", err)
	}

	// Get file info for size
	stat, err := file.Stat()
	if err != nil {
		file.Close()
		return nil, fmt.Errorf("failed to stat ZIP file: %v", err)
	}

	// Create ZIP reader
	zipReader, err := zip.NewReader(file, stat.Size())
	if err != nil {
		file.Close()
		return nil, fmt.Errorf("failed to create ZIP reader: %v", err)
	}

	// Filter to get CSV/TXT files
	csvFiles, isGTFS := filterAndDetectFormat(zipReader.File)
	if len(csvFiles) == 0 {
		file.Close()
		return nil, fmt.Errorf("no CSV or TXT files found in ZIP archive")
	}

	reader := &ZipCsvReader{
		zipReader:  zipReader,
		zipFile:    file,
		csvFiles:   csvFiles,
		currentIdx: -1,
		nbf:        nbf,
		csvInfo:    csvInfo,
		isGTFS:     isGTFS,
	}

	// Start reading from first file
	err = reader.nextFile()
	if err != nil {
		reader.Close(context.Background())
		return nil, err
	}

	return reader, nil
}

// filterAndDetectFormat returns CSV or TXT files from the ZIP and detects if it's GTFS
func filterAndDetectFormat(files []*zip.File) ([]*zip.File, bool) {
	var txtFiles []*zip.File
	var csvFiles []*zip.File

	// Separate TXT and CSV files
	for _, file := range files {
		if file.FileInfo().IsDir() {
			continue
		}

		name := strings.ToLower(filepath.Base(file.Name))
		ext := filepath.Ext(name)

		if ext == ".txt" {
			txtFiles = append(txtFiles, file)
		} else if ext == ".csv" {
			csvFiles = append(csvFiles, file)
		}
	}

	// Check if this looks like a GTFS feed (has required TXT files)
	isGTFS := checkGTFS(txtFiles)
	if isGTFS {
		return txtFiles, true
	}

	return csvFiles, false
}

// checkGTFS checks if the TXT files look like a GTFS feed
func checkGTFS(txtFiles []*zip.File) bool {
	if len(txtFiles) == 0 {
		return false
	}

	fileNames := make(map[string]bool)
	for _, file := range txtFiles {
		name := strings.ToLower(filepath.Base(file.Name))
		fileNames[name] = true
	}

	// Check if all required GTFS files are present
	for _, required := range GTFSRequiredFiles {
		if !fileNames[required] {
			return false
		}
	}

	return true
}

// nextFile advances to the next CSV file in the ZIP
func (r *ZipCsvReader) nextFile() error {
	// Close current reader
	if r.currentReader != nil {
		r.currentReader.Close(context.Background())
		r.currentReader = nil
	}

	r.currentIdx++
	if r.currentIdx >= len(r.csvFiles) {
		return io.EOF
	}

	// Open the next file
	file := r.csvFiles[r.currentIdx]
	fileReader, err := file.Open()
	if err != nil {
		return fmt.Errorf("failed to open file %s in ZIP: %v", file.Name, err)
	}

	// Create CSV reader for this file
	r.currentReader, err = csv.NewCSVReader(r.nbf, fileReader, r.csvInfo)
	if err != nil {
		fileReader.Close()
		return fmt.Errorf("failed to create CSV reader for %s: %v", file.Name, err)
	}

	return nil
}

// ReadRow reads the next row from the current file (implements table.Reader)
func (r *ZipCsvReader) ReadRow(ctx context.Context) (row.Row, error) {
	for {
		if r.currentReader == nil {
			return nil, io.EOF
		}

		row, err := r.currentReader.ReadRow(ctx)
		if err == io.EOF {
			// Try next file
			err = r.nextFile()
			if err != nil {
				return nil, err
			}
			continue
		}

		return row, err
	}
}

// ReadSqlRow reads the next row from the current file (implements table.SqlRowReader)
func (r *ZipCsvReader) ReadSqlRow(ctx context.Context) (sql.Row, error) {
	for {
		if r.currentReader == nil {
			return nil, io.EOF
		}

		row, err := r.currentReader.ReadSqlRow(ctx)
		if err == io.EOF {
			// Try next file
			err = r.nextFile()
			if err != nil {
				return nil, err
			}
			continue
		}

		return row, err
	}
}

// GetSchema returns the schema (implements table.Reader)
func (r *ZipCsvReader) GetSchema() schema.Schema {
	if r.currentReader != nil {
		return r.currentReader.GetSchema()
	}
	return nil
}

// Close closes the reader (implements table.Closer)
func (r *ZipCsvReader) Close(ctx context.Context) error {
	if r.currentReader != nil {
		r.currentReader.Close(ctx)
	}
	if r.zipFile != nil {
		return r.zipFile.Close()
	}
	return nil
}

// nopWriteCloser wraps an io.Writer to implement io.WriteCloser
type nopWriteCloser struct {
	io.Writer
}

func (nwc nopWriteCloser) Close() error {
	return nil
}

// NewZipCsvWriter creates a writer that outputs CSV files to a ZIP archive
func NewZipCsvWriter(writer io.WriteCloser, tableName string, sch schema.Schema, csvInfo *csv.CSVFileInfo, useGTFS bool) (*ZipCsvWriter, error) {
	zipWriter := zip.NewWriter(writer)

	// Determine filename extension
	ext := ".csv"
	if useGTFS {
		ext = ".txt"
	}

	filename := tableName + ext

	// Create file in ZIP
	fileWriter, err := zipWriter.Create(filename)
	if err != nil {
		writer.Close()
		return nil, fmt.Errorf("failed to create file %s in ZIP: %v", filename, err)
	}

	// Wrap fileWriter to implement WriteCloser
	writeCloser := nopWriteCloser{fileWriter}

	// Create CSV writer
	csvWriter, err := csv.NewCSVWriter(writeCloser, sch, csvInfo)
	if err != nil {
		writer.Close()
		return nil, fmt.Errorf("failed to create CSV writer: %v", err)
	}

	return &ZipCsvWriter{
		zipWriter:  zipWriter,
		csvWriter:  csvWriter,
		fileWriter: fileWriter,
		closer:     writer,
		sch:        sch,
	}, nil
}

// WriteSqlRow writes a row to the CSV file in the ZIP (implements table.SqlRowWriter)
func (w *ZipCsvWriter) WriteSqlRow(ctx *sql.Context, row sql.Row) error {
	return w.csvWriter.WriteSqlRow(ctx, row)
}

// GetSchema returns the schema
func (w *ZipCsvWriter) GetSchema() schema.Schema {
	return w.sch
}

// Close closes the writer and finalizes the ZIP file (implements table.Closer)
func (w *ZipCsvWriter) Close(ctx context.Context) error {
	var err error

	if w.csvWriter != nil {
		err = w.csvWriter.Close(ctx)
		if err != nil {
			w.closer.Close()
			return err
		}
	}

	if w.zipWriter != nil {
		err = w.zipWriter.Close()
		if err != nil {
			w.closer.Close()
			return err
		}
	}

	return w.closer.Close()
}
