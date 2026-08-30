package main

import (
	"archive/zip"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"novelhub/pkg/bookparser/epub"
)

func main() {
	fmt.Println("=== NovelHub Book Doctor & EPUB Repair Engine ===")

	if len(os.Args) < 2 {
		printUsage()
		return
	}

	command := os.Args[1]

	switch command {
	case "validate", "check":
		if len(os.Args) < 3 {
			fmt.Println("Usage: go run ./cmd/bookdoctor validate <path/to/book.epub>")
			os.Exit(1)
		}
		epubPath := os.Args[2]
		runValidate(epubPath)

	case "repair", "fix":
		repairCmd := flag.NewFlagSet("repair", flag.ExitOnError)
		outFlag := repairCmd.String("out", "", "Output path for repaired EPUB (defaults to <name>_repaired.epub)")
		_ = repairCmd.Parse(os.Args[2:])

		args := repairCmd.Args()
		if len(args) < 1 {
			fmt.Println("Usage: go run ./cmd/bookdoctor repair [--out output.epub] <path/to/book.epub>")
			os.Exit(1)
		}
		epubPath := args[0]
		outPath := *outFlag
		if outPath == "" {
			ext := filepath.Ext(epubPath)
			base := stringsTrimSuffix(epubPath, ext)
			outPath = base + "_repaired" + ext
		}
		runRepair(epubPath, outPath)

	case "test", "demo":
		runDemo()

	default:
		printUsage()
	}
}

func printUsage() {
	fmt.Println("Commands:")
	fmt.Println("  validate <book.epub>             Check and diagnose EPUB health and structural issues")
	fmt.Println("  repair [--out out.epub] <book.epub> Repair XML entities, unclosed tags, manifest, spine & mimetype")
	fmt.Println("  demo                             Run end-to-end demo on a synthetic corrupted EPUB")
}

func runValidate(epubPath string) {
	fmt.Printf("[*] Diagnosing EPUB: %s\n", epubPath)
	report, err := epub.ValidateEPUB(epubPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: Validation failed: %v\n", err)
		os.Exit(1)
	}

	printReport(report)
}

func runRepair(srcPath, dstPath string) {
	fmt.Printf("[*] Analyzing and repairing: %s\n", srcPath)
	res, err := epub.RepairEPUB(srcPath, dstPath, epub.DefaultRepairOptions())
	if err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: Repair failed: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("\n[+] Applied %d automatic structural fixes:\n", res.FixedCount)
	for _, log := range res.Logs {
		fmt.Printf("   • %s\n", log)
	}

	fmt.Printf("\n[+] Saved repaired EPUB to: %s\n", dstPath)
	fmt.Println("\n=== Post-Repair Diagnostic Report ===")
	printReport(res.Report)
}

func runDemo() {
	tmpDir, err := os.MkdirTemp("", "bookdoctor-demo-*")
	if err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: %v\n", err)
		os.Exit(1)
	}
	defer os.RemoveAll(tmpDir)

	brokenPath := filepath.Join(tmpDir, "corrupted_sample.epub")
	repairedPath := filepath.Join("data", "repaired_sample.epub")
	_ = os.MkdirAll("data", 0755)

	// Create corrupted EPUB
	f, err := os.Create(brokenPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: %v\n", err)
		os.Exit(1)
	}
	zw := zip.NewWriter(f)

	// Flaw 1: compressed mimetype
	mw, _ := zw.CreateHeader(&zip.FileHeader{Name: "mimetype", Method: zip.Deflate})
	_, _ = mw.Write([]byte("application/epub+zip"))

	// Flaw 2: container.xml
	cw, _ := zw.Create("META-INF/container.xml")
	_, _ = cw.Write([]byte(`<?xml version="1.0"?><container version="1.0" xmlns="urn:oasis:names:tc:opendocument:xmlns:container"><rootfiles><rootfile full-path="OEBPS/content.opf" media-type="application/oebps-package+xml"/></rootfiles></container>`))

	// Flaw 3: OPF with missing dc:language, missing file in manifest, missing spine itemref
	ow, _ := zw.Create("OEBPS/content.opf")
	_, _ = ow.Write([]byte(`<?xml version="1.0" encoding="UTF-8"?>
<package xmlns="http://www.idpf.org/2007/opf" unique-identifier="BookID" version="2.0">
  <metadata xmlns:dc="http://purl.org/dc/elements/1.1/">
    <dc:title>Demo Corrupted Book</dc:title>
  </metadata>
  <manifest>
    <item id="ch1" href="ch1.xhtml" media-type="application/xhtml+xml"/>
    <item id="ghost" href="ghost.xhtml" media-type="application/xhtml+xml"/>
  </manifest>
  <spine>
    <itemref idref="ch1"/>
    <itemref idref="ghost"/>
    <itemref idref="phantom"/>
  </spine>
</package>`))

	// Flaw 4: XHTML with unclosed <br>, <img>, &nbsp;, bare &
	ch1w, _ := zw.Create("OEBPS/ch1.xhtml")
	_, _ = ch1w.Write([]byte(`<html><head></head><body><h1>Chapter 1</h1><p>Text with&nbsp;HTML entity & lone & ampersand.<br><hr><img src="missing.jpg"></p></body></html>`))

	_ = zw.Close()
	_ = f.Close()

	fmt.Println("1. Validating Corrupted Demo EPUB...")
	initialReport, _ := epub.ValidateEPUB(brokenPath)
	printReport(initialReport)

	fmt.Println("\n2. Executing Auto-Repair Engine...")
	runRepair(brokenPath, repairedPath)
}

func printReport(report epub.ValidationReport) {
	status := "HEALTHY / VALID"
	if !report.Valid {
		status = "HAS ISSUES / INVALID"
	}
	fmt.Printf("Status:   %s\n", status)
	fmt.Printf("Summary:  %d Errors, %d Warnings, %d Info items\n", report.Errors, report.Warnings, report.Infos)
	if len(report.Issues) > 0 {
		fmt.Println("Issues Detected:")
		for i, issue := range report.Issues {
			fixableStr := ""
			if issue.Fixable {
				fixableStr = " [Auto-Fixable]"
			}
			fileStr := ""
			if issue.File != "" {
				fileStr = fmt.Sprintf(" (%s)", issue.File)
			}
			fmt.Printf("  %2d. [%-7s] %s%s: %s%s\n", i+1, issue.Severity, issue.Code, fileStr, issue.Message, fixableStr)
		}
	} else {
		fmt.Println("🎉 No structural issues detected. Book is in perfect condition!")
	}
}

func stringsTrimSuffix(s, suffix string) string {
	if suffix != "" && len(s) >= len(suffix) && s[len(s)-len(suffix):] == suffix {
		return s[:len(s)-len(suffix)]
	}
	return s
}
