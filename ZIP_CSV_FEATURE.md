# ZIP CSV Import/Export Feature for Dolt

This document describes the ZIP CSV import/export functionality added to Dolt, enabling users to work with ZIP archives containing CSV files, including GTFS (General Transit Feed Specification) transit data.

## Overview

The ZIP CSV feature adds support for importing and exporting data from ZIP archives containing CSV files. This is particularly useful for:

- **Transit Data (GTFS)**: Import/export GTFS feeds which are distributed as ZIP files containing multiple `.txt` files with CSV data
- **Data Distribution**: Package multiple CSV files or compress large CSV files for easier distribution
- **Legacy Systems**: Work with systems that distribute data in ZIP+CSV format

## Key Features

### Automatic Format Detection
- **CSV Files**: Detects `.csv` files within ZIP archives
- **GTFS Detection**: Automatically detects GTFS format by checking for required transit files (`agency.txt`, `stops.txt`, `routes.txt`, `trips.txt`, `stop_times.txt`)
- **File Extension Handling**: Supports both `.csv` and `.txt` file extensions

### Seamless Integration
- Uses existing `dolt table import` and `dolt table export` commands
- Supports all existing CSV import options (delimiters, headers, column mapping, etc.)
- Automatic schema inference from CSV headers

## Usage

### Importing from ZIP Files

```bash
# Import CSV files from a ZIP archive
dolt table import -c users data.zip

# Import with specific CSV options
dolt table import -c users data.zip --delim="|" --no-header --columns="id,name,age"

# Import GTFS transit data (auto-detected)
dolt table import -c transit gtfs_feed.zip

# Explicitly specify file type
dolt table import -c users data.unknown --file-type=zip
```

### Exporting to ZIP Files

```bash
# Export table to ZIP archive containing CSV file
dolt table export users exported_data.zip

# The ZIP will contain a file named "users.csv" with the table data
```

### File Format Support

The implementation supports:

| File Extension in ZIP | Format | Use Case |
|----------------------|---------|----------|
| `.csv` | Standard CSV | Regular data files |
| `.txt` | CSV format in TXT files | GTFS transit data |

## GTFS Support

GTFS (General Transit Feed Specification) files are automatically detected when a ZIP archive contains the required transit files:

**Required Files:**
- `agency.txt`
- `stops.txt` 
- `routes.txt`
- `trips.txt`
- `stop_times.txt`

**Optional Files** (also supported):
- `calendar.txt`
- `calendar_dates.txt`
- `fare_attributes.txt`
- `fare_rules.txt`
- `shapes.txt`
- `frequencies.txt`
- `transfers.txt`
- `feed_info.txt`

When GTFS format is detected, the system processes `.txt` files as CSV data.

## Technical Implementation

### Architecture

The feature is implemented through several key components:

1. **Data Format**: New `ZipCsvFile` format added to `mvdata.DataFormat`
2. **Reader**: `ZipCsvReader` implements `table.SqlRowReader` interface
3. **Writer**: `ZipCsvWriter` implements `table.SqlRowWriter` interface
4. **Integration**: Seamless integration with existing import/export infrastructure

### File Processing

1. **ZIP Archive Handling**: Uses Go's `archive/zip` package for reading/writing ZIP files
2. **Format Detection**: Examines ZIP contents to determine if it's GTFS or regular CSV
3. **Multi-File Processing**: Processes all CSV/TXT files within the ZIP sequentially
4. **Schema Inference**: Uses existing CSV schema inference for column types and names

### Code Location

- **Core Implementation**: `go/libraries/doltcore/table/untyped/zipcsv/`
- **Data Format Integration**: `go/libraries/doltcore/mvdata/`
- **Command Integration**: `go/cmd/dolt/commands/tblcmds/`
- **Tests**: `integration-tests/bats/zip-csv-import-export.bats`

## Examples

### Example 1: Regular CSV ZIP Import

```bash
# Create a ZIP file with CSV data
echo "id,name,age
1,John Doe,25
2,Jane Smith,30" > users.csv
zip users.zip users.csv

# Import into Dolt
dolt table import -c users users.zip

# Verify import
dolt sql -q "SELECT * FROM users"
```

### Example 2: GTFS Transit Data

```bash
# Download a GTFS feed (example)
wget https://example.com/transit/gtfs.zip

# Import transit data
dolt table import -c transit_data gtfs.zip

# Query transit data
dolt sql -q "SELECT * FROM transit_data LIMIT 10"
```

### Example 3: Round-trip Export/Import

```bash
# Export existing table to ZIP
dolt table export customers customers.zip

# Import exported data to new table
dolt table import -c customers_copy customers.zip

# Verify data integrity
dolt sql -q "SELECT COUNT(*) FROM customers"
dolt sql -q "SELECT COUNT(*) FROM customers_copy"
```

## Testing

Comprehensive integration tests are available in `integration-tests/bats/zip-csv-import-export.bats`:

```bash
# Run tests with required dependencies
nix develop --command bash -c "cd integration-tests && bats bats/zip-csv-import-export.bats"
```

Test coverage includes:
- Basic ZIP CSV import/export
- GTFS format detection
- CSV parsing options (delimiters, headers, columns)
- Round-trip data integrity
- File type parameter usage

## Development Environment

A Nix flake is provided for development with all required dependencies:

```bash
# Enter development environment
nix develop

# All necessary tools are available:
# - Go toolchain
# - Bats testing framework  
# - zip/unzip utilities
```

## Future Enhancements

Potential improvements for future versions:

1. **Multi-table GTFS Import**: Import each GTFS file as separate tables
2. **Compression Options**: Support for different compression levels
3. **Progress Reporting**: Progress indicators for large ZIP files
4. **Validation**: Built-in GTFS validation
5. **Incremental Updates**: Support for updating existing tables from ZIP files

## Compatibility

- **Dolt Version**: Compatible with current Dolt version
- **Go Version**: Requires Go 1.19+
- **Operating Systems**: All platforms supported by Dolt
- **File Formats**: Standard ZIP files with CSV/TXT content

## Status

✅ **COMPLETED** - Full ZIP CSV import/export functionality is implemented and tested.

This feature successfully addresses the wishlist item for CSV ZIP file import/export with GTFS support, making it easier to work with transit data and compressed CSV datasets in Dolt.