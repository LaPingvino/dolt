# Git Integration Design with Chunking

## Overview

This document outlines the design for Dolt's Git integration feature, enabling export/import of Dolt repositories to/from Git repositories (GitHub, GitLab, etc.) with smart chunking to handle large datasets.

## Problem Statement

### File Size Constraints
- **GitHub**: 100MB file limit, 5GB repository warnings
- **Git Performance**: Large files slow clone/fetch operations
- **Real-world Data**: Dolt tables can easily exceed these limits
- **Collaboration**: Teams want to use existing Git workflows for data versioning

### Requirements
1. Export complete Dolt repositories to Git repositories
2. Import Git repositories back to Dolt with full fidelity
3. Handle arbitrarily large tables through chunking
4. Preserve schema information and metadata
5. Support incremental updates and synchronization
6. Work with existing Git hosting platforms

## Architecture

### File Structure in Git Repository
```
repo/
├── .dolt-metadata/
│   ├── manifest.json         # Repository-level metadata
│   ├── schema.sql            # Complete database schema
│   └── tables/
│       ├── users.json        # Table metadata (chunking info)
│       └── orders.json       # Table metadata
├── data/
│   ├── users/
│   │   ├── users_000001.csv  # First chunk (50MB max)
│   │   ├── users_000002.csv  # Second chunk
│   │   └── users_000003.csv  # Final chunk
│   └── orders/
│       └── orders_000001.csv # Single chunk (under limit)
└── README.md                 # Human-readable repository info
```

### Chunking Strategies

#### Strategy 1: Size-based Chunking (Default)
```json
{
  "table": "users",
  "chunking": "size",
  "max_chunk_size": "50MB",
  "chunks": [
    {
      "file": "users_000001.csv",
      "rows": 100000,
      "row_range": [1, 100000],
      "size": 52428800
    },
    {
      "file": "users_000002.csv", 
      "rows": 87543,
      "row_range": [100001, 187543],
      "size": 45123456
    }
  ]
}
```

#### Strategy 2: Column-based Partitioning (Advanced)
```json
{
  "table": "events",
  "chunking": "column",
  "partition_column": "event_date",
  "chunks": [
    {
      "file": "events_2023.csv",
      "filter": "event_date >= '2023-01-01' AND event_date < '2024-01-01'",
      "rows": 2500000
    },
    {
      "file": "events_2024.csv",
      "filter": "event_date >= '2024-01-01' AND event_date < '2025-01-01'",
      "rows": 1200000
    }
  ]
}
```

#### Strategy 3: Compressed Chunking (Storage Optimized)
```json
{
  "table": "large_data",
  "chunking": "compressed",
  "compression": "gzip",
  "max_chunk_size": "50MB",
  "chunks": [
    {
      "file": "large_data_000001.csv.gz",
      "uncompressed_size": 150000000,
      "compressed_size": 45000000,
      "rows": 500000
    }
  ]
}
```

## Implementation

### Core Components

#### 1. Git Bridge Interface
```go
type GitBridge interface {
    Export(ctx context.Context, doltRepo *env.DoltEnv, gitRepoURL string, opts ExportOptions) error
    Import(ctx context.Context, gitRepoURL string, targetDir string, opts ImportOptions) error
    Sync(ctx context.Context, doltRepo *env.DoltEnv, gitRepoURL string, opts SyncOptions) error
}
```

#### 2. Chunking Engine
```go
type ChunkingStrategy interface {
    ShouldChunk(table string, estimatedSize int64) bool
    CreateChunks(ctx context.Context, table string, reader TableReader) ([]ChunkInfo, error)
    ReassembleChunks(ctx context.Context, chunks []ChunkInfo) (TableReader, error)
}

type ChunkInfo struct {
    FileName    string
    RowCount    int64
    SizeBytes   int64
    RowRange    [2]int64  // [start, end] row indices
    Filter      string    // SQL WHERE clause for column-based chunking
}
```

#### 3. Export Pipeline
```go
func (gb *GitBridge) Export(ctx context.Context, doltRepo *env.DoltEnv, gitRepoURL string, opts ExportOptions) error {
    // 1. Clone or create Git repository
    gitRepo := gb.prepareGitRepo(gitRepoURL)
    
    // 2. Export metadata and schema
    gb.exportMetadata(doltRepo, gitRepo)
    gb.exportSchema(doltRepo, gitRepo)
    
    // 3. Process each table
    tables := doltRepo.GetTableNames()
    for _, tableName := range tables {
        // 4. Determine chunking strategy
        strategy := gb.selectChunkingStrategy(tableName, opts)
        
        // 5. Export table data with chunking
        gb.exportTable(ctx, doltRepo, tableName, strategy, gitRepo)
    }
    
    // 6. Commit and push changes
    return gb.commitAndPush(gitRepo, "Export from Dolt")
}
```

### Commands Interface

#### Export Command
```bash
# Basic export
dolt git export github.com/user/dataset-repo

# With custom chunking
dolt git export --chunk-size=25MB github.com/user/dataset-repo

# Column-based chunking for time series data
dolt git export --chunk-by=date_column github.com/user/dataset-repo

# Compressed export
dolt git export --compress=gzip github.com/user/dataset-repo
```

#### Import Command  
```bash
# Basic import
dolt git import github.com/user/dataset-repo

# Import to specific directory
dolt git import github.com/user/dataset-repo ./imported-data

# Import specific tables only
dolt git import --tables=users,orders github.com/user/dataset-repo
```

#### Sync Command
```bash
# Bidirectional sync
dolt git sync github.com/user/dataset-repo

# Push-only sync
dolt git sync --push-only github.com/user/dataset-repo

# Pull-only sync  
dolt git sync --pull-only github.com/user/dataset-repo
```

### Chunking Algorithm

#### Size-based Chunking Implementation
```go
func (s *SizeBasedChunking) CreateChunks(ctx context.Context, table string, reader TableReader) ([]ChunkInfo, error) {
    var chunks []ChunkInfo
    chunkIndex := 1
    currentSize := int64(0)
    currentRows := int64(0)
    startRow := int64(1)
    
    // Create CSV writer for current chunk
    chunkWriter := s.createChunkWriter(table, chunkIndex)
    
    for {
        row, err := reader.ReadRow(ctx)
        if err == io.EOF {
            break
        }
        
        // Write row and track size
        rowSize := s.writeRowAndMeasure(chunkWriter, row)
        currentSize += rowSize
        currentRows++
        
        // Check if chunk size limit reached
        if currentSize >= s.maxChunkSize {
            // Finalize current chunk
            chunkInfo := ChunkInfo{
                FileName:  s.getChunkFileName(table, chunkIndex),
                RowCount:  currentRows,
                SizeBytes: currentSize,
                RowRange:  [2]int64{startRow, startRow + currentRows - 1},
            }
            chunks = append(chunks, chunkInfo)
            
            // Start new chunk
            chunkIndex++
            startRow += currentRows
            currentSize = 0
            currentRows = 0
            chunkWriter = s.createChunkWriter(table, chunkIndex)
        }
    }
    
    // Handle final chunk if any data remains
    if currentRows > 0 {
        chunkInfo := ChunkInfo{
            FileName:  s.getChunkFileName(table, chunkIndex),
            RowCount:  currentRows, 
            SizeBytes: currentSize,
            RowRange:  [2]int64{startRow, startRow + currentRows - 1},
        }
        chunks = append(chunks, chunkInfo)
    }
    
    return chunks, nil
}
```

### Integration with Bundle Architecture

#### Leveraging Bundle Experience
- **Metadata handling**: Reuse bundle's metadata structures
- **Compression**: Apply bundle compression techniques to chunks
- **Error handling**: Similar patterns for data validation
- **Testing**: Adapt bundle test patterns for Git workflows

#### Shared Components
```go
// Reuse from bundle implementation
type RepositoryMetadata struct {
    FormatVersion string    `json:"format_version"`
    CreatedAt     time.Time `json:"created_at"`
    Creator       string    `json:"creator"`
    Description   string    `json:"description"`
    Branch        string    `json:"branch"`
    CommitHash    string    `json:"commit_hash"`
}

// Extend for Git-specific needs
type GitExportMetadata struct {
    RepositoryMetadata
    ChunkingStrategy string            `json:"chunking_strategy"`
    MaxChunkSize     int64             `json:"max_chunk_size"`
    Tables          []TableMetadata    `json:"tables"`
}
```

## Performance Considerations

### Chunking Performance
- **Streaming processing**: Process tables in chunks to avoid memory issues
- **Parallel processing**: Export multiple tables concurrently
- **Incremental updates**: Only re-export changed chunks
- **Compression ratios**: Monitor and optimize chunk sizes based on content

### Git Operations
- **Shallow clones**: Use shallow clones for faster initial operations
- **LFS integration**: Consider Git LFS for very large chunks
- **Batch commits**: Group related changes into single commits
- **Progress reporting**: Show progress for long-running operations

## Error Handling and Recovery

### Chunking Errors
- **Partial failures**: Resume from last successful chunk
- **Size estimation errors**: Adjust chunking strategy dynamically
- **Corrupt chunks**: Validate chunk integrity during reassembly

### Git Integration Errors
- **Network failures**: Implement retry logic with exponential backoff
- **Authentication**: Support multiple authentication methods
- **Merge conflicts**: Provide clear resolution strategies
- **Repository state**: Validate Git repository state before operations

## Testing Strategy

### Unit Tests
- Chunking algorithms with various data sizes and types
- Metadata serialization/deserialization
- Individual table export/import operations

### Integration Tests
- End-to-end export/import with real Git repositories
- Large dataset handling (>1GB test data)
- Multiple table scenarios with different chunking strategies
- Network failure simulation and recovery

### Performance Tests
- Chunking overhead measurement
- Memory usage profiling during large exports
- Git operation timing benchmarks

## Future Enhancements

### Advanced Features
- **Smart chunking**: ML-based optimal chunk size prediction
- **Semantic chunking**: Chunk by logical data boundaries
- **Delta exports**: Only export changed data since last sync
- **Schema evolution**: Handle schema changes across Git commits

### Integration Opportunities
- **GitHub Actions**: Automated data validation workflows
- **GitLab CI**: Data pipeline integration
- **Git hooks**: Automatic validation on push
- **Branch-based data versioning**: Map Dolt branches to Git branches

## Implementation Timeline

### Phase 1: Core Export (Week 1)
- Basic Git repository creation and file writing
- Size-based chunking implementation
- Metadata and schema export
- Single table export functionality

### Phase 2: Import and Basic Sync (Week 2)
- Git repository reading and parsing
- Chunk reassembly logic
- Basic table import functionality
- Bidirectional sync foundation

### Phase 3: Advanced Features (Week 3)
- Multiple chunking strategies
- Compression support
- Error handling and recovery
- Performance optimizations

### Phase 4: Production Readiness (Week 4)
- Comprehensive testing
- Documentation and examples
- Performance benchmarking
- Security review