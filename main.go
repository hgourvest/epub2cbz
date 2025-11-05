package main

import (
	"archive/zip"
	"encoding/xml"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"runtime/debug"
	"strconv"
	"strings"
	"sync"

	"golang.org/x/net/html"
)

type Container struct {
	Rootfiles struct {
		Rootfile struct {
			FullPath string `xml:"full-path,attr"`
		} `xml:"rootfile"`
	} `xml:"rootfiles"`
}

type Package struct {
	Metadata Metadata `xml:"metadata"`
	Manifest struct {
		Items []struct {
			ID   string `xml:"id,attr"`
			Href string `xml:"href,attr"`
		} `xml:"item"`
	} `xml:"manifest"`
	Spine struct {
		Itemrefs []struct {
			IDRef string `xml:"idref,attr"`
		} `xml:"itemref"`
	} `xml:"spine"`
}

type Metadata struct {
	XMLName    xml.Name `xml:"metadata"`
	Identifier []string `xml:"http://purl.org/dc/elements/1.1/ identifier"`
	Title      []string `xml:"http://purl.org/dc/elements/1.1/ title"`
	Language   []string `xml:"http://purl.org/dc/elements/1.1/ language"`
	Creator    []string `xml:"http://purl.org/dc/elements/1.1/ creator"`
	Publisher  []string `xml:"http://purl.org/dc/elements/1.1/ publisher"`
	Date       []string `xml:"http://purl.org/dc/elements/1.1/ date"`
	Rights     []string `xml:"http://purl.org/dc/elements/1.1/ rights"`
	Series     []string `xml:"http://purl.org/dc/elements/1.1/ series"`
	SeriesID   []string `xml:"http://purl.org/dc/elements/1.1/ seriesid"`
	Number     []string `xml:"http://purl.org/dc/elements/1.1/ number"`
}

type XHTML struct {
	Body struct {
		Div struct {
			Img struct {
				Src string `xml:"src,attr"`
			} `xml:"img"`
		} `xml:"div"`
	} `xml:"body"`
}

type ComicInfo struct {
	XMLName             xml.Name              `xml:"ComicInfo"`
	Title               string                `xml:"Title,omitempty"`
	Series              string                `xml:"Series,omitempty"`
	Number              string                `xml:"Number,omitempty"`
	Count               int                   `xml:"Count,omitempty"`
	Volume              int                   `xml:"Volume,omitempty"`
	AlternateSeries     string                `xml:"AlternateSeries,omitempty"`
	AlternateNumber     string                `xml:"AlternateNumber,omitempty"`
	AlternateCount      int                   `xml:"AlternateCount,omitempty"`
	Summary             string                `xml:"Summary,omitempty"`
	Notes               string                `xml:"Notes,omitempty"`
	Year                int                   `xml:"Year,omitempty"`
	Month               int                   `xml:"Month,omitempty"`
	Day                 int                   `xml:"Day,omitempty"`
	Writer              string                `xml:"Writer,omitempty"`
	Penciller           string                `xml:"Penciller,omitempty"`
	Inker               string                `xml:"Inker,omitempty"`
	Colorist            string                `xml:"Colorist,omitempty"`
	Letterer            string                `xml:"Letterer,omitempty"`
	CoverArtist         string                `xml:"CoverArtist,omitempty"`
	Editor              string                `xml:"Editor,omitempty"`
	Publisher           string                `xml:"Publisher,omitempty"`
	Imprint             string                `xml:"Imprint,omitempty"`
	Genre               string                `xml:"Genre,omitempty"`
	Web                 string                `xml:"Web,omitempty"`
	PageCount           int                   `xml:"PageCount,omitempty"`
	LanguageISO         string                `xml:"LanguageISO,omitempty"`
	Format              string                `xml:"Format,omitempty"`
	BlackAndWhite       string                `xml:"BlackAndWhite,omitempty"`
	Manga               string                `xml:"Manga,omitempty"`
	Characters          string                `xml:"Characters,omitempty"`
	Teams               string                `xml:"Teams,omitempty"`
	Locations           string                `xml:"Locations,omitempty"`
	ScanInformation     string                `xml:"ScanInformation,omitempty"`
	StoryArc            string                `xml:"StoryArc,omitempty"`
	SeriesGroup         string                `xml:"SeriesGroup,omitempty"`
	AgeRating           string                `xml:"AgeRating,omitempty"`
	Pages               *ArrayOfComicPageInfo `xml:"Pages,omitempty"`
	CommunityRating     string                `xml:"CommunityRating,omitempty"`
	MainCharacterOrTeam string                `xml:"MainCharacterOrTeam,omitempty"`
	Review              string                `xml:"Review,omitempty"`
}

type ArrayOfComicPageInfo struct {
	Page []ComicPageInfo `xml:"Page"`
}

type ComicPageInfo struct {
	Image       int    `xml:"Image,attr"`
	Type        string `xml:"Type,attr,omitempty"`
	DoublePage  bool   `xml:"DoublePage,attr,omitempty"`
	ImageSize   int64  `xml:"ImageSize,attr,omitempty"`
	Key         string `xml:"Key,attr,omitempty"`
	Bookmark    string `xml:"Bookmark,attr,omitempty"`
	ImageWidth  int    `xml:"ImageWidth,attr,omitempty"`
	ImageHeight int    `xml:"ImageHeight,attr,omitempty"`
}

// createComicInfo creates a ComicInfo.xml structure from OPF metadata
func createComicInfo(metadata Metadata) *ComicInfo {
	comicInfo := &ComicInfo{
		Title:       getFirst(metadata.Title),
		Series:      getFirst(metadata.Series),
		Number:      getFirst(metadata.Number),
		Publisher:   getFirst(metadata.Publisher),
		LanguageISO: getFirst(metadata.Language),
		Notes:       "Generated from EPUB metadata",
	}

	// Extract year from date if possible
	if len(metadata.Date) > 0 {
		dateStr := metadata.Date[0]
		if len(dateStr) >= 4 {
			if year, err := strconv.Atoi(dateStr[:4]); err == nil {
				comicInfo.Year = year
			}
		}
	}

	// Set Manga to Yes if series is in Japanese (simplified heuristic)
	if comicInfo.Series != "" {
		// Check if the series title contains Japanese characters
		if containsJapanese(comicInfo.Series) {
			comicInfo.Manga = "Yes"
		} else {
			comicInfo.Manga = "No"
		}
	} else {
		comicInfo.Manga = "Unknown"
	}

	// Map creator to writer (or penciller if appropriate)
	creator := getFirst(metadata.Creator)
	if creator != "" {
		// For manga, often the creator is both writer and penciller
		comicInfo.Writer = creator
		comicInfo.Penciller = creator
	}

	// Set default values according to schema
	if comicInfo.BlackAndWhite == "" {
		comicInfo.BlackAndWhite = "Unknown"
	}
	if comicInfo.AgeRating == "" {
		comicInfo.AgeRating = "Unknown"
	}

	return comicInfo
}

// getFirst returns the first element of a slice or an empty string if the slice is empty
func getFirst(items []string) string {
	if len(items) > 0 {
		return items[0]
	}
	return ""
}

// containsJapanese checks if a string contains Japanese characters
func containsJapanese(s string) bool {
	for _, r := range s {
		if (r >= 0x3040 && r <= 0x309F) || // Hiragana
			(r >= 0x30A0 && r <= 0x30FF) || // Katakana
			(r >= 0x4E00 && r <= 0x9FBF) { // Kanji
			return true
		}
	}
	return false
}

// hasMetadata checks if there is any useful metadata to include in ComicInfo.xml
func hasMetadata(metadata Metadata) bool {
	return len(metadata.Title) > 0 ||
		len(metadata.Creator) > 0 ||
		len(metadata.Publisher) > 0 ||
		len(metadata.Series) > 0 ||
		len(metadata.Date) > 0 ||
		len(metadata.Language) > 0 ||
		len(metadata.Identifier) > 0 ||
		len(metadata.Number) > 0
}

// getVersion returns the version of the application
func getVersion() string {
	info, ok := debug.ReadBuildInfo()
	if ok && info.Main.Version != "" {
		return info.Main.Version
	}
	// If build info is not available, return a default version
	return "v1.0.0"
}

func main() {
	var recursive bool
	var showVersion bool
	flag.BoolVar(&recursive, "r", false, "process subdirectories recursively")
	flag.BoolVar(&showVersion, "v", false, "show version information")
	flag.Parse()

	if showVersion {
		version := getVersion()
		fmt.Printf("epub2cbz version %s\n", version)
		return
	}

	if len(flag.Args()) < 1 {
		log.Fatal("Usage: epub2cbz [-r] [-v] <epub_file.epub | source_dir> [output_dir]")
	}

	sourcePath := flag.Arg(0)
	var outputPath string
	if len(flag.Args()) > 1 {
		outputPath = flag.Arg(1)
	}

	// Check if source is a directory
	sourceInfo, err := os.Stat(sourcePath)
	if err != nil {
		log.Fatal("Error accessing source path:", err)
	}

	if sourceInfo.IsDir() {
		// Process all .epub files in the directory based on recursive flag
		processDirectory(sourcePath, outputPath, recursive)
	} else {
		// Process single .epub file
		if err := processFile(sourcePath, outputPath); err != nil {
			log.Fatal(err)
		}
	}
}

func processDirectory(sourceDir string, outputDir string, recursive bool) {
	var epubFiles []string

	if recursive {
		// Walk the directory recursively
		err := filepath.Walk(sourceDir, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if !info.IsDir() && strings.HasSuffix(strings.ToLower(path), ".epub") {
				epubFiles = append(epubFiles, path)
			}
			return nil
		})
		if err != nil {
			log.Fatal("Error walking directory:", err)
		}
	} else {
		// Only process files in the top-level directory (non-recursive)
		entries, err := os.ReadDir(sourceDir)
		if err != nil {
			log.Fatal("Error reading directory:", err)
		}
		for _, entry := range entries {
			if !entry.IsDir() && strings.HasSuffix(strings.ToLower(entry.Name()), ".epub") {
				epubFiles = append(epubFiles, filepath.Join(sourceDir, entry.Name()))
			}
		}
	}

	if len(epubFiles) == 0 {
		if recursive {
			log.Fatal("No .epub files found in directory or subdirectories:", sourceDir)
		} else {
			log.Fatal("No .epub files found in directory:", sourceDir)
		}
	}

	// Create output directory if specified
	if outputDir != "" {
		err := os.MkdirAll(outputDir, 0755)
		if err != nil {
			log.Fatal("Error creating output directory:", err)
		}
	}

	var wg sync.WaitGroup
	// Limit the number of goroutines to the number of available CPUs
	maxConcurrency := runtime.NumCPU()
	semaphore := make(chan struct{}, maxConcurrency)

	// Process each .epub file
	for _, epubPath := range epubFiles {
		wg.Add(1)
		semaphore <- struct{}{}

		go func(path string) {
			defer wg.Done()
			defer func() { <-semaphore }()

			fmt.Printf("Processing %s...\n", path)

			var finalOutputPath string
			if outputDir != "" {
				// Generate output path preserving directory structure if recursive
				if recursive {
					relPath, err := filepath.Rel(sourceDir, path)
					if err != nil {
						log.Printf("Error getting relative path for %s: %v", path, err)
						return
					}
					// Create corresponding output directory structure
					outputDirPath := filepath.Join(outputDir, filepath.Dir(relPath))
					err = os.MkdirAll(outputDirPath, 0755)
					if err != nil {
						log.Printf("Error creating output directory structure for %s: %v", path, err)
						return
					}
					// Generate output path in the output directory
					baseName := strings.TrimSuffix(filepath.Base(path), ".epub")
					finalOutputPath = filepath.Join(outputDirPath, baseName+".cbz")
				} else {
					// Just put output in the output directory without subdirectory structure
					baseName := strings.TrimSuffix(filepath.Base(path), ".epub")
					finalOutputPath = filepath.Join(outputDir, baseName+".cbz")
				}
			} else {
				// Use default naming in source directory
				finalOutputPath = ""
			}

			if err := processFile(path, finalOutputPath); err != nil {
				log.Printf("ERROR processing %s: %v", path, err)
			}
		}(epubPath)
	}

	wg.Wait()
}

// findAndOpenFile searches for a file by name in the zip archive and returns an open reader.
func findAndOpenFile(zipReader *zip.ReadCloser, fileName string) (io.ReadCloser, error) {
	for _, f := range zipReader.File {
		if f.Name == fileName {
			return f.Open()
		}
	}
	return nil, fmt.Errorf("file not found in archive: %s", fileName)
}

func processFile(epubPath string, outputPath string) error {
	// Validate input file
	if filepath.Ext(epubPath) != ".epub" {
		return fmt.Errorf("input file must have .epub extension")
	}

	// Generate output path if not provided
	if outputPath == "" {
		outputPath = epubPath[:len(epubPath)-len(".epub")] + ".cbz"
	}

	// Open the EPUB file
	zipReader, err := zip.OpenReader(epubPath)
	if err != nil {
		return fmt.Errorf("error opening EPUB file: %w", err)
	}
	defer zipReader.Close()

	// 1. Find the vol.opf file
	var volOPFPath string
	containerFile, err := findAndOpenFile(zipReader, "META-INF/container.xml")
	if err != nil {
		return fmt.Errorf("error finding container.xml: %w", err)
	}
	defer containerFile.Close()

	var container Container
	if err := xml.NewDecoder(containerFile).Decode(&container); err != nil {
		return fmt.Errorf("error decoding container.xml: %w", err)
	}
	volOPFPath = container.Rootfiles.Rootfile.FullPath

	if volOPFPath == "" {
		return fmt.Errorf("vol.opf file not found in container")
	}

	// 2. Read vol.opf to get the metadata and pages
	var pages []string
	var metadata Metadata
	opfFile, err := findAndOpenFile(zipReader, volOPFPath)
	if err != nil {
		return fmt.Errorf("error finding vol.opf: %w", err)
	}
	defer opfFile.Close()

	var pkg Package
	if err := xml.NewDecoder(opfFile).Decode(&pkg); err != nil {
		return fmt.Errorf("error decoding vol.opf: %w", err)
	}

	// Store the metadata for later use
	metadata = pkg.Metadata

	// Find hrefs of pages via spine
	pageMap := make(map[string]string)
	for _, item := range pkg.Manifest.Items {
		pageMap[item.ID] = item.Href
	}

	for _, ref := range pkg.Spine.Itemrefs {
		href, exists := pageMap[ref.IDRef]
		if exists {
			// Convert relative path to absolute path based on volOPFPath
			absPath := filepath.Join(filepath.Dir(volOPFPath), href)
			// Normalize path separators to forward slashes for ZIP/EPUB compatibility
			absPath = filepath.ToSlash(absPath)
			absPath = strings.TrimPrefix(absPath, "/")
			pages = append(pages, absPath)
		}
	}

	if len(pages) == 0 {
		return fmt.Errorf("no pages found in spine")
	}

	// 3. Open each page and extract images
	zipWriter, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("error creating ZIP file: %w", err)
	}
	defer zipWriter.Close()

	zipw := zip.NewWriter(zipWriter)
	defer zipw.Close()

	// Variables to track images
	imageIndex := 0
	var imgSrcs []string
	for _, pageHref := range pages {
		for _, f := range zipReader.File {
			if f.Name == pageHref {
				file, err := f.Open()
				if err != nil {
					log.Printf("Error opening %s: %v", pageHref, err)
					continue
				}
				// Read the content of the page
				content, err := io.ReadAll(file)
				file.Close() // Close the file immediately after reading
				if err != nil {
					log.Printf("Error reading %s: %v", pageHref, err)
					continue
				}

				// Extract images
				imgSrcs = extractImagesFromXHTML(string(content), pageHref, imgSrcs)

				break
			}
		}
	}
	for _, src := range imgSrcs {
		addImageToZip(zipw, zipReader, src, imageIndex, len(imgSrcs))
		imageIndex++
	}

	// Generate and add ComicInfo.xml to the ZIP if metadata exists
	if hasMetadata(metadata) {
		comicInfo := createComicInfo(metadata)
		comicInfoXML, err := xml.MarshalIndent(comicInfo, "", "  ")
		if err != nil {
			log.Printf("Error marshaling ComicInfo: %v", err)
		} else {
			// Add XML declaration to the beginning of the XML
			comicInfoContent := xml.Header + string(comicInfoXML)

			// Create the ComicInfo.xml entry in the ZIP
			comicInfoFile, err := zipw.Create("ComicInfo.xml")
			if err != nil {
				log.Printf("Error creating ComicInfo.xml in ZIP: %v", err)
			} else {
				_, err = comicInfoFile.Write([]byte(comicInfoContent))
				if err != nil {
					log.Printf("Error writing ComicInfo.xml to ZIP: %v", err)
				}
			}
		}
	}

	fmt.Printf("Images extracted to %s\n", outputPath)
	return nil
}

// extractImagesFromHTML extracts image paths from HTML content using XML parser
func extractImagesFromXHTML(htmlContent string, pageHref string, srcs []string) []string {
	doc, err := html.Parse(strings.NewReader(htmlContent))
	if err != nil {
		log.Printf("Error parsing HTML from %s: %v", pageHref, err)
		return srcs
	}

	var f func(*html.Node)
	f = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "img" {
			for _, a := range n.Attr {
				if a.Key == "src" {
					imgPath := filepath.Join(filepath.Dir(pageHref), a.Val)
					imgPath = filepath.ToSlash(imgPath)
					imgPath = strings.TrimPrefix(imgPath, "/")
					srcs = append(srcs, imgPath)
					break
				}
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			f(c)
		}
	}
	f(doc)
	return srcs
}

// normalizeImageName renames images with the format "pageX.extension"
func normalizeImageName(originalName string, index int, totalFiles int) string {
	// Extract file extension
	ext := filepath.Ext(originalName)

	// Calculate number of digits needed for total
	totalDigits := len(strconv.Itoa(totalFiles))

	// Create new name with prefix "page" + padded index
	return fmt.Sprintf("page%0*d%s", totalDigits, index, ext)
}

// addImageToZip adds an image from the EPUB to the output ZIP
func addImageToZip(zipw *zip.Writer, zipReader *zip.ReadCloser, imgPath string, imageIndex int, total int) {
	for _, f := range zipReader.File {
		if f.Name == imgPath {
			srcFile, err := f.Open()
			if err != nil {
				log.Printf("Error opening image %s: %v", imgPath, err)
				return
			}
			defer srcFile.Close()

			// Create entry in ZIP
			dstFile, err := zipw.Create(filepath.Base(normalizeImageName(imgPath, imageIndex, total)))
			if err != nil {
				log.Printf("Error creating entry in ZIP: %v", err)
				return
			}

			// Copy content
			_, err = io.Copy(dstFile, srcFile)
			if err != nil {
				log.Printf("Error copying image %s: %v", imgPath, err)
				return
			}
			return
		}
	}
	log.Printf("Image not found in EPUB: %s", imgPath)
}
