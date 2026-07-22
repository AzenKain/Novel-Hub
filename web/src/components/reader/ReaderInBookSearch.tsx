import React, { useState } from 'react';
import { Search, Loader2, X } from 'lucide-react';
import { useTranslation } from 'react-i18next';
import DOMPurify from 'dompurify';
import { readerService } from '@/services';
import type { SearchSnippet } from '@/types';

interface ReaderInBookSearchProps {
  bookId: string;
  onSelectResult: (chapterId: string, offset: number) => void;
  onClose: () => void;
}

export const ReaderInBookSearch: React.FC<ReaderInBookSearchProps> = ({
  bookId,
  onSelectResult,
  onClose,
}) => {
  const { t } = useTranslation();
  const [query, setQuery] = useState('');
  const [loading, setLoading] = useState(false);
  const [results, setResults] = useState<SearchSnippet[]>([]);
  const [searched, setSearched] = useState(false);

  const handleSearch = async (e: React.SyntheticEvent) => {
    e.preventDefault();
    if (!query.trim()) return;

    setLoading(true);
    setSearched(true);
    try {
      const res = await readerService.searchInBook(bookId, query.trim());
      if (res?.status) {
        setResults(res.data || []);
      }
    } catch (err) {
      console.error('Failed to search in book:', err);
      setResults([]);
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="bg-base-200 border border-base-300 p-4 rounded-xl shadow-lg w-80 max-h-96 flex flex-col gap-3">
      <div className="flex items-center justify-between">
        <h3 className="font-bold text-sm flex items-center gap-1.5">
          <Search className="h-4 w-4 text-primary" />
          {t('reader.in_book_search', 'Search in Book')}
        </h3>
        <button type="button" onClick={onClose} className="btn btn-xs btn-ghost btn-circle">
          <X className="h-4 w-4" />
        </button>
      </div>

      <form onSubmit={handleSearch} className="flex gap-2">
        <input
          type="text"
          value={query}
          onChange={(e) => setQuery(e.target.value)}
          placeholder={t('reader.search_placeholder', 'Search text...')}
          className="input input-bordered input-xs w-full focus:input-primary"
        />
        <button type="submit" disabled={loading} className="btn btn-primary btn-xs">
          {loading ? <Loader2 className="h-3 w-3 animate-spin" /> : t('common.search', 'Search')}
        </button>
      </form>

      <div className="overflow-y-auto flex-1 flex flex-col gap-2 divide-y divide-base-300 pr-1">
        {results.map((res, idx) => (
          <div
            key={idx}
            onClick={() => onSelectResult(res.chapter_id, res.offset)}
            className="pt-2 cursor-pointer hover:bg-base-300/50 p-2 rounded transition-colors"
          >
            <div className="font-semibold text-xs text-primary">{res.chapter_title}</div>
            <div className="text-xs opacity-80 line-clamp-2 mt-0.5" dangerouslySetInnerHTML={{ __html: DOMPurify.sanitize(res.snippet) }} />
          </div>
        ))}

        {searched && !loading && results.length === 0 && (
          <div className="text-xs opacity-60 text-center py-4">
            {t('reader.no_results', 'No matches found.')}
          </div>
        )}
      </div>
    </div>
  );
};
