# 📋 NovelHub Development Roadmap & Task Specifications

Tài liệu này tổng hợp và phân loại các nhiệm vụ phát triển tiếp theo cho **NovelHub**. Tất cả các tính năng dưới đây được chọn lọc kỹ lưỡng, **100% không vi phạm bản quyền / thương hiệu**, tuân thủ các tiêu chuẩn mở (Open Standards) và đảm bảo an toàn pháp lý tuyệt đối cho dự án nguồn mở self-hosted.

---

## 🧭 Nguyên tắc chọn lọc & Bản quyền (Copyright Safety)

1. **Chuẩn mở & Không bản quyền (Open Standards Only):** Sử dụng các giao thức mở chuẩn quốc tế (IETF WebDAV RFC 4918, S3 API, Anki Open Format, IDPF EPUB 2/3 Specifications, Piper TTS ONNX models).
2. **Không cào web lậu / Trang thương mại (Zero Proprietary Web Scraping):** Loại bỏ các module crawl tiểu thuyết từ các trang thương mại có bản quyền. Toàn bộ nội dung sách và audiobook do người dùng tự quản lý và sở hữu cục bộ.
3. **Local-First & ByoK (Bring-Your-Own-Key):** Mọi tính năng AI xử lý trực tiếp qua Local LLM (Ollama) hoặc qua API key cá nhân do người dùng cấu hình trong Admin Settings.
4. **Kiến trúc chuẩn NovelHub:** Mọi tính năng mới bắt buộc tuân thủ quy chuẩn `AGENTS.md` (3 lớp Controller <-> Service <-> Repository, 100% SQLC queries, Cache-by-IDs + Singleflight, Fiber v3, Bun + React 19 + TanStack Query + Zustand, đồng bộ 16 ngôn ngữ i18n).

---

## 🎯 DANH SÁCH CÁC TRỤ CỘT TÍNH NĂNG ĐƯỢC CHỌN LỌC

```text
┌────────────────────────────────────────────────────────────────────────────────────────┐
│                        NOVELHUB ROADMAP - COPYRIGHT-SAFE TRACKS                        │
├────────────────────────┬────────────────────────┬──────────────────────────────────────┤
│ 1. AI Reading Copilot  │ 2. Open Protocols      │ 3. Storage & Infrastructure          │
│ • Chat with Book (RAG) │ • WebDAV Server        │ • S3 / R2 / MinIO Storage Driver     │
│ • Character Dossier    │ • Calibre Server API   │ • Offsite Snapshot Cloud Backup      │
│ • Smart Chapter Recap  │ • Anki Flashcard (.apkg│ • Book Doctor & EPUB Structure Repair│
├────────────────────────┴────────────────────────┴──────────────────────────────────────┤
│ 4. Neural Local Audio Engine                    │ 5. Social & Collaborative Reading    │
│ • Server-side Piper/Kokoro TTS to M4B           │ • Book Clubs & Group Reading         │
│ • Background Audio Synthesis Queue              │ • Shared Highlights & Discussion     │
└─────────────────────────────────────────────────┴──────────────────────────────────────┘
```

---

## 🚀 Track 1: In-Book AI Copilot & Semantic Assistant (Hỏi đáp & Trợ lý AI)

### Mục tiêu

Cung cấp khả năng tương tác thông minh với sách mà người dùng sở hữu trong thư viện. Sử dụng SQLite FTS5 làm bộ trích xuất ngữ cảnh RAG cục bộ và cho phép kết nối với Local LLM (Ollama) hoặc Cloud LLM (OpenAI, Gemini, Anthropic, DeepSeek).

### Nhiệm vụ chi tiết

- [ ] **1.1. Cấu hình AI Provider trong Admin Settings**
  - **Database:** Bổ sung setting keys trong `app_settings` (`ai.provider`, `ai.endpoint`, `ai.api_key`, `ai.model`, `ai.temperature`, `ai.max_tokens`, `ai.enabled`).
  - **Backend (`internal/services/aiService.go`):** Tạo service kết nối đa LLM (hỗ trợ OpenAI format, Anthropic format, Gemini format, Ollama format) sử dụng `pkg/netx` SSRF-safe client.
  - **Admin UI (`web/src/components/admin/settings/AISettingsModal.tsx`):** Giao diện cấu hình, kiểm tra kết nối API key và chọn model.

- [ ] **1.2. Chat With Your Book (In-Book RAG Assistant)**
  - **Backend:** Endpoint `POST /api/v1/reader/books/:id/chat` nhận câu hỏi + chapter range.
  - **RAG Engine:** Trích xuất đoạn văn liên quan nhất thông qua `chapters_fts` FTS5 ranking (BM25) và tạo prompt context gửi sang LLM.
  - **Frontend UI (`web/src/components/reader/ReaderAiCopilotPanel.tsx`):** Panel chat bên phải reader với stream response, trích dẫn số trang/chương sách làm dẫn chứng.

- [ ] **1.3. Character Dossier & Lore Explorer (Hồ sơ Nhân vật & Thuật ngữ)**
  - **Database:** Tạo bảng `book_entities` (`id`, `book_id`, `entity_type`, `name`, `aliases_json`, `description`, `first_chapter_id`).
  - **Backend:** Background job phân tích trích xuất thực thể theo từng chương mà không làm lộ nội dung các chương sau.
  - **Frontend:** Popup thông tin nhân vật/thuật ngữ khi bấm vào tên nhân vật trong nội dung sách.

- [ ] **1.4. Smart Chapter Recap ("Previously on...")**
  - **Backend:** Tự động sinh tóm tắt 3-5 câu về diễn biến các chương gần nhất trước vị trí đọc hiện tại (`reading_progress`).
  - **Frontend:** Hiển thị banner/card tóm tắt nhanh khi người đọc mở lại sách sau hơn 7 ngày không đọc.

- [x] **1.5. Anki Flashcard Export (.apkg / CSV)**
  - **Backend (`pkg/anki/`):** Xuất toàn bộ highlights/annotations của cuốn sách sang file `.apkg` (Anki SQLite deck) hoặc file CSV chuẩn Anki gồm Mặt trước (từ/câu trích), Mặt sau (ghi chú/nghĩa), Ngữ cảnh (đoạn văn trích dẫn).
  - **Frontend:** Nút xuất "Anki Deck (.apkg)" và "CSV (.csv)" trong Highlights Export Card.
  - **Native Tool:** CLI probe `cmd/ankiprobe` kiểm tra và sinh deck Anki độc lập.

---

## 🌐 Track 2: Open Protocols & Ecosystem Expansion (Giao thức Mở & Hệ sinh thái)

### Mục tiêu

Mở rộng khả năng tương thích của NovelHub với mọi ứng dụng đọc sách trên thị trường thông qua các tiêu chuẩn mở quốc tế mà không cần app phụ thuộc.

### Nhiệm vụ chi tiết

- [x] **2.1. WebDAV Server Protocol (RFC 4918)**
  - **Backend (`pkg/webdav/`, `internal/routes/webdavRoutes.go`, `internal/controllers/webdavController.go`, `internal/services/webdavService.go`):** Triển khai WebDAV server chuẩn IETF RFC 4918 hỗ trợ đầy đủ `OPTIONS`, `PROPFIND` (Depth 0/1/infinity), `GET` (stream range bytes 206 Partial Content), `HEAD`.
  - **Security & Scope (`internal/middlewares/webdavAuthMiddleware.go`):** Xác thực qua HTTP Basic Auth (Email + Password / Magic Code), Bearer Token, Query token `?token=...`, kiểm tra quyền RBAC và sandboxed theo Library của user.
  - **Native CLI Probe (`cmd/webdavprobe/`):** Công cụ CLI kiểm thử trực tiếp các request WebDAV trên live server hoặc mock in-memory.
  - **Frontend UI (`web/src/components/profile/WebDAVSyncCard.tsx`):** Card hướng dẫn kết nối WebDAV kèm URL, nút sao chép, mã QR Code và hướng dẫn cấu hình cho Moon+ Reader, KyBook 3, FBReader, Foliate, Zotero. Đồng bộ 16 ngôn ngữ i18n.
  - **Full test coverage:** 100% test pass (`pkg/webdav`, `webdavService_test.go`, `webdavController_test.go`, `webdavAuthMiddleware_test.go`).

- [x] **2.2. Calibre Content Server API Emulation**
  - **Codec & Standards (`pkg/calibre/codec.go`):** Bộ mã hóa/giải mã tên UTF-8 hex chuẩn Calibre (`EncodeName`, `DecodeName`) kèm kiểm tra tính hợp lệ UTF-8 (`utf8.Valid`).
  - **DTOs (`internal/dtos/request/calibre_server.go`, `internal/dtos/response/calibre_server.go`):** Mô hình hóa chuẩn 100% Calibre Content Server JSON schema (`library-info`, `categories`, `category_detail`, `books_in`, `search`, `books`, `book`).
  - **Security & RBAC (`internal/middlewares/calibreAuthMiddleware.go`):** Xác thực HTTP Basic Auth, Bearer token, query token `?token=...`, cookie, và chế độ Guest. Tự động kiểm tra và trả về 403 Forbidden đối với tài khoản bị cấm (banned), đồng thời bảo vệ giới hạn tần suất request (Rate Limiting).
  - **Service & Controller (`internal/services/calibreServerService.go`, `internal/controllers/calibreServerController.go`, `internal/routes/calibreServerRoutes.go`):** Triển khai đầy đủ các endpoint `/calibre/ajax/*` và `/calibre/get/:what/:book_id`. Hỗ trợ phân quyền đọc sách (`CanReadBook`), lọc sách theo quyền thư viện (`FilterReadableBooks`), phân quyền tải file (`CanDownloadBook`), chống path traversal qua `localfs.SafeJoin`, tối ưu hóa đếm sách nhanh qua `ListBookIDsByLibrary`, và áp dụng giới hạn phân trang tối đa 100 book IDs chống DoS.
  - **Full Test Coverage:** 100% unit tests & e2e HTTP integration tests (`pkg/calibre/codec_test.go`, `calibreAuthMiddleware_test.go`, `calibreServerService_test.go`, `calibreServerController_test.go`, `cmd/api/calibre_contract_test.go`).

---

## 🎙️ Track 3: Local Neural TTS to Audiobook Engine (Tạo Sách nói Cục bộ)

### Mục tiêu

Cung cấp khả năng chuyển đổi trực tiếp sách chữ (EPUB, TXT, MD) thành sách nói chaptered M4B chất lượng cao bằng engine AI Neural TTS mã nguồn mở (Piper TTS / Kokoro) chạy hoàn toàn trên server, không phụ thuộc API bên thứ 3.

### Nhiệm vụ chi tiết

- [ ] **3.1. Engine Tích hợp Piper-TTS Backend (`pkg/neuraltts/`)**
  - Đóng gói hoặc tích hợp driver giao tiếp với binary `piper` ONNX gọn nhẹ thuần CPU/GPU.
  - Hỗ trợ tải các model giọng đọc đa ngôn ngữ (Tiếng Việt, Tiếng Anh, Tiếng Nhật, v.v.).

- [ ] **3.2. Worker tổng hợp Audiobook nền (`pkg/worker`)**
  - **Job:** `synthesize_audiobook` nhận `book_id` và danh sách `chapter_ids`.
  - Render từng chương thành audio PCM/WAV -> ghép vào engine WaxFlow -> đóng gói thành file `.m4b` với chapter markers đầy đủ.
  - Tự động tạo bản ghi `book_files` định dạng audio và gán vào thư viện Audiobook.

- [ ] **3.3. UI Quản lý & Sinh Audiobook**
  - **Frontend:** Nút "Generate Audiobook" trong trang chi tiết sách.
  - Modal chọn giọng đọc (Voice Model), tốc độ đọc, và chọn chương cần render kèm thanh tiến trình render realtime (SSE / WebSocket / Polling).

---

## ☁️ Track 4: Storage Tiering & Cloud Backup (Hạ tầng Lưu trữ & Sao lưu)

### Mục tiêu

Tách biệt tầng lưu trữ dữ liệu lớn (sách, audiobook) để hỗ trợ Cloud Object Storage chuẩn S3, giải phóng dung lượng đĩa cho các máy chủ/VPS nhỏ, đồng thời tự động hóa sao lưu an toàn ra ngoài (offsite backup).

### Nhiệm vụ chi tiết

- [ ] **4.1. Storage Driver Abstraction (`pkg/storage/`)**
  - Định nghĩa interface `StorageDriver` (`Read`, `Write`, `Delete`, `Stat`, `OpenRange`).
  - Triển khai `LocalStorageDriver` (mặc định - sandboxed filesystem).
  - Triển khai `S3StorageDriver` (hỗ trợ AWS S3, Cloudflare R2, MinIO, Backblaze B2).
  - Cấu hình qua Admin Settings UI (`storage.driver`, `storage.s3.endpoint`, `storage.s3.bucket`, `storage.s3.access_key`, `storage.s3.secret_key`).

- [ ] **4.2. Offsite Cloud Backup Replication**
  - **Backend (`internal/services/backupService.go`):** Bổ sung cơ chế tự động upload bản snapshot SQLite nén (`.db.gz`) sau khi tạo snapshot thành công lên S3 Bucket / WebDAV / Google Drive.
  - **Admin UI:** Lịch trình sao lưu tự động và danh sách các bản backup đã lưu trên Cloud.

- [x] **4.3. Book Doctor & EPUB Structure Repair**
  - **Backend (`pkg/bookparser/epub/repair.go`):** Module chẩn đoán và tự động sửa lỗi cấu trúc EPUB:
    - Sửa lỗi unclosed XML/XHTML tags gây crash trình đọc (`<br>`, `<img>`, `<hr>`).
    - Chuẩn hóa thực thể XML (`&nbsp;` -> `&#160;`, `&copy;` -> `&#169;`, escape bare `&`).
    - Tạo lại/sửa file mục lục `toc.ncx` / `nav.xhtml` nếu bị thiếu hoặc lệch spine.
    - Sửa lỗi file `mimetype` (1st entry, uncompressed/stored) và `container.xml` đúng chuẩn IDPF.
    - Reconcile manifest và spine: loại bỏ item hỏng, khai báo unmanifested media/XHTML.
  - **Native CLI Probe (`cmd/bookdoctor/`):** CLI tool `validate`, `repair`, `demo` kiểm thử trực tiếp trên terminal.
  - **UI (`web/src/components/book-detail/BookDoctorModal.tsx`):** Modal chẩn đoán, xem danh sách lỗi phân loại theo mức độ (Errors, Warnings, Infos), tùy chọn auto-repair và xem live repair logs.
  - **Full test coverage:** 100% test pass cả backend (`repair_test.go`, `bookService_doctor_test.go`) lẫn frontend (`bookDoctorService.test.ts`).

---

## 👥 Track 5: Book Clubs & Collaborative Reading (Câu lạc bộ Sách & Ghi chú Chia sẻ)

### Mục tiêu

Xây dựng trải nghiệm đọc sách tương tác giữa các tài khoản trong cùng một instance NovelHub (gia đình, bạn bè, nhóm nghiên cứu) một cách an toàn và riêng tư.

### Nhiệm vụ chi tiết

- [ ] **5.1. Quản lý Book Clubs (Câu lạc bộ Đọc sách)**
  - **Database:** Bảng `book_clubs` (`id`, `name`, `description`, `owner_id`, `created_at`), `book_club_members` (`club_id`, `user_id`, `role`), `book_club_books` (`club_id`, `book_id`, `target_date`).
  - **Backend (`internal/services/clubService.go`):** CRUD Club, mời thành viên, gán sách đọc chung theo tuần/tháng.
  - **Frontend (`web/src/pages/user/BookClubsPage.tsx`):** Trang quản lý nhóm, danh sách thành viên và sách đang đọc chung.

- [ ] **5.2. Shared Annotations & Discussion Threads (Thảo luận theo đoạn văn)**
  - **Database:** Bảng `highlight_discussions` (`id`, `highlight_id`, `user_id`, `content`, `created_at`).
  - **Backend:** Thêm quyền `highlight.share` và API lấy các bình luận theo highlight.
  - **Frontend:** Trình đọc hiển thị highlight được chia sẻ từ bạn bè trong nhóm đọc kèm bảng bình luận trao đổi trực tiếp bên cạnh đoạn trích.

---

## 📅 Lộ trình Triển khai Đề xuất (Phased Milestones)

### 🔹 Giai đoạn 1: v1.1.0 — Open Protocols & Storage Tiering

- Triển khai **WebDAV Server Protocol (RFC 4918)**.
- Triển khai **Calibre Content Server API Emulation**.
- Triển khai **Storage Driver Abstraction (S3 / Cloudflare R2 / MinIO)**.
- Triển khai **Book Doctor & EPUB Structure Auto-Repair Engine**.

### 🔹 Giai đoạn 2: v1.2.0 — AI Reading Copilot & Local Intelligence

- Cấu hình AI Provider trong Admin Settings (Ollama, OpenAI, Gemini, Claude, DeepSeek).
- Triển khai **Chat with Book (RAG trên SQLite FTS5)**.
- Triển khai **Character Dossier & Lore Explorer**.
- Triển khai **Smart Chapter Recap ("Previously on...")**.
- Triển khai **Xuất Anki Flashcard (.apkg)**.

### 🔹 Giai đoạn 3: v1.3.0 — Neural TTS & Social Reading

- Triển khai **Server-Side Neural TTS (Piper-TTS) sang Audiobook M4B**.
- Triển khai **Book Clubs & Shared Highlights/Discussion Threads**.
- Triển khai **Offsite Cloud Backup Replication (S3 / WebDAV)**.

---

## 📌 Checklist Quy tắc Kỹ thuật Bắt buộc (Mandatory Technical Checklist)

- [ ] **Zero Raw SQL:** Mọi truy vấn database phải nằm trong `db/query/*.sql` và sinh mã qua `make sqlc`.
- [ ] **Cache-by-IDs & Singleflight:** Toàn bộ truy vấn đọc danh sách phải qua RAM Cache Theine với Singleflight wrapper.
- [ ] **Security First:** Dùng `pkg/localfs.SafeJoin` chống path traversal, `pkg/netx.NewSafeHTTPClient` chống SSRF, `pkg/jsonx` cho mọi xử lý JSON.
- [ ] **Bun & React 19 Frontend:** Toàn bộ component dưới 400 dòng, khai báo kiểu tại `web/src/types/`, dùng TanStack Query + Zustand, tuyệt đối không hardcode text (đồng bộ đủ 16 file ngôn ngữ trong `web/public/locales/`).
- [ ] **100% Test Coverage:** Mọi service, controller và parser mới phải có unit test Go (`*_test.go`) và Vitest frontend (`*.test.ts`).
