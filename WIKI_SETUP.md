# GitHub Wiki Setup Instructions

## Overview

The documentation has been converted from nested files to GitHub Wiki format. The wiki pages are ready to be uploaded to https://github.com/nhalm/skimatik/wiki

## Wiki Pages Created

The following markdown files in the repository root are ready to be copied to the GitHub Wiki:

1. **Home.md** - Main wiki homepage
2. **Quick-Start-Guide.md** - Installation and basic usage
3. **Shared-Utilities-Guide.md** - Database operations and retry patterns  
4. **Embedding-Patterns.md** - Repository composition patterns
5. **Examples-and-Tutorials.md** - Hands-on learning with real applications
6. **Error-Handling-Guide.md** - Comprehensive error management
7. **Configuration-Reference.md** - Complete configuration options

## Manual Setup Steps

### 1. Enable Wiki (if not already enabled)
```bash
gh repo edit nhalm/skimatik --enable-wiki
```

### 2. Create Wiki Pages
Go to https://github.com/nhalm/skimatik/wiki and create each page:

1. Click "Create the first page"
2. Copy content from `Home.md` 
3. Click "Save Page"
4. For each additional page:
   - Click "New Page"
   - Use the filename without .md extension as the page title
   - Copy the content from the corresponding .md file
   - Click "Save Page"

### 3. Page Titles
- **Home** (from Home.md)
- **Quick Start Guide** (from Quick-Start-Guide.md)
- **Shared Utilities Guide** (from Shared-Utilities-Guide.md)
- **Embedding Patterns** (from Embedding-Patterns.md)  
- **Examples and Tutorials** (from Examples-and-Tutorials.md)
- **Error Handling Guide** (from Error-Handling-Guide.md)
- **Configuration Reference** (from Configuration-Reference.md)

### 4. Verify Links
The wiki pages contain cross-references using the format `[Page Title](Page-Title)`. GitHub will automatically convert these to working wiki links.

## After Wiki Setup

Once the wiki is set up:

1. Delete the .md files from the repository root (they're no longer needed)
2. Users can access documentation at https://github.com/nhalm/skimatik/wiki
3. The README.md already contains links to the wiki pages

## Benefits of Wiki Format

- **Better Navigation**: Sidebar navigation for easy browsing
- **Search Functionality**: Built-in search across all documentation  
- **Version Control**: Wiki pages are Git-controlled
- **Collaborative**: Team members can easily contribute
- **Integration**: Links directly from main repository
- **Organization**: Clean structure instead of nested files

The documentation is now properly organized and ready for the wiki format! 