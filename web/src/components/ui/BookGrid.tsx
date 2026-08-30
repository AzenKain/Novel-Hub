import type { Book } from "@/types";
import React from 'react';
import { useTranslation } from 'react-i18next';
import { BookCard } from './BookCard';

interface BookGridProps {
  books: Book[];
  onBookClick: (book: Book) => void;
  compact?: boolean;
}

export const BookCardSkeleton: React.FC<{ compact?: boolean }> = ({ compact }) => (
  <div className={`card card-compact bg-base-100/60 border border-base-200/60 rounded-xl overflow-hidden shadow-2xs ${compact ? 'gap-1' : 'gap-1.5'}`}>
    <div className={`relative ${compact ? 'aspect-3/4' : 'aspect-[3/4.12]'} w-full animate-shimmer rounded-t-xl`} />
    <div className={`card-body ${compact ? 'p-1.5 gap-1.5' : 'p-2.5 gap-2'} flex-1 flex flex-col justify-start`}>
      <div className="h-3.5 w-4/5 rounded-md animate-shimmer" />
      <div className="h-3 w-1/2 rounded-md animate-shimmer opacity-70" />
    </div>
  </div>
);

export const BookGridSkeleton: React.FC<{ count?: number; compact?: boolean }> = ({ count = 10, compact }) => (
  <section
    className={
      compact
        ? "grid grid-cols-3 gap-2.5 sm:grid-cols-4 md:grid-cols-5 lg:grid-cols-6 2xl:grid-cols-7"
        : "grid grid-cols-2 gap-4 sm:grid-cols-3 md:grid-cols-4 lg:grid-cols-4 xl:grid-cols-5 2xl:grid-cols-6"
    }
  >
    {Array.from({ length: count }).map((_, i) => (
      <BookCardSkeleton key={i} compact={compact} />
    ))}
  </section>
);

export const BookGrid: React.FC<BookGridProps> = ({ books, onBookClick, compact }) => {
  const { t } = useTranslation();
  if (!books || books.length === 0) {
    return null;
  }

  return (
    <section
      className={
        compact
          ? "grid grid-cols-3 gap-2.5 sm:grid-cols-4 md:grid-cols-5 lg:grid-cols-6 2xl:grid-cols-7"
          : "grid grid-cols-2 gap-4 sm:grid-cols-3 md:grid-cols-4 lg:grid-cols-4 xl:grid-cols-5 2xl:grid-cols-6"
      }
      aria-label={t("library.book_grid")}
    >
      {books.map((book) => (
        <BookCard key={book.id} book={book} onClick={onBookClick} compact={compact} />
      ))}
    </section>
  );
};

