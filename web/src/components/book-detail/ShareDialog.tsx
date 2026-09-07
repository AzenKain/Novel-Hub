import { Check, Copy, X } from "lucide-react";
import React from "react";
import { createPortal } from "react-dom";

import type { Book } from "@/types";
import { CustomQRCode } from "@/components/common/CustomQRCode";

type ShareDialogProps = {
  open: boolean;
  book: Book;
  shareUrl: string;
  copied: boolean;
  t: (key: string, fallback: string) => string;
  onClose: () => void;
  onCopy: () => void;
};

export const ShareDialog: React.FC<ShareDialogProps> = ({
  open,
  book,
  shareUrl,
  copied,
  t,
  onClose,
  onCopy,
}) => {
  if (!open) return null;

  return createPortal(
    <div className="fixed inset-0 z-50 flex items-center justify-center p-3 sm:p-4">
      <button
        className="absolute inset-0 bg-black/50"
        aria-label={t("common.close", "Close")}
        onClick={onClose}
      />
      <section className="relative z-10 w-full max-w-md max-h-[80dvh] sm:max-h-[85vh] flex flex-col rounded-2xl sm:rounded-3xl border border-base-300 bg-base-100 p-0 shadow-2xl overflow-hidden">
        {/* Fixed Header */}
        <div className="px-4 sm:px-6 py-3.5 sm:py-4 border-b border-base-200 flex items-center justify-between gap-3 shrink-0 bg-base-100">
          <div className="min-w-0 flex-1">
            <h2 className="text-base sm:text-lg font-bold leading-tight truncate">
              {t("library.share_book", "Share book")}
            </h2>
            <p className="mt-0.5 text-xs text-base-content/60 truncate">{book.title}</p>
          </div>
          <button
            type="button"
            className="btn btn-ghost btn-circle btn-sm -mr-1.5 text-base-content/70 hover:text-base-content shrink-0"
            onClick={onClose}
            aria-label={t("common.close", "Close")}
          >
            <X className="w-4 h-4 sm:w-5 sm:h-5 stroke-[2.5]" />
          </button>
        </div>

        {/* Scrollable Body */}
        <div className="p-4 sm:p-6 overflow-y-auto flex-1 min-h-0 space-y-4">
          <CustomQRCode value={shareUrl} size={200} />

          <div className="rounded-xl border border-base-200 bg-base-100 p-3 shadow-2xs">
            <div className="mb-1 text-xs font-bold uppercase tracking-wider text-base-content/45">
              {t("library.share_link", "Share link")}
            </div>
            <div className="flex gap-2">
              <input
                className="input input-bordered input-sm min-w-0 flex-1 bg-base-100 font-mono text-xs"
                value={shareUrl}
                readOnly
              />
              <button className="btn btn-primary btn-sm gap-2" onClick={onCopy}>
                {copied ? (
                  <Check className="h-4 w-4" />
                ) : (
                  <Copy className="h-4 w-4" />
                )}
                {copied ? t("common.copied", "Copied") : t("common.copy", "Copy")}
              </button>
            </div>
          </div>
        </div>
      </section>
    </div>,
    document.body,
  );
};
