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
	"strings"
	"time"

	"github.com/fatih/color"

	"github.com/dolthub/dolt/go/cmd/dolt/cli"
	"github.com/dolthub/dolt/go/cmd/dolt/commands"
	"github.com/dolthub/dolt/go/cmd/dolt/errhand"
	"github.com/dolthub/dolt/go/libraries/doltcore/doltdb"
	"github.com/dolthub/dolt/go/libraries/doltcore/env"
	"github.com/dolthub/dolt/go/libraries/utils/argparser"
	eventsapi "github.com/dolthub/eventsapi_schema/dolt/services/eventsapi/v1alpha1"
)

var statusDocs = cli.CommandDocumentationContent{
	ShortDesc: `Show the Git working directory status.`,
	LongDesc: `{{.EmphasisLeft}}dolt git status{{.EmphasisRight}} displays the state of the working directory and staging area for Git operations.

This command shows:
- Tables staged for Git commit (ready to be pushed)
- Tables with modifications since the last Git export
- New tables that haven't been staged yet
- Information about the current Git remote configuration

The status helps track which tables will be included in the next {{.EmphasisLeft}}dolt git commit{{.EmphasisRight}} and {{.EmphasisLeft}}dolt git push{{.EmphasisRight}} operations.

Status categories:
- {{.EmphasisLeft}}Staged{{.EmphasisRight}}: Tables ready for the next Git commit
- {{.EmphasisLeft}}Modified{{.EmphasisRight}}: Tables changed since last Git export
- {{.EmphasisLeft}}Untracked{{.EmphasisRight}}: New tables not yet staged

Examples:
{{.EmphasisLeft}}# Show current Git status{{.EmphasisRight}}
dolt git status

{{.EmphasisLeft}}# Show detailed status information{{.EmphasisRight}}
dolt git status --verbose

{{.EmphasisLeft}}# Show status with table size information{{.EmphasisRight}}
dolt git status --show-sizes
`,
	Synopsis: []string{
		"",
	},
}

type StatusCmd struct{}

// Name returns the name of the Dolt cli command
func (cmd StatusCmd) Name() string {
	return "status"
}

// Description returns a description of the command
func (cmd StatusCmd) Description() string {
	return "Show the Git working directory status."
}

// RequiresRepo indicates this command requires a Dolt repository
func (cmd StatusCmd) RequiresRepo() bool {
	return true
}

// Docs returns the documentation for this command
func (cmd StatusCmd) Docs() *cli.CommandDocumentation {
	ap := cmd.ArgParser()
	return cli.NewCommandDocumentation(statusDocs, ap)
}

// ArgParser returns the argument parser for this command
func (cmd StatusCmd) ArgParser() *argparser.ArgParser {
	ap := argparser.NewArgParserWithMaxArgs(cmd.Name(), 0)

	ap.SupportsFlag("verbose", "v", "Show detailed information about tables")
	ap.SupportsFlag("show-sizes", "s", "Show estimated table sizes")
	ap.SupportsFlag("porcelain", "", "Give the output in porcelain format (machine-readable)")

	return ap
}

// EventType returns the type of the event to log
func (cmd StatusCmd) EventType() eventsapi.ClientEventType {
	return eventsapi.ClientEventType_STATUS
}

// GitStatusInfo contains information about the Git status
type GitStatusInfo struct {
	StagedTables    []string
	ModifiedTables  []string
	UntrackedTables []string
	LastExportTime  *time.Time
	RemoteURL       string
	CurrentBranch   string
}

// TableStatusInfo contains detailed information about a table's status
type TableStatusInfo struct {
	Name          string
	Status        string // "staged", "modified", "untracked"
	RowCount      int64
	EstimatedSize int64
	LastModified  time.Time
}

// Exec executes the git status command
func (cmd StatusCmd) Exec(ctx context.Context, commandStr string, args []string, dEnv *env.DoltEnv, cliCtx cli.CliContext) int {
	ap := cmd.ArgParser()
	help, _ := cli.HelpAndUsagePrinters(cli.CommandDocsForCommandString(commandStr, statusDocs, ap))
	apr := cli.ParseArgsOrDie(ap, args, help)

	verbose := apr.Contains("verbose")
	showSizes := apr.Contains("show-sizes")
	porcelain := apr.Contains("porcelain")

	// Gather status information
	statusInfo, err := gatherGitStatusInfo(ctx, dEnv, showSizes || verbose)
	if err != nil {
		return commands.HandleVErrAndExitCode(errhand.BuildDError("failed to gather status information: %v", err).Build(), nil)
	}

	// Display status
	if porcelain {
		displayPorcelainStatus(statusInfo)
	} else {
		displayHumanStatus(statusInfo, verbose, showSizes)
	}

	return 0
}

// gatherGitStatusInfo collects all status information
func gatherGitStatusInfo(ctx context.Context, dEnv *env.DoltEnv, includeDetails bool) (*GitStatusInfo, error) {
	// Get all tables in the repository
	root, err := dEnv.WorkingRoot(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get working root: %v", err)
	}

	allTables, err := root.GetTableNames(ctx, doltdb.DefaultSchemaName, true)
	if err != nil {
		return nil, fmt.Errorf("failed to get table names: %v", err)
	}

	// Load staged tables
	gitStagingDir := filepath.Join(dEnv.GetDoltDir(), ".dolt-git")
	stagedTables, err := loadStagedTables(gitStagingDir)
	if err != nil {
		// If we can't load staged tables, assume none are staged
		stagedTables = []string{}
	}

	// Load last export information
	lastExportTime, err := getLastExportTime(gitStagingDir)
	if err != nil {
		lastExportTime = nil // No previous export
	}

	// Determine table statuses
	var modifiedTables, untrackedTables []string

	for _, table := range allTables {
		if contains(stagedTables, table) {
			// Table is staged, no need to categorize further
			continue
		}

		// Check if table has been modified since last export
		if lastExportTime != nil {
			// TODO: Check actual table modification time
			// For now, assume all non-staged tables are modified
			modifiedTables = append(modifiedTables, table)
		} else {
			// No previous export, so table is untracked
			untrackedTables = append(untrackedTables, table)
		}
	}

	// Get remote information
	remoteURL, err := getConfiguredRemote(gitStagingDir)
	if err != nil {
		remoteURL = ""
	}

	// TODO: Get current branch from Dolt
	currentBranch := "main"

	return &GitStatusInfo{
		StagedTables:    stagedTables,
		ModifiedTables:  modifiedTables,
		UntrackedTables: untrackedTables,
		LastExportTime:  lastExportTime,
		RemoteURL:       remoteURL,
		CurrentBranch:   currentBranch,
	}, nil
}

// displayHumanStatus displays status in human-readable format
func displayHumanStatus(status *GitStatusInfo, verbose, showSizes bool) {
	// Header
	cli.Println(color.CyanString("On branch %s", status.CurrentBranch))

	if status.RemoteURL != "" {
		cli.Println(color.CyanString("Remote: %s", status.RemoteURL))
	}

	if status.LastExportTime != nil {
		cli.Println(color.CyanString("Last export: %s", status.LastExportTime.Format("2006-01-02 15:04:05")))
	} else {
		cli.Println(color.YellowString("No previous Git export found"))
	}

	cli.Println() // Empty line

	// Staged changes
	if len(status.StagedTables) > 0 {
		cli.Println(color.GreenString("Changes staged for Git commit:"))
		for _, table := range status.StagedTables {
			if verbose || showSizes {
				// TODO: Get actual table info
				cli.Println(color.GreenString("  staged:    %s", table))
			} else {
				cli.Println(color.GreenString("  %s", table))
			}
		}
		cli.Println() // Empty line
	}

	// Modified tables
	if len(status.ModifiedTables) > 0 {
		cli.Println(color.RedString("Tables not staged for Git commit:"))
		cli.Println(color.RedString("  (use \"dolt git add <table>...\" to stage for commit)"))
		cli.Println()
		for _, table := range status.ModifiedTables {
			if verbose || showSizes {
				cli.Println(color.RedString("  modified:  %s", table))
			} else {
				cli.Println(color.RedString("  %s", table))
			}
		}
		cli.Println() // Empty line
	}

	// Untracked tables
	if len(status.UntrackedTables) > 0 {
		cli.Println(color.RedString("Untracked tables:"))
		cli.Println(color.RedString("  (use \"dolt git add <table>...\" to include in Git commits)"))
		cli.Println()
		for _, table := range status.UntrackedTables {
			if verbose || showSizes {
				cli.Println(color.RedString("  untracked: %s", table))
			} else {
				cli.Println(color.RedString("  %s", table))
			}
		}
		cli.Println() // Empty line
	}

	// Summary and suggestions
	totalTables := len(status.StagedTables) + len(status.ModifiedTables) + len(status.UntrackedTables)
	if totalTables == 0 {
		cli.Println(color.GreenString("No tables in repository"))
	} else if len(status.StagedTables) == 0 {
		if len(status.ModifiedTables) > 0 || len(status.UntrackedTables) > 0 {
			cli.Println(color.YellowString("No changes staged for Git commit"))
			cli.Println(color.CyanString("Use 'dolt git add .' to stage all tables"))
		}
	} else {
		cli.Println(color.GreenString("Ready to commit %d table(s) to Git", len(status.StagedTables)))
		cli.Println(color.CyanString("Use 'dolt git commit -m \"message\"' to commit staged changes"))
	}
}

// displayPorcelainStatus displays status in machine-readable format
func displayPorcelainStatus(status *GitStatusInfo) {
	for _, table := range status.StagedTables {
		cli.Println("A  " + table)
	}
	for _, table := range status.ModifiedTables {
		cli.Println(" M " + table)
	}
	for _, table := range status.UntrackedTables {
		cli.Println("?? " + table)
	}
}

// getLastExportTime gets the timestamp of the last Git export
func getLastExportTime(gitStagingDir string) (*time.Time, error) {
	exportInfoFile := filepath.Join(gitStagingDir, "last_export.txt")

	if _, err := os.Stat(exportInfoFile); os.IsNotExist(err) {
		return nil, fmt.Errorf("no export info found")
	}

	content, err := os.ReadFile(exportInfoFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read export info: %v", err)
	}

	timestamp, err := time.Parse(time.RFC3339, strings.TrimSpace(string(content)))
	if err != nil {
		return nil, fmt.Errorf("failed to parse export timestamp: %v", err)
	}

	return &timestamp, nil
}

// getConfiguredRemote gets the configured Git remote URL
func getConfiguredRemote(gitStagingDir string) (string, error) {
	remoteFile := filepath.Join(gitStagingDir, "remote_url.txt")

	if _, err := os.Stat(remoteFile); os.IsNotExist(err) {
		return "", fmt.Errorf("no remote configured")
	}

	content, err := os.ReadFile(remoteFile)
	if err != nil {
		return "", fmt.Errorf("failed to read remote info: %v", err)
	}

	return strings.TrimSpace(string(content)), nil
}

// saveExportTime saves the timestamp of a Git export
func saveExportTime(gitStagingDir string, timestamp time.Time) error {
	exportInfoFile := filepath.Join(gitStagingDir, "last_export.txt")
	content := timestamp.Format(time.RFC3339)
	return os.WriteFile(exportInfoFile, []byte(content), 0644)
}

// saveRemoteURL saves the Git remote URL
func saveRemoteURL(gitStagingDir, remoteURL string) error {
	remoteFile := filepath.Join(gitStagingDir, "remote_url.txt")
	return os.WriteFile(remoteFile, []byte(remoteURL), 0644)
}
