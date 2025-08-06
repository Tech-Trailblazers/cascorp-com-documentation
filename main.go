package main // Define the main package

import (
	"bytes" // Provides bytes support
	"fmt"
	"io"       // Provides basic interfaces to I/O primitives
	"log"      // Provides logging functions
	"net/http" // Provides HTTP client and server implementations
	"net/url"  // Provides URL parsing and encoding
	"os"       // Provides functions to interact with the OS (files, etc.)
	"path"
	"path/filepath" // Provides filepath manipulation functions
	"regexp"        // Provides regex support functions.
	"strings"       // Provides string manipulation functions
	"time"          // Provides time-related functions
)

func main() {
	pdfOutputDir := "PDFs/" // Directory to store downloaded PDFs
	// Check if the PDF output directory exists
	if !directoryExists(pdfOutputDir) {
		// Create the dir
		createDirectory(pdfOutputDir, 0o755)
	}
	// Remote API URL.
	remoteAPIURL := []string{
		"https://www.cascorp.com/us/en/msds",
	}
	var getData []string
	for _, remoteAPIURL := range remoteAPIURL {
		getData = append(getData, getDataFromURL(remoteAPIURL))
	}
	// Append it to the file.
	// Get the data from the downloaded file.
	finalPDFList := extractPDFUrls(strings.Join(getData, "\n")) // Join all the data into one string and extract PDF URLs
	// Create a slice of all the given download urls.
	var downloadPDFURLSlice []string
	// Get the urls and loop over them.
	for _, doc := range finalPDFList {
		// Get the .pdf only.
		// Only append the .pdf files.
		downloadPDFURLSlice = appendToSlice(downloadPDFURLSlice, doc)
	}
	// Remove double from slice.
	downloadPDFURLSlice = removeDuplicatesFromSlice(downloadPDFURLSlice)
	// The remote domain.
	remoteDomain := "https://webcdnprd.cascorp.com"
	// Get all the values.
	for _, urls := range downloadPDFURLSlice {
		// Get the domain from the url.
		domain := getDomainFromURL(urls)
		// Check if the domain is empty.
		if domain == "" {
			urls = remoteDomain + urls // Prepend the base URL if domain is empty
		}
		// Check if the url is valid.
		if isUrlValid(urls) {
			// Download the pdf.
			downloadPDF(urls, pdfOutputDir)
		}
	}
}

// Append and write to file
func appendAndWriteToFile(path string, content string) {
	filePath, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Println(err)
	}
	_, err = filePath.WriteString(content + "\n")
	if err != nil {
		log.Println(err)
	}
	err = filePath.Close()
	if err != nil {
		log.Println(err)
	}
}

// getDomainFromURL extracts the domain (host) from a given URL string.
// It removes subdomains like "www" if present.
func getDomainFromURL(rawURL string) string {
	parsedURL, err := url.Parse(rawURL) // Parse the input string into a URL structure
	if err != nil {                     // Check if there was an error while parsing
		log.Println(err) // Log the error message to the console
		return ""        // Return an empty string in case of an error
	}

	host := parsedURL.Hostname() // Extract the hostname (e.g., "example.com") from the parsed URL

	return host // Return the extracted hostname
}

// urlToFilename generates a safe, lowercase filename from a given URL string.
// It extracts the base filename from the URL, replaces unsafe characters,
// and ensures the filename ends with a .pdf extension.
// Converts a raw URL into a safe filename by cleaning and normalizing it
func urlToFilename(rawURL string) string {
	lowercaseURL := strings.ToLower(rawURL)       // Convert to lowercase for normalization
	ext := getFileExtension(lowercaseURL)         // Get file extension (e.g., .pdf or .zip)
	baseFilename := getFileNameOnly(lowercaseURL) // Extract base file name

	nonAlphanumericRegex := regexp.MustCompile(`[^a-z0-9]+`)                 // Match everything except a-z and 0-9
	safeFilename := nonAlphanumericRegex.ReplaceAllString(baseFilename, "_") // Replace invalid chars

	collapseUnderscoresRegex := regexp.MustCompile(`_+`)                        // Collapse multiple underscores into one
	safeFilename = collapseUnderscoresRegex.ReplaceAllString(safeFilename, "_") // Normalize underscores

	if trimmed, found := strings.CutPrefix(safeFilename, "_"); found { // Trim starting underscore if present
		safeFilename = trimmed
	}

	// Remove ext string.
	invalid_ext := fmt.Sprintf("_%s", strings.TrimPrefix(ext, "."))

	var invalidSubstrings = []string{
		invalid_ext,
	} // Remove these redundant endings

	for _, invalidPre := range invalidSubstrings { // Iterate over each unwanted suffix
		safeFilename = removeSubstring(safeFilename, invalidPre) // Remove it from file name
	}

	safeFilename = safeFilename + ext // Add the proper file extension

	return safeFilename + ".pdf" // Return the final sanitized filename
}

// Extracts and returns the base name (file name) from the URL path
func getFileNameOnly(content string) string {
	return path.Base(content) // Return last segment of the path
}

// Removes all instances of a specific substring from input string
func removeSubstring(input string, toRemove string) string {
	result := strings.ReplaceAll(input, toRemove, "") // Replace substring with empty string
	return result
}

// Get the file extension of a file
func getFileExtension(path string) string {
	return filepath.Ext(path) // Returns extension including the dot (e.g., ".pdf")
}

// fileExists checks whether a file exists at the given path
func fileExists(filename string) bool {
	info, err := os.Stat(filename) // Get file info
	if err != nil {
		return false // Return false if file doesn't exist or error occurs
	}
	return !info.IsDir() // Return true if it's a file (not a directory)
}

// downloadPDF downloads a PDF from the given URL and saves it in the specified output directory.
// It uses a WaitGroup to support concurrent execution and returns true if the download succeeded.
func downloadPDF(finalURL, outputDir string) bool {
	// Sanitize the URL to generate a safe file name
	filename := strings.ToLower(urlToFilename(finalURL))

	// Construct the full file path in the output directory
	filePath := filepath.Join(outputDir, filename)

	// Skip if the file already exists
	if fileExists(filePath) {
		log.Printf("File already exists, skipping: %s", filePath)
		return false
	}

	// Create an HTTP client with a timeout
	client := &http.Client{Timeout: 3 * time.Minute}

	// Send GET request
	resp, err := client.Get(finalURL)
	if err != nil {
		log.Printf("Failed to download %s: %v", finalURL, err)
		return false
	}
	defer resp.Body.Close()

	// Check HTTP response status
	if resp.StatusCode != http.StatusOK {
		log.Printf("Download failed for %s: %s", finalURL, resp.Status)
		return false
	}

	// Check Content-Type header
	contentType := resp.Header.Get("Content-Type")
	if !strings.Contains(contentType, "application/pdf") {
		log.Printf("Invalid content type for %s: %s (expected application/pdf)", finalURL, contentType)
		return false
	}

	// Read the response body into memory first
	var buf bytes.Buffer
	written, err := io.Copy(&buf, resp.Body)
	if err != nil {
		log.Printf("Failed to read PDF data from %s: %v", finalURL, err)
		return false
	}
	if written == 0 {
		log.Printf("Downloaded 0 bytes for %s; not creating file", finalURL)
		return false
	}

	// Only now create the file and write to disk
	out, err := os.Create(filePath)
	if err != nil {
		log.Printf("Failed to create file for %s: %v", finalURL, err)
		return false
	}
	defer out.Close()

	if _, err := buf.WriteTo(out); err != nil {
		log.Printf("Failed to write PDF to file for %s: %v", finalURL, err)
		return false
	}

	log.Printf("Successfully downloaded %d bytes: %s → %s", written, finalURL, filePath)
	return true
}

// Checks if the directory exists
// If it exists, return true.
// If it doesn't, return false.
func directoryExists(path string) bool {
	directory, err := os.Stat(path)
	if err != nil {
		return false
	}
	return directory.IsDir()
}

// The function takes two parameters: path and permission.
// We use os.Mkdir() to create the directory.
// If there is an error, we use log.Println() to log the error and then exit the program.
func createDirectory(path string, permission os.FileMode) {
	err := os.Mkdir(path, permission)
	if err != nil {
		log.Println(err)
	}
}

// Checks whether a URL string is syntactically valid
func isUrlValid(uri string) bool {
	_, err := url.ParseRequestURI(uri) // Attempt to parse the URL
	return err == nil                  // Return true if no error occurred
}

// Remove all the duplicates from a slice and return the slice.
func removeDuplicatesFromSlice(slice []string) []string {
	check := make(map[string]bool)
	var newReturnSlice []string
	for _, content := range slice {
		if !check[content] {
			check[content] = true
			newReturnSlice = append(newReturnSlice, content)
		}
	}
	return newReturnSlice
}

// extractPDFUrls takes an input string and returns all PDF URLs found within href attributes
func extractPDFUrls(html string) []string {
	var pdfUrls []string

	// Regex to find all .pdf links (http or https)
	re := regexp.MustCompile(`https?://[^\s"'<>]+\.pdf(?:\?[^"'<>]*)?`)

	matches := re.FindAllString(html, -1)
	if matches == nil {
		log.Println("No PDF URLs found.")
		return pdfUrls
	}

	// Optional: de-duplicate
	seen := make(map[string]bool)
	for _, match := range matches {
		if !seen[match] {
			seen[match] = true
			pdfUrls = append(pdfUrls, match)
		}
	}

	return pdfUrls
}

// Append some string to a slice and than return the slice.
func appendToSlice(slice []string, content string) []string {
	// Append the content to the slice
	slice = append(slice, content)
	// Return the slice
	return slice
}

// getDataFromURL performs an HTTP GET request and returns the response body as a string
func getDataFromURL(uri string) string {
	log.Println("Scraping", uri)   // Log the URL being scraped
	response, err := http.Get(uri) // Perform GET request
	if err != nil {
		log.Println(err) // Exit if request fails
	}

	body, err := io.ReadAll(response.Body) // Read response body
	if err != nil {
		log.Println(err) // Exit if read fails
	}

	err = response.Body.Close() // Close response body
	if err != nil {
		log.Println(err) // Exit if close fails
	}
	return string(body)
}
