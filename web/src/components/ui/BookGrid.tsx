import type { Book } from "@/types";
import React from 'react';
import { BookCard } from './BookCard';

interface BookGridProps {
  books: Book[];
  onBookClick: (book: Book) => void;
  compact?: boolean;
}

export const BookGrid: React.FC<BookGridProps> = ({ books, onBookClick, compact }) => {
  if (!books || books.length === 0) {
    return null;
  }

  return (
    <section
      className={
        compact
          ? "grid grid-cols-3 gap-2.5 sm:grid-cols-4 md:grid-cols-5 lg:grid-cols-6 xl:grid-cols-8 2xl:grid-cols-10 min-[1900px]:grid-cols-12"
          : "grid grid-cols-2 gap-4 sm:grid-cols-3 md:grid-cols-4 lg:grid-cols-5 xl:grid-cols-6 2xl:grid-cols-8 min-[1900px]:grid-cols-10"
      }
      aria-label="Book grid"
    >
      {books.map((book) => (
        <BookCard key={book.id} book={book} onClick={onBookClick} compact={compact} />
      ))}
    </section>
  );
};

