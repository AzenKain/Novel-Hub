import { Check, Copy, X } from "lucide-react";
import React from "react";

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

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center p-4">
      <button
        className="absolute inset-0 bg-black/60 backdrop-blur-sm"
        aria-label={t("common.close", "Close")}
        onClick={onClose}
      />
      <section className="relative z-10 w-full max-w-md rounded-2xl border border-base-300 bg-base-100 p-5 shadow-2xl space-y-4">
        <div className="flex items-start justify-between gap-3">
          <div>
            <h2 className="text-xl font-black">
              {t("library.share_book", "Share book")}
            </h2>
            <p className="mt-1 text-sm text-base-content/55">{book.title}</p>
          </div>
          <button
            className="btn btn-ghost btn-circle btn-sm"
            onClick={onClose}
            aria-label={t("common.close", "Close")}
          >
            <X className="h-4 w-4" />
          </button>
        </div>

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
              {copied ? <Check className="h-4 w-4" /> : <Copy className="h-4 w-4" />}
              {copied
                ? t("common.copied", "Copied")
                : t("common.copy", "Copy")}
            </button>
          </div>
        </div>
      </section>
    </div>
  );
};
