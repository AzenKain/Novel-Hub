import type { ReaderTheme } from "@/stores";

export type ReaderThemeClasses = {
  readerBg: string;
  proseClass: string;
  sidebarBg: string;
  headerBg: string;
  linkColor: string;
  linkColorHover: string;
};

export function getReaderThemeClasses(theme: ReaderTheme): ReaderThemeClasses {
  if (theme === "light") {
    return {
      readerBg: "bg-gray-50 text-gray-900",
      proseClass: "prose",
      sidebarBg: "bg-gray-50 border-gray-200",
      headerBg: "bg-white/90 border-gray-200",
      linkColor: "#2563eb",
      linkColorHover: "#1d4ed8",
    };
  }
  if (theme === "sepia") {
    return {
      readerBg: "bg-[#f4ecd8] text-[#5b4636]",
      proseClass: "prose prose-sepia",
      sidebarBg: "bg-[#f4ecd8] border-[#5b4636]/10",
      headerBg: "bg-[#e8e0cc]/90 border-[#5b4636]/10",
      linkColor: "#7c593c",
      linkColorHover: "#5c3d24",
    };
  }
  if (theme === "warm") {
    return {
      readerBg: "bg-[#fffcf0] text-[#3a352a]",
      proseClass: "prose prose-stone",
      sidebarBg: "bg-[#fffcf0] border-[#3a352a]/10",
      headerBg: "bg-[#f5f1e1]/90 border-[#3a352a]/10",
      linkColor: "#8c6239",
      linkColorHover: "#6e4722",
    };
  }
  if (theme === "coffee") {
    return {
      readerBg: "bg-[#2b211a] text-[#d6c8b3]",
      proseClass: "prose prose-invert",
      sidebarBg: "bg-[#2b211a] border-[#d6c8b3]/10",
      headerBg: "bg-[#1f1712]/90 border-[#d6c8b3]/10",
      linkColor: "#d6c8b3",
      linkColorHover: "#ebdcc5",
    };
  }
  if (theme === "dim") {
    return {
      readerBg: "bg-[#222222] text-[#a0a0a0]",
      proseClass: "prose prose-invert",
      sidebarBg: "bg-[#222222] border-[#a0a0a0]/10",
      headerBg: "bg-[#1a1a1a]/90 border-[#a0a0a0]/10",
      linkColor: "#3b82f6",
      linkColorHover: "#60a5fa",
    };
  }
  if (theme === "eink") {
    return {
      readerBg: "bg-white text-black",
      proseClass: "prose text-black",
      sidebarBg: "bg-white border-black",
      headerBg: "bg-white border-black",
      linkColor: "#000000",
      linkColorHover: "#000000",
    };
  }
  if (theme === "custom") {
    return {
      readerBg: "bg-[var(--custom-reader-bg,#1e1e2e)] text-[var(--custom-reader-text,#e5e7eb)]",
      proseClass: "prose",
      sidebarBg: "bg-[var(--custom-reader-bg,#1e1e2e)] border-white/10",
      headerBg: "bg-[var(--custom-reader-bg,#1e1e2e)]/90 border-white/10",
      linkColor: "var(--custom-reader-accent,#38bdf8)",
      linkColorHover: "var(--custom-reader-accent,#38bdf8)",
    };
  }
  return {
    readerBg: "bg-[#13141b] text-gray-200",
    proseClass: "prose prose-invert",
    sidebarBg: "bg-[#13141b] border-white/10",
    headerBg: "bg-[#13141b]/90 border-white/10",
    linkColor: "#3b82f6",
    linkColorHover: "#2563eb",
  };
}
