# Extension VBook dành cho NovelHub

Thư mục này chứa extension dành cho ứng dụng **VBook** giúp người dùng đọc thư viện truyện từ máy chủ **NovelHub** cá nhân.

## Cấu trúc thư mục Extension

```text
vbook/
├── plugin.json       # Tệp cấu hình metadata & đường dẫn script
├── icon.png          # Biểu tượng icon ứng dụng NovelHub
├── plugin.zip        # Tệp đóng gói extension để cài đặt vào VBook
├── README.md         # Hướng dẫn sử dụng
└── src/
    ├── home.js       # Tải trang chủ (Khám phá)
    ├── genre.js      # Tải danh mục thể loại/sê-ri
    ├── gen.js        # Tải danh sách sách phân trang
    ├── search.js     # Tìm kiếm sách
    ├── detail.js     # Tải chi tiết sách
    ├── toc.js        # Tải mục lục (danh sách chương)
    └── chap.js       # Tải nội dung chương
```

## Hướng dẫn sử dụng & Cấu hình URL máy chủ

1. Mở tệp `vbook/plugin.json`.
2. Thay đổi địa chỉ máy chủ trong trường `"source"` thành địa chỉ NovelHub của bạn:
   ```json
   "metadata": {
     "source": "https://novelhub.your-domain.com"
   }
   ```
3. Nếu máy chủ của bạn yêu cầu xác thực (không cho phép khách truy cập), thêm tham số `token` hoặc cấu hình Header trong extension.

## Hướng dẫn Pull Request sang Darkrai9x/vbook-extensions

Để thêm extension NovelHub vào kho lưu trữ chính thức của VBook:
1. Fork repository [Darkrai9x/vbook-extensions](https://github.com/Darkrai9x/vbook-extensions).
2. Copy thư mục `novelhub` (chứa `plugin.json` và thư mục `src/`) vào thư mục của repo đó.
3. Tạo Pull Request sang `Darkrai9x/vbook-extensions`.
