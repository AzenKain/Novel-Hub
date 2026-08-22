import type { TFunction } from "i18next";
import { ChevronLeft } from "lucide-react";
import React from "react";

type ReaderPageControlsProps = {
  t: TFunction;
  mode: "floating" | "footer";
  canGoPrev: boolean;
  canGoNext: boolean;
  onPrev: () => void;
  onNext: () => void;
};

export const ReaderPageControls: React.FC<ReaderPageControlsProps> = ({
  t,
  mode,
  canGoPrev,
  canGoNext,
  onPrev,
  onNext,
}) => {
  if (mode === "floating") {
    return (
      <>
        <button
          type="button"
          onClick={onPrev}
          className="reader-floating-btn btn btn-circle absolute left-4 top-1/2 z-20 -translate-y-1/2 animate-none shadow-xl hidden lg:flex opacity-40 hover:opacity-100 transition-all cursor-pointer"
          title={t("reader.prev_page", "Previous Page")}
          aria-label={t("reader.prev_page", "Previous Page")}
        >
          <ChevronLeft className="h-6 w-6" />
        </button>
        <button
          type="button"
          onClick={onNext}
          className="reader-floating-btn btn btn-circle absolute right-4 top-1/2 z-20 -translate-y-1/2 animate-none shadow-xl hidden lg:flex opacity-40 hover:opacity-100 transition-all cursor-pointer"
          title={t("reader.next_page", "Next Page")}
          aria-label={t("reader.next_page", "Next Page")}
        >
          <ChevronLeft className="h-6 w-6 rotate-180" />
        </button>
      </>
    );
  }

  return (
    <div className="mt-16 flex items-center justify-between border-t border-current/10 pt-8 pb-8">
      <button
        onClick={onPrev}
        disabled={!canGoPrev}
        className="reader-outline-btn btn animate-none rounded-full"
      >
        <ChevronLeft className="h-4 w-4" />
        {t("reader.prev_chapter", "Previous")}
      </button>

      <button
        onClick={onNext}
        disabled={!canGoNext}
        className="reader-action-btn btn animate-none rounded-full"
      >
        {t("reader.next_chapter", "Next")}
        <ChevronLeft className="h-4 w-4 rotate-180" />
      </button>
    </div>
  );
};
