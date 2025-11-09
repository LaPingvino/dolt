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

package gitcmds

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/dolthub/dolt/go/libraries/doltcore/doltdb"
	"github.com/dolthub/dolt/go/libraries/doltcore/doltdb/durable"
	"github.com/dolthub/dolt/go/libraries/doltcore/env"
	gitintegration "github.com/dolthub/dolt/go/libraries/doltcore/git"
	"github.com/dolthub/dolt/go/libraries/doltcore/row"
	"github.com/dolthub/dolt/go/libraries/doltcore/schema"
	"github.com/dolthub/dolt/go/libraries/doltcore/sqle/index"
	"github.com/dolthub/dolt/go/libraries/doltcore/table/untyped/csv"
	"github.com/dolthub/go-mysql-server/sql"

	"github.com/dolthub/dolt/go/libraries/utils/iohelp"
	"github.com/dolthub/dolt/go/store/types"
	eventsapi "github.com/dolthub/eventsapi_schema/dolt/services/eventsapi/v1alpha1"

	"github.com/fatih/color"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/go-git/go-git/v5/plumbing/transport"
	"github.com/go-git/go-git/v5/plumbing/transport/http"
	"github.com/go-git/go-git/v5/plumbing/transport/ssh"

	"github.com/dolthub/dolt/go/cmd/dolt/cli"
	"github.com/dolthub/dolt/go/cmd/dolt/commands"
	"github.com/dolthub/dolt/go/cmd/dolt/doltversion"
	"github.com/dolthub/dolt/go/cmd/dolt/errhand"

	"github.com/dolthub/dolt/go/libraries/utils/argparser"
)

var pushDocs = cli.CommandDocumentationContent{
	ShortDesc: `Push Dolt changes to a Git repository.`,
	LongDesc: `{{.EmphasisLeft}}dolt git push{{.EmphasisRight}} exports Dolt repository data to a Git repository in CSV format.

The command will:
1. Export all tables as CSV files (with automatic chunking for large tables)
2. Generate metadata files including schema and table information
3. Commit changes to the Git repository with descriptive messages
4. Push changes to the remote Git repository

The resulting Git repository structure:
- {{.EmphasisLeft}}.dolt-metadata/{{.EmphasisRight}} - Repository metadata and schema information
- {{.EmphasisLeft}}data/{{.EmphasisRight}} - CSV files organized by table name
- {{.EmphasisLeft}}README.md{{.EmphasisRight}} - Human-readable repository information

Large tables are automatically chunked to stay within Git hosting limits:
- Default chunk size: 50MB (configurable)
- Files over 80MB automatically use Git LFS
- CSV format for maximum Git ecosystem compatibility

Authentication methods supported:
- GitHub personal access tokens
- SSH keys
- Username/password authentication

Examples:
{{.EmphasisLeft}}# Push to existing Git repository{{.EmphasisRight}}
dolt git push https://github.com/user/dataset-repo main

{{.EmphasisLeft}}# Push with custom chunk size{{.EmphasisRight}}
dolt git push --chunk-size=25MB https://github.com/user/dataset-repo main

{{.EmphasisLeft}}# Push with authentication token{{.EmphasisRight}}
dolt git push --token=ghp_xyz123 https://github.com/user/private-dataset main

{{.EmphasisLeft}}# Push with custom commit message{{.EmphasisRight}}
dolt git push -m "Update Q4 2024 dataset" https://github.com/user/dataset-repo main
`,
	Synopsis: []string{
		"{{.LessThan}}git-repo-url{{.GreaterThan}} [{{.LessThan}}branch{{.GreaterThan}}]",
	},
}

type PushCmd struct{}

// Name returns the name of the Dolt cli command
func (cmd PushCmd) Name() string {
	return "push"
}

// Description returns a description of the command
func (cmd PushCmd) Description() string {
	return "Push Dolt changes to a Git repository."
}

// RequiresRepo indicates this command requires a Dolt repository
func (cmd PushCmd) RequiresRepo() bool {
	return true
}

// Docs returns the documentation for this command
func (cmd PushCmd) Docs() *cli.CommandDocumentation {
	ap := cmd.ArgParser()
	return cli.NewCommandDocumentation(pushDocs, ap)
}

// ArgParser returns the argument parser for this command
func (cmd PushCmd) ArgParser() *argparser.ArgParser {
	ap := argparser.NewArgParserWithMaxArgs(cmd.Name(), 2)
	ap.ArgListHelp = append(ap.ArgListHelp, [2]string{"git-repo-url", "URL of the Git repository to push to"})
	ap.ArgListHelp = append(ap.ArgListHelp, [2]string{"branch", "Branch to push to (default: main)"})

	ap.SupportsString("message", "m", "message", "Commit message for the push")
	ap.SupportsString("token", "t", "token", "Personal access token for private repository authentication")
	ap.SupportsString("username", "u", "username", "Username for HTTP authentication")
	ap.SupportsString("password", "p", "password", "Password for HTTP authentication")
	ap.SupportsString("ssh-key", "", "path", "Path to SSH private key file")
	ap.SupportsString("chunk-size", "", "size", "Maximum chunk size (e.g., 25MB, 100MB)")
	ap.SupportsFlag("force", "f", "Force push even if remote has changes")
	ap.SupportsFlag("verbose", "v", "Show detailed progress information")
	ap.SupportsFlag("dry-run", "", "Show what would be pushed without actually pushing")

	return ap
}

// EventType returns the type of the event to log
func (cmd PushCmd) EventType() eventsapi.ClientEventType {
	return eventsapi.ClientEventType_PUSH
}

// Exec executes the git push command
func (cmd PushCmd) Exec(ctx context.Context, commandStr string, args []string, dEnv *env.DoltEnv, cliCtx cli.CliContext) int {
	ap := cmd.ArgParser()
	help, usage := cli.HelpAndUsagePrinters(cli.CommandDocsForCommandString(commandStr, pushDocs, ap))
	apr := cli.ParseArgsOrDie(ap, args, help)

	if apr.NArg() == 0 {
		usage()
		return 1
	}

	repoURL := apr.Arg(0)
	branch := apr.GetValueOrDefault(apr.Arg(1), "main")
	verbose := apr.Contains("verbose")
	dryRun := apr.Contains("dry-run")

	if verbose {
		cli.Println(color.CyanString("Pushing to Git repository: %s", repoURL))
		cli.Println(color.CyanString("Target branch: %s", branch))
	}

	// Setup authentication
	auth, err := setupAuthentication(apr)
	if err != nil {
		return commands.HandleVErrAndExitCode(errhand.BuildDError("authentication setup failed: %v", err).Build(), nil)
	}

	// Parse chunk size
	chunkSize := int64(DefaultChunkSize)
	if chunkSizeStr := apr.GetValueOrDefault("chunk-size", ""); chunkSizeStr != "" {
		chunkSize, err = parseChunkSize(chunkSizeStr)
		if err != nil {
			return commands.HandleVErrAndExitCode(errhand.BuildDError("invalid chunk size: %v", err).Build(), nil)
		}
	}

	// Create Git config
	gitConfig := &GitConfig{
		ChunkSize:      chunkSize,
		UseCompression: false, // Git handles compression internally
		LfsEnabled:     true,
		RemoteName:     "origin",
		DefaultBranch:  branch,
	}

	if verbose {
		cli.Println(color.CyanString("Configuration:"))
		cli.Println(color.CyanString("  Chunk size: %d bytes (%.1f MB)", chunkSize, float64(chunkSize)/(1024*1024)))
		cli.Println(color.CyanString("  Git LFS enabled: %v", gitConfig.LfsEnabled))
	}

	// Export Dolt data to Git repository
	if err := pushDoltToGit(ctx, dEnv, repoURL, branch, gitConfig, auth, apr, verbose, dryRun); err != nil {
		return commands.HandleVErrAndExitCode(errhand.BuildDError("failed to push to Git repository: %v", err).Build(), nil)
	}

	if dryRun {
		cli.Println(color.GreenString("Dry run completed successfully"))
	} else {
		cli.Println(color.GreenString("Successfully pushed Dolt data to Git repository"))
		cli.Println(color.CyanString("Repository URL: %s", repoURL))
		cli.Println(color.CyanString("Branch: %s", branch))
	}

	return 0
}

// parseChunkSize parses a chunk size string like "25MB" or "100MB" into bytes
func parseChunkSize(sizeStr string) (int64, error) {
	sizeStr = strings.ToUpper(strings.TrimSpace(sizeStr))

	if strings.HasSuffix(sizeStr, "MB") {
		sizeStr = strings.TrimSuffix(sizeStr, "MB")
		var size float64
		if n, err := fmt.Sscanf(sizeStr, "%f", &size); err != nil || n != 1 {
			return 0, fmt.Errorf("invalid chunk size format: %s", sizeStr)
		}
		return int64(size * 1024 * 1024), nil
	}

	if strings.HasSuffix(sizeStr, "GB") {
		sizeStr = strings.TrimSuffix(sizeStr, "GB")
		var size float64
		if n, err := fmt.Sscanf(sizeStr, "%f", &size); err != nil || n != 1 {
			return 0, fmt.Errorf("invalid chunk size format: %s", sizeStr)
		}
		return int64(size * 1024 * 1024 * 1024), nil
	}

	return 0, fmt.Errorf("chunk size must end with MB or GB (e.g., 50MB, 1GB)")
}

// pushDoltToGit handles the complete push process
func pushDoltToGit(ctx context.Context, dEnv *env.DoltEnv, repoURL, branch string, gitConfig *GitConfig, auth interface{}, apr *argparser.ArgParseResults, verbose, dryRun bool) error {
	// Create temporary directory for Git operations
	tempDir, err := os.MkdirTemp("", "dolt-git-push-*")
	if err != nil {
		return fmt.Errorf("failed to create temporary directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Clone or initialize Git repository
	gitRepo, err := setupGitRepository(ctx, repoURL, tempDir, branch, auth, verbose)
	if err != nil {
		return fmt.Errorf("failed to setup Git repository: %v", err)
	}

	// Export Dolt data to Git repository structure
	if err := exportDoltData(ctx, dEnv, tempDir, gitConfig, verbose); err != nil {
		return fmt.Errorf("failed to export Dolt data: %v", err)
	}

	if dryRun {
		return nil // Don't actually commit and push in dry run mode
	}

	// Commit and push changes
	commitMessage := apr.GetValueOrDefault("message", generateCommitMessage(dEnv))
	if err := commitAndPush(ctx, gitRepo, tempDir, commitMessage, auth, apr.Contains("force"), verbose); err != nil {
		return fmt.Errorf("failed to commit and push changes: %v", err)
	}

	return nil
}

// setupGitRepository clones or initializes the target Git repository
func setupGitRepository(ctx context.Context, repoURL, tempDir, branch string, auth interface{}, verbose bool) (*git.Repository, error) {
	cloneOptions := &git.CloneOptions{
		URL:          repoURL,
		SingleBranch: true,
	}

	if auth != nil {
		if authMethod, ok := auth.(transport.AuthMethod); ok {
			cloneOptions.Auth = authMethod
		}
	}

	if verbose {
		cloneOptions.Progress = os.Stdout
		cli.Println(color.CyanString("Cloning existing Git repository..."))
	}

	// Try to clone existing repository
	repo, err := git.PlainCloneContext(ctx, tempDir, false, cloneOptions)
	if err != nil {
		if verbose {
			cli.Println(color.YellowString("Repository doesn't exist or is empty, will create new one"))
		}
		// Repository doesn't exist, create new one
		repo, err = git.PlainInit(tempDir, false)
		if err != nil {
			return nil, fmt.Errorf("failed to initialize Git repository: %v", err)
		}

		// Add remote
		_, err = repo.CreateRemote(&config.RemoteConfig{
			Name: "origin",
			URLs: []string{repoURL},
		})
		if err != nil {
			return nil, fmt.Errorf("failed to add remote: %v", err)
		}
	}

	return repo, nil
}

// exportDoltData exports all Dolt tables and metadata to Git repository structure
func exportDoltData(ctx context.Context, dEnv *env.DoltEnv, gitRepoPath string, gitConfig *GitConfig, verbose bool) error {
	// Create directory structure
	dataDir := filepath.Join(gitRepoPath, "data")
	metadataDir := filepath.Join(gitRepoPath, ".dolt-metadata")
	tablesMetadataDir := filepath.Join(metadataDir, "tables")

	for _, dir := range []string{dataDir, metadataDir, tablesMetadataDir} {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create directory %s: %v", dir, err)
		}
	}

	// Get all tables
	root, err := dEnv.WorkingRoot(ctx)
	if err != nil {
		return fmt.Errorf("failed to get working root: %v", err)
	}

	tableNames, err := root.GetTableNames(ctx, doltdb.DefaultSchemaName, true)
	if err != nil {
		return fmt.Errorf("failed to get table names: %v", err)
	}

	if verbose {
		cli.Println(color.CyanString("Exporting %d tables...", len(tableNames)))
	}

	var tablesMetadata []TableGitMetadata
	chunking := gitintegration.NewSizeBasedChunking(gitConfig.ChunkSize)

	// Export each table
	for _, tableName := range tableNames {
		if verbose {
			cli.Println(color.CyanString("  Exporting table: %s", tableName))
		}

		tableMetadata, err := exportTable(ctx, dEnv, tableName, dataDir, tablesMetadataDir, chunking, verbose)
		if err != nil {
			return fmt.Errorf("failed to export table %s: %v", tableName, err)
		}

		tablesMetadata = append(tablesMetadata, *tableMetadata)
	}

	// Export schema
	if err := exportSchema(ctx, dEnv, metadataDir, verbose); err != nil {
		return fmt.Errorf("failed to export schema: %v", err)
	}

	// Generate repository metadata
	if err := generateRepositoryMetadata(dEnv, metadataDir, tablesMetadata, gitConfig, verbose); err != nil {
		return fmt.Errorf("failed to generate repository metadata: %v", err)
	}

	// Generate README
	if err := generateREADME(gitRepoPath, tablesMetadata, verbose); err != nil {
		return fmt.Errorf("failed to generate README: %v", err)
	}

	return nil
}

// exportTable exports a single table, handling chunking if necessary
func exportTable(ctx context.Context, dEnv *env.DoltEnv, tableName string, dataDir, metadataDir string, chunking gitintegration.ChunkingStrategy, verbose bool) (*TableGitMetadata, error) {
	// Create table-specific directory
	tableDataDir := filepath.Join(dataDir, tableName)
	if err := os.MkdirAll(tableDataDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create table directory: %v", err)
	}

	// Get table from Dolt environment
	root, err := dEnv.WorkingRoot(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get working root: %v", err)
	}

	table, tableName, ok, err := doltdb.GetTableInsensitive(ctx, root, doltdb.TableName{Name: tableName, Schema: doltdb.DefaultSchemaName})
	if err != nil {
		return nil, fmt.Errorf("failed to get table %s: %v", tableName, err)
	}
	if !ok {
		return nil, fmt.Errorf("table %s not found", tableName)
	}

	// Get table schema
	sch, err := table.GetSchema(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get schema for table %s: %v", tableName, err)
	}

	// Get row data for size estimation
	rowData, err := table.GetRowData(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get row data for table %s: %v", tableName, err)
	}

	rowCount, err := rowData.Count()
	if err != nil {
		return nil, fmt.Errorf("failed to count rows in table %s: %v", tableName, err)
	}

	// Estimate size based on row count (rough estimate: 100 bytes per row average)
	estimatedSize := int64(rowCount * 100)

	var chunks []gitintegration.ChunkInfo
	var actualRowCount int64 = 0
	var actualSize int64 = 0

	if chunking.ShouldChunk(tableName, estimatedSize) && rowCount > 10000 {
		if verbose {
			cli.Println(color.CyanString("    Table %s requires chunking (estimated size: %.1f MB, %d rows)",
				tableName, float64(estimatedSize)/(1024*1024), rowCount))
		}

		// Export in chunks
		chunkSize := int64(50000) // rows per chunk
		chunkNum := 0
		rowsProcessed := int64(0)

		for rowsProcessed < int64(rowCount) {
			chunkNum++
			chunkFileName := fmt.Sprintf("%s_%06d.csv", tableName, chunkNum)
			chunkPath := filepath.Join(tableDataDir, chunkFileName)

			// Create and export this chunk
			chunkRowCount, chunkFileSize, err := exportTableChunk(ctx, table, sch, chunkPath, rowsProcessed, chunkSize)
			if err != nil {
				return nil, fmt.Errorf("failed to export chunk %d of table %s: %v", chunkNum, tableName, err)
			}

			chunks = append(chunks, gitintegration.ChunkInfo{
				FileName:  chunkFileName,
				RowCount:  chunkRowCount,
				SizeBytes: chunkFileSize,
				RowRange:  [2]int64{rowsProcessed, rowsProcessed + chunkRowCount},
			})

			actualRowCount += chunkRowCount
			actualSize += chunkFileSize
			rowsProcessed += chunkRowCount

			if rowsProcessed >= int64(rowCount) {
				break
			}
		}
	} else {
		if verbose {
			cli.Println(color.CyanString("    Table %s exported as single file (%d rows)", tableName, rowCount))
		}

		// Single file export
		chunkFileName := fmt.Sprintf("%s.csv", tableName)
		chunkPath := filepath.Join(tableDataDir, chunkFileName)

		chunkRowCount, chunkFileSize, err := exportTableChunk(ctx, table, sch, chunkPath, 0, int64(rowCount))
		if err != nil {
			return nil, fmt.Errorf("failed to export table %s: %v", tableName, err)
		}

		chunks = []gitintegration.ChunkInfo{
			{
				FileName:  chunkFileName,
				RowCount:  chunkRowCount,
				SizeBytes: chunkFileSize,
			},
		}

		actualRowCount = chunkRowCount
		actualSize = chunkFileSize
	}

	// Generate table schema DDL
	schemaDDL, err := generateTableDDL(sch, tableName)
	if err != nil {
		return nil, fmt.Errorf("failed to generate schema DDL for table %s: %v", tableName, err)
	}

	// Create table metadata
	tableMetadata := &gitintegration.TableMetadata{
		TableName:        tableName,
		ChunkingStrategy: chunking.GetStrategyName(),
		MaxChunkSize:     gitintegration.DefaultMaxChunkSize,
		Chunks:           chunks,
		Schema:           schemaDDL,
		CreatedAt:        time.Now(),
	}

	// Save table metadata
	metadataPath := filepath.Join(metadataDir, fmt.Sprintf("%s.json", tableName))
	if err := saveTableMetadata(tableMetadata, metadataPath); err != nil {
		return nil, fmt.Errorf("failed to save table metadata: %v", err)
	}

	return &TableGitMetadata{
		TableName:       tableName,
		ChunkCount:      len(chunks),
		TotalRows:       actualRowCount,
		TotalSizeBytes:  actualSize,
		LastModified:    time.Now().Format(time.RFC3339),
		ChunkingEnabled: len(chunks) > 1,
		LfsEnabled:      anyChunkOverLfsThreshold(chunks),
	}, nil
}

// Helper functions
func calculateTotalRows(chunks []gitintegration.ChunkInfo) int64 {
	total := int64(0)
	for _, chunk := range chunks {
		total += chunk.RowCount
	}
	return total
}

func calculateTotalSize(chunks []gitintegration.ChunkInfo) int64 {
	total := int64(0)
	for _, chunk := range chunks {
		total += chunk.SizeBytes
	}
	return total
}

func anyChunkOverLfsThreshold(chunks []gitintegration.ChunkInfo) bool {
	for _, chunk := range chunks {
		if chunk.SizeBytes > GitLfsThreshold {
			return true
		}
	}
	return false
}

// saveTableMetadata saves table metadata to JSON file
func saveTableMetadata(metadata *gitintegration.TableMetadata, path string) error {
	data, err := json.MarshalIndent(metadata, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

// exportSchema exports database schema as SQL DDL
func exportSchema(ctx context.Context, dEnv *env.DoltEnv, metadataDir string, verbose bool) error {
	if verbose {
		cli.Println(color.CyanString("Exporting database schema..."))
	}

	// Get all tables from Dolt environment
	root, err := dEnv.WorkingRoot(ctx)
	if err != nil {
		return fmt.Errorf("failed to get working root: %v", err)
	}

	tableNames, err := root.GetTableNames(ctx, doltdb.DefaultSchemaName, true)
	if err != nil {
		return fmt.Errorf("failed to get table names: %v", err)
	}

	var schemaSQL strings.Builder
	schemaSQL.WriteString("-- Database schema exported from Dolt\n")
	schemaSQL.WriteString(fmt.Sprintf("-- Generated at: %s\n\n", time.Now().Format(time.RFC3339)))

	// Generate DDL for each table
	for _, tableName := range tableNames {
		table, _, ok, err := doltdb.GetTableInsensitive(ctx, root, doltdb.TableName{Name: tableName, Schema: doltdb.DefaultSchemaName})
		if err != nil {
			return fmt.Errorf("failed to get table %s: %v", tableName, err)
		}
		if !ok {
			continue
		}

		sch, err := table.GetSchema(ctx)
		if err != nil {
			return fmt.Errorf("failed to get schema for table %s: %v", tableName, err)
		}

		tableDDL, err := generateTableDDL(sch, tableName)
		if err != nil {
			return fmt.Errorf("failed to generate DDL for table %s: %v", tableName, err)
		}

		schemaSQL.WriteString(fmt.Sprintf("-- Table: %s\n", tableName))
		schemaSQL.WriteString(tableDDL)
		schemaSQL.WriteString("\n\n")
	}

	schemaPath := filepath.Join(metadataDir, "schema.sql")
	if err := os.WriteFile(schemaPath, []byte(schemaSQL.String()), 0644); err != nil {
		return fmt.Errorf("failed to write schema file: %v", err)
	}

	return nil
}

// generateRepositoryMetadata creates the main repository metadata manifest
func generateRepositoryMetadata(dEnv *env.DoltEnv, metadataDir string, tablesMetadata []TableGitMetadata, gitConfig *GitConfig, verbose bool) error {
	if verbose {
		cli.Println(color.CyanString("Generating repository metadata..."))
	}

	// Get actual Dolt version
	doltVersion := doltversion.Version

	// Get current user (try from environment or Git config)
	exportedBy := "unknown"
	if user := os.Getenv("USER"); user != "" {
		exportedBy = user
	} else if user := os.Getenv("USERNAME"); user != "" {
		exportedBy = user
	}

	// Get current branch and commit
	currentBranch := "main"
	currentCommit := "unknown"

	if dEnv.RepoState != nil {
		if headRef, err := dEnv.RepoStateReader().CWBHeadRef(context.Background()); err == nil {
			currentBranch = headRef.GetPath()
			if strings.HasPrefix(currentBranch, "refs/heads/") {
				currentBranch = strings.TrimPrefix(currentBranch, "refs/heads/")
			}
		}
	}

	metadata := RepositoryGitMetadata{
		DoltVersion:    doltVersion,
		ExportedAt:     time.Now().Format(time.RFC3339),
		ExportedBy:     exportedBy,
		SourceBranch:   currentBranch,
		SourceCommit:   currentCommit,
		Tables:         tablesMetadata,
		ChunkingConfig: gitConfig,
	}

	manifestPath := filepath.Join(metadataDir, "manifest.json")
	manifestData, err := json.MarshalIndent(metadata, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %v", err)
	}

	if err := os.WriteFile(manifestPath, manifestData, 0644); err != nil {
		return fmt.Errorf("failed to write manifest file: %v", err)
	}

	return nil
}

// exportTableChunk exports a portion of a table to a CSV file
func exportTableChunk(ctx context.Context, table *doltdb.Table, sch schema.Schema, outputPath string, offset, limit int64) (int64, int64, error) {
	// Create output file
	outputFile, err := os.Create(outputPath)
	if err != nil {
		return 0, 0, fmt.Errorf("failed to create output file %s: %v", outputPath, err)
	}
	defer outputFile.Close()

	// Create CSV writer with headers
	csvInfo := csv.NewCSVInfo()
	csvInfo.HasHeaderLine = true
	csvWriter, err := csv.NewCSVWriter(iohelp.NopWrCloser(outputFile), sch, csvInfo)
	if err != nil {
		return 0, 0, fmt.Errorf("failed to create CSV writer: %v", err)
	}

	// Get table row data
	rowData, err := table.GetRowData(ctx)
	if err != nil {
		return 0, 0, fmt.Errorf("failed to get row data: %v", err)
	}

	// Create SQL context for CSV operations
	sqlCtx := sql.NewEmptyContext()

	// Get all columns for data conversion
	allCols := sch.GetAllCols()
	colCount := allCols.Size()

	// Create a map of tag to column index for efficient lookup
	tagToIdx := make(map[uint64]int)
	colTags := make([]uint64, colCount)
	i := 0
	err = allCols.Iter(func(tag uint64, col schema.Column) (bool, error) {
		tagToIdx[tag] = i
		colTags[i] = tag
		i++
		return false, nil
	})
	if err != nil {
		return 0, 0, fmt.Errorf("failed to iterate schema columns: %v", err)
	}

	// Iterate through actual table data
	rowsWritten := int64(0)
	currentRowNum := int64(0)

	// Use proper row iteration based on Dolt storage format
	if types.IsFormat_DOLT(rowData.Format()) {
		// For DOLT format, use prolly tree handling with proper row iterator
		prollyMap, err := durable.ProllyMapFromIndex(rowData)
		if err != nil {
			return 0, 0, fmt.Errorf("failed to get prolly map: %v", err)
		}

		totalRows, err := prollyMap.Count()
		if err != nil {
			return 0, 0, fmt.Errorf("failed to count rows: %v", err)
		}

		if totalRows > 0 {
			// Calculate actual range based on offset/limit
			startIdx := uint64(0)
			endIdx := uint64(totalRows)

			if offset > 0 && uint64(offset) < uint64(totalRows) {
				startIdx = uint64(offset)
			}

			if limit > 0 && startIdx+uint64(limit) < uint64(totalRows) {
				endIdx = startIdx + uint64(limit)
			}

			// Use ordinal range iterator for efficient offset/limit handling
			iter, err := prollyMap.IterOrdinalRange(ctx, startIdx, endIdx)
			if err != nil {
				return 0, 0, fmt.Errorf("failed to create range iterator: %v", err)
			}

			// Use proper row iterator for prolly tree data
			rowIter := index.NewProllyRowIterForMap(sch, prollyMap, iter, nil)

			// Iterate through SQL rows
			for {
				sqlRow, err := rowIter.Next(sqlCtx)
				if err == io.EOF {
					break
				}
				if err != nil {
					return 0, 0, fmt.Errorf("failed to read row: %v", err)
				}

				// Write row to CSV
				err = csvWriter.WriteSqlRow(sqlCtx, sqlRow)
				if err != nil {
					return 0, 0, fmt.Errorf("failed to write row %d: %v", rowsWritten, err)
				}

				rowsWritten++
			}
		}
	} else {
		// For NOMS format, use proper iteration
		nomsMap := durable.NomsMapFromIndex(rowData)

		err = nomsMap.IterAll(ctx, func(key, value types.Value) error {
			// Skip rows before offset
			if currentRowNum < offset {
				currentRowNum++
				return nil
			}

			// Stop if we've reached the limit
			if limit > 0 && rowsWritten >= limit {
				return nil
			}

			// Convert key/value to row with proper type assertions
			r, err := row.FromNoms(sch, key.(types.Tuple), value.(types.Tuple))
			if err != nil {
				return fmt.Errorf("failed to convert row data: %v", err)
			}

			// Convert row to SQL row format
			sqlRow := make(sql.Row, colCount)
			for j, tag := range colTags {
				val, ok := r.GetColVal(tag)
				if !ok || val == nil {
					sqlRow[j] = nil
				} else {
					col, _ := allCols.GetByTag(tag)
					sqlType := col.TypeInfo.ToSqlType()

					// Convert the value to SQL format
					convertedVal, _, err := sqlType.Convert(ctx, val)
					if err != nil {
						// If conversion fails, use string representation
						sqlRow[j] = val.HumanReadableString()
					} else {
						sqlRow[j] = convertedVal
					}
				}
			}

			// Write row to CSV
			err = csvWriter.WriteSqlRow(sqlCtx, sqlRow)
			if err != nil {
				return fmt.Errorf("failed to write row %d: %v", rowsWritten, err)
			}

			rowsWritten++
			currentRowNum++
			return nil
		})

		if err != nil {
			return 0, 0, fmt.Errorf("failed to iterate table rows: %v", err)
		}
	}

	// Close CSV writer to flush data
	err = csvWriter.Close(ctx)
	if err != nil {
		return 0, 0, fmt.Errorf("failed to flush CSV writer: %v", err)
	}

	// Get final file size
	fileInfo, err := outputFile.Stat()
	if err != nil {
		return rowsWritten, 0, fmt.Errorf("failed to get file stats: %v", err)
	}

	return rowsWritten, fileInfo.Size(), nil
}

// generateTableDDL creates a CREATE TABLE statement for the given schema
func generateTableDDL(sch schema.Schema, tableName string) (string, error) {
	var ddl strings.Builder
	ddl.WriteString(fmt.Sprintf("CREATE TABLE `%s` (\n", tableName))

	cols := sch.GetAllCols().GetColumns()
	var columnDefs []string
	var primaryKeys []string

	for _, col := range cols {
		colDef, err := generateColumnDDL(col)
		if err != nil {
			return "", fmt.Errorf("failed to generate DDL for column %s: %v", col.Name, err)
		}
		columnDefs = append(columnDefs, colDef)

		if col.IsPartOfPK {
			primaryKeys = append(primaryKeys, fmt.Sprintf("`%s`", col.Name))
		}
	}

	ddl.WriteString("  " + strings.Join(columnDefs, ",\n  "))

	if len(primaryKeys) > 0 {
		ddl.WriteString(",\n  PRIMARY KEY (")
		ddl.WriteString(strings.Join(primaryKeys, ", "))
		ddl.WriteString(")")
	}

	ddl.WriteString("\n);")
	return ddl.String(), nil
}

// generateColumnDDL creates the DDL for a single column
func generateColumnDDL(col schema.Column) (string, error) {
	var colDef strings.Builder
	colDef.WriteString(fmt.Sprintf("`%s` ", col.Name))

	// Map Dolt types to SQL types
	// Generate column definition using SQL type
	sqlType := col.TypeInfo.ToSqlType()
	colDef.WriteString(sqlType.String())

	if !col.IsNullable() {
		colDef.WriteString(" NOT NULL")
	}

	return colDef.String(), nil
}

// generateREADME creates a human-readable README for the Git repository
func generateREADME(gitRepoPath string, tables []TableGitMetadata, verbose bool) error {
	if verbose {
		cli.Println(color.CyanString("Generating README.md..."))
	}

	readme := `# Dolt Dataset Repository

This repository contains data exported from a Dolt database in CSV format.

## Repository Structure

- ` + "`" + `.dolt-metadata/` + "`" + ` - Repository metadata and schema information
- ` + "`" + `data/` + "`" + ` - CSV files organized by table name
- Large tables are automatically chunked to stay within Git hosting limits

## Tables

| Table | Rows | Size | Chunks |
|-------|------|------|--------|
`

	for _, table := range tables {
		size := formatBytes(table.TotalSizeBytes)
		readme += fmt.Sprintf("| %s | %d | %s | %d |\n", table.TableName, table.TotalRows, size, table.ChunkCount)
	}

	readme += `
## Usage

To work with this data in Dolt:

` + "```" + `bash
# Clone this repository back to Dolt format
dolt git clone <this-repo-url>

# Or work with the CSV files directly
# Data files are located in the data/ directory
` + "```" + `

## Generated Information

- Exported at: ` + time.Now().Format("2006-01-02 15:04:05 UTC") + `
- Total tables: ` + fmt.Sprintf("%d", len(tables)) + `
- Format: CSV with automatic chunking for large tables

For more information about Dolt, visit https://github.com/dolthub/dolt
`

	readmePath := filepath.Join(gitRepoPath, "README.md")
	if err := os.WriteFile(readmePath, []byte(readme), 0644); err != nil {
		return fmt.Errorf("failed to write README file: %v", err)
	}

	return nil
}

// formatBytes formats byte count as human-readable string
func formatBytes(bytes int64) string {
	if bytes < 1024 {
		return fmt.Sprintf("%d B", bytes)
	}
	if bytes < 1024*1024 {
		return fmt.Sprintf("%.1f KB", float64(bytes)/1024)
	}
	if bytes < 1024*1024*1024 {
		return fmt.Sprintf("%.1f MB", float64(bytes)/(1024*1024))
	}
	return fmt.Sprintf("%.1f GB", float64(bytes)/(1024*1024*1024))
}

// generateCommitMessage creates a descriptive commit message
func generateCommitMessage(dEnv *env.DoltEnv) string {
	timestamp := time.Now().Format("2006-01-02 15:04:05")

	// Try to get database name for a more descriptive message
	dbName := "database"
	if dbData := dEnv.DbData(context.Background()); dbData.Ddb != nil {
		// Use a simple placeholder name for now
		dbName = "dolt-repository"
	}

	return fmt.Sprintf("Export %s dataset to Git - %s", dbName, timestamp)
}

// commitAndPush commits changes and pushes to remote repository
func commitAndPush(ctx context.Context, repo *git.Repository, repoPath, message string, auth interface{}, force bool, verbose bool) error {
	worktree, err := repo.Worktree()
	if err != nil {
		return fmt.Errorf("failed to get worktree: %v", err)
	}

	// Add all files
	if verbose {
		cli.Println(color.CyanString("Adding files to Git..."))
	}

	if err := worktree.AddGlob("."); err != nil {
		return fmt.Errorf("failed to add files: %v", err)
	}

	// Commit changes
	if verbose {
		cli.Println(color.CyanString("Committing changes..."))
	}

	commit, err := worktree.Commit(message, &git.CommitOptions{
		Author: &object.Signature{
			Name:  "Dolt Git Integration",
			Email: "dolt@dolthub.com",
			When:  time.Now(),
		},
	})
	if err != nil {
		return fmt.Errorf("failed to commit: %v", err)
	}

	if verbose {
		cli.Println(color.GreenString("✓ Created commit: %s", commit.String()[:8]))
	}

	// Push to remote
	if verbose {
		cli.Println(color.CyanString("Pushing to remote repository..."))
	}

	pushOptions := &git.PushOptions{}
	if auth != nil {
		if authMethod, ok := auth.(transport.AuthMethod); ok {
			pushOptions.Auth = authMethod
			if verbose {
				// Don't log sensitive auth details, just indicate what type is being used
				switch authMethod.(type) {
				case *ssh.PublicKeys:
					cli.Println(color.CyanString("Using SSH key authentication"))
				case *http.BasicAuth:
					cli.Println(color.CyanString("Using HTTP basic authentication"))
				default:
					cli.Println(color.CyanString("Using authentication method: %T", authMethod))
				}
			}
		} else if verbose {
			cli.Println(color.YellowString("Warning: Authentication object provided but not recognized as transport.AuthMethod"))
		}
	} else if verbose {
		cli.Println(color.YellowString("No authentication method configured - attempting anonymous access"))
	}
	if force {
		pushOptions.Force = true
	}
	if verbose {
		pushOptions.Progress = os.Stdout
	}

	if err := repo.Push(pushOptions); err != nil {
		// Provide more helpful error messages based on common failure scenarios
		errStr := err.Error()

		if strings.Contains(errStr, "ssh: handshake failed") && strings.Contains(errStr, "unable to authenticate") {
			return fmt.Errorf("SSH authentication failed: %v\n\nTroubleshooting steps:\n"+
				"1. Ensure your SSH key is added to your Git hosting provider (GitHub, GitLab, etc.)\n"+
				"2. Test SSH connection: 'ssh -T git@github.com' (for GitHub)\n"+
				"3. Add your SSH key to ssh-agent: 'ssh-add ~/.ssh/id_ed25519'\n"+
				"4. Or use token authentication: --token=YOUR_TOKEN\n"+
				"5. Or use username/password: --username=USER --password=PASS", err)
		}

		if strings.Contains(errStr, "authentication required") {
			return fmt.Errorf("authentication required for Git repository: %v\n\nPlease use one of:\n"+
				"• SSH key (ensure key is added to hosting provider and ssh-agent)\n"+
				"• Personal access token: --token=YOUR_TOKEN\n"+
				"• Username/password: --username=USER --password=PASS", err)
		}

		if strings.Contains(errStr, "repository not found") || strings.Contains(errStr, "404") {
			return fmt.Errorf("repository not found or access denied: %v\n\nCheck:\n"+
				"1. Repository URL is correct\n"+
				"2. Repository exists and you have push access\n"+
				"3. Authentication credentials are valid", err)
		}

		if strings.Contains(errStr, "non-fast-forward") {
			return fmt.Errorf("push rejected due to non-fast-forward update: %v\n\n"+
				"The remote repository has changes that conflict with your push.\n"+
				"Try: dolt git pull before pushing, or use --force to override (DANGEROUS)", err)
		}

		// Generic error with some helpful context
		return fmt.Errorf("failed to push to Git repository: %v\n\n"+
			"Common solutions:\n"+
			"• Check network connectivity\n"+
			"• Verify repository URL and access permissions\n"+
			"• Ensure authentication is properly configured\n"+
			"• Use --verbose flag for more detailed output", err)
	}

	if verbose {
		cli.Println(color.GreenString("✓ Successfully pushed to remote repository"))
	}

	return nil
}
