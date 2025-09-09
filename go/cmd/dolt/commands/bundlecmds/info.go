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

package bundlecmds

import (
	"context"
	"fmt"
	"os"

	"github.com/fatih/color"

	"github.com/dolthub/dolt/go/cmd/dolt/cli"
	"github.com/dolthub/dolt/go/cmd/dolt/commands"
	"github.com/dolthub/dolt/go/cmd/dolt/errhand"
	"github.com/dolthub/dolt/go/libraries/doltcore/env"
	"github.com/dolthub/dolt/go/libraries/doltcore/table/bundle"
	"github.com/dolthub/dolt/go/libraries/utils/argparser"
	eventsapi "github.com/dolthub/eventsapi_schema/dolt/services/eventsapi/v1alpha1"
)

var infoDocs = cli.CommandDocumentationContent{
	ShortDesc: `Display information about a bundle file.`,
	LongDesc: `{{.EmphasisLeft}}dolt bundle info{{.EmphasisRight}} displays metadata and information about a bundle file.

This command shows:
- Bundle format version and creation details
- Creator and description information
- Source repository information (branch, commit hash)
- List of tables contained in the bundle
- Bundle file size

Use this command to inspect bundle files before cloning or to verify bundle contents.
`,
	Synopsis: []string{
		"{{.LessThan}}bundle_file{{.GreaterThan}}",
	},
}

type InfoCmd struct{}

// Name returns the name of the Dolt cli command
func (cmd InfoCmd) Name() string {
	return "info"
}

// Description returns a description of the command
func (cmd InfoCmd) Description() string {
	return "Display information about a bundle file."
}

// Docs returns the documentation for this command
func (cmd InfoCmd) Docs() *cli.CommandDocumentation {
	ap := cmd.ArgParser()
	return cli.NewCommandDocumentation(infoDocs, ap)
}

// ArgParser returns the argument parser for this command
func (cmd InfoCmd) ArgParser() *argparser.ArgParser {
	ap := argparser.NewArgParserWithMaxArgs(cmd.Name(), 1)
	ap.ArgListHelp = append(ap.ArgListHelp, [2]string{"bundle_file", "Path to the bundle file to inspect"})
	return ap
}

// RequiresRepo indicates whether this command requires a Dolt repository
func (cmd InfoCmd) RequiresRepo() bool {
	return false
}

// EventType returns the type of the event to log
func (cmd InfoCmd) EventType() eventsapi.ClientEventType {
	return eventsapi.ClientEventType_CLONE
}

// Exec executes the command
func (cmd InfoCmd) Exec(ctx context.Context, commandStr string, args []string, dEnv *env.DoltEnv, cliCtx cli.CliContext) int {
	ap := cmd.ArgParser()
	help, usage := cli.HelpAndUsagePrinters(cli.CommandDocsForCommandString(commandStr, infoDocs, ap))
	apr := cli.ParseArgsOrDie(ap, args, help)

	if apr.NArg() == 0 {
		usage()
		return 1
	}

	bundlePath := apr.Arg(0)

	// Validate bundle file exists
	if !fileExists(bundlePath) {
		cli.PrintErrln(color.RedString("Bundle file does not exist: %s", bundlePath))
		return 1
	}

	// Get file size
	stat, err := os.Stat(bundlePath)
	if err != nil {
		verr := errhand.BuildDError("Failed to stat bundle file").AddCause(err).Build()
		return commands.HandleVErrAndExitCode(verr, usage)
	}

	// Open bundle and read metadata
	reader, err := bundle.OpenBundleReader(bundlePath)
	if err != nil {
		verr := errhand.BuildDError("Failed to open bundle file").AddCause(err).Build()
		return commands.HandleVErrAndExitCode(verr, usage)
	}
	defer reader.Close(ctx)

	info := reader.GetBundleInfo()
	tables := reader.GetTables()

	// Display bundle information
	cli.Println(color.CyanString("Bundle File: %s", bundlePath))
	cli.Println()

	// File information
	fileSize := float64(stat.Size())
	var sizeStr string
	if fileSize < 1024 {
		sizeStr = fmt.Sprintf("%.0f bytes", fileSize)
	} else if fileSize < 1024*1024 {
		sizeStr = fmt.Sprintf("%.1f KB", fileSize/1024)
	} else if fileSize < 1024*1024*1024 {
		sizeStr = fmt.Sprintf("%.1f MB", fileSize/(1024*1024))
	} else {
		sizeStr = fmt.Sprintf("%.1f GB", fileSize/(1024*1024*1024))
	}

	cli.Println(color.YellowString("File Information:"))
	cli.Printf("  Size: %s\n", sizeStr)
	cli.Printf("  Modified: %s\n", stat.ModTime().Format("2006-01-02 15:04:05"))
	cli.Println()

	// Bundle metadata
	cli.Println(color.YellowString("Bundle Metadata:"))
	cli.Printf("  Format Version: %s\n", info.FormatVersion)
	cli.Printf("  Created: %s\n", info.CreatedAt.Format("2006-01-02 15:04:05"))
	cli.Printf("  Creator: %s\n", info.Creator)
	cli.Printf("  Description: %s\n", info.Description)
	cli.Println()

	// Source repository information
	cli.Println(color.YellowString("Source Repository:"))
	cli.Printf("  Branch: %s\n", info.Branch)
	if len(info.CommitHash) > 12 {
		cli.Printf("  Commit: %s (%s)\n", info.CommitHash[:12], info.CommitHash)
	} else {
		cli.Printf("  Commit: %s\n", info.CommitHash)
	}
	if info.RepoRoot != "" {
		cli.Printf("  Original Path: %s\n", info.RepoRoot)
	}
	cli.Println()

	// Tables information
	cli.Println(color.YellowString("Tables:"))
	if len(tables) == 0 {
		cli.Printf("  No tables found in bundle\n")
	} else {
		cli.Printf("  Total Tables: %d\n", len(tables))
		for i, table := range tables {
			if i < 10 { // Show first 10 tables
				cli.Printf("  - %s\n", table)
			} else if i == 10 {
				cli.Printf("  ... and %d more tables\n", len(tables)-10)
				break
			}
		}
	}
	cli.Println()

	// Usage suggestions
	cli.Println(color.GreenString("Usage:"))
	cli.Printf("  Clone this bundle:  dolt bundle clone %s [directory]\n", bundlePath)
	if len(tables) > 0 {
		cli.Printf("  Import table data:  dolt table import -c <table_name> %s\n", bundlePath)
	}

	return 0
}
