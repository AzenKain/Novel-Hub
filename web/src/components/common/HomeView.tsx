import { usePublicSettings } from "@/hooks/useSettings";
import { ArrowRight, BookMarked, Library, Sparkles } from "lucide-react";
import { Link } from "react-router-dom";

export function HomeView() {
  const settings = usePublicSettings();
  const title = settings?.site?.title || "NovelHub";
  const description = settings?.site?.description || "Local novel library manager";

  return (
    <div className="home-container">
      {/* Decorative Top Line Grid */}
      <div className="decor-grid" aria-hidden="true">
        <div className="decor-line"></div>
        <div className="decor-line"></div>
        <div className="decor-line"></div>
      </div>

      <header className="home-header">
        <div className="home-brand">
          <div className="home-brand-mark"></div>
          <div>
            <h1>{title}</h1>
            <p>{description}</p>
          </div>
        </div>
        <Link to="/admin" className="admin-link-btn">
          <span>Go to Admin</span>
          <ArrowRight size={16} />
        </Link>
      </header>

      <main className="home-hero">
        <section className="hero-text-area">
          <div className="hero-badge">
            <Sparkles size={14} className="sparkle-icon" />
            <span>Local-First & Light Novel Focused</span>
          </div>
          <h2>Dòng chảy câu chữ từ chiếc tủ sách cá nhân của bạn.</h2>
          <p>
            NovelHub là giải pháp quản lý và thưởng thức tiểu thuyết, sách điện tử nhẹ nhàng, hiện đại thay thế cho Calibre. 
            Giữ trọn vẹn sự riêng tư với cơ chế lưu trữ cục bộ, quét thư mục thông minh, và trải nghiệm đọc trôi chảy ngay trên trình duyệt.
          </p>
          <div className="hero-ctas">
            <Link to="/admin" className="cta-button primary">
              <span>Mở trang Quản trị</span>
              <Library size={18} />
            </Link>
            <a href="#features" className="cta-button secondary">
              Tìm hiểu thêm
            </a>
          </div>
        </section>

        <section className="hero-visual-area">
          <div className="book-card-3d">
            <div className="book-spine"></div>
            <div className="book-cover-art">
              <small>NovelHub Edition</small>
              <strong>Vườn Anh Đào Số Hóa</strong>
              <div className="book-cover-footer">
                <span>Vol. 1</span>
                <span>Self-hosted</span>
              </div>
            </div>
            <div className="book-pages-side"></div>
          </div>
        </section>
      </main>

      <section id="features" className="home-features">
        <h3>Tính năng cốt lõi</h3>
        <div className="features-grid">
          <div className="feature-card">
            <div className="feature-icon"><Library size={20} /></div>
            <h4>Quản lý thông minh</h4>
            <p>Hỗ trợ cả chế độ liên kết (Reference) giữ nguyên file gốc lẫn chế độ sao chép (Managed) tự động phân loại thư mục.</p>
          </div>
          <div className="feature-card">
            <div className="feature-icon"><BookMarked size={20} /></div>
            <h4>Trình đọc EPUB</h4>
            <p>Trình đọc tối ưu, stream từng chương và tài nguyên ảnh trực tiếp từ ZIP/EPUB mà không gây tốn tài nguyên RAM.</p>
          </div>
          <div className="feature-card">
            <div className="feature-icon"><Sparkles size={20} /></div>
            <h4>Tự động hoá Jobs</h4>
            <p>Quét thư viện nhanh chóng, tự sinh ảnh bìa thumbnail, trích xuất text phục vụ tìm kiếm toàn văn FTS trong nền.</p>
          </div>
        </div>
      </section>

      <footer className="home-footer">
        <p>&copy; {new Date().getFullYear()} NovelHub. Nền tảng tủ sách local-first tinh gọn.</p>
      </footer>
    </div>
  );
}
