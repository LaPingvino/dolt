# Dolt Git Integration - TODO & Issues

**Status:** 🔧 **NEEDS FIXES**  
**Date:** December 2024  
**Priority:** HIGH - Critical functionality issues discovered

---

## 🚨 Critical Issues Discovered

During the deployment of the holywritings/bahaiwritings dataset to GitHub, several critical issues were identified that prevent the Git integration from working properly in production:

### Issue #1: Empty CSV Export Files
**Status:** 🔴 CRITICAL  
**Description:** The Git export process is generating empty CSV files instead of actual data.

**Evidence:**
- GitHub repository shows CSV files in `data/` directory
- Files appear to be created but contain no actual table data
- Table metadata shows row counts, but exported files are empty

**Impact:** 
- Complete data loss during export process
- Git repositories unusable for actual data collaboration
- Users cannot access the exported data

**Root Cause Analysis Needed:**
- CSV export function may have streaming/buffer issues
- Table iteration might not be reading actual row data
- Chunking process could be creating empty chunks
- File writing permissions or path issues

### Issue #2: Git History Not Preserved
**Status:** 🔴 CRITICAL  
**Description:** Dolt commit history is completely ignored during Git export.

**Evidence:**
- Only shows single "export" commit in Git repository
- Rich Dolt commit history with meaningful messages lost
- No branch information or merge history preserved
- Historical context and data evolution completely missing

**Impact:**
- Loss of valuable version control information
- Cannot track data changes over time in Git
- Defeats primary purpose of version-controlled data sharing
- Collaboration workflows severely limited

**Root Cause Analysis Needed:**
- Export process only creates snapshot, not history
- No mapping between Dolt commits and Git commits
- Branch handling not implemented
- Merge history conversion missing

### Issue #3: Metadata vs Reality Mismatch
**Status:** 🟡 MEDIUM  
**Description:** Repository metadata shows correct information, but actual exports don't match.

**Evidence:**
- README.md shows correct table counts and row numbers
- Metadata files contain proper schema information
- But actual CSV files don't contain the data
- Disconnect between analysis phase and export phase

---

## 📋 Current Working Components

### ✅ Infrastructure That Works
- **Command registration** - All `dolt git` commands available
- **Authentication** - SSH keys, tokens working properly
- **Dry-run testing** - Process validation works
- **Metadata generation** - Schema and repository info correct
- **GitHub connectivity** - Push/pull operations successful
- **Chunking framework** - Size-based chunking logic in place
- **Progress reporting** - User feedback and logging working

### ✅ Successful Test Scenarios
- **Small synthetic data** - Works with manually created test tables
- **Command workflow** - `add`, `commit`, `status`, `log` all functional
- **Repository structure** - Proper directory layout created
- **README generation** - Human-readable documentation created

---

## 🔧 Required Fixes

### Priority 1: Fix CSV Data Export
**Estimated Effort:** High  
**Components to Fix:**
- `go/libraries/doltcore/git/chunking.go` - CSV export functions
- `go/cmd/dolt/commands/gitcmds/push.go` - Export pipeline
- Table reading and streaming logic
- File writing and buffer flushing

**Action Items:**
1. Debug CSV export pipeline with real data
2. Add comprehensive logging to export process
3. Test with various table sizes and types
4. Verify file writing and permissions
5. Add data validation during export
6. Test round-trip: export → import → verify

### Priority 2: Implement History Preservation
**Estimated Effort:** Very High  
**Components to Create/Modify:**
- New history mapping system
- Commit translation layer
- Branch handling logic
- Merge history preservation

**Action Items:**
1. Design Dolt commit → Git commit mapping
2. Implement iterative commit export (not just snapshot)
3. Handle branch creation and switching
4. Preserve commit messages, authors, timestamps
5. Map merge commits properly
6. Add commit graph visualization
7. Test with complex branching scenarios

### Priority 3: Enhanced Error Handling & Validation
**Estimated Effort:** Medium  
**Action Items:**
1. Add data validation at each export step
2. Verify CSV contents match source tables
3. Add rollback capability for failed exports
4. Improve error messages with specific guidance
5. Add integrity checks and warnings

### Priority 4: Round-Trip Testing & Validation
**Estimated Effort:** Medium  
**Action Items:**
1. Implement automated round-trip testing
2. Data comparison tools for export/import verification
3. Performance benchmarks for large datasets
4. Regression test suite for various table types

---

## 🧪 Testing Strategy

### Test Cases Needed
1. **Small tables** (< 1MB) - Basic functionality
2. **Medium tables** (1-50MB) - Single chunk export
3. **Large tables** (> 50MB) - Multi-chunk export
4. **Complex schemas** - Various data types, constraints
5. **Historical data** - Multiple commits, branches, merges
6. **Mixed workloads** - Multiple tables with different sizes

### Validation Requirements
- **Data integrity** - Every row and column preserved exactly
- **Schema preservation** - All constraints and types maintained
- **Performance** - Reasonable export times for large datasets
- **History fidelity** - All commits and branches represented
- **Human readability** - CSV files viewable and usable on GitHub

---

## 🎯 Implementation Plan

### Phase 1: Critical Bug Fixes (Week 1-2)
1. **Fix CSV export pipeline**
   - Debug empty file generation
   - Implement proper data streaming
   - Add validation and logging
   - Test with real datasets

2. **Basic history preservation**
   - Single branch commit history export
   - Preserve commit messages and metadata
   - Test with linear history

### Phase 2: Advanced Features (Week 3-4)  
1. **Full history support**
   - Multi-branch handling
   - Merge commit preservation
   - Complex history scenarios

2. **Enhanced validation**
   - Automated integrity checking
   - Round-trip testing
   - Performance optimization

### Phase 3: Production Readiness (Week 5-6)
1. **Comprehensive testing**
   - Large dataset validation
   - Performance benchmarking
   - Edge case handling

2. **Documentation and examples**
   - User guides for complex scenarios
   - Best practices documentation
   - Troubleshooting guides

---

## 🔍 Debugging Approach

### Immediate Next Steps
1. **Create minimal test case**
   - Single small table with known data
   - Step through export process manually
   - Identify exact point of failure

2. **Add detailed logging**
   - Log every step of CSV generation
   - Track table reading and file writing
   - Monitor memory usage and performance

3. **Test export components in isolation**
   - Test table reading separately
   - Test CSV writing separately  
   - Test chunking logic separately
   - Test file operations separately

### Long-term Investigation
1. **Review export architecture**
   - Analyze data flow through export pipeline
   - Identify bottlenecks and failure points
   - Compare with successful bundle export logic

2. **Study successful implementations**
   - Review how bundle export handles large data
   - Learn from CSV export in other Dolt components
   - Analyze streaming vs. batch processing trade-offs

---

## 📚 Resources and References

### Key Files to Examine
- `go/libraries/doltcore/git/chunking.go` - Core export logic
- `go/cmd/dolt/commands/gitcmds/push.go` - Push command implementation  
- `go/libraries/doltcore/table/` - Table reading interfaces
- `go/libraries/doltcore/schema/` - Schema handling

### Related Successful Implementations
- Bundle export/import logic
- CSV import/export commands
- Table streaming in SQL queries
- Commit history handling in core Dolt

---

## 🎉 Vision: What Success Looks Like

When these fixes are complete, the Git integration will enable:

1. **Perfect data fidelity** - Every byte exported equals every byte imported
2. **Complete history preservation** - Full Dolt commit graph visible in Git
3. **Seamless collaboration** - Teams can use Git workflows on data naturally
4. **Scale without limits** - Handle multi-GB datasets with automatic chunking
5. **Human accessibility** - Browse data directly on GitHub/GitLab
6. **Version control workflows** - Branch, merge, and diff data like code

The holywritings/bahaiwritings dataset will serve as the perfect proof-of-concept: a large, real-world dataset with rich history, successfully exported to GitHub and usable by anyone familiar with Git workflows.

---

**Next Actions:**
1. Stop current deployment (data is corrupted anyway)
2. Focus on fixing CSV export with small test data
3. Implement proper data validation
4. Once basic export works, tackle history preservation
5. Return to large dataset deployment when fixes are complete

**Goal:** Transform Dolt's Git integration from "almost working" to "production ready" for real-world data collaboration.