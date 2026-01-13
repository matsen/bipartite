// Package pdf handles PDF path resolution and opening.
package pdf

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
)

// Opener handles resolving and opening PDF files.
type Opener struct {
	pdfRoot   string
	pdfReader string
}

// NewOpener creates a new PDF opener with the given configuration.
func NewOpener(pdfRoot, pdfReader string) *Opener {
	if pdfReader == "" {
		pdfReader = "system"
	}
	return &Opener{
		pdfRoot:   pdfRoot,
		pdfReader: pdfReader,
	}
}

// ResolvePath resolves a relative PDF path to an absolute path.
func (o *Opener) ResolvePath(relativePath string) (string, error) {
	if o.pdfRoot == "" {
		return "", fmt.Errorf("pdf_root not configured")
	}
	if relativePath == "" {
		return "", fmt.Errorf("no PDF path specified")
	}

	fullPath := filepath.Join(o.pdfRoot, relativePath)

	// Check if file exists
	if _, err := os.Stat(fullPath); err != nil {
		if os.IsNotExist(err) {
			return "", fmt.Errorf("PDF not found: %s", fullPath)
		}
		return "", fmt.Errorf("checking PDF: %w", err)
	}

	return fullPath, nil
}

// Open opens a PDF file using the configured reader.
// The fullPath should be an absolute path to an existing PDF file.
func (o *Opener) Open(fullPath string) error {
	// Fail fast if file doesn't exist
	if _, err := os.Stat(fullPath); err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("PDF file does not exist: %s", fullPath)
		}
		return fmt.Errorf("checking PDF file: %w", err)
	}

	var cmd *exec.Cmd

	switch runtime.GOOS {
	case "darwin":
		cmd = o.darwinCommand(fullPath)
	case "linux":
		cmd = o.linuxCommand(fullPath)
	default:
		return fmt.Errorf("unsupported platform: %s", runtime.GOOS)
	}

	return cmd.Start()
}

// darwinCommand returns the command to open a PDF on macOS.
func (o *Opener) darwinCommand(path string) *exec.Cmd {
	switch o.pdfReader {
	case "skim":
		return exec.Command("open", "-a", "Skim", path)
	case "preview":
		return exec.Command("open", "-a", "Preview", path)
	default: // "system"
		return exec.Command("open", path)
	}
}

// linuxCommand returns the command to open a PDF on Linux.
func (o *Opener) linuxCommand(path string) *exec.Cmd {
	switch o.pdfReader {
	case "zathura":
		return exec.Command("zathura", path)
	case "evince":
		return exec.Command("evince", path)
	case "okular":
		return exec.Command("okular", path)
	default: // "system"
		return exec.Command("xdg-open", path)
	}
}
