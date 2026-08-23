import React, { useRef, useEffect } from "react";
import { ChevronRight } from "lucide-react";
import { BookCard } from "@/components/ui/BookCard";
import type { Book } from "@/types";

interface HorizontalBookShelfProps {
  books: Book[];
  onBookClick: (book: Book) => void;
  title?: React.ReactNode;
  subtitle?: React.ReactNode;
  icon?: React.ReactNode;
  headerActions?: React.ReactNode;
  onViewAll?: () => void;
  viewAllText?: string;
  emptyMessage?: string;
  itemWidthClass?: string;
  className?: string;
}

export const HorizontalBookShelf: React.FC<HorizontalBookShelfProps> = ({
  books,
  onBookClick,
  title,
  subtitle,
  icon,
  headerActions,
  onViewAll,
  viewAllText,
  emptyMessage,
  itemWidthClass = "w-36 sm:w-44",
  className = "",
}) => {
  const containerRef = useRef<HTMLDivElement>(null);
  const isDraggingRef = useRef(false);
  const isDownRef = useRef(false);
  const startXRef = useRef(0);
  const scrollLeftRef = useRef(0);
  const lastXRef = useRef(0);
  const lastTimeRef = useRef(0);
  const velocityRef = useRef(0);
  const animIdRef = useRef<number | null>(null);

  const stopInertia = () => {
    if (animIdRef.current !== null) {
      cancelAnimationFrame(animIdRef.current);
      animIdRef.current = null;
    }
  };

  const handleMouseDown = (e: React.MouseEvent) => {
    const el = containerRef.current;
    if (!el) return;
    if (e.button !== 0) return; // Left click only

    stopInertia();
    isDownRef.current = true;
    isDraggingRef.current = false;
    startXRef.current = e.pageX;
    scrollLeftRef.current = el.scrollLeft;
    lastXRef.current = e.pageX;
    lastTimeRef.current = performance.now();
    velocityRef.current = 0;

    // Eliminate snap & smooth scroll lag during direct mouse drag
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
    };

    const onMouseUp = () => {
      isDownRef.current = false;
      window.removeEventListener("mousemove", onMouseMove);
      window.removeEventListener("mouseup", onMouseUp);

      if (containerRef.current) {
        containerRef.current.classList.remove("cursor-grabbing");
        containerRef.current.classList.add("cursor-grab");

        // Physics-based momentum glide
        let velocity = velocityRef.current * 16;
        if (Math.abs(velocity) > 1.2) {
          const stepInertia = () => {
            const track = containerRef.current;
            if (!track) return;
            track.scrollLeft -= velocity;
            velocity *= 0.93; // smooth friction

            if (Math.abs(velocity) > 0.3) {
              animIdRef.current = requestAnimationFrame(stepInertia);
            } else {
              track.style.scrollBehavior = "";
              track.style.scrollSnapType = "";
              animIdRef.current = null;
            }
          };
          animIdRef.current = requestAnimationFrame(stepInertia);
        } else {
          containerRef.current.style.scrollBehavior = "";
          containerRef.current.style.scrollSnapType = "";
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

  const handleBookClick = (book: Book) => {
    if (isDraggingRef.current) return;
    onBookClick(book);
  };

  const content = (
    <>
      {title && (
        <div className="mb-4 flex items-center justify-between gap-3">
          <div className="flex items-center gap-3 min-w-0">
            {icon}
            <div className="min-w-0">
              <h3 className="text-lg font-black truncate">{title}</h3>
              {subtitle && (
                <p className="text-sm text-base-content/50 truncate">{subtitle}</p>
              )}
            </div>
          </div>

          <div className="flex items-center gap-3 shrink-0">
            {headerActions}
            {onViewAll && (
              <button
                type="button"
                onClick={onViewAll}
                className="btn btn-ghost btn-xs text-xs font-bold text-primary hover:bg-primary/10 gap-1 rounded-lg cursor-pointer"
              >
                <span>{viewAllText || "Xem toàn bộ"}</span>
                <ChevronRight className="w-3.5 h-3.5" />
              </button>
            )}
          </div>
        </div>
      )}

      {!title && headerActions && (
        <div className="flex items-center justify-end gap-1 mb-2">
          {headerActions}
        </div>
      )}

      {!books || books.length === 0 ? (
        emptyMessage ? (
          <div className="rounded-xl border border-dashed border-base-300 bg-base-100 p-8 text-center text-sm text-base-content/45 shadow-2xs">
            {emptyMessage}
          </div>
        ) : null
      ) : (
        <div
          ref={containerRef}
          onMouseDown={handleMouseDown}
          onDragStart={(e) => e.preventDefault()}
          className="flex gap-4 overflow-x-auto pb-1 scrollbar-none snap-x snap-mandatory items-stretch select-none cursor-grab active:cursor-grabbing -mx-1 px-1 touch-pan-x"
          style={{ scrollbarWidth: "none", msOverflowStyle: "none" }}
        >
          {books.map((book) => (
            <div
              key={book.id}
              className={`${itemWidthClass} shrink-0 flex flex-col snap-start pointer-events-auto`}
            >
              <BookCard book={book} onClick={handleBookClick} />
            </div>
          ))}
        </div>
      )}
    </>
  );

  if (title) {
    return (
      <section
        className={`rounded-2xl bg-base-100 shadow-sm border border-base-200 p-4 sm:p-5 ${className}`}
      >
        {content}
      </section>
    );
  }

  return <div className={`relative ${className}`}>{content}</div>;
};
