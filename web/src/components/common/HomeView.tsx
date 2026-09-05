import { usePublicSettings } from "@/hooks/useSettings";
import { ArrowRight, BookMarked, Library, Sparkles } from "lucide-react";
import { useTranslation } from "react-i18next";
import { Link } from "react-router-dom";

export function HomeView() {
  const { t } = useTranslation();
  const settings = usePublicSettings();
  const title = settings?.site?.title || "NovelHub";
  const siteLogo = settings?.site?.logo || "/logo.svg";
  const description =
    settings?.site?.description || t("home.default_description");

  return (
    <div className="home-container">
      <div className="decor-grid" aria-hidden="true">
        <div className="decor-line"></div>
        <div className="decor-line"></div>
        <div className="decor-line"></div>
      </div>

      <header className="home-header">
        <div className="home-brand flex items-center gap-2.5">
          {siteLogo ? (
            <img
              src={siteLogo}
              alt={t("common.alt_logo", "Logo")}
              className="h-11 w-auto max-w-[56px] object-contain shrink-0 drop-shadow-sm"
            />
          ) : (
            <div className="home-brand-mark"></div>
          )}
          <div>
            <h1 className="text-lg font-black leading-none tracking-tight text-base-content">
              {title}
            </h1>
            <p className="mt-1 text-[11px] font-semibold uppercase tracking-widest text-base-content/50">
              {description}
            </p>
          </div>
        </div>
        <Link to="/admin" className="admin-link-btn">
          <span>{t("home.go_to_admin")}</span>
          <ArrowRight size={16} />
        </Link>
      </header>

      <main className="home-hero">
        <section className="hero-text-area">
          <div className="hero-badge">
            <Sparkles size={14} className="sparkle-icon" />
            <span>{t("home.badge")}</span>
          </div>
          <h2>{t("home.hero_title")}</h2>
          <p>{t("home.hero_desc")}</p>
          <div className="hero-ctas">
            <Link to="/admin" className="cta-button primary">
              <span>{t("home.open_admin")}</span>
              <Library size={18} />
            </Link>
            <a href="#features" className="cta-button secondary">
              {t("home.learn_more")}
            </a>
          </div>
        </section>

        <section className="hero-visual-area">
          <div className="book-card-3d">
            <div className="book-spine"></div>
            <div className="book-cover-art">
              <small>{t("home.edition")}</small>
              <strong>{t("home.demo_book_title")}</strong>
              <div className="book-cover-footer">
                <span>{t("home.vol_1")}</span>
                <span>{t("home.self_hosted")}</span>
              </div>
            </div>
            <div className="book-pages-side"></div>
          </div>
        </section>
      </main>

      <section id="features" className="home-features">
        <h3>{t("home.features_title")}</h3>
        <div className="features-grid">
          <div className="feature-card">
            <div className="feature-icon">
              <Library size={20} />
            </div>
            <h4>{t("home.feature_manage_title")}</h4>
            <p>{t("home.feature_manage_desc")}</p>
          </div>
          <div className="feature-card">
            <div className="feature-icon">
              <BookMarked size={20} />
            </div>
            <h4>{t("home.feature_reader_title")}</h4>
            <p>{t("home.feature_reader_desc")}</p>
          </div>
          <div className="feature-card">
            <div className="feature-icon">
              <Sparkles size={20} />
            </div>
            <h4>{t("home.feature_jobs_title")}</h4>
            <p>{t("home.feature_jobs_desc")}</p>
          </div>
        </div>
      </section>

      <footer className="home-footer">
        <p>{t("home.footer", { year: new Date().getFullYear() })}</p>
      </footer>
    </div>
  );
}
