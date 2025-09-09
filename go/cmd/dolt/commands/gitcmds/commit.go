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
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/fatih/color"

	"github.com/dolthub/dolt/go/cmd/dolt/cli"
	"github.com/dolthub/dolt/go/cmd/dolt/commands"
	"github.com/dolthub/dolt/go/cmd/dolt/errhand"
	"github.com/dolthub/dolt/go/libraries/doltcore/env"
	"github.com/dolthub/dolt/go/libraries/utils/argparser"
	eventsapi "github.com/dolthub/eventsapi_schema/dolt/services/eventsapi/v1alpha1"
)

var commitDocs = cli.CommandDocumentationContent{
	ShortDesc: `Commit staged table changes for Git.`,
	LongDesc: `{{.EmphasisLeft}}dolt git commit{{.EmphasisRight}} commits staged table changes, preparing them for the next Git push operation.

This command:
1. Records the current state of all staged tables
2. Creates a commit record with the provided message
3. Clears the staging area
4. Prepares the committed changes for {{.EmphasisLeft}}dolt git push{{.EmphasisRight}}

The commit operation is local and does not immediately push to Git. Use {{.EmphasisLeft}}dolt git push{{.EmphasisRight}} to upload the committed changes to a Git repository.

Staged tables can be viewed with {{.EmphasisLeft}}dolt git status{{.EmphasisRight}} before committing.

A commit message is required and should describe the changes being committed. The message will be used when pushing to Git repositories.

Examples:
{{.EmphasisLeft}}# Commit staged changes with message{{.EmphasisRight}}
dolt git commit -m "Add Q4 2024 sales data"

{{.EmphasisLeft}}# Commit with detailed message{{.EmphasisRight}}
dolt git commit -m "Update customer database

- Add 1,000 new customer records
- Update contact information for existing customers
- Add new fields for customer preferences"

{{.EmphasisLeft}}# Commit with verbose output{{.EmphasisRight}}
dolt git commit -m "Update inventory data" --verbose
`,
	Synopsis: []string{
		"-m {{.LessThan}}message{{.GreaterThan}}",
	},
}

type CommitCmd struct{}

// Name returns the name of the Dolt cli command
func (cmd CommitCmd) Name() string {
	return "commit"
}

// Description returns a description of the command
func (cmd CommitCmd) Description() string {
	return "Commit staged table changes for Git."
}

// RequiresRepo indicates this command requires a Dolt repository
func (cmd CommitCmd) RequiresRepo() bool {
	return true
}

// Docs returns the documentation for this command
func (cmd CommitCmd) Docs() *cli.CommandDocumentation {
	ap := cmd.ArgParser()
	return cli.NewCommandDocumentation(commitDocs, ap)
}

// ArgParser returns the argument parser for this command
func (cmd CommitCmd) ArgParser() *argparser.ArgParser {
	ap := argparser.NewArgParserWithMaxArgs(cmd.Name(), 0)

	ap.SupportsString("message", "m", "message", "Commit message (required)")
	ap.SupportsString("author", "", "author", "Override commit author (format: \"Name <email>\")")
	ap.SupportsFlag("verbose", "v", "Show detailed information about committed tables")
	ap.SupportsFlag("dry-run", "", "Show what would be committed without actually committing")

	return ap
}

// EventType returns the type of the event to log
func (cmd CommitCmd) EventType() eventsapi.ClientEventType {
	return eventsapi.ClientEventType_COMMIT
}

// GitCommitInfo contains information about a Git commit
type GitCommitInfo struct {
	ID         string    `json:"id"`
	Message    string    `json:"message"`
	Author     string    `json:"author"`
	Timestamp  time.Time `json:"timestamp"`
	Tables     []string  `json:"tables"`
	DoltCommit string    `json:"dolt_commit,omitempty"`
	DoltBranch string    `json:"dolt_branch,omitempty"`
}

// Exec executes the git commit command
func (cmd CommitCmd) Exec(ctx context.Context, commandStr string, args []string, dEnv *env.DoltEnv, cliCtx cli.CliContext) int {
	ap := cmd.ArgParser()
	help, _ := cli.HelpAndUsagePrinters(cli.CommandDocsForCommandString(commandStr, commitDocs, ap))
	apr := cli.ParseArgsOrDie(ap, args, help)

	// Check for required message
	message := apr.GetValueOrDefault("message", "")
	if message == "" {
		cli.PrintErrln(color.RedString("Error: Commit message is required"))
		cli.PrintErrln(color.CyanString("Use: dolt git commit -m \"Your commit message\""))
		return 1
	}

	verbose := apr.Contains("verbose")
	dryRun := apr.Contains("dry-run")

	// Load staged tables
	gitStagingDir := filepath.Join(dEnv.GetDoltDir(), ".dolt-git")
	stagedTables, err := loadStagedTables(gitStagingDir)
	if err != nil {
		return commands.HandleVErrAndExitCode(errhand.BuildDError("failed to load staged tables: %v", err).Build(), nil)
	}

	if len(stagedTables) == 0 {
		cli.PrintErrln(color.YellowString("No changes staged for commit"))
		cli.PrintErrln(color.CyanString("Use 'dolt git add <table>' to stage tables for commit"))
		cli.PrintErrln(color.CyanString("Use 'dolt git status' to see available tables"))
		return 1
	}

	if verbose || dryRun {
		cli.Println(color.CyanString("Tables to be committed:"))
		for _, table := range stagedTables {
			cli.Println(color.GreenString("  %s", table))
		}
	}

	if dryRun {
		cli.Println(color.YellowString("Dry run - would commit %d table(s)", len(stagedTables)))
		cli.Println(color.CyanString("Commit message: %s", message))
		return 0
	}

	// Create commit
	author := apr.GetValueOrDefault("author", getDefaultAuthor())
	commitInfo := &GitCommitInfo{
		ID:         generateCommitID(),
		Message:    message,
		Author:     author,
		Timestamp:  time.Now(),
		Tables:     stagedTables,
		DoltCommit: "", // TODO: Get current Dolt commit hash
		DoltBranch: "", // TODO: Get current Dolt branch
	}

	if err := createGitCommit(ctx, dEnv, commitInfo, verbose); err != nil {
		return commands.HandleVErrAndExitCode(errhand.BuildDError("failed to create commit: %v", err).Build(), nil)
	}

	// Success message
	cli.Println(color.GreenString("Created commit %s", commitInfo.ID[:8]))
	cli.Println(color.CyanString("Committed %d table(s): %s", len(stagedTables), strings.Join(stagedTables, ", ")))

	if verbose {
		cli.Println(color.CyanString("Author: %s", author))
		cli.Println(color.CyanString("Message: %s", message))
		cli.Println(color.CyanString("Timestamp: %s", commitInfo.Timestamp.Format("2006-01-02 15:04:05")))
	}

	cli.Println()
	cli.Println(color.CyanString("Use 'dolt git push <remote> <branch>' to push this commit to a Git repository"))

	return 0
}

// createGitCommit creates a new Git commit record
func createGitCommit(ctx context.Context, dEnv *env.DoltEnv, commitInfo *GitCommitInfo, verbose bool) error {
	gitStagingDir := filepath.Join(dEnv.GetDoltDir(), ".dolt-git")

	// Ensure Git staging directory exists
	if err := os.MkdirAll(gitStagingDir, 0755); err != nil {
		return fmt.Errorf("failed to create Git staging directory: %v", err)
	}

	// Save commit information
	commitsDir := filepath.Join(gitStagingDir, "commits")
	if err := os.MkdirAll(commitsDir, 0755); err != nil {
		return fmt.Errorf("failed to create commits directory: %v", err)
	}

	if err := saveCommitInfo(commitsDir, commitInfo); err != nil {
		return fmt.Errorf("failed to save commit info: %v", err)
	}

	// Update HEAD to point to this commit
	if err := updateGitHEAD(gitStagingDir, commitInfo.ID); err != nil {
		return fmt.Errorf("failed to update HEAD: %v", err)
	}

	// Record this as the latest commit for push
	if err := updateLatestCommit(gitStagingDir, commitInfo.ID); err != nil {
		return fmt.Errorf("failed to update latest commit: %v", err)
	}

	// Clear staging area
	if err := clearStagingArea(gitStagingDir); err != nil {
		return fmt.Errorf("failed to clear staging area: %v", err)
	}

	// Record commit time as last activity
	if err := saveExportTime(gitStagingDir, commitInfo.Timestamp); err != nil {
		// This is not critical, just log it
		if verbose {
			cli.PrintErrln(color.YellowString("Warning: Failed to update last activity time: %v", err))
		}
	}

	if verbose {
		cli.Println(color.GreenString("✓ Saved commit information"))
		cli.Println(color.GreenString("✓ Updated HEAD reference"))
		cli.Println(color.GreenString("✓ Cleared staging area"))
	}

	return nil
}

// saveCommitInfo saves commit information to disk
func saveCommitInfo(commitsDir string, commitInfo *GitCommitInfo) error {
	data, err := json.MarshalIndent(commitInfo, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal commit info: %v", err)
	}

	commitFile := filepath.Join(commitsDir, commitInfo.ID+".json")
	if err := os.WriteFile(commitFile, data, 0644); err != nil {
		return fmt.Errorf("failed to write commit file: %v", err)
	}

	return nil
}

// updateGitHEAD updates the HEAD reference to point to the new commit
func updateGitHEAD(gitStagingDir, commitID string) error {
	headFile := filepath.Join(gitStagingDir, "HEAD")
	return os.WriteFile(headFile, []byte(commitID), 0644)
}

// updateLatestCommit updates the latest commit reference for push operations
func updateLatestCommit(gitStagingDir, commitID string) error {
	latestFile := filepath.Join(gitStagingDir, "latest_commit.txt")
	return os.WriteFile(latestFile, []byte(commitID), 0644)
}

// clearStagingArea clears all staged tables
func clearStagingArea(gitStagingDir string) error {
	stagingFile := filepath.Join(gitStagingDir, "staged_tables.txt")

	// Remove staging file or write empty content
	if err := os.WriteFile(stagingFile, []byte(""), 0644); err != nil {
		return fmt.Errorf("failed to clear staging file: %v", err)
	}

	return nil
}

// generateCommitID generates a unique commit identifier
func generateCommitID() string {
	// Create a hash based on current time and some randomness
	h := sha256.New()
	h.Write([]byte(time.Now().Format(time.RFC3339Nano)))
	h.Write([]byte(fmt.Sprintf("%d", time.Now().UnixNano())))

	hash := h.Sum(nil)
	return hex.EncodeToString(hash)[:40] // Git-style 40 character hash
}

// getDefaultAuthor gets the default commit author
func getDefaultAuthor() string {
	// TODO: Get from Dolt config or Git config
	// For now, return a placeholder
	if user := os.Getenv("USER"); user != "" {
		return fmt.Sprintf("%s <dolt@dolthub.com>", user)
	}
	return "Dolt User <dolt@dolthub.com>"
}

// getLatestCommitID gets the ID of the latest commit
func getLatestCommitID(gitStagingDir string) (string, error) {
	latestFile := filepath.Join(gitStagingDir, "latest_commit.txt")

	if _, err := os.Stat(latestFile); os.IsNotExist(err) {
		return "", fmt.Errorf("no commits found")
	}

	content, err := os.ReadFile(latestFile)
	if err != nil {
		return "", fmt.Errorf("failed to read latest commit: %v", err)
	}

	return strings.TrimSpace(string(content)), nil
}

// loadCommitInfo loads commit information from disk
func loadCommitInfo(commitsDir, commitID string) (*GitCommitInfo, error) {
	commitFile := filepath.Join(commitsDir, commitID+".json")

	data, err := os.ReadFile(commitFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read commit file: %v", err)
	}

	var commitInfo GitCommitInfo
	if err := json.Unmarshal(data, &commitInfo); err != nil {
		return nil, fmt.Errorf("failed to unmarshal commit info: %v", err)
	}

	return &commitInfo, nil
}
