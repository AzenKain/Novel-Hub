import { Check, Copy, Download, Sparkles, X } from "lucide-react";
import React, { useCallback, useEffect, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import { toast } from "react-toastify";

interface QuoteCardModalProps {
  open: boolean;
  quote?: string;
  imageUrl?: string;
  bookTitle?: string;
  bookAuthor?: string;
  bookCover?: string;
  chapterTitle?: string;
  onClose: () => void;
}

type CardTheme = "modern" | "vintage" | "aurora" | "glass";

export const QuoteCardModal: React.FC<QuoteCardModalProps> = ({
  open,
  quote = "",
  imageUrl,
  bookTitle = "Unknown Book",
  bookAuthor = "Unknown Author",
  bookCover,
  chapterTitle,
  onClose,
}) => {
  const { t } = useTranslation();
  const canvasRef = useRef<HTMLCanvasElement>(null);
  const [theme, setTheme] = useState<CardTheme>("aurora");
  const [copiedImage, setCopiedImage] = useState(false);
  const [coverImgElement, setCoverImgElement] = useState<HTMLImageElement | null>(null);
  const [quoteImgElement, setQuoteImgElement] = useState<HTMLImageElement | null>(null);

  // Preload book cover image
  useEffect(() => {
    if (!bookCover) {
      setCoverImgElement(null);
      return;
    }
    const img = new Image();
    img.crossOrigin = "anonymous";
    img.src = bookCover;
    img.onload = () => setCoverImgElement(img);
    img.onerror = () => setCoverImgElement(null);
  }, [bookCover]);

  // Preload quote artwork image (for illustration bookmarks)
  useEffect(() => {
    if (!imageUrl) {
      setQuoteImgElement(null);
      return;
    }
    const img = new Image();
    img.crossOrigin = "anonymous";
    img.src = imageUrl;
    img.onload = () => setQuoteImgElement(img);
    img.onerror = () => setQuoteImgElement(null);
  }, [imageUrl]);

  const drawCard = useCallback(() => {
    const canvas = canvasRef.current;
    if (!canvas) return;
    const ctx = canvas.getContext("2d");
    if (!ctx) return;

    // High Resolution Canvas (1080 x 1350 - Standard Instagram Portrait 4:5)
    const width = 1080;
    const height = 1350;
    canvas.width = width;
    canvas.height = height;

    // Reset base text metrics
    ctx.textBaseline = "top";
    ctx.textAlign = "left";

    // 1. Draw Background Theme
    if (theme === "vintage") {
      // Warm Paper Texture Style
      const grad = ctx.createLinearGradient(0, 0, width, height);
      grad.addColorStop(0, "#f7f2e7");
      grad.addColorStop(1, "#ebdcc4");
      ctx.fillStyle = grad;
      ctx.fillRect(0, 0, width, height);

      // Subtle Vintage Border
      ctx.strokeStyle = "#c8b393";
      ctx.lineWidth = 4;
      ctx.strokeRect(40, 40, width - 80, height - 80);
      ctx.lineWidth = 1;
      ctx.strokeRect(48, 48, width - 96, height - 96);
    } else if (theme === "aurora") {
      // Modern Vibrant Mesh Gradient
      const grad = ctx.createLinearGradient(0, 0, width, height);
      grad.addColorStop(0, "#0f172a");
      grad.addColorStop(0.5, "#1e1b4b");
      grad.addColorStop(1, "#030712");
      ctx.fillStyle = grad;
      ctx.fillRect(0, 0, width, height);

      const radial1 = ctx.createRadialGradient(width * 0.8, height * 0.2, 50, width * 0.8, height * 0.2, 500);
      radial1.addColorStop(0, "rgba(168, 85, 247, 0.35)");
      radial1.addColorStop(1, "transparent");
      ctx.fillStyle = radial1;
      ctx.fillRect(0, 0, width, height);

      const radial2 = ctx.createRadialGradient(width * 0.2, height * 0.7, 50, width * 0.2, height * 0.7, 550);
      radial2.addColorStop(0, "rgba(56, 189, 248, 0.3)");
      radial2.addColorStop(1, "transparent");
      ctx.fillStyle = radial2;
      ctx.fillRect(0, 0, width, height);
    } else if (theme === "glass") {
      const grad = ctx.createLinearGradient(0, 0, width, height);
      grad.addColorStop(0, "#090d16");
      grad.addColorStop(1, "#111827");
      ctx.fillStyle = grad;
      ctx.fillRect(0, 0, width, height);

      const radial = ctx.createRadialGradient(width * 0.2, height * 0.8, 50, width * 0.2, height * 0.8, 600);
      radial.addColorStop(0, "rgba(13, 148, 136, 0.25)");
      radial.addColorStop(1, "transparent");
      ctx.fillStyle = radial;
      ctx.fillRect(0, 0, width, height);
    } else {
      // Modern OLED Dark
      ctx.fillStyle = "#0c0d12";
      ctx.fillRect(0, 0, width, height);

      ctx.strokeStyle = "rgba(255, 255, 255, 0.08)";
      ctx.lineWidth = 4;
      ctx.strokeRect(40, 40, width - 80, height - 80);
    }

    // 2. Compute Bottom Attribution Card Geometry
    const bottomCardHeight = 160;
    const bottomCardY = height - bottomCardHeight - 60; // 1350 - 160 - 60 = 1130

    // 3. Draw Content: Image Card OR Text Quote
    if (quoteImgElement) {
      // IMAGE QUOTE CARD
      // Check if there is a personal user note to display
      const cleanQuote = (quote || "").trim();
      const hasCaption = Boolean(
        cleanQuote &&
          !cleanQuote.startsWith("http") &&
          !cleanQuote.startsWith("[Minh họa]") &&
          cleanQuote !== bookTitle &&
          cleanQuote !== chapterTitle
      );

      const topPadding = 60;
      const captionHeight = hasCaption ? 50 : 0;
      const captionGap = hasCaption ? 20 : 0;
      const maxImageBottom = bottomCardY - 30 - captionHeight - captionGap;
      const maxAreaH = maxImageBottom - topPadding; // ~980px
      const maxAreaW = width - 140; // 940px

      const imgW = quoteImgElement.naturalWidth || 800;
      const imgH = quoteImgElement.naturalHeight || 600;
      const imgRatio = imgW / imgH;
      const boxRatio = maxAreaW / maxAreaH;

      let drawW = maxAreaW;
      let drawH = maxAreaH;

      if (imgRatio > boxRatio) {
        drawW = maxAreaW;
        drawH = maxAreaW / imgRatio;
      } else {
        drawH = maxAreaH;
        drawW = maxAreaH * imgRatio;
      }

      const drawX = (width - drawW) / 2;
      const drawY = topPadding + (maxAreaH - drawH) / 2;

      // Draw image with rounded corners
      ctx.save();
      ctx.beginPath();
      if (typeof ctx.roundRect === "function") {
        ctx.roundRect(drawX, drawY, drawW, drawH, 18);
      } else {
        ctx.rect(drawX, drawY, drawW, drawH);
      }
      ctx.clip();
      ctx.drawImage(quoteImgElement, drawX, drawY, drawW, drawH);
      ctx.restore();

      // Border around artwork
      ctx.beginPath();
      if (typeof ctx.roundRect === "function") {
        ctx.roundRect(drawX, drawY, drawW, drawH, 18);
      } else {
        ctx.rect(drawX, drawY, drawW, drawH);
      }
      ctx.strokeStyle = theme === "vintage" ? "rgba(180, 140, 90, 0.45)" : "rgba(255, 255, 255, 0.18)";
      ctx.lineWidth = 3;
      ctx.stroke();

      // Caption / Note if provided (Strictly placed below image and above bottom card)
      if (hasCaption) {
        const captionY = drawY + drawH + 16;
        ctx.font = `italic 500 28px "Georgia", "Merriweather", "Noto Serif", serif`;
        ctx.fillStyle = theme === "vintage" ? "#382818" : "#f1f5f9";
        ctx.textAlign = "center";
        ctx.textBaseline = "top";
        const displayCaption = cleanQuote.length > 80 ? cleanQuote.slice(0, 77) + "..." : cleanQuote;
        ctx.fillText(displayCaption, width / 2, captionY);
        ctx.textAlign = "left";
      }
    } else {
      // TEXT QUOTE CARD
      // Decorative Quotation Mark
      ctx.save();
      if (theme === "vintage") {
        ctx.fillStyle = "rgba(160, 110, 60, 0.2)";
      } else if (theme === "aurora") {
        ctx.fillStyle = "rgba(217, 70, 239, 0.2)";
      } else {
        ctx.fillStyle = "rgba(255, 255, 255, 0.08)";
      }
      ctx.font = "bold 200px Georgia, serif";
      ctx.textBaseline = "top";
      ctx.fillText("“", 80, 100);
      ctx.restore();

      // Text Wrapping & Typography for Quote
      const safeQuote = (quote || "").trim() || "...";
      const maxTextWidth = width - 200;
      let fontSize = 46;
      if (safeQuote.length > 250) fontSize = 36;
      if (safeQuote.length > 450) fontSize = 30;
      if (safeQuote.length < 80) fontSize = 56;

      ctx.font = `italic 500 ${fontSize}px "Georgia", "Merriweather", "Noto Serif", serif`;
      ctx.fillStyle = theme === "vintage" ? "#382818" : "#f1f5f9";
      ctx.textBaseline = "top";

      const words = safeQuote.split(/\s+/);
      const lines: string[] = [];
      let currentLine = "";

      for (const word of words) {
        const testLine = currentLine ? `${currentLine} ${word}` : word;
        const metrics = ctx.measureText(testLine);
        if (metrics.width > maxTextWidth) {
          lines.push(currentLine);
          currentLine = word;
        } else {
          currentLine = testLine;
        }
      }
      if (currentLine) lines.push(currentLine);

      // Limit maximum lines to fit comfortably above bottom card
      const maxTextZoneHeight = bottomCardY - 260;
      const lineHeight = fontSize * 1.5;
      const maxLines = Math.floor(maxTextZoneHeight / lineHeight);
      const displayedLines = lines.slice(0, maxLines);
      if (lines.length > maxLines) {
        displayedLines[maxLines - 1] += "...";
      }

      const startY = 220;
      displayedLines.forEach((line, i) => {
        ctx.fillText(line, 100, startY + i * lineHeight);
      });

      // Divider Line
      const dividerY = Math.min(startY + displayedLines.length * lineHeight + 35, bottomCardY - 30);
      ctx.beginPath();
      ctx.moveTo(100, dividerY);
      ctx.lineTo(240, dividerY);
      ctx.strokeStyle = theme === "vintage" ? "#b48c5a" : "rgba(255, 255, 255, 0.25)";
      ctx.lineWidth = 4;
      ctx.stroke();
    }

    // 4. Book & Author Attribution Card (Bottom)
    ctx.textBaseline = "top";
    ctx.textAlign = "left";

    if (coverImgElement) {
      try {
        const coverWidth = 100;
        const coverHeight = 140;
        ctx.save();
        ctx.beginPath();
        if (typeof ctx.roundRect === "function") {
          ctx.roundRect(100, bottomCardY, coverWidth, coverHeight, 10);
        } else {
          ctx.rect(100, bottomCardY, coverWidth, coverHeight);
        }
        ctx.clip();
        ctx.drawImage(coverImgElement, 100, bottomCardY, coverWidth, coverHeight);
        ctx.restore();

        // Border around cover
        ctx.strokeStyle = theme === "vintage" ? "rgba(180, 140, 90, 0.4)" : "rgba(255,255,255,0.15)";
        ctx.lineWidth = 2;
        ctx.beginPath();
        if (typeof ctx.roundRect === "function") {
          ctx.roundRect(100, bottomCardY, coverWidth, coverHeight, 10);
        } else {
          ctx.rect(100, bottomCardY, coverWidth, coverHeight);
        }
        ctx.stroke();

        // Text beside cover
        const textX = 225;
        ctx.font = "bold 32px -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif";
        ctx.fillStyle = theme === "vintage" ? "#1e140a" : "#ffffff";
        ctx.fillText(bookTitle.length > 34 ? bookTitle.slice(0, 34) + "..." : bookTitle, textX, bottomCardY + 15);

        ctx.font = "500 24px -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif";
        ctx.fillStyle = theme === "vintage" ? "#6c5035" : "rgba(255, 255, 255, 0.65)";
        ctx.fillText(bookAuthor.length > 40 ? bookAuthor.slice(0, 40) + "..." : bookAuthor, textX, bottomCardY + 60);

        if (chapterTitle) {
          ctx.font = "400 20px -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif";
          ctx.fillStyle = theme === "vintage" ? "#9c7c5b" : "rgba(255, 255, 255, 0.45)";
          ctx.fillText(chapterTitle.length > 45 ? chapterTitle.slice(0, 45) + "..." : chapterTitle, textX, bottomCardY + 100);
        }
      } catch (e) {
        console.warn("[QuoteCard] cover draw fallback", e);
      }
    } else {
      ctx.font = "bold 34px -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif";
      ctx.fillStyle = theme === "vintage" ? "#1e140a" : "#ffffff";
      ctx.fillText(bookTitle.length > 45 ? bookTitle.slice(0, 45) + "..." : bookTitle, 100, bottomCardY + 20);

      ctx.font = "500 26px -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif";
      ctx.fillStyle = theme === "vintage" ? "#6c5035" : "rgba(255, 255, 255, 0.65)";
      ctx.fillText(bookAuthor.length > 50 ? bookAuthor.slice(0, 50) + "..." : bookAuthor, 100, bottomCardY + 68);

      if (chapterTitle) {
        ctx.font = "400 20px -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif";
        ctx.fillStyle = theme === "vintage" ? "#9c7c5b" : "rgba(255, 255, 255, 0.45)";
        ctx.fillText(chapterTitle.length > 55 ? chapterTitle.slice(0, 55) + "..." : chapterTitle, 100, bottomCardY + 108);
      }
    }

    // 5. Watermark Footer
    ctx.font = "600 20px -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif";
    ctx.fillStyle = theme === "vintage" ? "rgba(100, 70, 40, 0.5)" : "rgba(255, 255, 255, 0.35)";
    ctx.textAlign = "right";
    ctx.fillText("NovelHub", width - 80, height - 45);
    ctx.textAlign = "left";
  }, [theme, quote, imageUrl, quoteImgElement, bookTitle, bookAuthor, chapterTitle, coverImgElement]);

  useEffect(() => {
    if (open) {
      const tId = setTimeout(drawCard, 60);
      return () => clearTimeout(tId);
    }
  }, [open, drawCard]);

  const handleDownload = () => {
    const canvas = canvasRef.current;
    if (!canvas) return;
    const link = document.createElement("a");
    link.download = `quote_${bookTitle.replace(/\s+/g, "_").slice(0, 20)}.png`;
    link.href = canvas.toDataURL("image/png");
    link.click();
    toast.success(t("reader.quote_downloaded", "Đã tải ảnh trích dẫn về máy!"));
  };

  const handleCopyImage = async () => {
    const canvas = canvasRef.current;
    if (!canvas) return;
    try {
      canvas.toBlob(async (blob) => {
        if (!blob) return;
        await navigator.clipboard.write([
          new ClipboardItem({ "image/png": blob }),
        ]);
        setCopiedImage(true);
        toast.success(t("reader.quote_copied", "Đã sao chép ảnh vào Clipboard!"));
        setTimeout(() => setCopiedImage(false), 2000);
      });
    } catch {
      toast.error(t("reader.quote_copy_failed", "Không thể sao chép ảnh"));
    }
  };

  if (!open) return null;

  return (
    <dialog
      className="modal modal-open z-60 bg-black/75 backdrop-blur-sm animate-in fade-in duration-200"
      data-reader-modal="true"
    >
      <div className="modal-box max-w-lg p-5 rounded-2xl border border-[var(--reader-ui-border,rgba(255,255,255,0.12))] shadow-2xl bg-[var(--reader-ui-surface-strong,#1e202b)] text-[var(--reader-ui-text,#e2e8f0)] flex flex-col gap-4">
        {/* Modal Header */}
        <div className="flex items-center justify-between border-b border-[var(--reader-ui-border,rgba(255,255,255,0.1))] pb-3">
          <div className="flex items-center gap-2 font-bold text-sm text-[var(--reader-ui-text)]">
            <Sparkles className="w-4 h-4 text-[var(--reader-ui-accent,#38bdf8)]" />
            <span>{t("reader.quote_card_title", "Tạo ảnh trích dẫn")}</span>
          </div>
          <button
            type="button"
            onClick={onClose}
            className="btn btn-xs btn-circle bg-[var(--reader-ui-soft,rgba(255,255,255,0.06))] hover:bg-[var(--reader-ui-hover,rgba(255,255,255,0.1))] text-[var(--reader-ui-text)] border border-[var(--reader-ui-border)]"
          >
            <X size={14} />
          </button>
        </div>

        {/* Canvas Preview */}
        <div className="flex justify-center items-center py-1">
          <div className="relative rounded-xl overflow-hidden shadow-2xl border border-[var(--reader-ui-border,rgba(255,255,255,0.15))] max-h-[55vh] flex justify-center bg-black/20">
            <canvas
              ref={canvasRef}
              className="max-h-[55vh] w-auto h-auto object-contain rounded-xl"
            />
          </div>
        </div>

        {/* Theme Selectors */}
        <div className="flex items-center justify-between gap-2 px-1">
          <span className="text-xs font-semibold opacity-70">
            {t("reader.quote_theme", "Chủ đề")}:
          </span>
          <div className="flex items-center gap-1.5">
            {[
              { id: "aurora", label: "Aurora", bg: "bg-linear-to-r from-purple-900 to-sky-900" },
              { id: "modern", label: "OLED", bg: "bg-[#0c0d12] border border-white/20" },
              { id: "vintage", label: "Vintage", bg: "bg-[#f5e6cb] text-black" },
              { id: "glass", label: "Emerald", bg: "bg-linear-to-r from-teal-950 to-slate-900" },
            ].map((th) => (
              <button
                key={th.id}
                type="button"
                onClick={() => setTheme(th.id as CardTheme)}
                className={`btn btn-xs rounded-lg text-[11px] font-medium px-2.5 transition-all cursor-pointer ${
                  theme === th.id
                    ? "ring-2 ring-[var(--reader-ui-accent,#38bdf8)] ring-offset-1 ring-offset-[var(--reader-ui-surface-strong,#1e202b)] opacity-100 font-bold scale-105"
                    : "opacity-60 hover:opacity-100"
                } ${th.bg}`}
              >
                {th.label}
              </button>
            ))}
          </div>
        </div>

        {/* Modal Actions */}
        <div className="flex items-center justify-end gap-2 pt-2 border-t border-[var(--reader-ui-border,rgba(255,255,255,0.1))]">
          <button
            type="button"
            onClick={onClose}
            className="btn btn-sm rounded-xl bg-[var(--reader-ui-soft)] hover:bg-[var(--reader-ui-hover)] text-[var(--reader-ui-text)] border border-[var(--reader-ui-border)]"
          >
            {t("common.cancel", "Đóng")}
          </button>
          <button
            type="button"
            onClick={handleCopyImage}
            className="btn btn-sm rounded-xl gap-1.5 bg-[var(--reader-ui-soft)] hover:bg-[var(--reader-ui-hover)] text-[var(--reader-ui-text)] border border-[var(--reader-ui-border)] font-semibold"
          >
            {copiedImage ? <Check size={14} className="text-success" /> : <Copy size={14} />}
            <span>{copiedImage ? t("reader.copied", "Đã chép") : t("reader.copy_image", "Sao chép ảnh")}</span>
          </button>
          <button
            type="button"
            onClick={handleDownload}
            className="btn btn-sm rounded-xl gap-1.5 bg-[var(--reader-ui-accent,#38bdf8)] text-[var(--reader-ui-accent-text,#08111d)] border-0 hover:opacity-90 font-bold"
          >
            <Download size={14} />
            <span>{t("reader.download_image", "Tải ảnh về")}</span>
          </button>
        </div>
      </div>
    </dialog>
  );
};
