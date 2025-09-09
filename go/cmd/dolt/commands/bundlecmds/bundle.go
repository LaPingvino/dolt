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
	"os"

	"github.com/dolthub/dolt/go/cmd/dolt/cli"
	"github.com/dolthub/dolt/go/libraries/doltcore/env"
	"github.com/dolthub/dolt/go/libraries/utils/argparser"
	eventsapi "github.com/dolthub/eventsapi_schema/dolt/services/eventsapi/v1alpha1"
)

const (
	descriptionParam = "description"
	forceParam       = "force"
)

// fileExists checks if a file exists
func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// getUserName returns the current user name for bundle metadata
func getUserName() string {
	if user := os.Getenv("USER"); user != "" {
		return user
	}
	if user := os.Getenv("USERNAME"); user != "" {
		return user
	}
	if hostname, err := os.Hostname(); err == nil {
		return "user@" + hostname
	}
	return "unknown"
}

var bundleDocs = cli.CommandDocumentationContent{
	ShortDesc: `Work with Dolt bundle files.`,
	LongDesc: `{{.EmphasisLeft}}dolt bundle{{.EmphasisRight}} provides commands for creating, cloning, and inspecting Dolt bundle files.

A bundle file is a single SQLite file that contains a complete Dolt repository, including:
- All commits and branch history
- Current working set data
- Repository metadata and configuration

Bundle files are useful for:
- Sharing complete datasets as single files
- Creating portable backups of repositories
- Distributing data without requiring Git-style clone operations
- Working offline with complete repository copies

The bundle format is similar to Git bundles but optimized for Dolt's data model and includes the current working set data.

Available subcommands:
- {{.EmphasisLeft}}create{{.EmphasisRight}}: Create a bundle file from the current repository
- {{.EmphasisLeft}}clone{{.EmphasisRight}}: Clone a repository from a bundle file
- {{.EmphasisLeft}}info{{.EmphasisRight}}: Display information about a bundle file
`,
	Synopsis: []string{
		"create [-f] [--description {{.LessThan}}desc{{.GreaterThan}}] {{.LessThan}}bundle_file{{.GreaterThan}}",
		"clone [-f] {{.LessThan}}bundle_file{{.GreaterThan}} [{{.LessThan}}directory{{.GreaterThan}}]",
		"info {{.LessThan}}bundle_file{{.GreaterThan}}",
	},
}

type BundleCmd struct {
	Subcommands []cli.Command
}

var bundleCommands = []cli.Command{
	CreateCmd{},
	CloneCmd{},
	InfoCmd{},
}

// NewBundleCmd creates a new bundle command with all subcommands
func NewBundleCmd() BundleCmd {
	return BundleCmd{
		Subcommands: bundleCommands,
	}
}

// Name returns the name of the Dolt cli command
func (cmd BundleCmd) Name() string {
	return "bundle"
}

// Description returns a description of the command
func (cmd BundleCmd) Description() string {
	return "Work with Dolt bundle files."
}

// RequiresRepo indicates whether this command requires a Dolt repository
func (cmd BundleCmd) RequiresRepo() bool {
	return false // Bundle commands can work outside of repositories (clone, info)
}

// Docs returns the documentation for this command
func (cmd BundleCmd) Docs() *cli.CommandDocumentation {
	ap := cmd.ArgParser()
	return cli.NewCommandDocumentation(bundleDocs, ap)
}

// ArgParser returns the argument parser for this command
func (cmd BundleCmd) ArgParser() *argparser.ArgParser {
	ap := argparser.NewArgParserWithMaxArgs(cmd.Name(), 0)
	ap.SupportsString("help", "", "command", "Show help for a specific command")
	return ap
}

// EventType returns the type of the event to log
func (cmd BundleCmd) EventType() eventsapi.ClientEventType {
	return eventsapi.ClientEventType_CLONE
}

// Exec executes the command
func (cmd BundleCmd) Exec(ctx context.Context, commandStr string, args []string, dEnv *env.DoltEnv, cliCtx cli.CliContext) int {
	subCommandHandler := cli.NewSubCommandHandler(cmd.Name(), cmd.Description(), cmd.Subcommands)
	return subCommandHandler.Exec(ctx, commandStr, args, dEnv, cliCtx)
}
