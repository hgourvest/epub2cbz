# epub2cbz

A command-line tool to convert EPUB files to CBZ format (comic book archive format).

## Description

epub2cbz is a Go application that extracts images from EPUB files and packages them into CBZ (Comic Book ZIP) archives. This tool is particularly useful for converting digital comic books or manga from EPUB format to CBZ format for use with comic book readers.

## Features

- Extract images from EPUB files
- Preserve page order when extracting images
- Convert to CBZ format (ZIP archive with .cbz extension)
- Recursive directory processing (optional, disabled by default)
- Preserve directory structure in output (when processing directories recursively)
- Generate ComicInfo.xml metadata file (when EPUB contains metadata)

## Installation

To build from source:

1. Make sure you have Go installed (version 1.16 or higher)
2. Clone the repository:
   ```bash
   git clone <repository-url>
   cd epub2cbz
   ```
3. Build the binary:
   ```bash
   go build
   ```

## Usage

### Convert a single EPUB file
```bash
./epub2cbz <input.epub> [output.cbz]
```

### Convert all EPUB files in a directory and subdirectories (recursive)
```bash
./epub2cbz -r <input_directory> [output_directory]
```

### Convert only EPUB files in a specific directory (non-recursive, default behavior)
```bash
./epub2cbz <input_directory> [output_directory]
```

## Options

- `-r` (boolean): Process subdirectories recursively. Default is `false`.

## Examples

1. Convert a single file:
   ```bash
   ./epub2cbz my_comic.epub my_comic.cbz
   ```

2. Convert a single file with automatic output name:
   ```bash
   ./epub2cbz my_comic.epub
   # Creates my_comic.cbz in the same directory
   ```

3. Convert all EPUB files in a directory and subdirectories:
   ```bash
   ./epub2cbz -r /path/to/epubs /path/to/output
   ```

4. Convert only EPUB files in the top directory (not subdirectories, default):
   ```bash
   ./epub2cbz /path/to/epubs /path/to/output
   ```

## Output

The tool creates CBZ files with images named in sequential order (e.g., page001.jpg, page002.png, etc.) to ensure proper reading order in comic book readers.

When processing directories recursively, the output directory structure mirrors the input structure.

## Metadata Support

When EPUB files contain metadata (title, creator, publisher, series, etc.), the tool will automatically generate a ComicInfo.xml file in the output CBZ archive. This metadata enhances compatibility with comic book readers that support metadata display and organization.

The ComicInfo.xml file is only generated when the source EPUB contains useful metadata, avoiding unnecessary empty metadata files in the archive.