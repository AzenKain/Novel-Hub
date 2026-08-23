package tex

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestTeXParser(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "paper.tex")

	content := `\documentclass{article}
\title{Quantum Mechanics Introduction}
\author{Richard Feynman}
\date{1965}
\begin{document}
\maketitle

\section{Introduction}
This is a \textbf{bold introduction} with \textit{italic text} and \underline{underlined text}.

\begin{center}
Centered poem or quotation
\end{center}

\section{Core Principles}
Here are the principles:
\begin{itemize}
\item Superposition
\item Entanglement
\end{itemize}

\end{document}`

	if err := os.WriteFile(path, []byte(content), 0600); err != nil {
		t.Fatalf("write tex: %v", err)
	}

	parser := NewParser()
	meta, err := parser.ParseMetadata(path)
	if err != nil {
		t.Fatalf("ParseMetadata: %v", err)
	}
	if meta.Title != "Quantum Mechanics Introduction" {
		t.Fatalf("title = %q, want Quantum Mechanics Introduction", meta.Title)
	}
	if meta.Author != "Richard Feynman" {
		t.Fatalf("author = %q, want Richard Feynman", meta.Author)
	}

	spine, err := parser.ParseSpine(path)
	if err != nil || len(spine) < 2 {
		t.Fatalf("ParseSpine expected >=2 sections, got %d (err: %v)", len(spine), err)
	}

	sec1, err := parser.GetChapterContent(path, spine[0].ContentPath)
	if err != nil {
		t.Fatalf("GetChapterContent sec 0: %v", err)
	}

	if !strings.Contains(sec1, "<b>bold introduction</b>") || !strings.Contains(sec1, "<i>italic text</i>") {
		t.Errorf("expected formatted text in sec1, got: %s", sec1)
	}
	if !strings.Contains(sec1, `<div align="center">`) {
		t.Errorf("expected centered div in sec1, got: %s", sec1)
	}
}

func TestTeXParserWithImagesAndFigures(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "paper_with_images.tex")
	imgPath := filepath.Join(dir, "chart.png")
	if err := os.WriteFile(imgPath, []byte("fake-image-png-data"), 0600); err != nil {
		t.Fatalf("write img: %v", err)
	}

	content := `\documentclass{article}
\title{Physics Paper}
\author{Marie Curie}
\begin{document}
\section{Experimental Setup}
Here is the experimental setup:
\begin{figure}
  \includegraphics{chart.png}
  \caption{Decay curve measurement}
\end{figure}

And standalone diagram:
\includegraphics[width=0.8\textwidth]{diagram.jpg}
\end{document}`

	if err := os.WriteFile(path, []byte(content), 0600); err != nil {
		t.Fatalf("write tex: %v", err)
	}

	parser := NewParser()
	images, err := parser.ListImages(path)
	if err != nil {
		t.Fatalf("ListImages: %v", err)
	}
	if len(images) == 0 || images[0] != "chart.png" {
		t.Errorf("expected chart.png in ListImages, got: %v", images)
	}

	asset, err := parser.GetAsset(path, "chart.png")
	if err != nil {
		t.Fatalf("GetAsset: %v", err)
	}
	if string(asset) != "fake-image-png-data" {
		t.Errorf("got asset data %q, want fake-image-png-data", string(asset))
	}

	spine, err := parser.ParseSpine(path)
	if err != nil || len(spine) == 0 {
		t.Fatalf("ParseSpine: %v", err)
	}

	secContent, err := parser.GetChapterContent(path, spine[0].ContentPath)
	if err != nil {
		t.Fatalf("GetChapterContent: %v", err)
	}

	if !strings.Contains(secContent, "<figure") || !strings.Contains(secContent, "chart.png") || !strings.Contains(secContent, "Decay curve measurement</figcaption>") {
		t.Errorf("expected figure with caption in HTML, got: %s", secContent)
	}
	if !strings.Contains(secContent, "diagram.jpg") {
		t.Errorf("expected diagram.jpg in HTML, got: %s", secContent)
	}
}
