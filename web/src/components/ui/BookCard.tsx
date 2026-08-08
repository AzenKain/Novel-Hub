import type { Book } from "@/types";
import React from 'react';
import { useTranslation } from 'react-i18next';
import { useNavigate } from 'react-router-dom';
import { parseMetadata } from '@/lib/bookDetail';
import { getMediaUrl } from '@/config/api';

interface BookCardProps {
  book: Book;
  onClick: (book: Book) => void;
  compact?: boolean;
}

const GRADIENTS = [
  "from-[#e85d83] via-[#4657b8] to-[#182033]",
  "from-[#1f9a94] via-[#5c6de0] to-[#20315f]",
  "from-[#c38a28] via-[#e85d83] to-[#4e4a95]",
  "from-[#7f8bd9] via-[#f08ca7] to-[#213255]",
  "from-[#182033] via-[#4657b8] to-[#1f9a94]",
  "from-[#df6071] via-[#c38a28] to-[#35418d]"
];

export const BookCard: React.FC<BookCardProps> = ({ book, onClick, compact }) => {
  const { t } = useTranslation();
  const navigate = useNavigate();
  const charCode = book.id ? book.id.charCodeAt(0) : 0;
  const gradientClass = GRADIENTS[charCode % 6];
  const format = (book.files?.[0]?.format || "BOOK").toUpperCase();
  
  const meta = book.metadata_json ? parseMetadata(book.metadata_json) : {};
  const series = meta.series;
  const author_name = book.author_name || book.author_id || t('library.unknown_author', 'Unknown');

  return (
    <article 
      className="group card card-compact bg-base-100 border border-base-200 shadow-sm cursor-pointer rounded-xl overflow-hidden transition-[border-color,box-shadow,background-color] duration-200 ease-out hover:border-primary/35 hover:shadow-md focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary/35 h-full flex flex-col justify-between"
      onClick={() => onClick(book)}
      tabIndex={0}
      onKeyDown={(event) => {
        if (event.key === "Enter" || event.key === " ") {
          event.preventDefault();
          onClick(book);
        }
      }}
    >
      <figure className={`relative ${compact ? 'aspect-[3/4]' : 'aspect-[3/4.12]'} w-full text-white flex flex-col justify-between ${compact ? 'p-2.5' : 'p-4'} bg-linear-to-br ${gradientClass} shrink-0`}>
        {book.cover_url ? (
          <>
            <img src={getMediaUrl(book.cover_url)} alt={book.title} loading="lazy" className="absolute inset-0 w-full h-full object-cover transition-[filter] duration-150 ease-out motion-reduce:transition-none group-hover:brightness-105" />
            <span className="absolute inset-0 bg-primary/0 transition-colors duration-200 ease-out group-hover:bg-primary/3" />
          </>
        ) : (
          <>
            <div className="absolute inset-0 bg-black/25 pointer-events-none" />
            <small className={`z-10 font-black ${compact ? 'text-[8px]' : 'text-[10px]'} tracking-wider opacity-90`}>NOVEL</small>
            <strong className={`z-10 ${compact ? 'text-sm' : 'text-lg'} leading-tight line-clamp-3 font-bold`} style={{ textShadow: '0 2px 4px rgba(0,0,0,0.5)' }}>{book.title}</strong>
            <small className={`z-10 font-black ${compact ? 'text-[8px]' : 'text-[10px]'} tracking-wider opacity-90 self-end`}>{format}</small>
          </>
        )}
      </figure>
      <div className={`card-body ${compact ? 'p-1.5 gap-0.5' : 'p-2 gap-1'} flex-1 flex flex-col justify-start`}>
        <strong className={`${compact ? 'text-xs' : 'text-base'} line-clamp-2 leading-tight transition-colors duration-150 group-hover:text-primary`} title={book.title}>{book.title}</strong>
        <p 
          className={`${compact ? 'text-[11px]' : 'text-sm'} text-base-content/70 line-clamp-1 hover:text-primary hover:underline cursor-pointer`}
          title={author_name}
          onClick={(e) => {
            e.stopPropagation();
            navigate(`/?nav=authors&facet=author&name=${encodeURIComponent(author_name)}`);
          }}
        >
          {author_name}
        </p>
        {series && (
          <p 
            className={`${compact ? 'text-[11px]' : 'text-sm'} text-secondary/80 font-medium line-clamp-1 hover:text-secondary hover:underline cursor-pointer`}
            title={series}
            onClick={(e) => {
              e.stopPropagation();
              navigate(`/?nav=series&facet=series&name=${encodeURIComponent(series)}`);
            }}
          >
            {series}
          </p>
        )}
      </div>
    </article>
  );
};
