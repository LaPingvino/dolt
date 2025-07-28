// Copyright 2025 Dolthub, Inc.
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

package nbs

import (
	"fmt"
	"io"
	"os"

	"github.com/dolthub/dolt/go/libraries/utils/file"
	"github.com/dolthub/dolt/go/store/hash"
)

type prefixList interface {
	getNumChunks() int
	getPrefix(idx uint32) uint64
}

type slicePrefixList []uint64

var _ prefixList = slicePrefixList{}

func (s slicePrefixList) getNumChunks() int {
	return len(s)
}

func (s slicePrefixList) getPrefix(idx uint32) uint64 {
	return s[idx]
}

type archiveIndexReader interface {
	prefixList
	getSpanIndex(idx uint32) uint64
	getChunkRef(idx int) (dict, data uint32)
	getSuffix(idx uint64) suffix
	io.Closer
}

// mmapIndexReader lazily loads archive index data from a memory-mapped file.
type mmapIndexReader struct {
	data          *file.MmapData
	indexSize     uint64
	byteSpanCount uint32
	chunkCount    uint32

	// Byte offsets within the mapped region for each section
	spanIndexOffset int64
	prefixesOffset  int64
	chunkRefsOffset int64
	suffixesOffset  int64
}

// newMmapIndexReader creates a new memory-mapped index reader.
func newMmapIndexReader(fileHandle *os.File, footer archiveFooter) (*mmapIndexReader, error) {
	// Calculate the total index span
	indexSpan := footer.totalIndexSpan()

	// Calculate section offsets within the mapped region
	spanIndexOffset := int64(0)
	prefixesOffset := spanIndexOffset + int64(footer.byteSpanCount)*int64(uint64Size)
	chunkRefsOffset := prefixesOffset + int64(footer.chunkCount)*int64(uint64Size)
	suffixesOffset := chunkRefsOffset + int64(footer.chunkCount)*2*int64(uint32Size)

	// Memory map the entire index section
	mappedData, err := file.Mmap(fileHandle, int64(indexSpan.offset), int(indexSpan.length))
	if err != nil {
		return nil, fmt.Errorf("failed to mmap index: %w", err)
	}

	return &mmapIndexReader{
		data:            mappedData,
		indexSize:       footer.indexSize,
		byteSpanCount:   footer.byteSpanCount,
		chunkCount:      footer.chunkCount,
		spanIndexOffset: spanIndexOffset,
		prefixesOffset:  prefixesOffset,
		chunkRefsOffset: chunkRefsOffset,
		suffixesOffset:  suffixesOffset,
	}, nil
}

func (m *mmapIndexReader) getNumChunks() int {
	return int(m.chunkCount)
}

// getSpanIndex returns the span index value at the given position
func (m *mmapIndexReader) getSpanIndex(idx uint32) uint64 {
	if idx == 0 {
		return 0 // Null span to simplify logic, matching original implementation
	}
	if idx > m.byteSpanCount {
		return 0
	}

	offset := m.spanIndexOffset + int64(idx-1)*int64(uint64Size)
	return m.data.GetUint64(offset)
}

// getPrefix returns the prefix value at the given index
func (m *mmapIndexReader) getPrefix(idx uint32) uint64 {
	if idx >= m.chunkCount {
		return 0
	}
	offset := m.prefixesOffset + int64(idx)*int64(uint64Size)
	return m.data.GetUint64(offset)
}

// getChunkRef returns the dictionary and data references for the chunk at the given index
func (m *mmapIndexReader) getChunkRef(idx int) (dict, data uint32) {
	if idx < 0 || idx >= int(m.chunkCount) {
		return 0, 0
	}

	// Chunk refs are stored as pairs of uint32s
	offset := m.chunkRefsOffset + int64(idx)*2*int64(uint32Size)
	dict = m.data.GetUint32(offset)
	data = m.data.GetUint32(offset + int64(uint32Size))
	return
}

// getSuffix returns the suffix for the chunk at the given index
func (m *mmapIndexReader) getSuffix(idx uint64) (suf suffix) {
	if idx >= uint64(m.chunkCount) {
		return suffix{}
	}

	start := m.suffixesOffset + int64(idx)*hash.SuffixLen
	_, _ = m.data.ReadAt(suf[:], start)
	return
}

// close unmaps the memory region
func (m *mmapIndexReader) Close() error {
	// Currently we never unmap mmapped indexes in order to prevent a data race with the AutoIncrementTracker.
	/*if m.data != nil {
		data := m.data
		m.data = nil
		err := data.munmap()
		return err
	}*/
	return nil
}
