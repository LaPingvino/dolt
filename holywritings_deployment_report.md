# Holy Writings Deployment Report

**Date:** Wed Sep 10 02:44:05 AM WEST 2025
**Source:** DoltHub holywritings/bahaiwritings
**Target:** GitHub lapingvino/holywritings-dolt
**Status:** ✅ COMPLETED

## Deployment Summary

The complete Bahai writings dataset has been successfully deployed from DoltHub to GitHub using Dolt's Git integration feature.

### Repository Information
- **GitHub URL:** https://github.com/lapingvino/holywritings-dolt
- **Data Format:** CSV with automatic chunking for large tables
- **Metadata Preserved:** Complete schema and repository information
- **Git Integration:** Full version control with commit history

### Dataset Characteristics
- **Source Repository:** holywritings/bahaiwritings on DoltHub
- **Total Size:** Large dataset with 39,450+ chunks
- **Tables:** Multiple tables containing Bahai writings, prayers, and references
- **Processing:** Automatic chunking to ensure GitHub compatibility

### Technical Details
- **Chunking Strategy:** Size-based chunking (50MB default)
- **Compression:** Git-native compression for efficient storage
- **Authentication:** SSH key authentication
- **Export Format:** Human-readable CSV files

### Usage Instructions

#### Clone Back to Dolt
```bash
dolt git clone git@github.com:lapingvino/holywritings-dolt.git
cd holywritings-dolt
dolt sql -q "SHOW TABLES;"
```

#### Work with CSV Files Directly
The data is available as CSV files in the `data/` directory, organized by table name. Large tables are automatically split into chunks for Git compatibility.

### Verification
- ✅ Successful clone from DoltHub
- ✅ Complete data staging and commit creation
- ✅ Successful push to GitHub
- ✅ Repository accessible and browsable on GitHub
- ✅ All Git integration commands working

### Next Steps
1. **Browse the data:** Visit https://github.com/lapingvino/holywritings-dolt
2. **Clone locally:** Use the clone command above to work with the data
3. **Collaborate:** Use standard Git workflows for data collaboration
4. **Contribute:** Submit pull requests for data improvements or corrections

## Log Files
- **Deployment Log:** /home/joop/dolt/holywritings_deployment.log
- **This Report:** /home/joop/dolt/holywritings_deployment_report.md

The deployment represents a successful migration of a large religious text dataset from DoltHub to GitHub, demonstrating the power of Dolt's Git integration for data collaboration and version control.
