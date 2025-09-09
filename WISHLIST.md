# Wishlist for new features in Dolt

- Bundle support (probably sqlite-based)
- Clone from/to Git(hub)
- Built-in command line table editor with viewer (kinda like visidata, might even just rely on a local copy of it, but might just be a fancier `dolt sql` interface)
- Enable using jj style instead of git style (probably option to turn it on and add commands like dolt new and dolt desc)
- Import/export to csv zip files, including specifically gtfs files

Each wish list item is a separate issue and should be worked on in a separate branch. For every change, add a design description to this document and correct each section to describe progress on the item.

## JJ-style workflow

This section documents the design and implementation progress of the JJ-style workflow in Dolt.

### Design

The JJ-style workflow is an alternative to the Git-style workflow that is designed to be more intuitive and user-friendly. It is based on the following core concepts:

*   **Changes vs. Commits:** A "change" is a mutable description of a set of modifications to the database. A "commit" is an immutable snapshot of the database at a point in time. In the JJ-style workflow, the user primarily interacts with changes.
*   **Parallel Repository:** When `dolt.jj.enabled` is true, Dolt will maintain a parallel, hidden copy of the database in the `.dolt` directory. This copy will be used to store the immutable commits, while the working copy will be treated as a mutable commit.
*   **Operation Log:** Dolt will maintain an operation log that records every command that modifies the repository's history. This will be the foundation for the `dolt undo` command.
*   **Automatic Rebase:** When a commit is modified, all its descendants will be automatically rebased on top of the modified commit.
*   **Native Conflict Resolution:** When a merge conflict occurs, the conflicting files will be marked as "conflicted" in the working copy. The user will be able to resolve the conflicts by editing the files in the working copy. Once the conflicts are resolved, the user can run `dolt commit` to create a new commit that resolves the conflict.

### Configuration

The JJ-style workflow can be enabled by setting the `dolt.jj.enabled` configuration option to `true`.

### Commands

The following commands will be modified or added to support the JJ-style workflow:

*   **`dolt new`**: Creates a new, empty change and sets it as the current change.
*   **`dolt add`**: Only for tracking new tables.
*   **`dolt commit`**: Creates an immutable commit from the current "change."
*   **`dolt desc`**: Edits the description of the *current change*.
*   **`dolt status`**: Shows modifications in the current "change" and any conflict information.
*   **`dolt log`**: Shows the history of immutable commits.
*   **`dolt rebase`**: Reorders and modifies the history of "changes."
*   **`dolt pull` and `dolt fetch`**: Will automatically rebase local changes on top of remote changes to maintain a linear history.
*   **`dolt undo`**: Reverts the last operation.
*   **`dolt changes`**: Lists all "changes" in the repository.
*   **`dolt edit <commit>`**: Creates a new change that is a copy of the specified commit.

### Implementation Progress

**Phase 1: Core Commands**

*   [ ] `dolt new`
*   [ ] `dolt desc`
*   [ ] `dolt commit`
*   [ ] `dolt add`
*   [ ] `dolt status`
*   [ ] `dolt log`
*   [ ] `dolt changes`
*   [ ] `dolt edit`

**Phase 2: Automatic Rebase and Native Conflict Resolution**

*   [ ] Automatic Rebase
*   [ ] Native Conflict Resolution

**Phase 3: Advanced Commands**

*   [ ] `dolt undo`
*   [ ] `dolt rebase`


## Bundle support

A bundle is a single file that you can clone from and push to, like git bundles. This makes it easier to clone and share datasets. If this uses sqlite, the sqlite file could just contain a checkout plus a table with the contents of the .dolt directory. A dolt bundle fsck command could fix discrepancies that come from manual manipulation of the bundle outside of dolt.

### Design

Bundle files are SQLite-based archives that contain complete Dolt repositories, including:

* **Complete Repository Data:** All commits, branches, and repository history
* **Working Set Data:** Current table data and schemas
* **Compressed Storage:** Gzip compression for efficient file sizes
* **Metadata:** Creation info, descriptions, and source repository details

### Implementation Progress

**Phase 1: Core Bundle Infrastructure**

* [x] `BundleFile` data format implementation
* [x] SQLite-based bundle reader/writer with compression
* [x] Repository data archival (complete .dolt directory)
* [x] Bundle metadata storage and retrieval
* [x] Integration with existing data movement infrastructure

**Phase 2: Bundle Commands**

* [x] `dolt bundle create` - Create bundles from repositories
* [x] `dolt bundle clone` - Clone repositories from bundles  
* [x] `dolt bundle info` - Inspect bundle contents and metadata
* [x] Command integration and CLI interface
* [x] Final compilation and integration fixes

**Status:** ✅ **COMPLETED** - Full bundle functionality implemented and tested successfully.

**Usage Examples:**
```bash
# Create a bundle from current repository
dolt bundle create --description "Dataset v1.0" dataset.bundle

# Clone repository from bundle
dolt bundle clone dataset.bundle my-dataset

# View bundle information
dolt bundle info dataset.bundle
```

## Clone from/to Git(hub)

It would be nice to be able to clone a dataset from github or gitlab, or from a local git repository. This would be useful for sharing datasets with collaborators, and for cloning a dataset from a remote server without having to rely on Dolthub specifically, and to add support for gitea, gitlab and forgejo, instead of needing to do a Doltlab installation etc.

## Table editor with viewer

This is a feature that I've been wanting for a while, and I think it's a good idea to have. It would be nice to be able to view the data in a table, and edit it in a table editor. This would be useful for data entry, and for exploring the data. It would also be useful for making changes to the data, like adding a new column, or changing the type of a column. It's probably good to enable both sql commands and table editor commands, so that you can use the table editor to make changes to the data, but also use sql to query the data.

## Enable using jj style instead of git style

Many people struggle with the two step process of the staging area that Dolt copies from Git, and jj fixes this in a great way. It might using Dolt efficiently by people not used to git a lot easier, especially when paired with the table editor.

## Import/export to csv zip files, gtfs support

It would be massive to be able to version transit data and easily import and export it, also csv zip files are probably easier than sqlite based bundles to work with for many people.

### Design

This feature adds support for importing and exporting ZIP archives containing CSV files. The implementation includes:

* **ZIP CSV Format Support:** A new `ZipCsvFile` data format that handles ZIP archives containing CSV files
* **GTFS Auto-detection:** Automatic detection of GTFS (General Transit Feed Specification) files by examining ZIP contents for required .txt files (agency.txt, stops.txt, routes.txt, trips.txt, stop_times.txt)
* **Flexible File Extensions:** Support for both .csv files in ZIP archives and .txt files for GTFS compatibility
* **Unified Interface:** Integration with existing import/export commands using the same syntax

### Implementation

**Core Components:**

* **ZipCsvReader:** Implements `table.SqlRowReader` interface to read CSV files from ZIP archives
* **ZipCsvWriter:** Implements `table.SqlRowWriter` interface to write CSV files to ZIP archives  
* **Format Detection:** Automatically detects GTFS format by checking for required transit files
* **Schema Inference:** Uses existing CSV schema inference for column types and names

**Command Integration:**

* `dolt table import -c <table> <file.zip>` - Import CSV files from ZIP archive
* `dolt table export <table> <file.zip>` - Export table data to CSV file in ZIP archive
* Support for all existing CSV import options (--delimiter, --no-header, --columns, etc.)

### Implementation Progress

**Phase 1: Core ZIP CSV Support**

* [x] `ZipCsvFile` data format implementation
* [x] ZIP archive reading with CSV file filtering
* [x] GTFS format auto-detection based on file contents
* [x] Integration with existing import/export commands
* [x] Schema inference from CSV headers
* [x] Support for both .csv and .txt file extensions

**Phase 2: Testing and Validation**

* [x] Basic import functionality from ZIP archives
* [x] Basic export functionality to ZIP archives
* [x] GTFS file detection and processing
* [x] Integration with existing CSV parsing options

**Status:** ✅ **COMPLETED** - Full ZIP CSV import/export functionality is implemented and working.

**Usage Examples:**
```bash
# Import CSV files from a ZIP archive
dolt table import -c users data.zip

# Export table to CSV file in ZIP archive  
dolt table export users exported_data.zip

# Import GTFS transit data (auto-detected from .txt files)
dolt table import -c transit_data gtfs_feed.zip
```
