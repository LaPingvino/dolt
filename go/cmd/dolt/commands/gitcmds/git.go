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

	"github.com/dolthub/dolt/go/cmd/dolt/cli"
	"github.com/dolthub/dolt/go/libraries/doltcore/env"
	"github.com/dolthub/dolt/go/libraries/utils/argparser"
	eventsapi "github.com/dolthub/eventsapi_schema/dolt/services/eventsapi/v1alpha1"
)

var gitDocs = cli.CommandDocumentationContent{
	ShortDesc: `Git integration for Dolt repositories.`,
	LongDesc: `{{.EmphasisLeft}}dolt git{{.EmphasisRight}} provides Git-compatible commands for working with Dolt repositories in Git hosting platforms.

Dolt Git integration enables:
- Cloning Dolt repositories from Git repositories (GitHub, GitLab, etc.)
- Pushing Dolt data changes to Git repositories as CSV files
- Pulling updates from Git repositories back into Dolt
- Using familiar Git workflows for data versioning and collaboration

The git commands automatically handle:
- Large table chunking to stay within Git hosting file size limits
- CSV format conversion for Git compatibility and human readability
- Schema preservation through metadata files
- Commit history mapping between Dolt and Git

Available subcommands:
- {{.EmphasisLeft}}clone{{.EmphasisRight}}: Clone a Git repository containing Dolt data
- {{.EmphasisLeft}}push{{.EmphasisRight}}: Push Dolt changes to a Git repository
- {{.EmphasisLeft}}pull{{.EmphasisRight}}: Pull Git repository changes into Dolt
- {{.EmphasisLeft}}add{{.EmphasisRight}}: Stage table changes for Git commit
- {{.EmphasisLeft}}commit{{.EmphasisRight}}: Commit staged changes to Git
- {{.EmphasisLeft}}status{{.EmphasisRight}}: Show Git working directory status
- {{.EmphasisLeft}}log{{.EmphasisRight}}: Show Git commit history
`,
	Synopsis: []string{
		"clone <git-repo-url> [<directory>]",
		"push [<remote>] [<branch>]",
		"pull [<remote>] [<branch>]",
		"add [<table-name>|.]",
		"commit -m <message>",
		"status",
		"log [--oneline] [-n <count>]",
	},
}

type GitCmd struct {
	Subcommands []cli.Command
}

var gitCommands = []cli.Command{
	CloneCmd{},
	PushCmd{},
	PullCmd{},
	AddCmd{},
	CommitCmd{},
	StatusCmd{},
	LogCmd{},
}

// NewGitCmd creates a new git command with all subcommands
func NewGitCmd() GitCmd {
	return GitCmd{
		Subcommands: gitCommands,
	}
}

// Name returns the name of the Dolt cli command
func (cmd GitCmd) Name() string {
	return "git"
}

// Description returns a description of the command
func (cmd GitCmd) Description() string {
	return "Git integration for Dolt repositories."
}

// RequiresRepo indicates whether this command requires a Dolt repository
func (cmd GitCmd) RequiresRepo() bool {
	return false // Some git commands (like clone) don't require existing repos
}

// Docs returns the documentation for this command
func (cmd GitCmd) Docs() *cli.CommandDocumentation {
	ap := cmd.ArgParser()
	return cli.NewCommandDocumentation(gitDocs, ap)
}

// ArgParser returns the argument parser for this command
func (cmd GitCmd) ArgParser() *argparser.ArgParser {
	ap := argparser.NewArgParserWithMaxArgs(cmd.Name(), 0)
	ap.SupportsString("help", "", "command", "Show help for a specific Git command")
	return ap
}

// EventType returns the type of the event to log
func (cmd GitCmd) EventType() eventsapi.ClientEventType {
	return eventsapi.ClientEventType_CLONE
}

// Exec executes the command
func (cmd GitCmd) Exec(ctx context.Context, commandStr string, args []string, dEnv *env.DoltEnv, cliCtx cli.CliContext) int {
	subCommandHandler := cli.NewSubCommandHandler(cmd.Name(), cmd.Description(), cmd.Subcommands)
	return subCommandHandler.Exec(ctx, commandStr, args, dEnv, cliCtx)
}

// Common constants and helper functions for Git commands
const (
	DefaultChunkSize = 50 * 1024 * 1024 // 50MB default chunk size for Git compatibility
	GitLfsThreshold  = 80 * 1024 * 1024 // Files larger than 80MB should use Git LFS
)

// GitConfig holds configuration for Git operations
type GitConfig struct {
	ChunkSize      int64  // Maximum size per CSV chunk
	UseCompression bool   // Whether to use gzip compression (generally false for Git)
	LfsEnabled     bool   // Whether to use Git LFS for large files
	RemoteName     string // Default remote name (usually "origin")
	DefaultBranch  string // Default branch name (usually "main")
}

// DefaultGitConfig returns the default configuration for Git operations
func DefaultGitConfig() *GitConfig {
	return &GitConfig{
		ChunkSize:      DefaultChunkSize,
		UseCompression: false, // Git handles compression internally
		LfsEnabled:     true,  // Enable LFS for files over threshold
		RemoteName:     "origin",
		DefaultBranch:  "main",
	}
}

// GitRepository represents a Git repository for Dolt operations
type GitRepository struct {
	URL        string
	LocalPath  string
	RemoteName string
	Branch     string
	Config     *GitConfig
}

// TableGitMetadata contains Git-specific metadata for a table
type TableGitMetadata struct {
	TableName       string `json:"table_name"`
	ChunkCount      int    `json:"chunk_count"`
	TotalRows       int64  `json:"total_rows"`
	TotalSizeBytes  int64  `json:"total_size_bytes"`
	LastModified    string `json:"last_modified"`
	ChunkingEnabled bool   `json:"chunking_enabled"`
	LfsEnabled      bool   `json:"lfs_enabled"`
}

// RepositoryGitMetadata contains Git-specific metadata for the entire repository
type RepositoryGitMetadata struct {
	DoltVersion    string             `json:"dolt_version"`
	ExportedAt     string             `json:"exported_at"`
	ExportedBy     string             `json:"exported_by"`
	SourceBranch   string             `json:"source_branch"`
	SourceCommit   string             `json:"source_commit"`
	Tables         []TableGitMetadata `json:"tables"`
	ChunkingConfig *GitConfig         `json:"chunking_config"`
}
