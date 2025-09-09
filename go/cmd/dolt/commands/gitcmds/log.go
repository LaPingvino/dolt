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
	"sort"
	"strconv"
	"strings"

	"github.com/fatih/color"

	"github.com/dolthub/dolt/go/cmd/dolt/cli"
	"github.com/dolthub/dolt/go/cmd/dolt/commands"
	"github.com/dolthub/dolt/go/cmd/dolt/errhand"
	"github.com/dolthub/dolt/go/libraries/doltcore/env"
	"github.com/dolthub/dolt/go/libraries/utils/argparser"
	eventsapi "github.com/dolthub/eventsapi_schema/dolt/services/eventsapi/v1alpha1"
)

var logDocs = cli.CommandDocumentationContent{
	ShortDesc: `Show Git commit history.`,
	LongDesc: `{{.EmphasisLeft}}dolt git log{{.EmphasisRight}} displays the commit history for Git operations.

This command shows commits created with {{.EmphasisLeft}}dolt git commit{{.EmphasisRight}} in reverse chronological order (newest first).

For each commit, the following information is displayed:
- Commit ID (hash)
- Author name and email
- Commit timestamp
- Commit message
- Tables included in the commit
- Associated Dolt commit information (if available)

The log helps track the history of changes that have been committed for Git operations and can be used to understand what data was exported when.

Output formats:
- Default: Full commit information with details
- {{.EmphasisLeft}}--oneline{{.EmphasisRight}}: Condensed one-line format
- {{.EmphasisLeft}}--stat{{.EmphasisRight}}: Include statistics about tables in each commit

Examples:
{{.EmphasisLeft}}# Show all Git commits{{.EmphasisRight}}
dolt git log

{{.EmphasisLeft}}# Show last 5 commits{{.EmphasisRight}}
dolt git log -n 5

{{.EmphasisLeft}}# Show commits in one-line format{{.EmphasisRight}}
dolt git log --oneline

{{.EmphasisLeft}}# Show commits with table statistics{{.EmphasisRight}}
dolt git log --stat

{{.EmphasisLeft}}# Show last 10 commits with statistics{{.EmphasisRight}}
dolt git log -n 10 --stat
`,
	Synopsis: []string{
		"[--oneline] [--stat] [-n {{.LessThan}}count{{.GreaterThan}}]",
	},
}

type LogCmd struct{}

// Name returns the name of the Dolt cli command
func (cmd LogCmd) Name() string {
	return "log"
}

// Description returns a description of the command
func (cmd LogCmd) Description() string {
	return "Show Git commit history."
}

// RequiresRepo indicates this command requires a Dolt repository
func (cmd LogCmd) RequiresRepo() bool {
	return true
}

// Docs returns the documentation for this command
func (cmd LogCmd) Docs() *cli.CommandDocumentation {
	ap := cmd.ArgParser()
	return cli.NewCommandDocumentation(logDocs, ap)
}

// ArgParser returns the argument parser for this command
func (cmd LogCmd) ArgParser() *argparser.ArgParser {
	ap := argparser.NewArgParserWithMaxArgs(cmd.Name(), 0)

	ap.SupportsString("n", "", "count", "Limit the number of commits to show")
	ap.SupportsFlag("oneline", "", "Show commits in one-line format")
	ap.SupportsFlag("stat", "", "Show statistics for each commit")
	ap.SupportsFlag("verbose", "v", "Show detailed information")

	return ap
}

// EventType returns the type of the event to log
func (cmd LogCmd) EventType() eventsapi.ClientEventType {
	return eventsapi.ClientEventType_LOG
}

// Exec executes the git log command
func (cmd LogCmd) Exec(ctx context.Context, commandStr string, args []string, dEnv *env.DoltEnv, cliCtx cli.CliContext) int {
	ap := cmd.ArgParser()
	help, _ := cli.HelpAndUsagePrinters(cli.CommandDocsForCommandString(commandStr, logDocs, ap))
	apr := cli.ParseArgsOrDie(ap, args, help)

	oneline := apr.Contains("oneline")
	showStat := apr.Contains("stat")
	verbose := apr.Contains("verbose")

	// Parse limit
	limit := -1 // No limit by default
	if limitStr := apr.GetValueOrDefault("n", ""); limitStr != "" {
		var err error
		limit, err = strconv.Atoi(limitStr)
		if err != nil || limit <= 0 {
			cli.PrintErrln(color.RedString("Error: Invalid limit value: %s", limitStr))
			return 1
		}
	}

	// Load and display commit history
	if err := showGitLog(ctx, dEnv, limit, oneline, showStat, verbose); err != nil {
		return commands.HandleVErrAndExitCode(errhand.BuildDError("failed to show git log: %v", err).Build(), nil)
	}

	return 0
}

// showGitLog displays the Git commit log
func showGitLog(ctx context.Context, dEnv *env.DoltEnv, limit int, oneline, showStat, verbose bool) error {
	gitStagingDir := filepath.Join(dEnv.GetDoltDir(), ".dolt-git")
	commitsDir := filepath.Join(gitStagingDir, "commits")

	// Check if commits directory exists
	if _, err := os.Stat(commitsDir); os.IsNotExist(err) {
		cli.Println(color.YellowString("No Git commits found"))
		cli.Println(color.CyanString("Use 'dolt git commit -m \"message\"' to create commits"))
		return nil
	}

	// Load all commits
	commits, err := loadAllCommits(commitsDir)
	if err != nil {
		return fmt.Errorf("failed to load commits: %v", err)
	}

	if len(commits) == 0 {
		cli.Println(color.YellowString("No Git commits found"))
		return nil
	}

	// Sort commits by timestamp (newest first)
	sort.Slice(commits, func(i, j int) bool {
		return commits[i].Timestamp.After(commits[j].Timestamp)
	})

	// Apply limit
	if limit > 0 && limit < len(commits) {
		commits = commits[:limit]
	}

	// Display commits
	for i, commit := range commits {
		if oneline {
			displayCommitOneline(commit)
		} else {
			displayCommitFull(commit, showStat, verbose)
			// Add separator between commits (except for the last one)
			if i < len(commits)-1 {
				cli.Println()
			}
		}
	}

	// Summary
	if verbose {
		totalCommits := len(commits)
		if limit > 0 && limit < totalCommits {
			cli.Println()
			cli.Println(color.CyanString("Showing %d of %d total commits", limit, totalCommits))
		}
	}

	return nil
}

// loadAllCommits loads all commit information from the commits directory
func loadAllCommits(commitsDir string) ([]*GitCommitInfo, error) {
	files, err := os.ReadDir(commitsDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read commits directory: %v", err)
	}

	var commits []*GitCommitInfo

	for _, file := range files {
		if !file.IsDir() && strings.HasSuffix(file.Name(), ".json") {
			commitID := strings.TrimSuffix(file.Name(), ".json")

			commit, err := loadCommitInfo(commitsDir, commitID)
			if err != nil {
				// Skip corrupted commit files
				continue
			}

			commits = append(commits, commit)
		}
	}

	return commits, nil
}

// displayCommitOneline displays a commit in one-line format
func displayCommitOneline(commit *GitCommitInfo) {
	shortID := commit.ID[:8]
	shortMessage := commit.Message
	if len(shortMessage) > 50 {
		shortMessage = shortMessage[:47] + "..."
	}

	tableCount := len(commit.Tables)
	tableText := "table"
	if tableCount != 1 {
		tableText = "tables"
	}

	cli.Printf("%s %s (%d %s)\n",
		color.YellowString(shortID),
		shortMessage,
		tableCount,
		tableText)
}

// displayCommitFull displays a commit with full details
func displayCommitFull(commit *GitCommitInfo, showStat, verbose bool) {
	// Header with commit ID
	cli.Println(color.YellowString("commit %s", commit.ID))

	// Author
	cli.Println(color.WhiteString("Author: %s", commit.Author))

	// Date
	cli.Println(color.WhiteString("Date:   %s", commit.Timestamp.Format("Mon Jan 2 15:04:05 2006 -0700")))

	// Dolt information if available
	if verbose && (commit.DoltCommit != "" || commit.DoltBranch != "") {
		if commit.DoltBranch != "" {
			cli.Println(color.CyanString("Dolt Branch: %s", commit.DoltBranch))
		}
		if commit.DoltCommit != "" {
			cli.Println(color.CyanString("Dolt Commit: %s", commit.DoltCommit))
		}
	}

	cli.Println()

	// Commit message (indented)
	messageLines := strings.Split(commit.Message, "\n")
	for _, line := range messageLines {
		cli.Println(color.WhiteString("    %s", line))
	}

	// Tables information
	if len(commit.Tables) > 0 {
		cli.Println()
		if showStat {
			tableText := "table"
			if len(commit.Tables) != 1 {
				tableText = "tables"
			}
			cli.Println(color.CyanString("    %d %s changed:", len(commit.Tables), tableText))
			for _, table := range commit.Tables {
				cli.Println(color.GreenString("     %s", table))
			}
		} else {
			tablesList := strings.Join(commit.Tables, ", ")
			if len(tablesList) > 60 {
				// Break long table lists
				cli.Println(color.CyanString("    Tables: %s", tablesList[:57]+"..."))
			} else {
				cli.Println(color.CyanString("    Tables: %s", tablesList))
			}
		}
	}
}

// Helper function to check if a commit exists
func commitExists(commitsDir, commitID string) bool {
	commitFile := filepath.Join(commitsDir, commitID+".json")
	_, err := os.Stat(commitFile)
	return !os.IsNotExist(err)
}

// Helper function to get the latest commit ID
func getLatestCommitFromLog(commitsDir string) (string, error) {
	commits, err := loadAllCommits(commitsDir)
	if err != nil {
		return "", err
	}

	if len(commits) == 0 {
		return "", fmt.Errorf("no commits found")
	}

	// Sort by timestamp and return the newest
	sort.Slice(commits, func(i, j int) bool {
		return commits[i].Timestamp.After(commits[j].Timestamp)
	})

	return commits[0].ID, nil
}

// Helper function to count total commits
func getTotalCommitCount(commitsDir string) (int, error) {
	files, err := os.ReadDir(commitsDir)
	if err != nil {
		return 0, err
	}

	count := 0
	for _, file := range files {
		if !file.IsDir() && strings.HasSuffix(file.Name(), ".json") {
			count++
		}
	}

	return count, nil
}
