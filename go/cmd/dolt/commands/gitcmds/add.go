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

	"github.com/fatih/color"

	"github.com/dolthub/dolt/go/cmd/dolt/cli"
	"github.com/dolthub/dolt/go/cmd/dolt/commands"
	"github.com/dolthub/dolt/go/cmd/dolt/errhand"
	"github.com/dolthub/dolt/go/libraries/doltcore/doltdb"
	"github.com/dolthub/dolt/go/libraries/doltcore/env"
	"github.com/dolthub/dolt/go/libraries/utils/argparser"
	eventsapi "github.com/dolthub/eventsapi_schema/dolt/services/eventsapi/v1alpha1"
)

var addDocs = cli.CommandDocumentationContent{
	ShortDesc: `Stage table changes for Git commit.`,
	LongDesc: `{{.EmphasisLeft}}dolt git add{{.EmphasisRight}} stages table changes for the next Git commit operation.

This command marks tables as ready to be included in the next {{.EmphasisLeft}}dolt git commit{{.EmphasisRight}} and {{.EmphasisLeft}}dolt git push{{.EmphasisRight}} operations.

The staging area tracks which tables have been modified and should be included in the next Git export. This provides control over which changes are committed together.

Tables can be staged individually by name, or all tables can be staged using {{.EmphasisLeft}}.{{.EmphasisRight}}

The staged changes are stored in the repository's Git staging information and persist until committed or reset.

Examples:
{{.EmphasisLeft}}# Stage a specific table{{.EmphasisRight}}
dolt git add users

{{.EmphasisLeft}}# Stage multiple tables{{.EmphasisRight}}
dolt git add users orders products

{{.EmphasisLeft}}# Stage all tables{{.EmphasisRight}}
dolt git add .

{{.EmphasisLeft}}# Stage all tables with verbose output{{.EmphasisRight}}
dolt git add . --verbose
`,
	Synopsis: []string{
		"{{.LessThan}}table-name{{.GreaterThan}}...",
		".",
	},
}

type AddCmd struct{}

// Name returns the name of the Dolt cli command
func (cmd AddCmd) Name() string {
	return "add"
}

// Description returns a description of the command
func (cmd AddCmd) Description() string {
	return "Stage table changes for Git commit."
}

// RequiresRepo indicates this command requires a Dolt repository
func (cmd AddCmd) RequiresRepo() bool {
	return true
}

// Docs returns the documentation for this command
func (cmd AddCmd) Docs() *cli.CommandDocumentation {
	ap := cmd.ArgParser()
	return cli.NewCommandDocumentation(addDocs, ap)
}

// ArgParser returns the argument parser for this command
func (cmd AddCmd) ArgParser() *argparser.ArgParser {
	ap := argparser.NewArgParserWithVariableArgs(cmd.Name())
	ap.ArgListHelp = append(ap.ArgListHelp, [2]string{"table-name", "Name of table to stage (use '.' to stage all tables)"})

	ap.SupportsFlag("verbose", "v", "Show detailed information about staged tables")
	ap.SupportsFlag("dry-run", "", "Show what would be staged without actually staging")

	return ap
}

// EventType returns the type of the event to log
func (cmd AddCmd) EventType() eventsapi.ClientEventType {
	return eventsapi.ClientEventType_ADD
}

// Exec executes the git add command
func (cmd AddCmd) Exec(ctx context.Context, commandStr string, args []string, dEnv *env.DoltEnv, cliCtx cli.CliContext) int {
	ap := cmd.ArgParser()
	help, usage := cli.HelpAndUsagePrinters(cli.CommandDocsForCommandString(commandStr, addDocs, ap))
	apr := cli.ParseArgsOrDie(ap, args, help)

	if apr.NArg() == 0 {
		usage()
		return 1
	}

	verbose := apr.Contains("verbose")
	dryRun := apr.Contains("dry-run")

	// Get all available tables
	root, err := dEnv.WorkingRoot(ctx)
	if err != nil {
		return commands.HandleVErrAndExitCode(errhand.BuildDError("failed to get working root: %v", err).Build(), nil)
	}

	allTables, err := root.GetTableNames(ctx, doltdb.DefaultSchemaName, true)
	if err != nil {
		return commands.HandleVErrAndExitCode(errhand.BuildDError("failed to get table names: %v", err).Build(), nil)
	}

	// Determine which tables to stage
	var tablesToStage []string

	for i := 0; i < apr.NArg(); i++ {
		arg := apr.Arg(i)
		if arg == "." {
			// Stage all tables
			tablesToStage = allTables
			break
		} else {
			// Check if table exists
			if !contains(allTables, arg) {
				cli.PrintErrln(color.RedString("Error: Table '%s' does not exist", arg))
				cli.PrintErrln(color.YellowString("Available tables: %s", strings.Join(allTables, ", ")))
				return 1
			}
			tablesToStage = append(tablesToStage, arg)
		}
	}

	if len(tablesToStage) == 0 {
		cli.PrintErrln(color.YellowString("No tables to stage"))
		return 0
	}

	// Remove duplicates
	tablesToStage = removeDuplicates(tablesToStage)

	if verbose {
		cli.Println(color.CyanString("Tables to stage: %s", strings.Join(tablesToStage, ", ")))
	}

	if dryRun {
		cli.Println(color.YellowString("Dry run - would stage %d table(s):", len(tablesToStage)))
		for _, table := range tablesToStage {
			cli.Println(color.CyanString("  %s", table))
		}
		return 0
	}

	// Stage the tables
	if err := stageTables(ctx, dEnv, tablesToStage, verbose); err != nil {
		return commands.HandleVErrAndExitCode(errhand.BuildDError("failed to stage tables: %v", err).Build(), nil)
	}

	// Show success message
	if len(tablesToStage) == 1 {
		cli.Println(color.GreenString("Staged table: %s", tablesToStage[0]))
	} else {
		cli.Println(color.GreenString("Staged %d tables: %s", len(tablesToStage), strings.Join(tablesToStage, ", ")))
	}

	cli.Println(color.CyanString("Use 'dolt git status' to see staged changes"))
	cli.Println(color.CyanString("Use 'dolt git commit -m \"message\"' to commit staged changes"))

	return 0
}

// stageTables stages the specified tables for Git operations
func stageTables(ctx context.Context, dEnv *env.DoltEnv, tables []string, verbose bool) error {
	// Create .dolt-git directory for staging information if it doesn't exist
	gitStagingDir := filepath.Join(dEnv.GetDoltDir(), ".dolt-git")
	if err := os.MkdirAll(gitStagingDir, 0755); err != nil {
		return fmt.Errorf("failed to create Git staging directory: %v", err)
	}

	// Load existing staged tables
	stagedTables, err := loadStagedTables(gitStagingDir)
	if err != nil {
		return fmt.Errorf("failed to load existing staged tables: %v", err)
	}

	// Add new tables to staging
	for _, table := range tables {
		if !contains(stagedTables, table) {
			stagedTables = append(stagedTables, table)
			if verbose {
				cli.Println(color.GreenString("  ✓ Staged table: %s", table))
			}
		} else {
			if verbose {
				cli.Println(color.YellowString("  • Table already staged: %s", table))
			}
		}
	}

	// Save updated staging information
	if err := saveStagedTables(gitStagingDir, stagedTables); err != nil {
		return fmt.Errorf("failed to save staged tables: %v", err)
	}

	return nil
}

// loadStagedTables loads the list of currently staged tables
func loadStagedTables(gitStagingDir string) ([]string, error) {
	stagingFile := filepath.Join(gitStagingDir, "staged_tables.txt")

	// If file doesn't exist, return empty list
	if _, err := os.Stat(stagingFile); os.IsNotExist(err) {
		return []string{}, nil
	}

	content, err := os.ReadFile(stagingFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read staged tables file: %v", err)
	}

	lines := strings.Split(strings.TrimSpace(string(content)), "\n")
	var tables []string
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" {
			tables = append(tables, line)
		}
	}

	return tables, nil
}

// saveStagedTables saves the list of staged tables
func saveStagedTables(gitStagingDir string, tables []string) error {
	stagingFile := filepath.Join(gitStagingDir, "staged_tables.txt")

	content := strings.Join(tables, "\n")
	if content != "" {
		content += "\n"
	}

	if err := os.WriteFile(stagingFile, []byte(content), 0644); err != nil {
		return fmt.Errorf("failed to write staged tables file: %v", err)
	}

	return nil
}

// Helper functions
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

func removeDuplicates(slice []string) []string {
	seen := make(map[string]bool)
	var result []string

	for _, item := range slice {
		if !seen[item] {
			seen[item] = true
			result = append(result, item)
		}
	}

	return result
}
