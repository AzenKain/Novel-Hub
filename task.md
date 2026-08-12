# NovelHub Extended Roadmap & Missing Features (task2.md)

This file tracks the outstanding feature gaps identified when comparing NovelHub with competitors like **Audiobookshelf, Tome, BookOrbit, Kavita, and Calibre-Web Automated**. 

---

## 📦 EPIC 9: Advanced Audiobooks & Podcast Management
> **Target Competitor**: Audiobookshelf (ABS)

- [x] **9.1 In-App Audio Merger**
  - [x] Support selecting multiple raw audio files (MP3, M4A, FLAC) from a folder or upload batch.
  - [x] Merge them into a single, chaptered `.m4b` file in a background worker task.
  - [x] Extract and preserve chapter markers based on original file splits or file names.
- [x] **9.2 Chapter Lookup Engine**
  - [x] Integrate with the **Audnexus API** to automatically look up chapter timings/names using the Audible ASIN.
  - [x] Add an in-app chapter editor allowing manual adjustments of chapter names and times.
- [x] **9.3 Podcast RSS Ingestion**
  - [x] Support subscribing to external podcast RSS feeds.
  - [x] Automatically check and download new episodes, saving them directly to the user's managed storage.

---

## 📈 EPIC 10: Gamified Reading Insights & Advanced Stats
> **Target Competitor**: Tome, Audiobookshelf

- [x] **10.1 Reading Streaks & Goals**
  - [x] Compute and display "Reading Streaks" (number of consecutive days read).
  - [x] Track personal reading goals (e.g. books per year or words per day progress bar).
- [x] **10.2 Smart Completion Estimator (ETA)**
  - [x] Analyze the user's reading pace (words or pages read per minute).
  - [x] Calculate and display the estimated time remaining to finish the current chapter and the entire book.
- [x] **10.3 Comprehensive Stats Center**
  - [x] Create a dedicated dashboard displaying library statistics (e.g., breakdown by genre, author, format, and publisher).
  - [x] Add listening history insights (listening hours by day/month, average speed) for audiobooks.

---

## 📡 EPIC 11: External Scrobbling & Integrations
> **Target Competitor**: BookOrbit, Kavita

- [ ] **11.1 Social Scrobbling**
  - [ ] Support syncing reading status (currently reading, completed, rating) with **StoryGraph** and **Hardcover** APIs.
  - [ ] Integrate with **Goodreads** API to update shelf status.
- [ ] **11.2 Readwise Integration**
  - [ ] Support automatic export of local highlights and annotations (mapped via EPUB CFI) to the **Readwise** API.
- [ ] **11.3 Obsidian / Logseq Connector**
  - [ ] Implement a markdown-export endpoint or webhook to push book annotations directly into personal knowledge bases.

---

## 🤖 EPIC 12: Library Automation & Formats Conversion
> **Target Competitor**: Calibre-Web Automated

- [ ] **12.1 Dynamic Format Converters**
  - [ ] Add support for converting files between different document formats (e.g. converting MOBI or PDF to EPUB) on upload or download request.
- [ ] **12.2 Fuzzy Duplicate Merging**
  - [ ] Enhance duplicate detection to fuzzy-match books by title/author/ISBN.
  - [ ] Support merging duplicate entries, grouping multiple formats under a single book metadata record.