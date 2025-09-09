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

#### Strategy 3: Git LFS Chunking (Very Large Files)
```json
{
  "table": "large_data",
  "chunking": "lfs",
  "lfs_enabled": true,
  "max_chunk_size": "80MB",
  "chunks": [
    {
      "file": "large_data_000001.csv",
      "size": 150000000,
      "rows": 500000,
      "lfs_pointer": true
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
    LfsPointer  bool      // true if file should use Git LFS
}
```

#### 3. Push Pipeline
```go
func (gb *GitBridge) Push(ctx context.Context, doltRepo *env.DoltEnv, gitRepoURL string, opts PushOptions) error {
    // 1. Open or clone Git repository
    gitRepo := gb.ensureGitRepo(gitRepoURL)
    
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
    
    // 6. Git add, commit and push changes
    return gb.addCommitPush(gitRepo, opts.CommitMessage)
}
```

### Commands Interface

#### Git-Native Commands
```bash
# Clone a dataset repository
dolt git clone github.com/user/dataset-repo [directory]

# Add changes to staging area
dolt git add .
dolt git add table_name

# Commit changes with message
dolt git commit -m "Update dataset with new records"

# Push changes to remote
dolt git push origin main
dolt git push --chunk-size=25MB origin main

# Pull changes from remote  
dolt git pull origin main

# Check status of working directory
dolt git status

# View commit history
dolt git log
dolt git log --oneline

# Configure chunking for specific tables
dolt git config table.large_table.chunk-size 80MB
dolt git config table.events.chunk-by date_column
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
                LfsPointer: currentSize > 80*1024*1024, // Use LFS for files > 80MB
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
            LfsPointer: currentSize > 80*1024*1024, // Use LFS for files > 80MB
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
type GitRepositoryMetadata struct {
    RepositoryMetadata
    ChunkingStrategy string            `json:"chunking_strategy"`
    MaxChunkSize     int64             `json:"max_chunk_size"`
    LfsEnabled       bool              `json:"lfs_enabled"`
    Tables          []TableMetadata    `json:"tables"`
}
```

## Performance Considerations

### Chunking Performance
- **Streaming processing**: Process tables in chunks to avoid memory issues
- **Parallel processing**: Export multiple tables concurrently
- **Incremental updates**: Only re-export changed chunks
- **Git-native efficiency**: Let Git handle compression and delta storage

### Git Operations
- **Shallow clones**: Use shallow clones for faster initial operations  
- **LFS integration**: Automatic LFS for chunks >80MB
- **Batch commits**: Group related changes into single commits
- **Progress reporting**: Show progress for long-running operations
- **Native Git commands**: Mirror standard Git workflow

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

### Phase 1: Core Git Commands (Week 1)
- `dolt git clone` - Clone repositories from Git
- `dolt git push` - Push Dolt changes to Git
- Size-based chunking with Git LFS integration
- Metadata and schema handling

### Phase 2: Full Git Workflow (Week 2)
- `dolt git pull` - Pull changes from Git repositories
- `dolt git add` - Stage table changes
- `dolt git commit` - Commit with proper messaging
- `dolt git status` - Show working directory status

### Phase 3: Advanced Features (Week 3)
- `dolt git log` - View commit history
- Multiple chunking strategies
- Git configuration integration
- Error handling and recovery

### Phase 4: Production Readiness (Week 4)
- Comprehensive testing
- Documentation and examples
- Performance benchmarking
- Git hosting platform compatibility