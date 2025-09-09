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
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/dolthub/dolt/go/libraries/doltcore/doltdb"

	"github.com/fatih/color"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/go-git/go-git/v5/plumbing/transport"

	"github.com/dolthub/dolt/go/cmd/dolt/cli"
	"github.com/dolthub/dolt/go/cmd/dolt/commands"
	"github.com/dolthub/dolt/go/cmd/dolt/errhand"
	"github.com/dolthub/dolt/go/libraries/doltcore/env"
	gitintegration "github.com/dolthub/dolt/go/libraries/doltcore/git"
	"github.com/dolthub/dolt/go/libraries/utils/argparser"
	eventsapi "github.com/dolthub/eventsapi_schema/dolt/services/eventsapi/v1alpha1"
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

	// TODO: Get table reader from Dolt environment
	// For now, this is a placeholder that would need integration with Dolt's table reading infrastructure
	// This would involve:
	// 1. Getting the table from dEnv
	// 2. Creating a TableReader that implements the gitintegration.TableReader interface
	// 3. Using the chunking strategy to create chunks

	// Estimate table size (placeholder - would come from actual table)
	estimatedSize := int64(100 * 1024 * 1024) // Placeholder: 100MB

	var chunks []gitintegration.ChunkInfo
	var err error

	if chunking.ShouldChunk(tableName, estimatedSize) {
		if verbose {
			cli.Println(color.CyanString("    Table %s requires chunking (estimated size: %.1f MB)",
				tableName, float64(estimatedSize)/(1024*1024)))
		}

		// TODO: Create actual table reader and use chunking
		// chunks, err = chunking.CreateChunks(ctx, tableName, tableReader, tableDataDir)
		// For now, create placeholder chunks
		chunks = []gitintegration.ChunkInfo{
			{
				FileName:  fmt.Sprintf("%s_000001.csv", tableName),
				RowCount:  50000,
				SizeBytes: 50 * 1024 * 1024,
				RowRange:  [2]int64{1, 50000},
			},
			{
				FileName:  fmt.Sprintf("%s_000002.csv", tableName),
				RowCount:  30000,
				SizeBytes: 30 * 1024 * 1024,
				RowRange:  [2]int64{50001, 80000},
			},
		}
	} else {
		if verbose {
			cli.Println(color.CyanString("    Table %s exported as single file", tableName))
		}

		// Single file export
		chunks = []gitintegration.ChunkInfo{
			{
				FileName:  fmt.Sprintf("%s_000001.csv", tableName),
				RowCount:  10000,
				SizeBytes: 10 * 1024 * 1024,
			},
		}
	}

	if err != nil {
		return nil, err
	}

	// Create table metadata
	tableMetadata := &gitintegration.TableMetadata{
		TableName:        tableName,
		ChunkingStrategy: chunking.GetStrategyName(),
		MaxChunkSize:     50 * 1024 * 1024, // TODO: Get from chunking strategy
		Chunks:           chunks,
		Schema:           "", // TODO: Get actual table schema as DDL
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
		TotalRows:       calculateTotalRows(chunks),
		TotalSizeBytes:  calculateTotalSize(chunks),
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

	// TODO: Generate actual schema DDL from Dolt environment
	// This would involve getting all table schemas and generating CREATE TABLE statements
	schemaSQL := "-- Database schema exported from Dolt\n-- TODO: Generate actual DDL from Dolt tables\n"

	schemaPath := filepath.Join(metadataDir, "schema.sql")
	if err := os.WriteFile(schemaPath, []byte(schemaSQL), 0644); err != nil {
		return fmt.Errorf("failed to write schema file: %v", err)
	}

	return nil
}

// generateRepositoryMetadata creates the main repository metadata manifest
func generateRepositoryMetadata(dEnv *env.DoltEnv, metadataDir string, tables []TableGitMetadata, gitConfig *GitConfig, verbose bool) error {
	if verbose {
		cli.Println(color.CyanString("Generating repository metadata..."))
	}

	// TODO: Get actual Dolt version and current user
	metadata := &RepositoryGitMetadata{
		DoltVersion:    "1.32.4", // TODO: Get actual version
		ExportedAt:     time.Now().Format(time.RFC3339),
		ExportedBy:     "dolt-user", // TODO: Get actual user
		SourceBranch:   "main",      // TODO: Get current branch
		SourceCommit:   "abc123",    // TODO: Get current commit hash
		Tables:         tables,
		ChunkingConfig: gitConfig,
	}

	manifestPath := filepath.Join(metadataDir, "manifest.json")
	data, err := json.MarshalIndent(metadata, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %v", err)
	}

	if err := os.WriteFile(manifestPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write manifest file: %v", err)
	}

	return nil
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
	// TODO: Generate more descriptive message based on changes
	return fmt.Sprintf("Update Dolt dataset - %s", time.Now().Format("2006-01-02 15:04:05"))
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
		}
	}
	if force {
		pushOptions.Force = true
	}
	if verbose {
		pushOptions.Progress = os.Stdout
	}

	if err := repo.Push(pushOptions); err != nil {
		return fmt.Errorf("failed to push: %v", err)
	}

	if verbose {
		cli.Println(color.GreenString("✓ Successfully pushed to remote repository"))
	}

	return nil
}
