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
	"time"

	"github.com/fatih/color"

	"github.com/dolthub/dolt/go/cmd/dolt/cli"
	"github.com/dolthub/dolt/go/cmd/dolt/commands"
	"github.com/dolthub/dolt/go/cmd/dolt/errhand"
	"github.com/dolthub/dolt/go/libraries/doltcore/env"
	"github.com/dolthub/dolt/go/libraries/doltcore/table/bundle"
	"github.com/dolthub/dolt/go/libraries/utils/argparser"
	eventsapi "github.com/dolthub/eventsapi_schema/dolt/services/eventsapi/v1alpha1"
)

var createDocs = cli.CommandDocumentationContent{
	ShortDesc: `Create a bundle file from the current Dolt repository.`,
	LongDesc: `{{.EmphasisLeft}}dolt bundle create{{.EmphasisRight}} creates a bundle file containing the entire Dolt repository, including all commits, branches, and working set data.

A bundle file is a single SQLite file that contains:
- Complete repository history (all commits and branches)
- Current working set data
- Repository metadata and configuration

Bundle files can be used for:
- Sharing complete datasets as single files
- Creating portable backups of repositories
- Distributing data without requiring Git-style clone operations
- Working offline with complete repository copies

The bundle format is similar to Git bundles but optimized for Dolt's data model.
`,
	Synopsis: []string{
		"[-f] [--description {{.LessThan}}desc{{.GreaterThan}}] {{.LessThan}}bundle_file{{.GreaterThan}}",
	},
}

type CreateCmd struct{}

// Name returns the name of the Dolt cli command
func (cmd CreateCmd) Name() string {
	return "create"
}

// Description returns a description of the command
func (cmd CreateCmd) Description() string {
	return "Create a bundle file from the current repository."
}

// Docs returns the documentation for this command
func (cmd CreateCmd) Docs() *cli.CommandDocumentation {
	ap := cmd.ArgParser()
	return cli.NewCommandDocumentation(createDocs, ap)
}

// ArgParser returns the argument parser for this command
func (cmd CreateCmd) ArgParser() *argparser.ArgParser {
	ap := argparser.NewArgParserWithMaxArgs(cmd.Name(), 1)
	ap.ArgListHelp = append(ap.ArgListHelp, [2]string{"bundle_file", "Path to the bundle file to create (should end in .bundle)"})
	ap.SupportsFlag(forceParam, "f", "Overwrite existing bundle file if it exists")
	ap.SupportsString(descriptionParam, "", "description", "Description of the bundle contents")
	return ap
}

// RequiresRepo indicates whether this command requires a Dolt repository
func (cmd CreateCmd) RequiresRepo() bool {
	return true
}

// EventType returns the type of the event to log
func (cmd CreateCmd) EventType() eventsapi.ClientEventType {
	return eventsapi.ClientEventType_CLONE
}

// Exec executes the command
func (cmd CreateCmd) Exec(ctx context.Context, commandStr string, args []string, dEnv *env.DoltEnv, cliCtx cli.CliContext) int {
	ap := cmd.ArgParser()
	help, usage := cli.HelpAndUsagePrinters(cli.CommandDocsForCommandString(commandStr, createDocs, ap))
	apr := cli.ParseArgsOrDie(ap, args, help)

	if apr.NArg() == 0 {
		usage()
		return 1
	}

	bundlePath := apr.Arg(0)
	force := apr.Contains(forceParam)
	description := apr.GetValueOrDefault(descriptionParam, "Bundle created from Dolt repository")

	// Validate bundle path
	if !force && fileExists(bundlePath) {
		cli.PrintErrln(color.RedString("Bundle file already exists: %s", bundlePath))
		cli.PrintErrln("Use --force to overwrite existing files")
		return 1
	}

	// Ensure bundle has proper extension
	if filepath.Ext(bundlePath) == "" {
		bundlePath += bundle.DefaultBundleExt
		cli.PrintErrln(color.YellowString("Adding .bundle extension: %s", bundlePath))
	}

	// Get current branch and commit info
	currentBranch, err := dEnv.RepoStateReader().CWBHeadRef(ctx)
	if err != nil {
		verr := errhand.BuildDError("Failed to get current branch").AddCause(err).Build()
		return commands.HandleVErrAndExitCode(verr, usage)
	}
	branchName := currentBranch.GetPath()

	ddb := dEnv.DoltDB(ctx)
	commit, err := ddb.ResolveCommitRef(ctx, currentBranch)
	if err != nil {
		verr := errhand.BuildDError("Failed to resolve current commit").AddCause(err).Build()
		return commands.HandleVErrAndExitCode(verr, usage)
	}

	hash, err := commit.HashOf()
	if err != nil {
		verr := errhand.BuildDError("Failed to get commit hash").AddCause(err).Build()
		return commands.HandleVErrAndExitCode(verr, usage)
	}
	commitHash := hash.String()

	// Create bundle info
	bundleInfo := &bundle.BundleInfo{
		FormatVersion: bundle.BundleFormatVersion,
		CreatedAt:     time.Now(),
		Creator:       getUserName(),
		Description:   description,
		RepoRoot:      dEnv.GetDoltDir(),
		Branch:        branchName,
		CommitHash:    commitHash,
	}

	cli.PrintErrln(color.CyanString("Creating bundle file: %s", bundlePath))
	cli.PrintErrln(fmt.Sprintf("Repository: %s", dEnv.GetDoltDir()))
	cli.PrintErrln(fmt.Sprintf("Branch: %s", branchName))
	cli.PrintErrln(fmt.Sprintf("Commit: %s", commitHash[:12]))

	// Create the bundle
	err = bundle.CreateBundle(ctx, bundlePath, filepath.Dir(dEnv.GetDoltDir()), bundleInfo)
	if err != nil {
		verr := errhand.BuildDError("Failed to create bundle").AddCause(err).Build()
		return commands.HandleVErrAndExitCode(verr, usage)
	}

	// Get bundle file size for reporting
	if stat, err := os.Stat(bundlePath); err == nil {
		size := float64(stat.Size()) / (1024 * 1024) // Convert to MB
		cli.PrintErrln(color.GreenString("Successfully created bundle: %s (%.1f MB)", bundlePath, size))
	} else {
		cli.PrintErrln(color.GreenString("Successfully created bundle: %s", bundlePath))
	}

	cli.PrintErrln("")
	cli.PrintErrln("Bundle can be used with:")
	cli.PrintErrln(fmt.Sprintf("  dolt bundle clone %s <directory>", bundlePath))
	cli.PrintErrln(fmt.Sprintf("  dolt bundle info %s", bundlePath))

	return 0
}
