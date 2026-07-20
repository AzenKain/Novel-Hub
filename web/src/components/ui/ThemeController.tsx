import { Theme, useSettingsStore } from '@/stores';
import { Coffee, Heart, Monitor, Moon, Sun } from 'lucide-react';
import React from 'react';
import { useTranslation } from 'react-i18next';

export const ThemeController = ({ className = 'dropdown-end' }: { className?: string }) => {
  const { t } = useTranslation();
  const { theme, setTheme } = useSettingsStore();

  const themes: { value: Theme; label: string; icon: React.ReactNode }[] = [
    { value: 'system', label: t('common.system', 'System'), icon: <Monitor className="w-4 h-4" /> },
    { value: 'winter', label: t('common.winter', 'Winter'), icon: <Sun className="w-4 h-4" /> },
    { value: 'night', label: t('common.night', 'Night'), icon: <Moon className="w-4 h-4" /> },
    { value: 'cupcake', label: t('common.cupcake', 'Cupcake'), icon: <Heart className="w-4 h-4" /> },
    { value: 'coffee', label: t('common.coffee', 'Coffee'), icon: <Coffee className="w-4 h-4" /> },
  ];

  return (
    <div className={`dropdown ${className}`}>
      <div tabIndex={0} role="button" className="btn btn-ghost btn-sm m-1">
        {themes.find(t => t.value === theme)?.icon}
      </div>
      <ul tabIndex={0} className="dropdown-content z-2 menu p-2 shadow bg-base-100 rounded-box w-40">
        {themes.map((tItem) => (
          <li key={tItem.value}>
            <button 
              onClick={() => setTheme(tItem.value)}
              className={theme === tItem.value ? 'active' : ''}
            >
              {tItem.icon}
              {tItem.label}
            </button>
          </li>
        ))}
      </ul>
    </div>
  );
};
