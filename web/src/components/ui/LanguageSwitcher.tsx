import { Language, useSettingsStore } from '@/stores';
import { Globe } from 'lucide-react';
import { useEffect } from 'react';
import { useTranslation } from 'react-i18next';

const LANGUAGES: { code: Language; label: string }[] = [
  { code: 'en', label: 'English' },
  { code: 'vi', label: 'Tiếng Việt' },
  { code: 'ja', label: '日本語' },
  { code: 'zh', label: '中文' },
  { code: 'ko', label: '한국어' },
];

export function LanguageSwitcher({ className = 'dropdown-end' }: { className?: string }) {
  const { i18n } = useTranslation();
  const { language, setLanguage } = useSettingsStore();

  useEffect(() => {
    if (i18n.language !== language) {
      i18n.changeLanguage(language);
    }
  }, [language, i18n]);

  return (
    <div className={`dropdown ${className}`}>
      <div tabIndex={0} role="button" className="btn btn-ghost btn-sm m-1 gap-1" title="Change Language">
        <Globe className="w-4 h-4" />
        <span className="text-xs font-medium uppercase">{language}</span>
      </div>
      <ul tabIndex={0} className="dropdown-content z-[2] menu p-2 shadow bg-base-100 rounded-box w-40">
        {LANGUAGES.map((lang) => (
          <li key={lang.code}>
            <button
              onClick={() => {
                setLanguage(lang.code);
                (document.activeElement as HTMLElement)?.blur();
              }}
              className={language === lang.code ? 'active' : ''}
            >
              {lang.label}
            </button>
          </li>
        ))}
      </ul>
    </div>
  );
}
