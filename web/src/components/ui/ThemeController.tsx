import { Theme, useSettingsStore } from "@/stores";
import {
  Coffee,
  Heart,
  Monitor,
  Moon,
  Sun,
  Sparkles,
  Flame,
  Zap,
  Compass,
  TreePine,
  Radio,
  Eye,
  Palette,
} from "lucide-react";
import React from "react";
import { useTranslation } from "react-i18next";
import { useShallow } from "zustand/react/shallow";

export const ThemeController = ({
  className = "dropdown-end",
}: {
  className?: string;
}) => {
  const { t } = useTranslation();
  const { theme, setTheme } = useSettingsStore(
    useShallow((state) => ({ theme: state.theme, setTheme: state.setTheme })),
  );

  const themes: { value: Theme; label: string; icon: React.ReactNode }[] = [
    {
      value: "system",
      label: t("common.system", "System"),
      icon: <Monitor className="w-4 h-4 text-primary" />,
    },
    {
      value: "winter",
      label: t("common.winter", "Winter"),
      icon: <Sun className="w-4 h-4 text-blue-400" />,
    },
    {
      value: "night",
      label: t("common.night", "Night"),
      icon: <Moon className="w-4 h-4 text-indigo-400" />,
    },
    {
      value: "nord",
      label: "Nord",
      icon: <Compass className="w-4 h-4 text-cyan-400" />,
    },
    {
      value: "dracula",
      label: "Dracula",
      icon: <Sparkles className="w-4 h-4 text-purple-400" />,
    },
    {
      value: "sunset",
      label: "Sunset",
      icon: <Flame className="w-4 h-4 text-orange-400" />,
    },
    {
      value: "cyberpunk",
      label: "Cyberpunk",
      icon: <Zap className="w-4 h-4 text-yellow-400" />,
    },
    {
      value: "catppuccin",
      label: "Catppuccin",
      icon: <Palette className="w-4 h-4 text-pink-400" />,
    },
    {
      value: "emerald",
      label: "Emerald",
      icon: <TreePine className="w-4 h-4 text-emerald-400" />,
    },
    {
      value: "synthwave",
      label: "Synthwave",
      icon: <Radio className="w-4 h-4 text-fuchsia-400" />,
    },
    {
      value: "cupcake",
      label: t("common.cupcake", "Cupcake"),
      icon: <Heart className="w-4 h-4 text-rose-400" />,
    },
    {
      value: "coffee",
      label: t("common.coffee", "Coffee"),
      icon: <Coffee className="w-4 h-4 text-amber-600" />,
    },
    {
      value: "dim",
      label: "Dim",
      icon: <Eye className="w-4 h-4 text-slate-400" />,
    },
  ];

  const activeIcon = themes.find((item) => item.value === theme)?.icon || (
    <Palette className="w-4 h-4" />
  );

  return (
    <div className={`dropdown ${className}`}>
      <div
        tabIndex={0}
        role="button"
        className="btn btn-ghost btn-sm m-1 flex items-center gap-1"
      >
        {activeIcon}
      </div>
      <ul
        tabIndex={0}
        className="dropdown-content z-50 menu p-2 shadow-xl bg-base-100 rounded-box w-48 max-h-80 overflow-y-auto border border-base-200 flex-nowrap"
      >
        {themes.map((tItem) => (
          <li key={tItem.value}>
            <button
              onClick={() => setTheme(tItem.value)}
              className={`flex items-center gap-2 text-sm ${theme === tItem.value ? "active font-semibold" : ""}`}
            >
              {tItem.icon}
              <span>{tItem.label}</span>
            </button>
          </li>
        ))}
      </ul>
    </div>
  );
};
