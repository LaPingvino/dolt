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
	"path/filepath"

	"github.com/fatih/color"

	"github.com/dolthub/dolt/go/cmd/dolt/cli"
	"github.com/dolthub/dolt/go/cmd/dolt/commands"
	"github.com/dolthub/dolt/go/cmd/dolt/errhand"
	"github.com/dolthub/dolt/go/libraries/doltcore/env"
	"github.com/dolthub/dolt/go/libraries/doltcore/table/bundle"
	"github.com/dolthub/dolt/go/libraries/utils/argparser"
	eventsapi "github.com/dolthub/eventsapi_schema/dolt/services/eventsapi/v1alpha1"
)

var cloneDocs = cli.CommandDocumentationContent{
	ShortDesc: `Clone a Dolt repository from a bundle file.`,
	LongDesc: `{{.EmphasisLeft}}dolt bundle clone{{.EmphasisRight}} extracts a bundle file to create a new Dolt repository.

A bundle file contains a complete Dolt repository including:
- All commits and branch history
- Current working set data
- Repository metadata and configuration

After cloning from a bundle, you will have a fully functional Dolt repository that can be used normally with all Dolt commands.

If no target directory is specified, a directory will be created based on the bundle filename.
`,
	Synopsis: []string{
		"[-f] {{.LessThan}}bundle_file{{.GreaterThan}} [{{.LessThan}}directory{{.GreaterThan}}]",
	},
}

type CloneCmd struct{}

// Name returns the name of the Dolt cli command
func (cmd CloneCmd) Name() string {
	return "clone"
}

// Description returns a description of the command
func (cmd CloneCmd) Description() string {
	return "Clone a repository from a bundle file."
}

// Docs returns the documentation for this command
func (cmd CloneCmd) Docs() *cli.CommandDocumentation {
	ap := cmd.ArgParser()
	return cli.NewCommandDocumentation(cloneDocs, ap)
}

// ArgParser returns the argument parser for this command
func (cmd CloneCmd) ArgParser() *argparser.ArgParser {
	ap := argparser.NewArgParserWithMaxArgs(cmd.Name(), 2)
	ap.ArgListHelp = append(ap.ArgListHelp, [2]string{"bundle_file", "Path to the bundle file to clone from"})
	ap.ArgListHelp = append(ap.ArgListHelp, [2]string{"directory", "Target directory for the cloned repository (optional)"})
	ap.SupportsFlag(forceParam, "f", "Remove target directory if it exists")
	return ap
}

// RequiresRepo indicates whether this command requires a Dolt repository
func (cmd CloneCmd) RequiresRepo() bool {
	return false
}

// EventType returns the type of the event to log
func (cmd CloneCmd) EventType() eventsapi.ClientEventType {
	return eventsapi.ClientEventType_CLONE
}

// Exec executes the command
func (cmd CloneCmd) Exec(ctx context.Context, commandStr string, args []string, dEnv *env.DoltEnv, cliCtx cli.CliContext) int {
	ap := cmd.ArgParser()
	help, usage := cli.HelpAndUsagePrinters(cli.CommandDocsForCommandString(commandStr, cloneDocs, ap))
	apr := cli.ParseArgsOrDie(ap, args, help)

	if apr.NArg() == 0 {
		usage()
		return 1
	}

	bundlePath := apr.Arg(0)
	force := apr.Contains(forceParam)

	// Validate bundle file exists
	if !fileExists(bundlePath) {
		cli.PrintErrln(color.RedString("Bundle file does not exist: %s", bundlePath))
		return 1
	}

	// Determine target directory
	var targetDir string
	if apr.NArg() > 1 {
		targetDir = apr.Arg(1)
	} else {
		// Generate directory name from bundle file
		bundleName := filepath.Base(bundlePath)
		if ext := filepath.Ext(bundleName); ext != "" {
			bundleName = bundleName[:len(bundleName)-len(ext)]
		}
		targetDir = bundleName
	}

	// Convert to absolute path
	absTargetDir, err := filepath.Abs(targetDir)
	if err != nil {
		verr := errhand.BuildDError("Invalid target directory: %s", targetDir).AddCause(err).Build()
		return commands.HandleVErrAndExitCode(verr, usage)
	}

	// Check if target directory exists
	if dirExists(absTargetDir) {
		if !force {
			cli.PrintErrln(color.RedString("Target directory already exists: %s", absTargetDir))
			cli.PrintErrln("Use --force to remove existing directory")
			return 1
		}

		cli.PrintErrln(color.YellowString("Removing existing directory: %s", absTargetDir))
		if err := os.RemoveAll(absTargetDir); err != nil {
			verr := errhand.BuildDError("Failed to remove existing directory").AddCause(err).Build()
			return commands.HandleVErrAndExitCode(verr, usage)
		}
	}

	// Read bundle info first to show what we're cloning
	reader, err := bundle.OpenBundleReader(bundlePath)
	if err != nil {
		verr := errhand.BuildDError("Failed to open bundle file").AddCause(err).Build()
		return commands.HandleVErrAndExitCode(verr, usage)
	}

	info := reader.GetBundleInfo()
	reader.Close(ctx)

	cli.PrintErrln(color.CyanString("Cloning bundle: %s", bundlePath))
	cli.PrintErrln(fmt.Sprintf("Target directory: %s", absTargetDir))
	cli.PrintErrln("Bundle info:")
	cli.PrintErrln(fmt.Sprintf("  Description: %s", info.Description))
	cli.PrintErrln(fmt.Sprintf("  Creator: %s", info.Creator))
	cli.PrintErrln(fmt.Sprintf("  Created: %s", info.CreatedAt.Format("2006-01-02 15:04:05")))
	cli.PrintErrln(fmt.Sprintf("  Branch: %s", info.Branch))
	cli.PrintErrln(fmt.Sprintf("  Commit: %s", info.CommitHash[:12]))
	cli.PrintErrln("")

	// Extract the bundle
	cli.PrintErrln("Extracting bundle...")
	err = bundle.ExtractBundle(ctx, bundlePath, absTargetDir)
	if err != nil {
		verr := errhand.BuildDError("Failed to extract bundle").AddCause(err).Build()
		return commands.HandleVErrAndExitCode(verr, usage)
	}

	cli.PrintErrln(color.GreenString("Successfully cloned repository to: %s", absTargetDir))
	cli.PrintErrln("")
	cli.PrintErrln("To start working with the repository:")
	cli.PrintErrln(fmt.Sprintf("  cd %s", targetDir))
	cli.PrintErrln("  dolt status")
	cli.PrintErrln("  dolt log --oneline -10")

	return 0
}

func dirExists(path string) bool {
	stat, err := os.Stat(path)
	return err == nil && stat.IsDir()
}
