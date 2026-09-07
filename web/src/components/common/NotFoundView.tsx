import React from "react";
import { Link, useNavigate } from "react-router-dom";
import { useTranslation } from "react-i18next";
import { ArrowLeft, BookOpen, BookX, Compass, Search } from "lucide-react";
import { usePublicSettings } from "@/hooks/useSettings";

export interface NotFoundViewProps {
  type?: "book" | "page";
  title?: string;
  description?: string;
  onGoBack?: () => void;
  homePath?: string;
}

export const NotFoundView: React.FC<NotFoundViewProps> = ({
  type = "book",
  title,
  description,
  onGoBack,
  homePath = "/",
}) => {
  const { t } = useTranslation();
  const navigate = useNavigate();
  const publicSettings = usePublicSettings();

  const siteLogo = publicSettings?.site?.logo || "/logo.svg";
  const siteTitle = publicSettings?.site?.title || "NovelHub";

  const defaultTitle =
    type === "book"
      ? t("reader.book_not_found", "Book not found")
      : t("error.page_not_found", "Page not found");

  const defaultDescription =
    type === "book"
      ? t(
          "error.book_not_found_desc",
          "This book might have been deleted, moved, or the link is invalid.",
        )
      : t(
          "error.page_not_found_desc",
          "The page you are looking for doesn't exist or has been moved.",
        );

  const displayTitle = title || defaultTitle;
  const displayDescription = description || defaultDescription;

  const handleGoBack = () => {
    if (onGoBack) {
      onGoBack();
    } else if (window.history.length > 1) {
      navigate(-1);
    } else {
      navigate(homePath);
    }
  };

  return (
    <div className="relative flex min-h-screen w-full flex-col items-center justify-center overflow-hidden bg-base-100 px-4 py-12 text-base-content selection:bg-primary/20 transition-colors duration-200">
      {/* Ambient background glow */}
      <div className="pointer-events-none absolute -top-32 -left-32 h-96 w-96 rounded-full bg-primary/15 blur-3xl" />
      <div className="pointer-events-none absolute -bottom-32 -right-32 h-96 w-96 rounded-full bg-secondary/15 blur-3xl" />
      <div className="pointer-events-none absolute inset-0 bg-[radial-gradient(ellipse_at_center,transparent_0%,var(--color-base-100)_70%)] opacity-70" />

      <div className="relative z-10 flex w-full max-w-md flex-col items-center text-center animate-in fade-in zoom-in-95 duration-200">
        {/* Site Logo & Brand Pill from Stores */}
        <Link
          to="/"
          className="group mb-8 inline-flex items-center gap-2.5 rounded-full border border-base-content/10 bg-base-200/80 px-4 py-2 shadow-xs backdrop-blur-md transition-all duration-200 hover:border-primary/40 hover:bg-base-200"
          title={siteTitle}
        >
          {siteLogo ? (
            <img
              src={siteLogo}
              alt={siteTitle}
              className="h-6 w-auto max-w-8 object-contain drop-shadow-xs transition-transform duration-200 group-hover:scale-105"
            />
          ) : (
            <div className="flex h-6 w-6 items-center justify-center rounded-md bg-linear-to-br from-primary to-secondary text-[10px] font-bold text-primary-content shadow-xs">
              NH
            </div>
          )}
          <span className="text-sm font-bold tracking-tight text-base-content/90 transition-colors group-hover:text-base-content">
            {siteTitle}
          </span>
        </Link>

        {/* Card Container */}
        <div className="w-full rounded-3xl border border-base-content/10 bg-base-100/90 p-6 sm:p-8 shadow-2xl backdrop-blur-xl transition-all">
          {/* Visual Icon with 404 Badge */}
          <div className="relative mx-auto mb-6 flex h-24 w-24 sm:h-28 sm:w-28 items-center justify-center">
            <div className="absolute inset-0 rounded-3xl bg-linear-to-tr from-primary/20 via-primary/10 to-secondary/15 border border-primary/25 shadow-inner" />
            {type === "book" ? (
              <BookX className="relative h-12 w-12 sm:h-14 sm:w-14 text-primary animate-pulse" />
            ) : (
              <Compass className="relative h-12 w-12 sm:h-14 sm:w-14 text-primary animate-pulse" />
            )}
            <div className="absolute -bottom-1 -right-1 rounded-full border border-base-100 bg-error px-2.5 py-0.5 text-[10px] font-black tracking-wider text-error-content uppercase shadow-md">
              404
            </div>
          </div>

          {/* Heading and Description */}
          <h1 className="mb-2.5 text-2xl sm:text-3xl font-black tracking-tight text-base-content">
            {displayTitle}
          </h1>
          <p className="mx-auto mb-8 max-w-sm text-sm sm:text-base leading-relaxed text-base-content/70">
            {displayDescription}
          </p>

          {/* Action Buttons */}
          <div className="flex flex-col sm:flex-row items-center justify-center gap-3 w-full">
            <button
              type="button"
              onClick={handleGoBack}
              className="btn btn-outline btn-sm sm:btn-md h-10 min-h-10 w-full sm:w-auto flex-1 rounded-xl px-4 font-semibold gap-2 border-base-content/20 hover:border-base-content/40 hover:bg-base-content/5"
            >
              <ArrowLeft className="h-4 w-4" />
              <span>{t("common.back", "Go Back")}</span>
            </button>
            <button
              type="button"
              onClick={() => navigate(homePath)}
              className="btn btn-primary btn-sm sm:btn-md h-10 min-h-10 w-full sm:w-auto flex-1 rounded-xl px-4 font-bold shadow-md shadow-primary/20 hover:shadow-primary/35 gap-2"
            >
              <BookOpen className="h-4 w-4" />
              <span>{t("error.go_to_library", "Go to Library")}</span>
            </button>
          </div>

          {/* Search Hint Footer */}
          <div className="mt-6 pt-4 border-t border-base-content/10 flex items-center justify-center gap-1.5 text-xs text-base-content/60">
            <span>
              {t("error.looking_for_other_books", "Looking for other books?")}
            </span>
            <Link
              to="/search"
              className="link link-primary font-medium inline-flex items-center gap-1 hover:underline"
            >
              <Search className="h-3.5 w-3.5" />
              <span>{t("error.search_books", "Search")}</span>
            </Link>
          </div>
        </div>
      </div>
    </div>
  );
};
