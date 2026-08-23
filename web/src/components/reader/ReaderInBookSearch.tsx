import React, { useState } from 'react';
import { Search, Loader2, X } from 'lucide-react';
import { useTranslation } from 'react-i18next';
import DOMPurify from 'dompurify';
import { readerService } from '@/services';
import type { SearchSnippet } from '@/types';

interface ReaderInBookSearchProps {
  book_id: string;
  onSelectResult: (chapter_id: string, offset: number) => void;
  onClose: () => void;
}

export const ReaderInBookSearch: React.FC<ReaderInBookSearchProps> = ({
  book_id,
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
      const res = await readerService.searchInBook(book_id, query.trim());
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
    <div className="reader-settings-panel p-4 rounded-2xl shadow-2xl w-80 max-h-96 flex flex-col gap-3 backdrop-blur-md border transition-colors duration-300">
      <div className="flex items-center justify-between border-b border-current/10 pb-2">
        <h3 className="font-bold text-xs uppercase tracking-wider opacity-70 flex items-center gap-1.5">
          <Search className="h-4 w-4 text-(--reader-ui-accent)" />
          {t('reader.in_book_search', 'Search in Book')}
        </h3>
        <button type="button" onClick={onClose} className="reader-control-btn btn btn-xs btn-ghost btn-circle">
          <X className="h-4 w-4" />
        </button>
      </div>

      <form onSubmit={handleSearch} className="flex gap-2">
        <input
          type="text"
          value={query}
          onChange={(e) => setQuery(e.target.value)}
          placeholder={t('reader.search_placeholder', 'Search text...')}
          autoFocus
          className="reader-input input input-bordered input-xs w-full text-xs"
        />
        <button type="submit" disabled={loading} className="reader-action-btn btn btn-xs px-3">
          {loading ? <Loader2 className="h-3 w-3 animate-spin" /> : t('common.search', 'Search')}
        </button>
      </form>

      <div className="overflow-y-auto flex-1 flex flex-col gap-2 divide-y divide-(--reader-ui-border)/40 pr-1">
        {results.map((res, idx) => (
          <div
            key={idx}
            onClick={() => onSelectResult(res.chapter_id, res.offset)}
            className="pt-2 cursor-pointer hover:bg-(--reader-ui-hover) p-2 rounded-lg transition-colors"
          >
            <div className="font-semibold text-xs text-(--reader-ui-accent)">{res.chapter_title}</div>
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
