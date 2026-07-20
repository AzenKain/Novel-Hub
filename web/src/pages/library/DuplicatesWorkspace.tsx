import { useDuplicatesQuery } from "@/hooks";
import { Loader2 } from "lucide-react";
import { useTranslation } from "react-i18next";
import { Link } from "react-router-dom";

export const DuplicatesWorkspace = () => {
  const { t } = useTranslation();
  const { data: duplicates = [], isLoading: loading } = useDuplicatesQuery();

  return (
    <div className="p-8 min-h-screen bg-base-200">
      <div className="mb-8 flex justify-between items-center bg-base-100 p-6 rounded-2xl shadow-sm border border-base-200">
        <div>
          <h1 className="text-3xl font-bold mb-2">{t('library.duplicate_files', 'Duplicate Files')}</h1>
          <p className="opacity-60">{t('library.manage_duplicates', 'Manage identical files detected by SHA-256 hash.')}</p>
        </div>
        <Link to="/" className="btn btn-ghost">
          {t('library.back_to_library', 'Back to Library')}
        </Link>
      </div>

      {loading ? (
        <div className="flex justify-center p-12">
          <Loader2 className="animate-spin text-primary w-8 h-8" />
        </div>
      ) : duplicates.length === 0 ? (
        <div className="text-center p-12 bg-base-100 rounded-2xl border border-base-200 shadow-sm opacity-60">
          <p>{t('library.no_duplicates', 'No duplicate files found. Your library is clean!')}</p>
        </div>
      ) : (
        <div className="flex flex-col gap-4">
          {duplicates.map((dup) => (
            <div key={dup.hash} className="card bg-base-100 shadow-sm border border-base-200">
              <div className="card-body p-6 gap-4">
                <div className="flex justify-between items-start">
                  <div>
                    <h3 className="font-semibold text-lg text-primary">{dup.duplicate_count} {t('library.identical_files', 'Identical Files')}</h3>
                    <code className="text-xs opacity-60 font-mono mt-1 block">SHA-256: {dup.hash}</code>
                  </div>
                </div>
                <div className="bg-base-200/50 rounded-lg p-4 border border-base-200">
                  <p className="text-sm font-medium mb-2 opacity-70">{t('library.file_ids', 'File IDs:')}</p>
                  <div className="flex flex-col gap-2">
                    {dup.file_ids.split(',').map((id, index) => (
                      <code key={index} className="text-xs font-mono opacity-80">{id}</code>
                    ))}
                  </div>
                </div>
              </div>
            </div>
          ))}
        </div>
      )}
    </div>
  );
};
