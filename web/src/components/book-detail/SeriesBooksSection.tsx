import React, { useRef, useEffect, useState } from "react";
import { useNavigate } from "react-router-dom";
import { useTranslation } from "react-i18next";
import { Layers, ChevronRight, ChevronLeft, BookOpen } from "lucide-react";
import { useSeriesBooksQuery } from "@/hooks/useBooksQuery";
import { getMediaUrl } from "@/config/api";
import type { Book } from "@/types";

interface SeriesBooksSectionProps {
  currentBookId: string;
  seriesId: string;
  seriesName: string;
  seriesIndex?: string;
}

export const SeriesBooksSection: React.FC<SeriesBooksSectionProps> = ({
  currentBookId,
  seriesId,
  seriesName,
}) => {
  const { t } = useTranslation();
  const navigate = useNavigate();
  const { data: books = [], isLoading } = useSeriesBooksQuery(
    seriesId,
    Boolean(seriesId),
  );

  const containerRef = useRef<HTMLDivElement>(null);
  const isDraggingRef = useRef(false);
  const isDownRef = useRef(false);
  const startXRef = useRef(0);
  const scrollLeftRef = useRef(0);
  const lastXRef = useRef(0);
  const lastTimeRef = useRef(0);
  const velocityRef = useRef(0);
  const animIdRef = useRef<number | null>(null);

  const [canScrollLeft, setCanScrollLeft] = useState(false);
  const [canScrollRight, setCanScrollRight] = useState(false);

  const updateScrollButtons = () => {
    const el = containerRef.current;
    if (!el) return;
    setCanScrollLeft(el.scrollLeft > 10);
    setCanScrollRight(el.scrollLeft < el.scrollWidth - el.clientWidth - 10);
  };

  useEffect(() => {
    updateScrollButtons();
  }, [books]);

  const stopInertia = () => {
    if (animIdRef.current !== null) {
      cancelAnimationFrame(animIdRef.current);
      animIdRef.current = null;
    }
  };

  const scrollByAmount = (offset: number) => {
    if (containerRef.current) {
      containerRef.current.scrollBy({
        left: offset,
        behavior: "smooth",
      });
      setTimeout(updateScrollButtons, 350);
    }
  };

  const handleMouseDown = (e: React.MouseEvent) => {
    const el = containerRef.current;
    if (!el || e.button !== 0) return;

    stopInertia();
    isDownRef.current = true;
    isDraggingRef.current = false;
    startXRef.current = e.pageX;
    scrollLeftRef.current = el.scrollLeft;
    lastXRef.current = e.pageX;
    lastTimeRef.current = performance.now();
    velocityRef.current = 0;

    el.style.scrollBehavior = "auto";
    el.style.scrollSnapType = "none";
    el.classList.add("cursor-grabbing");
    el.classList.remove("cursor-grab");

    const onMouseMove = (moveEvent: MouseEvent) => {
      if (!isDownRef.current || !containerRef.current) return;
      const currentX = moveEvent.pageX;
      const deltaX = currentX - startXRef.current;

      if (Math.abs(deltaX) > 4) {
        isDraggingRef.current = true;
      }

      const now = performance.now();
      const dt = Math.max(now - lastTimeRef.current, 1);
      const dx = currentX - lastXRef.current;
      velocityRef.current = dx / dt;

      lastXRef.current = currentX;
      lastTimeRef.current = now;

      containerRef.current.scrollLeft = scrollLeftRef.current - deltaX;
      updateScrollButtons();
    };

    const onMouseUp = () => {
      isDownRef.current = false;
      window.removeEventListener("mousemove", onMouseMove);
      window.removeEventListener("mouseup", onMouseUp);

      if (containerRef.current) {
        containerRef.current.classList.remove("cursor-grabbing");
        containerRef.current.classList.add("cursor-grab");

        const velocity = velocityRef.current * 16;
        if (Math.abs(velocity) > 1.2) {
          let currentVelocity = velocity;
          const stepInertia = () => {
            const track = containerRef.current;
            if (!track) return;
            track.scrollLeft -= currentVelocity;
            currentVelocity *= 0.94;

            if (Math.abs(currentVelocity) > 0.3) {
              animIdRef.current = requestAnimationFrame(stepInertia);
            } else {
              track.style.scrollBehavior = "";
              track.style.scrollSnapType = "";
              animIdRef.current = null;
              updateScrollButtons();
            }
          };
          animIdRef.current = requestAnimationFrame(stepInertia);
        } else {
          containerRef.current.style.scrollBehavior = "";
          containerRef.current.style.scrollSnapType = "";
          updateScrollButtons();
        }
      }

      setTimeout(() => {
        isDraggingRef.current = false;
      }, 60);
    };

    window.addEventListener("mousemove", onMouseMove, { passive: true });
    window.addEventListener("mouseup", onMouseUp);
  };

  useEffect(() => {
    return () => {
      stopInertia();
    };
  }, []);

  const handleWheel = (e: React.WheelEvent) => {
    if (!containerRef.current) return;
    if (Math.abs(e.deltaY) > Math.abs(e.deltaX)) {
      containerRef.current.scrollLeft += e.deltaY;
      updateScrollButtons();
    }
  };

  const handleBookClick = (b: Book) => {
    if (isDraggingRef.current) return;
    navigate(`/books/${encodeURIComponent(b.id)}`);
  };

  if (!seriesId || (books.length <= 1 && !isLoading)) {
    return null;
  }

  return (
    <div className="pt-4 border-t border-base-200 mt-4">
      <div className="flex items-center justify-between gap-2 mb-3 min-w-0">
        <div className="flex items-center gap-2 min-w-0 flex-wrap sm:flex-nowrap">
          <Layers className="w-4 h-4 text-primary shrink-0" />
          <h3 className="text-base font-bold text-base-content shrink-0">
            {t("book.in_same_series", "Books in this Series")}
          </h3>
          <span
            className="badge badge-sm badge-ghost font-medium truncate max-w-45 sm:max-w-xs md:max-w-sm"
            title={seriesName}
          >
            {seriesName}
          </span>
        </div>
        <button
          onClick={() =>
            navigate(
              `/?nav=series&facet=series&facet_id=${encodeURIComponent(seriesId)}&name=${encodeURIComponent(seriesName)}`,
            )
          }
          className="btn btn-ghost btn-xs gap-1 text-primary hover:bg-primary/10 shrink-0"
        >
          <span>{t("common.view_all", "View All")}</span>
          <ChevronRight className="w-3.5 h-3.5" />
        </button>
      </div>

      {isLoading ? (
        <div className="flex gap-3 overflow-x-auto pb-2">
          {[1, 2, 3, 4].map((i) => (
            <div
              key={i}
              className="w-28 shrink-0 flex flex-col gap-2 animate-pulse"
            >
              <div className="w-28 h-40 bg-base-300 rounded-lg" />
              <div className="h-3 bg-base-300 rounded w-3/4" />
            </div>
          ))}
        </div>
      ) : (
        <div className="relative group/series">
          {canScrollLeft && (
            <button
              type="button"
              aria-label="Scroll series left"
              onClick={() => scrollByAmount(-360)}
              className="hidden sm:flex absolute -left-3 top-1/2 -translate-y-1/2 z-20 w-8 h-8 items-center justify-center rounded-full bg-base-100/90 hover:bg-base-100 border border-base-200/80 shadow-md text-base-content/80 hover:text-primary backdrop-blur-md transition-all duration-200 opacity-0 group-hover/series:opacity-100 hover:scale-110 active:scale-95 cursor-pointer"
            >
              <ChevronLeft className="w-4 h-4" />
            </button>
          )}

          {canScrollRight && (
            <button
              type="button"
              aria-label="Scroll series right"
              onClick={() => scrollByAmount(360)}
              className="hidden sm:flex absolute -right-3 top-1/2 -translate-y-1/2 z-20 w-8 h-8 items-center justify-center rounded-full bg-base-100/90 hover:bg-base-100 border border-base-200/80 shadow-md text-base-content/80 hover:text-primary backdrop-blur-md transition-all duration-200 opacity-0 group-hover/series:opacity-100 hover:scale-110 active:scale-95 cursor-pointer"
            >
              <ChevronRight className="w-4 h-4" />
            </button>
          )}

          <div
            ref={containerRef}
            onMouseDown={handleMouseDown}
            onWheel={handleWheel}
            onScroll={updateScrollButtons}
            className="flex gap-3 overflow-x-auto pb-2 scrollbar-none scroll-smooth snap-x snap-mandatory cursor-grab select-none"
          >
            {books.map((b: Book) => {
              const isCurrent = b.id === currentBookId;
              return (
                <div
                  key={b.id}
                  onClick={() => handleBookClick(b)}
                  className={`group w-28 shrink-0 cursor-pointer flex flex-col gap-1.5 transition-transform hover:-translate-y-1 ${
                    isCurrent ? "opacity-100" : "opacity-85 hover:opacity-100"
                  }`}
                >
                  <div className="relative w-28 h-40 rounded-lg overflow-hidden border border-base-300 shadow-2xs bg-base-200 group-hover:border-primary transition-colors select-none">
                    {b.cover_url ? (
                      <img
                        src={getMediaUrl(b.cover_url)}
                        alt={b.title}
                        className="w-full h-full object-cover pointer-events-none"
                        loading="lazy"
                        draggable={false}
                      />
                    ) : (
                      <div className="w-full h-full flex items-center justify-center text-base-content/40">
                        <BookOpen className="w-8 h-8" />
                      </div>
                    )}

                    {isCurrent && (
                      <span className="absolute top-1.5 right-1.5 badge badge-primary badge-xs font-bold text-[9px] shadow-xs">
                        {t("common.current", "Current")}
                      </span>
                    )}
                  </div>

                  <div className="min-w-0">
                    <p
                      className="text-xs font-semibold text-base-content line-clamp-2 leading-snug group-hover:text-primary transition-colors"
                      title={b.title}
                    >
                      {b.title}
                    </p>
                    {b.author_name && (
                      <p className="text-[11px] text-base-content/60 truncate mt-0.5">
                        {b.author_name}
                      </p>
                    )}
                  </div>
                </div>
              );
            })}
          </div>
        </div>
      )}
    </div>
  );
};
