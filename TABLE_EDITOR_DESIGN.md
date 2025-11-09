# Dolt Table Editor Design Document

## Overview

This document outlines the design for a built-in command line table editor and viewer for Dolt, providing an interactive interface for browsing, editing, and managing table data directly from the terminal.

## Goals

### Primary Goals
- **Interactive Data Exploration**: Browse table data with navigation, sorting, and filtering
- **In-place Editing**: Edit cell values, add/remove rows and columns directly
- **SQL Integration**: Seamless integration with Dolt's SQL engine for complex operations
- **Familiar Interface**: Excel/spreadsheet-like experience in the terminal
- **Version Control Aware**: Integration with Dolt's versioning and commit workflow

### Secondary Goals
- **Performance**: Handle large datasets efficiently with pagination and lazy loading
- **Accessibility**: Keyboard-driven interface with intuitive shortcuts
- **Extensibility**: Plugin architecture for custom data processors and views
- **Import/Export**: Quick data import/export from the editor interface

## Architecture

### Core Components

#### 1. **TUI Framework** (`bubbletea` + `lipgloss` + `bubbles`)
```go
// Core editor application structure
type TableEditor struct {
    Model       tea.Model
    Table       *TableViewModel
    SQL         *SQLEngine
    KeyBindings *KeyBindingManager
    StatusBar   *StatusBar
    Commands    *CommandProcessor
}
```

#### 2. **Table View Model**
```go
type TableViewModel struct {
    Schema      schema.Schema
    Data        []sql.Row
    Cursor      CursorPosition
    Selection   Selection
    Filters     []Filter
    Sort        SortConfig
    Pagination  PaginationConfig
    EditMode    EditMode
}

type CursorPosition struct {
    Row    int
    Column int
}

type Selection struct {
    StartRow, EndRow       int
    StartColumn, EndColumn int
    Type                   SelectionType // Cell, Row, Column, Range
}
```

#### 3. **SQL Engine Integration**
```go
type SQLEngine struct {
    Context    context.Context
    Database   *sql.Database
    Connection *sql.Context
    Cache      *QueryCache
}

// Methods for data operations
func (e *SQLEngine) LoadTableData(tableName string, limit, offset int) ([]sql.Row, error)
func (e *SQLEngine) UpdateCell(table, column string, rowId interface{}, value interface{}) error
func (e *SQLEngine) InsertRow(table string, values []interface{}) error
func (e *SQLEngine) DeleteRows(table string, conditions []string) error
```

#### 4. **Command System**
```go
type CommandProcessor struct {
    History []Command
    Macros  map[string][]Command
}

type Command interface {
    Execute(ctx *EditorContext) error
    Undo(ctx *EditorContext) error
    Description() string
}

// Example commands
type EditCellCommand struct {
    TableName string
    RowIndex  int
    ColIndex  int
    OldValue  interface{}
    NewValue  interface{}
}
```

## User Interface Design

### Layout Structure
```
┌─────────────────────────────────────────────────────────────────────────────┐
│ Table: users (15 rows) • Branch: main • Modified: 3 changes                │ Status Bar
├─────────────────────────────────────────────────────────────────────────────┤
│ ID▼  │ Name         │ Email              │ Age  │ City         │ Status     │ Table Header
├──────┼──────────────┼────────────────────┼──────┼──────────────┼────────────┤
│ 001  │ John Doe     │ john@example.com   │ 28   │ San Francisco│ Active     │
│ 002  │ Jane Smith   │ jane@example.com   │ 34   │ New York     │ Active     │
│ 003  │ Bob Johnson  │ bob@example.com    │ 45   │ Chicago      │ Inactive   │ Table Data
│ [004]│ Alice Brown  │ alice@example.com  │ 29   │ Seattle      │ Active     │ (Current Row)
│ 005  │ Charlie Lee  │ charlie@example.com│ 31   │ Boston       │ Active     │
├─────────────────────────────────────────────────────────────────────────────┤
│ [E]dit [A]dd Row [D]elete [F]ilter [S]ort [Q]uery [C]ommit [?]Help         │ Command Bar
└─────────────────────────────────────────────────────────────────────────────┘
```

### Key Bindings
| Key | Action | Mode |
|-----|--------|------|
| `↑↓←→` | Navigate cells | Navigation |
| `Enter` | Edit current cell | Navigation |
| `Esc` | Cancel edit/exit mode | Edit/Filter |
| `Tab` | Next cell, confirm edit | Edit |
| `Shift+Tab` | Previous cell | Navigation |
| `Space` | Select current row | Navigation |
| `Ctrl+A` | Select all | Navigation |
| `Ctrl+C` | Copy selection | Navigation |
| `Ctrl+V` | Paste | Navigation |
| `Delete` | Delete selected rows | Navigation |
| `Ctrl+Z` | Undo last action | Any |
| `Ctrl+Y` | Redo action | Any |
| `Ctrl+F` | Open filter dialog | Navigation |
| `Ctrl+S` | Save changes | Any |
| `F1` | Help screen | Any |
| `:` | SQL command mode | Navigation |

## Features

### 1. Data Navigation
- **Pagination**: Handle large tables with configurable page sizes (default: 100 rows)
- **Scrolling**: Smooth horizontal and vertical scrolling
- **Jump to**: Quick navigation to specific rows/columns
- **Search**: Find and highlight specific values across columns

### 2. Data Editing
- **Cell Editing**: Click or Enter to edit individual cells with type validation
- **Row Operations**: Insert, duplicate, and delete rows
- **Column Operations**: Add, rename, reorder, and delete columns
- **Batch Operations**: Apply changes to multiple selected cells
- **Data Validation**: Type checking and constraint validation during editing

### 3. Data Visualization
- **Column Sizing**: Auto-fit, manual resize, fixed width modes
- **Data Types**: Visual indicators for different data types (numbers, dates, strings, nulls)
- **Highlighting**: Color coding for modified cells, errors, and selected ranges
- **Sorting Indicators**: Visual cues for sorted columns with sort order

### 4. Filtering and Sorting
- **Column Filters**: Per-column filtering with type-appropriate controls
- **Multiple Sorts**: Multi-column sorting with priority indicators
- **Quick Filters**: Common filter presets (non-null, unique values, ranges)
- **Search Filters**: Text search across all columns

### 5. SQL Integration
- **Command Mode**: Press `:` to enter SQL command mode
- **Query Execution**: Run SELECT queries and view results in-place
- **Schema Inspection**: View table schema, indexes, and constraints
- **Join Preview**: Quick join operations with related tables

## Implementation Phases

### Phase 1: Core Infrastructure (2-3 weeks)
- [x] Set up bubbletea application structure
- [x] Implement basic table view model
- [x] Create SQL engine integration layer
- [x] Basic navigation (arrow keys, cursor movement)
- [x] Simple data loading and display

### Phase 2: Essential Editing (2-3 weeks)
- [ ] Cell editing with proper input handling
- [ ] Data type validation and conversion
- [ ] Row insertion and deletion
- [ ] Undo/redo system implementation
- [ ] Save changes to database

### Phase 3: Advanced Features (3-4 weeks)
- [ ] Column operations (add, delete, rename)
- [ ] Filtering system with UI
- [ ] Sorting (single and multi-column)
- [ ] Copy/paste functionality
- [ ] Selection handling (ranges, rows, columns)

### Phase 4: SQL Integration (2-3 weeks)
- [ ] Command mode for SQL queries
- [ ] Query result display
- [ ] Schema browsing
- [ ] Table switching/selection

### Phase 5: Polish and Performance (2-3 weeks)
- [ ] Performance optimization for large datasets
- [ ] Improved error handling and user feedback
- [ ] Help system and documentation
- [ ] Configuration and customization options
- [ ] Integration testing with Dolt workflows

## Technical Considerations

### Performance
- **Lazy Loading**: Load data on-demand to handle large tables
- **Virtual Scrolling**: Only render visible rows for memory efficiency
- **Query Optimization**: Efficient SQL queries for data operations
- **Caching**: Cache frequently accessed data and metadata

### Error Handling
- **Validation**: Real-time validation during editing with clear error messages
- **Recovery**: Graceful handling of SQL errors and connection issues
- **Rollback**: Ability to rollback changes on errors
- **Conflict Resolution**: Handle concurrent modifications gracefully

### Integration with Dolt
- **Version Awareness**: Show current branch and commit status
- **Change Tracking**: Highlight modified data that hasn't been committed
- **Commit Integration**: Quick commit of changes from editor
- **Branch Operations**: Switch branches from within editor

## Command Implementation

### Core Commands
```bash
# Launch table editor
dolt table edit <table_name>

# Launch with specific options
dolt table edit users --limit=50 --filter="status='active'"

# Launch in read-only mode
dolt table edit users --read-only

# Launch with specific columns
dolt table edit users --columns="id,name,email"
```

### Configuration
```toml
# ~/.dolt/config/editor.toml
[table_editor]
default_page_size = 100
auto_save = true
show_row_numbers = true
highlight_changes = true
vim_mode = false

[table_editor.colors]
header = "bold blue"
modified = "yellow"
error = "red"
selected = "reverse"

[table_editor.keybindings]
# Custom key bindings can be defined here
```

## Example Usage Workflows

### 1. Data Exploration
```bash
# Open table for browsing
dolt table edit customers

# Navigate and explore data
# Use filters: Ctrl+F -> "city = 'New York'"
# Sort by column: Click column header or use sort dialog
# Search for specific values: Ctrl+Shift+F
```

### 2. Data Entry
```bash
# Open table for editing
dolt table edit products

# Add new rows: Press 'A' to add row
# Edit cells: Navigate and press Enter
# Save changes: Ctrl+S or ':commit'
```

### 3. Data Analysis
```bash
# Open with SQL integration
dolt table edit sales_data

# Switch to command mode: Press ':'
# Run analysis: "SELECT region, SUM(amount) FROM sales_data GROUP BY region"
# View results in integrated viewer
```

### 4. Data Cleaning
```bash
# Open problematic dataset
dolt table edit messy_data

# Use filters to find issues: "column IS NULL"
# Batch edit selected cells: Select range, press Enter
# Validate changes: Built-in type checking
# Commit clean data: ':commit -m "Clean null values"'
```

## Dependencies

### Required Libraries
- `github.com/charmbracelet/bubbletea` - TUI framework
- `github.com/charmbracelet/lipgloss` - Styling and layout
- `github.com/charmbracelet/bubbles` - Pre-built UI components
- Existing Dolt SQL engine integration

### Optional Enhancements
- `github.com/atotto/clipboard` - System clipboard integration
- `github.com/mattn/go-runewidth` - Better Unicode text handling
- `github.com/rivo/tview` - Alternative TUI framework (fallback option)

## Success Criteria

### Minimum Viable Product
- [ ] Browse table data with navigation
- [ ] Edit individual cells with validation
- [ ] Add and delete rows
- [ ] Save changes to Dolt database
- [ ] Basic error handling and user feedback

### Full Feature Set
- [ ] Complete editing capabilities (rows, columns, cells)
- [ ] Advanced filtering and sorting
- [ ] SQL command integration
- [ ] Copy/paste and selection handling
- [ ] Performance with large datasets (10K+ rows)
- [ ] Integration with Dolt version control workflow

### Quality Metrics
- [ ] Response time < 100ms for navigation operations
- [ ] Memory usage < 50MB for 1K row tables
- [ ] Zero data loss during editing operations
- [ ] Intuitive interface (< 5 minutes to learn basic operations)
- [ ] Comprehensive test coverage (>80%)

## Future Enhancements

- **Visual Query Builder**: Drag-and-drop interface for creating joins and filters
- **Data Visualization**: Built-in charts and graphs for numeric data
- **Export Options**: Quick export to CSV, Excel, JSON formats
- **Collaboration Features**: Real-time collaborative editing
- **Plugin System**: Custom data processors and formatters
- **Mobile/Web Version**: Browser-based version of the editor
- **AI Integration**: Natural language queries and data suggestions

---

This design provides a solid foundation for implementing a comprehensive table editor that integrates seamlessly with Dolt's existing architecture while providing a modern, efficient interface for data manipulation and exploration.