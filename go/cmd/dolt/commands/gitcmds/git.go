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
- {{.EmphasisLeft}}diagnostics{{.EmphasisRight}}: Diagnose Git authentication and connectivity issues
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
	DiagnosticsCmd{},
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

// DiagnosticsCmd provides Git authentication and connectivity diagnostics
type DiagnosticsCmd struct{}

func (cmd DiagnosticsCmd) Name() string {
	return "diagnostics"
}

func (cmd DiagnosticsCmd) Description() string {
	return "Diagnose Git authentication and connectivity issues."
}

func (cmd DiagnosticsCmd) RequiresRepo() bool {
	return false
}

func (cmd DiagnosticsCmd) Docs() *cli.CommandDocumentation {
	ap := cmd.ArgParser()
	return cli.NewCommandDocumentation(cli.CommandDocumentationContent{
		ShortDesc: "Diagnose Git authentication and connectivity issues",
		LongDesc: `The {{.EmphasisLeft}}dolt git diagnostics{{.EmphasisRight}} command helps troubleshoot common Git integration issues.

This command checks:
- SSH key availability and configuration
- SSH connectivity to common Git hosts (GitHub, GitLab)
- SSH agent status
- Git configuration
- Network connectivity

Use this command when experiencing authentication failures or connectivity issues with Git operations.`,
		Synopsis: []string{
			"[--host <hostname>]",
		},
	}, ap)
}

func (cmd DiagnosticsCmd) ArgParser() *argparser.ArgParser {
	ap := argparser.NewArgParserWithMaxArgs(cmd.Name(), 0)
	ap.SupportsString("host", "", "hostname", "Specific Git host to test (e.g., github.com, gitlab.com)")
	return ap
}

func (cmd DiagnosticsCmd) EventType() eventsapi.ClientEventType {
	return eventsapi.ClientEventType_CLONE
}

func (cmd DiagnosticsCmd) Exec(ctx context.Context, commandStr string, args []string, dEnv *env.DoltEnv, cliCtx cli.CliContext) int {
	ap := cmd.ArgParser()
	help, _ := cli.HelpAndUsagePrinters(cli.CommandDocsForCommandString(commandStr, gitDocs, ap))
	apr := cli.ParseArgsOrDie(ap, args, help)

	host := apr.GetValueOrDefault("host", "")

	return runGitDiagnostics(ctx, host)
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

// runGitDiagnostics performs comprehensive Git diagnostics
func runGitDiagnostics(ctx context.Context, targetHost string) int {
	fmt.Println("🔍 Dolt Git Integration Diagnostics")
	fmt.Println("===================================")
	fmt.Println()

	allPassed := true

	// Test 1: Check for SSH directory and keys
	fmt.Print("1. Checking SSH configuration...")
	if sshOk := checkSSHConfiguration(); sshOk {
		fmt.Println(" ✅ PASS")
	} else {
		fmt.Println(" ❌ FAIL")
		allPassed = false
	}

	// Test 2: Check SSH agent
	fmt.Print("2. Checking SSH agent...")
	if agentOk := checkSSHAgent(); agentOk {
		fmt.Println(" ✅ PASS")
	} else {
		fmt.Println(" ⚠️  WARN - SSH agent not available")
	}

	// Test 3: Test connectivity to Git hosts
	hosts := []string{"github.com", "gitlab.com"}
	if targetHost != "" {
		hosts = []string{targetHost}
	}

	for i, host := range hosts {
		fmt.Printf("%d. Testing SSH connectivity to %s...", i+3, host)
		if connOk := testSSHConnectivity(host); connOk {
			fmt.Println(" ✅ PASS")
		} else {
			fmt.Println(" ❌ FAIL")
			allPassed = false
		}
	}

	// Test 4: Check network connectivity
	fmt.Printf("%d. Testing HTTPS connectivity...", len(hosts)+3)
	if httpsOk := testHTTPSConnectivity(hosts); httpsOk {
		fmt.Println(" ✅ PASS")
	} else {
		fmt.Println(" ⚠️  WARN - Limited HTTPS connectivity")
	}

	fmt.Println()
	if allPassed {
		fmt.Println("🎉 All critical diagnostics passed!")
		fmt.Println("Your Git integration should work correctly.")
		return 0
	} else {
		fmt.Println("⚠️  Some diagnostics failed.")
		fmt.Println("\nRecommended actions:")
		fmt.Println("• Generate SSH keys: ssh-keygen -t ed25519 -C 'your_email@example.com'")
		fmt.Println("• Add SSH key to Git host (GitHub/GitLab settings)")
		fmt.Println("• Test SSH: ssh -T git@github.com")
		fmt.Println("• Start SSH agent: eval $(ssh-agent -s) && ssh-add")
		fmt.Println("• Or use token authentication: --token=YOUR_TOKEN")
		return 1
	}
}
