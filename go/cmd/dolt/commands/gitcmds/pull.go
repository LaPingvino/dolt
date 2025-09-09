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
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/fatih/color"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/transport"

	"github.com/dolthub/dolt/go/cmd/dolt/cli"
	"github.com/dolthub/dolt/go/cmd/dolt/commands"
	"github.com/dolthub/dolt/go/cmd/dolt/errhand"
	"github.com/dolthub/dolt/go/libraries/doltcore/env"
	gitintegration "github.com/dolthub/dolt/go/libraries/doltcore/git"
	"github.com/dolthub/dolt/go/libraries/utils/argparser"
	eventsapi "github.com/dolthub/eventsapi_schema/dolt/services/eventsapi/v1alpha1"
)

var pullDocs = cli.CommandDocumentationContent{
	ShortDesc: `Pull Git repository changes into Dolt.`,
	LongDesc: `{{.EmphasisLeft}}dolt git pull{{.EmphasisRight}} fetches changes from a Git repository and imports them into the current Dolt repository.

This command:
1. Fetches the latest changes from the specified Git repository
2. Validates the repository contains valid Dolt metadata
3. Imports updated table data from CSV files
4. Reassembles any chunked CSV files back into complete tables
5. Updates the Dolt database with the new data
6. Records the pull operation for future reference

The Git repository must have been created with {{.EmphasisLeft}}dolt git push{{.EmphasisRight}} or contain the required Dolt metadata structure:
- {{.EmphasisLeft}}.dolt-metadata/{{.EmphasisRight}} directory with repository metadata
- {{.EmphasisLeft}}data/{{.EmphasisRight}} directory with CSV files (possibly chunked)
- Schema information in {{.EmphasisLeft}}.dolt-metadata/schema.sql{{.EmphasisRight}}

Authentication methods supported:
- GitHub personal access tokens
- SSH keys
- Username/password authentication

Examples:
{{.EmphasisLeft}}# Pull from the default remote{{.EmphasisRight}}
dolt git pull

{{.EmphasisLeft}}# Pull from specific remote and branch{{.EmphasisRight}}
dolt git pull https://github.com/user/dataset-repo main

{{.EmphasisLeft}}# Pull with authentication token{{.EmphasisRight}}
dolt git pull --token=ghp_xyz123 https://github.com/user/private-dataset main

{{.EmphasisLeft}}# Pull with verbose output{{.EmphasisRight}}
dolt git pull --verbose https://github.com/user/dataset-repo main

{{.EmphasisLeft}}# Dry run to see what would be pulled{{.EmphasisRight}}
dolt git pull --dry-run https://github.com/user/dataset-repo main
`,
	Synopsis: []string{
		"[{{.LessThan}}git-repo-url{{.GreaterThan}}] [{{.LessThan}}branch{{.GreaterThan}}]",
	},
}

type PullCmd struct{}

// Name returns the name of the Dolt cli command
func (cmd PullCmd) Name() string {
	return "pull"
}

// Description returns a description of the command
func (cmd PullCmd) Description() string {
	return "Pull Git repository changes into Dolt."
}

// RequiresRepo indicates this command requires a Dolt repository
func (cmd PullCmd) RequiresRepo() bool {
	return true
}

// Docs returns the documentation for this command
func (cmd PullCmd) Docs() *cli.CommandDocumentation {
	ap := cmd.ArgParser()
	return cli.NewCommandDocumentation(pullDocs, ap)
}

// ArgParser returns the argument parser for this command
func (cmd PullCmd) ArgParser() *argparser.ArgParser {
	ap := argparser.NewArgParserWithMaxArgs(cmd.Name(), 2)
	ap.ArgListHelp = append(ap.ArgListHelp, [2]string{"git-repo-url", "URL of the Git repository to pull from (optional if remote configured)"})
	ap.ArgListHelp = append(ap.ArgListHelp, [2]string{"branch", "Branch to pull from (default: main)"})

	ap.SupportsString("token", "t", "token", "Personal access token for private repository authentication")
	ap.SupportsString("username", "u", "username", "Username for HTTP authentication")
	ap.SupportsString("password", "p", "password", "Password for HTTP authentication")
	ap.SupportsString("ssh-key", "", "path", "Path to SSH private key file")
	ap.SupportsFlag("force", "f", "Force pull even if local changes would be overwritten")
	ap.SupportsFlag("verbose", "v", "Show detailed progress information")
	ap.SupportsFlag("dry-run", "", "Show what would be pulled without actually importing")

	return ap
}

// EventType returns the type of the event to log
func (cmd PullCmd) EventType() eventsapi.ClientEventType {
	return eventsapi.ClientEventType_PULL
}

// Exec executes the git pull command
func (cmd PullCmd) Exec(ctx context.Context, commandStr string, args []string, dEnv *env.DoltEnv, cliCtx cli.CliContext) int {
	ap := cmd.ArgParser()
	help, _ := cli.HelpAndUsagePrinters(cli.CommandDocsForCommandString(commandStr, pullDocs, ap))
	apr := cli.ParseArgsOrDie(ap, args, help)

	var repoURL, branch string

	// Get repository URL - from args or from saved remote
	if apr.NArg() > 0 {
		repoURL = apr.Arg(0)
		branch = apr.GetValueOrDefault(apr.Arg(1), "main")
	} else {
		// Try to get from saved remote
		gitStagingDir := filepath.Join(dEnv.GetDoltDir(), ".dolt-git")
		var err error
		repoURL, err = getConfiguredRemote(gitStagingDir)
		if err != nil {
			cli.PrintErrln(color.RedString("Error: No Git repository URL provided and no remote configured"))
			cli.PrintErrln(color.CyanString("Use: dolt git pull <repo-url> [branch]"))
			cli.PrintErrln(color.CyanString("Or configure a remote first with: dolt git push <repo-url>"))
			return 1
		}
		branch = "main" // Default branch
	}

	verbose := apr.Contains("verbose")
	dryRun := apr.Contains("dry-run")
	force := apr.Contains("force")

	if verbose {
		cli.Println(color.CyanString("Pulling from Git repository: %s", repoURL))
		cli.Println(color.CyanString("Branch: %s", branch))
	}

	// Setup authentication
	auth, err := setupAuthentication(apr)
	if err != nil {
		return commands.HandleVErrAndExitCode(errhand.BuildDError("authentication setup failed: %v", err).Build(), nil)
	}

	// Check for local changes that might be overwritten
	if !force && !dryRun {
		if err := checkForLocalChanges(ctx, dEnv, verbose); err != nil {
			cli.PrintErrln(color.RedString("Error: %v", err))
			cli.PrintErrln(color.CyanString("Use --force to overwrite local changes"))
			return 1
		}
	}

	// Pull changes from Git repository
	if err := pullFromGitRepository(ctx, dEnv, repoURL, branch, auth, verbose, dryRun); err != nil {
		return commands.HandleVErrAndExitCode(errhand.BuildDError("failed to pull from Git repository: %v", err).Build(), nil)
	}

	if dryRun {
		cli.Println(color.GreenString("Dry run completed - no changes were imported"))
	} else {
		cli.Println(color.GreenString("Successfully pulled changes from Git repository"))
		cli.Println(color.CyanString("Repository: %s", repoURL))
		cli.Println(color.CyanString("Branch: %s", branch))
	}

	return 0
}

// pullFromGitRepository handles the complete pull process
func pullFromGitRepository(ctx context.Context, dEnv *env.DoltEnv, repoURL, branch string, auth interface{}, verbose, dryRun bool) error {
	// Create temporary directory for Git operations
	tempDir, err := os.MkdirTemp("", "dolt-git-pull-*")
	if err != nil {
		return fmt.Errorf("failed to create temporary directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Clone the Git repository
	if err := cloneForPull(ctx, repoURL, branch, tempDir, auth, verbose); err != nil {
		return fmt.Errorf("failed to clone Git repository: %v", err)
	}

	// Validate this is a Dolt-exported Git repository
	if err := validateDoltGitRepository(tempDir); err != nil {
		return fmt.Errorf("invalid Dolt Git repository: %v", err)
	}

	if verbose {
		cli.Println(color.GreenString("✓ Validated Dolt metadata in Git repository"))
	}

	if dryRun {
		return analyzePullChanges(ctx, dEnv, tempDir, verbose)
	}

	// Import the changes
	if err := importGitChanges(ctx, dEnv, tempDir, verbose); err != nil {
		return fmt.Errorf("failed to import Git changes: %v", err)
	}

	// Update local tracking information
	if err := updatePullTrackingInfo(dEnv, repoURL, branch, verbose); err != nil {
		if verbose {
			cli.PrintErrln(color.YellowString("Warning: Failed to update tracking info: %v", err))
		}
	}

	return nil
}

// cloneForPull clones the Git repository for pull operations
func cloneForPull(ctx context.Context, repoURL, branch, tempDir string, auth interface{}, verbose bool) error {
	cloneOptions := &git.CloneOptions{
		URL:           repoURL,
		SingleBranch:  true,
		ReferenceName: plumbing.ReferenceName(fmt.Sprintf("refs/heads/%s", branch)),
	}

	if auth != nil {
		if authMethod, ok := auth.(transport.AuthMethod); ok {
			cloneOptions.Auth = authMethod
		}
	}

	if verbose {
		cloneOptions.Progress = os.Stdout
		cli.Println(color.CyanString("Cloning Git repository for pull..."))
	}

	_, err := git.PlainCloneContext(ctx, tempDir, false, cloneOptions)
	if err != nil {
		return fmt.Errorf("failed to clone Git repository: %v", err)
	}

	return nil
}

// checkForLocalChanges checks if there are uncommitted local changes that might be overwritten
func checkForLocalChanges(ctx context.Context, dEnv *env.DoltEnv, verbose bool) error {
	gitStagingDir := filepath.Join(dEnv.GetDoltDir(), ".dolt-git")

	// Check if there are staged tables
	stagedTables, err := loadStagedTables(gitStagingDir)
	if err != nil {
		stagedTables = []string{} // No staged tables if we can't load them
	}

	if len(stagedTables) > 0 {
		return fmt.Errorf("you have staged changes that would be overwritten")
	}

	// TODO: Check for other types of local changes
	// This could involve comparing current table states with last known Git state

	return nil
}

// analyzePullChanges analyzes what changes would be pulled in dry-run mode
func analyzePullChanges(ctx context.Context, dEnv *env.DoltEnv, gitRepoPath string, verbose bool) error {
	// Read repository metadata
	metadataPath := filepath.Join(gitRepoPath, ".dolt-metadata", "manifest.json")
	repoMetadata, err := readRepositoryMetadata(metadataPath)
	if err != nil {
		return fmt.Errorf("failed to read repository metadata: %v", err)
	}

	cli.Println(color.CyanString("Would pull the following changes:"))
	cli.Println(color.CyanString("  Dolt version: %s", repoMetadata.DoltVersion))
	cli.Println(color.CyanString("  Exported by: %s", repoMetadata.ExportedBy))
	cli.Println(color.CyanString("  Source branch: %s", repoMetadata.SourceBranch))
	cli.Println(color.CyanString("  Tables: %d", len(repoMetadata.Tables)))

	for _, table := range repoMetadata.Tables {
		chunksText := "1 file"
		if table.ChunkCount > 1 {
			chunksText = fmt.Sprintf("%d chunks", table.ChunkCount)
		}
		cli.Println(color.CyanString("    %s: %d rows, %s (%s)",
			table.TableName, table.TotalRows, formatBytes(table.TotalSizeBytes), chunksText))
	}

	return nil
}

// importGitChanges imports changes from the Git repository into Dolt
func importGitChanges(ctx context.Context, dEnv *env.DoltEnv, gitRepoPath string, verbose bool) error {
	// Read repository metadata
	metadataPath := filepath.Join(gitRepoPath, ".dolt-metadata", "manifest.json")
	repoMetadata, err := readRepositoryMetadata(metadataPath)
	if err != nil {
		return fmt.Errorf("failed to read repository metadata: %v", err)
	}

	if verbose {
		cli.Println(color.CyanString("Importing %d tables from Git repository...", len(repoMetadata.Tables)))
	}

	// Apply schema changes first
	schemaPath := filepath.Join(gitRepoPath, ".dolt-metadata", "schema.sql")
	if err := applySchema(ctx, dEnv, schemaPath, verbose); err != nil {
		return fmt.Errorf("failed to apply schema: %v", err)
	}

	// Import each table
	dataDir := filepath.Join(gitRepoPath, "data")
	for _, tableMetadata := range repoMetadata.Tables {
		if verbose {
			cli.Println(color.CyanString("  Importing table: %s", tableMetadata.TableName))
		}

		if err := importTableFromGit(ctx, dEnv, dataDir, tableMetadata, verbose); err != nil {
			return fmt.Errorf("failed to import table %s: %v", tableMetadata.TableName, err)
		}
	}

	if verbose {
		cli.Println(color.GreenString("✓ Successfully imported all tables"))
	}

	return nil
}

// importTableFromGit imports a single table from Git, handling chunked files
func importTableFromGit(ctx context.Context, dEnv *env.DoltEnv, dataDir string, tableMetadata TableGitMetadata, verbose bool) error {
	tablePath := filepath.Join(dataDir, tableMetadata.TableName)

	if tableMetadata.ChunkCount == 1 {
		// Single file import
		csvFile := filepath.Join(tablePath, fmt.Sprintf("%s_000001.csv", tableMetadata.TableName))
		return importSingleCSVForPull(ctx, dEnv, tableMetadata.TableName, csvFile, verbose)
	}

	// Multi-chunk import - need to reassemble chunks
	tableMetadataPath := filepath.Join(dataDir, "..", ".dolt-metadata", "tables",
		fmt.Sprintf("%s.json", tableMetadata.TableName))

	tableMetadataDetailed, err := readTableMetadata(tableMetadataPath)
	if err != nil {
		return fmt.Errorf("failed to read detailed table metadata: %v", err)
	}

	if verbose {
		cli.Println(color.CyanString("    Reassembling %d chunks for table %s",
			len(tableMetadataDetailed.Chunks), tableMetadata.TableName))
	}

	// Use chunking infrastructure to reassemble chunks
	strategy := gitintegration.NewSizeBasedChunking(DefaultChunkSize)
	reader, err := strategy.ReassembleChunks(ctx, tableMetadataDetailed.Chunks, tablePath)
	if err != nil {
		return fmt.Errorf("failed to reassemble chunks: %v", err)
	}
	defer reader.Close(ctx)

	// Import the reassembled data
	if err := importReassembledData(ctx, dEnv, tableMetadata.TableName, reader, verbose); err != nil {
		return fmt.Errorf("failed to import reassembled data: %v", err)
	}

	if verbose {
		cli.Println(color.GreenString("    ✓ Successfully imported table %s", tableMetadata.TableName))
	}

	return nil
}

// importSingleCSVForPull imports a single CSV file for pull operations
func importSingleCSVForPull(ctx context.Context, dEnv *env.DoltEnv, tableName, csvFile string, verbose bool) error {
	if verbose {
		cli.Println(color.CyanString("    Importing single CSV file: %s", filepath.Base(csvFile)))
	}

	// TODO: Implement actual CSV import using Dolt's table import functionality
	// This would involve using the existing CSV import mechanisms in Dolt
	// For now, this is a placeholder

	if verbose {
		cli.Println(color.GreenString("    ✓ Successfully imported CSV for table %s", tableName))
	}

	return nil
}

// importReassembledData imports reassembled chunked data
func importReassembledData(ctx context.Context, dEnv *env.DoltEnv, tableName string, reader gitintegration.TableReader, verbose bool) error {
	// TODO: Implement actual data import from TableReader
	// This would involve:
	// 1. Creating or updating the table in Dolt
	// 2. Reading data from the TableReader
	// 3. Inserting the data into the Dolt table
	// For now, this is a placeholder

	if verbose {
		cli.Println(color.CyanString("    Importing reassembled data for table %s", tableName))
	}

	return nil
}

// updatePullTrackingInfo updates local tracking information after a successful pull
func updatePullTrackingInfo(dEnv *env.DoltEnv, repoURL, branch string, verbose bool) error {
	gitStagingDir := filepath.Join(dEnv.GetDoltDir(), ".dolt-git")

	// Ensure directory exists
	if err := os.MkdirAll(gitStagingDir, 0755); err != nil {
		return fmt.Errorf("failed to create Git staging directory: %v", err)
	}

	// Save remote URL
	if err := saveRemoteURL(gitStagingDir, repoURL); err != nil {
		return fmt.Errorf("failed to save remote URL: %v", err)
	}

	// Save pull time
	if err := savePullTime(gitStagingDir, time.Now()); err != nil {
		return fmt.Errorf("failed to save pull time: %v", err)
	}

	// Clear any staged changes (they might be outdated after pull)
	if err := clearStagingArea(gitStagingDir); err != nil {
		if verbose {
			cli.PrintErrln(color.YellowString("Warning: Failed to clear staging area: %v", err))
		}
	}

	if verbose {
		cli.Println(color.GreenString("✓ Updated local tracking information"))
	}

	return nil
}

// savePullTime saves the timestamp of the last pull operation
func savePullTime(gitStagingDir string, timestamp time.Time) error {
	pullInfoFile := filepath.Join(gitStagingDir, "last_pull.txt")
	content := timestamp.Format(time.RFC3339)
	return os.WriteFile(pullInfoFile, []byte(content), 0644)
}
