import type { TFunction } from "i18next";
import {
  ArrowLeft,
  Bookmark,
  BookOpen,
  ChevronRight,
  FileText,
  ListTree,
  PanelLeftClose,
  Search,
  Sparkles,
  Trash2,
  X,
} from "lucide-react";
import React, { useEffect, useMemo, useRef, useState } from "react";

import type { AudioBookmark, Book, Chapter, Highlight, ImageBookmark, NextInSeries } from "@/types";
import { ReaderHighlightsPanel } from "./ReaderHighlightsPanel";
import { getMediaUrl } from "@/config/api";

type ReaderSidebarProps = {
  t: TFunction;
  book: Book;
  chapters: Chapter[];
  currentChapter: Chapter | null;
  sidebarBg: string;
  sidebarRef: React.RefObject<HTMLElement | null>;
  onClose: () => void;
  onBack: () => void;
  onSelectChapter: (chapter: Chapter) => void;
  highlights?: Highlight[];
  onUpdateHighlight?: (id: string, color: string, note?: string) => void;
  onDeleteHighlight?: (id: string) => void;
  onSelectHighlight?: (highlight: Highlight) => void;
  onOpenQuoteCard?: (text?: string, imageUrl?: string) => void;
  nextInSeries?: NextInSeries | null;
  nextInReadList?: Book | null;
  onGoToNextInSeries?: () => void;
  onGoToNextInReadList?: () => void;
  isAudio?: boolean;
  audioBookmarks?: AudioBookmark[];
  onSelectAudioBookmark?: (time_sec: number) => void;
  onDeleteAudioBookmark?: (id: string) => void;
  isVisualContent?: boolean;
  imageBookmarks?: ImageBookmark[];
  onSelectImageBookmark?: (bookmark: ImageBookmark) => void;
  onDeleteImageBookmark?: (id: string) => void;
};

// Common CJK and standard punctuation delimiters for word boundaries
const DELIMITERS =
  "\\s\\d:._\\-—–/\\\\|~#*(\\[\\]{}【】〔〕《》「」『』,;?!'\"：、。！？～・·（）　";

// Section-level entries (volumes, parts, books, arcs, sagas, tomes, etc.)
export const SECTION_RE = new RegExp(
  "^(" +
    [
      // English / Romance / Germanic / Slavic
      "volume",
      "vol\\.?",
      "v\\.",
      "book",
      "bk\\.?",
      "tome",
      "tomo",
      "volumen",
      "partie",
      "parte",
      "livre",
      "libro",
      "buch",
      "band",
      "teil",
      "arc",
      "act",
      "acte",
      "acto",
      "atto",
      "akt",
      "saga",
      "season",
      "staffel",
      "tom",
      "czesc",
      "jilid",
      "bagian",
      "buku",
      "babak",
      // Vietnamese sections (Quyển, Tập, Phần, Bộ, Phân đoạn)
      "quyển(?:\\s+(?:thứ|số))?",
      "tập(?:\\s+(?:thứ|số))?",
      "phần(?:\\s+(?:thứ|số))?",
      "bộ(?:\\s+(?:thứ|số))?",
      // Russian
      "том",
      "часть",
      "книга",
      "сезон",
      "выпуск",
      // Thai
      "เล่ม",
      "ภาค",
      // Hindi
      "भाग",
      "खंड",
      // Arabic
      "مجلد",
      "جزء",
      "كتاب",
      "قسم",
      // CJK sections (卷, 部, 篇, 集, 册, 巻, 編, 권, 부, 막)
      "(?:第[0-9一二三四五六七八九十百]+|[0-9]+)?\\s*(?:卷|分卷|部|分部|篇|集|册|冊|巻|編)",
      "(?:제[0-9일이삼사오육칠팔구십]+|[0-9]+)?\\s*(?:권|부|막)",
      "巻之[0-9一二三四五六七八九十百]+",
      "卷[0-9一二三四五六七八九十百]+",
      // Part with number or standalone (e.g. "Part 1", "Part I", "Part One")
      "part(?:\\s+(?:[0-9ivxlcdm]+|one|two|three|four|five|six|seven|eight|nine|ten))?",
    ].join("|") +
    ")(?=$|[" +
    DELIMITERS +
    "])",
  "iu",
);

// Special entries (prologues, epilogues, side stories, extra, bonus, afterwords, cover, TOC, end chapters, etc.)
export const SPECIAL_RE = new RegExp(
  "^(" +
    [
      // --- Vietnamese ---
      // Mở đầu / Lời dẫn / Lời nói đầu / Lời tựa / v.v.
      "lời\\s+mở\\s+đầu",
      "lời\\s+nói\\s+đầu",
      "lời\\s+dẫn",
      "lời\\s+tựa",
      "phần\\s+mở(?:\\s+đầu)?",
      "đoạn\\s+mở(?:\\s+đầu)?",
      "mở\\s+đầu",
      "tiền\\s+truyện",
      "phi\\s+lộ",
      "khởi\\s+đầu",
      "dẫn\\s+nhập",
      "nhập\\s+môn",
      "chương\\s+mở(?:\\s+đầu)?",
      // Kết thúc / Vĩ thanh / Lời bạt / Chương cuối / Chương kết / Kết đoạn / v.v.
      "chương\\s+cuối(?:\\s+cùng)?",
      "chương\\s+kết",
      "hồi\\s+kết",
      "kết\\s+đoạn",
      "đoạn\\s+kết",
      "phần\\s+kết",
      "kết\\s+thúc",
      "kết\\s+chương",
      "kết\\s+cục",
      "vĩ\\s+thanh",
      "vĩ\\s+khúc",
      "chung\\s+chương",
      "chung\\s+khúc",
      "lời\\s+bạt",
      "lời\\s+kết",
      "hậu\\s+ký",
      "lời\\s+cảm\\s+ơn",
      "lời\\s+tri\\s+ân",
      "tổng\\s+kết",
      "cuối\\s+sách",
      "bạt",
      // Ngoại truyện / Phiên ngoại / Phụ chương / Chương đặc biệt / v.v.
      "ngoại\\s+truyện",
      "phiên\\s+ngoại",
      "phụ\\s+chương",
      "chương\\s+phụ",
      "chương\\s+đặc\\s+biệt",
      "ngoại\\s+chương",
      "truyện\\s+phụ",
      "mẩu\\s+chuyện",
      "đoạn\\s+ngắn",
      "chuyện\\s+bên\\s+lề",
      "khoảng\\s+nghỉ",
      "chương\\s+xen",
      // Bìa / Mục lục / Minh họa / Bản quyền / Phụ lục
      "bìa(?:\\s+(?:sách|trước|sau))?",
      "trang\\s+bìa",
      "mục\\s+lục",
      "minh\\s+họa",
      "tranh\\s+minh\\s+họa",
      "phụ\\s+lục",
      "thông\\s+tin\\s+(?:sách|xuất\\s+bản)",
      "giới\\s+thiệu\\s+nhân\\s+vật",
      "hồ\\s+sơ\\s+nhân\\s+vật",
      "bản\\s+quyền",
      "ghi\\s+chú",
      "chú\\s+thích",

      // --- English ---
      "prologue",
      "prolog",
      "prelude",
      "preface",
      "foreword",
      "introduction",
      "intro",
      "lead-in",
      "opening",
      "epilogue",
      "epilog",
      "afterword",
      "postscript",
      "postlude",
      "conclusion",
      "final\\s+chapter",
      "last\\s+chapter",
      "the\\s+end",
      "ending",
      "author'?s\\s+note",
      "author\\s+note",
      "coda",
      "finale",
      "acknowledgments?",
      "acknowledgements?",
      "valediction",
      "side\\s+stor(?:y|ies)",
      "extra(?:\\s+chapter)?",
      "bonus(?:\\s+chapter)?",
      "special(?:\\s+chapter)?",
      "interlude",
      "intermission",
      "spin-off",
      "omake",
      "gaiden",
      "short\\s+story",
      "one-?shot",
      "cover",
      "front\\s+cover",
      "back\\s+cover",
      "title\\s+page",
      "half\\s+title",
      "table\\s+of\\s+contents",
      "contents",
      "toc",
      "illustrations?",
      "color\\s+(?:pages?|inserts?|illustrations?)",
      "character\\s+(?:profiles?|intro|introductions?)",
      "characters?",
      "cast",
      "dramatis\\s+personae",
      "maps?",
      "glossary",
      "appendix",
      "credits",
      "copyright",
      "colophon",
      "about\\s+the\\s+author",

      // --- Chinese (Simplified & Traditional) ---
      "序章",
      "序言",
      "序幕",
      "前言",
      "楔子",
      "引子",
      "引言",
      "导言",
      "導言",
      "卷首",
      "开篇",
      "開篇",
      "发端",
      "發端",
      "题记",
      "題記",
      "终章",
      "終章",
      "尾声",
      "尾聲",
      "后记",
      "後記",
      "结语",
      "結語",
      "结局",
      "結局",
      "完结",
      "完結",
      "大结局",
      "大結局",
      "完结感言",
      "跋",
      "余话",
      "餘話",
      "后话",
      "後話",
      "终曲",
      "終曲",
      "最终章",
      "最終章",
      "最后一章",
      "最後一章",
      "谢幕",
      "謝幕",
      "感言",
      "番外(?:篇)?",
      "外传",
      "外傳",
      "幕间",
      "幕間",
      "插曲",
      "特别篇",
      "特別篇",
      "小剧场",
      "小劇場",
      "附章",
      "短篇",
      "花絮",
      "别篇",
      "別篇",
      "封面",
      "封底",
      "扉页",
      "扉頁",
      "彩页",
      "彩頁",
      "插图",
      "插圖",
      "人物(?:介绍|介紹|一览|一覽)",
      "目录",
      "目錄",
      "附录",
      "附錄",
      "版权",
      "版權",
      "设定",
      "設定",
      "地图",
      "地圖",
      "年表",
      "术语表",
      "術語表",

      // --- Japanese ---
      "プロローグ",
      "エピローグ",
      "序章",
      "序文",
      "前口上",
      "はじめに",
      "初めに",
      "終章",
      "終曲",
      "あとがき",
      "後記",
      "後書き",
      "解説",
      "結び",
      "おわりに",
      "終わりに",
      "大団円",
      "番外編",
      "番外",
      "特別編",
      "幕間",
      "外伝",
      "おまけ",
      "オマケ",
      "ショートストーリー",
      "短編",
      "閑話",
      "間奏",
      "表紙",
      "裏表紙",
      "口絵",
      "扉絵",
      "挿絵",
      "カラーイラスト",
      "登場人物(?:紹介)?",
      "キャラクター紹介",
      "目次",
      "付録",
      "奥付",
      "用語集",
      "地図",
      "設定資料",

      // --- Korean ---
      "프롤로그",
      "에필로그",
      "서문",
      "서장",
      "머리말",
      "종장",
      "종편",
      "후기",
      "작가의\\s*말",
      "맺음말",
      "완결(?:\\s*감상)?",
      "마지막\\s*이야기",
      "외전(?:편)?",
      "특별편",
      "막간",
      "보너스",
      "단편",
      "비하인드",
      "표지",
      "일러스트",
      "삽화",
      "등장인물(?:\\s*소개)?",
      "인물소개",
      "목차",
      "부록",
      "판권",
      "용어집",

      // --- Spanish, Portuguese, Italian ---
      "pr[oó]logo",
      "ep[ií]logo",
      "pref[aá]cio",
      "prefacio",
      "posf[aá]cio",
      "posfacio",
      "introducci[oó]n",
      "introdu[cç][aã]o",
      "introduzione",
      "preludio",
      "prel[uú]dio",
      "conclusi[oó]n",
      "conclus[aã]o",
      "conclusione",
      "cap[ií]tulo\\s+final",
      "capitolo\\s+finale",
      "[uú]ltimo\\s+cap[ií]tulo",
      "nota\\s+d[eo]l?\\s*autor",
      "agradecimientos?",
      "agradecimentos?",
      "ringraziamenti",
      "interludio",
      "interl[uú]dio",
      "cap[ií]tulo\\s+especial",
      "capitolo\\s+speciale",
      "historia\\s+secundaria",
      "hist[oó]ria\\s+paralela",
      "portada",
      "capa",
      "copertina",
      "cubierta",
      "[ií]ndice",
      "sum[aá]rio",
      "sommario",
      "ilustrac(?:i[oó]n|iones)",
      "ilustra[cç][oõ]es",
      "illustrazioni",
      "personajes?",
      "personagens",
      "personaggi",
      "ap[eé]ndice",
      "ap[eê]ndice",
      "appendice",

      // --- French ---
      "[eé]pilogue",
      "pr[eé]face",
      "avant-propos",
      "postface",
      "pr[eé]lude",
      "dernier\\s+chapitre",
      "mot\\s+de\\s+l'auteur",
      "remerciements?",
      "interm[eè]de",
      "hors-s[eé]rie",
      "chapitre\\s+sp[eé]cial",
      "couverture",
      "sommaire",
      "table\\s+des\\s+mati[eè]res",
      "personnages?",
      "annexe",

      // --- German ---
      "prolog",
      "epilog",
      "vorwort",
      "einleitung",
      "pr[aä]ludium",
      "einf[uü]hrung",
      "nachwort",
      "schlusswort",
      "fazit",
      "danksagung",
      "letztes\\s+kapitel",
      "zwischenspiel",
      "kurzgeschichte",
      "sonderkapitel",
      "titelblatt",
      "inhaltsverzeichnis",
      "farbseiten",
      "charaktere",
      "anhang",

      // --- Russian ---
      "пролог",
      "эпилог",
      "предисловие",
      "послесловие",
      "введение",
      "вступление",
      "заключение",
      "от\\s+автора",
      "благодарности",
      "последняя\\s+глава",
      "экстра",
      "интерлюдия",
      "спецвыпуск",
      "специальная\\s+глава",
      "побочная\\s+история",
      "обложка",
      "содержание",
      "оглавление",
      "действующие\\s+лица",
      "персонажи",
      "приложение",
      "примечания",

      // --- Indonesian ---
      "prakata",
      "(?:kata\\s+)?pengantar",
      "pendahuluan",
      "catatan\\s+penulis",
      "cerita\\s+sampingan",
      "bab\\s+khusus",
      "daftar\\s+isi",
      "sampul",
      "lampiran",

      // --- Thai ---
      "บทนำ",
      "คำนำ",
      "บทส่งท้าย",
      "ปัจฉิมลิขิต",
      "ตอนพิเศษ",
      "ตอนจบ",
      "สารบัญ",
      "ปก",
      "ภาพประกอบ",

      // --- Arabic ---
      "مقدمة",
      "تمهيد",
      "فاتحة",
      "تصدير",
      "خاتمة",
      "نهاية",
      "كلمة\\s+المؤلف",
      "شكر\\s+وتقدير",
      "فصل\\s+إضافي",
      "ملحق",
      "فهرس",
      "غلاف",
      "رسوم",

      // --- Hindi ---
      "प्रस्तावना",
      "भूमिका",
      "उपसंहार",
      "अंतिम\\s+अध्याय",
      "समाप्ति",
      "विशेष\\s+अध्याय",
      "परिशिष्ट",
      "विषय\\s+सूची",
      "आवरण",
      "चित्र",
    ].join("|") +
    ")(?=$|[" +
    DELIMITERS +
    "])",
  "iu",
);

// Regular chapter pattern check (to avoid classifying numbered chapters as special/section)
export const REGULAR_CHAPTER_PREFIX_RE = new RegExp(
  "^(" +
    [
      "chương\\s+[0-9ivxlcdm]+",
      "chương\\s+thứ\\s+[0-9ivxlcdm]+",
      "chapter\\s+[0-9ivxlcdm]+",
      "ch\\.?\\s*[0-9]+",
      "chap\\.?\\s*[0-9]+",
      "cap[ií]tulo\\s+[0-9ivxlcdm]+",
      "cap\\.?\\s*[0-9]+",
      "chapitre\\s+[0-9ivxlcdm]+",
      "kapitel\\s+[0-9ivxlcdm]+",
      "глава\\s+[0-9ivxlcdm]+",
      "hồi\\s+[0-9ivxlcdm]+",
      "hồi\\s+thứ\\s+[0-9ivxlcdm]+",
      "bab\\s+[0-9]+",
      "ตอนที่\\s*[0-9]+",
      "第[0-9一二三四五六七八九十百]+[章話话回]",
      "[0-9]+[章話话回]",
      "[0-9]+화",
      "제[0-9]+[화장]",
    ].join("|") +
    ")(?=$|[" +
    DELIMITERS +
    "])",
  "iu",
);

function cleanTitle(raw: string): string {
  let s = raw.trim();
  s = s.replace(/^[\[\(\{【〔《«「『<★◆#*\-_~—–\s]+/, "");
  s = s.replace(/[\]\)\}】〕》»」』>★◆#*\-_~—–\s]+$/, "");

  // Strip leading numbering if followed by special/section words (e.g. "0. Prologue", "00 - Bìa")
  const strippedNumeric = s.replace(
    /^(?:[0-9]{1,4}(?:[.:\-_—–\s]+|\s+))(?=[a-zA-Z\u00C0-\u024F\u1EA0-\u1EFF\u4E00-\u9FFF\u3040-\u30FF\uAC00-\uD7AF\u0E00-\u0E7F\u0600-\u06FF\u0900-\u097F\u0400-\u04FF])/u,
    "",
  );
  if (strippedNumeric !== s) {
    if (
      SPECIAL_RE.test(strippedNumeric.toLowerCase()) ||
      SECTION_RE.test(strippedNumeric.toLowerCase())
    ) {
      return strippedNumeric.trim();
    }
  }

  return s.trim();
}

export function getSidebarEntryKind(title: string): "section" | "special" | "chapter" {
  const cleaned = cleanTitle(title);
  const normalized = cleaned.toLowerCase();

  // If it's an explicit numbered chapter (e.g. "Chương 1: Mở đầu", "Chapter 5: Final Battle"), it's a chapter!
  if (REGULAR_CHAPTER_PREFIX_RE.test(normalized)) {
    return "chapter";
  }

  // Check special entries first (e.g. "Chương cuối", "Chương kết", "Lời mở đầu", "Epilogue", "番外", "0. Prologue")
  if (SPECIAL_RE.test(normalized)) {
    return "special";
  }

  // Check section entries (e.g. "Volume 1", "Quyển 1", "Tập 2", "Phần 1", "第1卷")
  if (SECTION_RE.test(normalized)) {
    return "section";
  }

  return "chapter";
}

function formatChapterTitle(chapter: Chapter, t: TFunction) {
  let displayTitle =
    chapter.title || `${t("reader.chapter", "Chapter")} ${chapter.chapter_index + 1}`;
  if (displayTitle.match(/\.(x)?html$/i)) {
    displayTitle = displayTitle
      .replace(/\.(x)?html$/i, "")
      .replace(/[-_]/g, " ");
    displayTitle = displayTitle.replace(/\b\w/g, (letter) =>
      letter.toUpperCase(),
    );
  }
  return displayTitle;
}

function formatAudioTime(seconds: number): string {
  const mins = Math.floor(seconds / 60);
  const secs = Math.floor(seconds % 60);
  return `${mins}:${secs.toString().padStart(2, "0")}`;
}

export const ReaderSidebar: React.FC<ReaderSidebarProps> = ({
  t,
  book,
  chapters,
  currentChapter,
  sidebarBg,
  sidebarRef,
  onClose,
  onBack,
  onSelectChapter,
  highlights,
  onUpdateHighlight,
  onDeleteHighlight,
  onSelectHighlight,
  onOpenQuoteCard,
  nextInSeries,
  nextInReadList,
  onGoToNextInSeries,
  onGoToNextInReadList,
  isAudio = false,
  audioBookmarks = [],
  onSelectAudioBookmark,
  onDeleteAudioBookmark,
  isVisualContent = false,
  imageBookmarks = [],
  onSelectImageBookmark,
  onDeleteImageBookmark,
}) => {
  const [activeTab, setActiveTab] = useState<"toc" | "highlights">("toc");
  const [searchQuery, setSearchQuery] = useState("");
  const activeChapterRef = useRef<HTMLLIElement | null>(null);
  const singleChapter = chapters.length === 1;
  const totalBookmarksCount =
    (highlights?.length || 0) + (imageBookmarks?.length || 0) + (audioBookmarks?.length || 0);

  const filteredChapters = useMemo(() => {
    if (!searchQuery.trim()) return chapters;
    const q = searchQuery.toLowerCase().trim();
    return chapters.filter((chapter) => {
      const displayTitle = formatChapterTitle(chapter, t).toLowerCase();
      const idxStr = String(chapter.chapter_index + 1);
      return displayTitle.includes(q) || idxStr.includes(q);
    });
  }, [chapters, searchQuery, t]);

  useEffect(() => {
    if (activeTab === "toc" && activeChapterRef.current) {
      activeChapterRef.current.scrollIntoView({ block: "nearest", behavior: "smooth" });
    }
  }, [activeTab, currentChapter?.id]);

  const renderNextVolumeCard = () => {
    if (nextInReadList) {
      const coverUrl = nextInReadList.cover_url || (nextInReadList as any).cover_path;
      return (
        <div className="mt-4 p-3.5 rounded-2xl border border-(--reader-ui-border) bg-(--reader-ui-soft) shadow-xs">
          <div className="flex items-center gap-1.5 mb-2.5">
            <span className="badge badge-xs font-bold gap-1 py-1.5 px-2 bg-(--reader-ui-accent-soft) text-(--reader-ui-accent) border border-(--reader-ui-border)">
              <BookOpen className="w-2.5 h-2.5" />
              {t("reader.next_in_read_list", "Next in read list")}
            </span>
          </div>
          <div className="flex items-center gap-3">
            {coverUrl ? (
              <img
                src={getMediaUrl(coverUrl, nextInReadList.id, nextInReadList.updated_at)}
                alt={nextInReadList.title}
                className="h-14 aspect-[3/4.2] rounded-lg object-cover bg-(--reader-ui-surface-strong) shrink-0 shadow-sm border border-(--reader-ui-border)"
              />
            ) : (
              <div className="h-14 aspect-[3/4.2] rounded-lg bg-(--reader-ui-surface-strong) flex items-center justify-center shrink-0 shadow-sm border border-(--reader-ui-border)">
                <BookOpen className="w-4 h-4 opacity-40 text-(--reader-ui-accent)" />
              </div>
            )}
            <div className="min-w-0 flex-1">
              <p className="text-xs font-bold truncate text-(--reader-ui-text)">
                {nextInReadList.title}
              </p>
              {nextInReadList.description && (
                <p className="text-[10px] opacity-60 truncate mt-0.5 font-medium text-(--reader-ui-muted)">
                  {nextInReadList.description}
                </p>
              )}
            </div>
          </div>
          <button
            type="button"
            onClick={onGoToNextInReadList}
            className="btn btn-xs w-full gap-1.5 rounded-xl mt-3 text-[11px] font-bold shadow-xs cursor-pointer bg-(--reader-ui-accent) text-(--reader-ui-accent-text) border-0 hover:brightness-105"
          >
            <BookOpen className="w-3 h-3" />
            {t("reader.read_next_in_list", "Read next volume")}
            <ChevronRight className="w-3 h-3" />
          </button>
        </div>
      );
    }

    if (nextInSeries) {
      const coverUrl = nextInSeries.cover_url || (nextInSeries as any).cover_path;
      const seriesTitle = nextInSeries.series_name || (nextInSeries as any).series;
      return (
        <div className="mt-4 p-3.5 rounded-2xl border border-(--reader-ui-border) bg-(--reader-ui-soft) shadow-xs">
          <div className="flex items-center gap-1.5 mb-2.5">
            <span className="badge badge-xs font-bold gap-1 py-1.5 px-2 bg-(--reader-ui-accent-soft) text-(--reader-ui-accent) border border-(--reader-ui-border)">
              <BookOpen className="w-2.5 h-2.5" />
              {t("reader.next_in_series", "Next in series")}
            </span>
          </div>
          <div className="flex items-center gap-3">
            {coverUrl ? (
              <img
                src={getMediaUrl(coverUrl, nextInSeries.book_id)}
                alt={nextInSeries.title}
                className="h-14 aspect-[3/4.2] rounded-lg object-cover bg-(--reader-ui-surface-strong) shrink-0 shadow-sm border border-(--reader-ui-border)"
              />
            ) : (
              <div className="h-14 aspect-[3/4.2] rounded-lg bg-(--reader-ui-surface-strong) flex items-center justify-center shrink-0 shadow-sm border border-(--reader-ui-border)">
                <BookOpen className="w-4 h-4 opacity-40 text-(--reader-ui-accent)" />
              </div>
            )}
            <div className="min-w-0 flex-1">
              <p className="text-xs font-bold truncate text-(--reader-ui-text)">
                {nextInSeries.title}
              </p>
              {seriesTitle && (
                <p className="text-[10px] opacity-60 truncate mt-0.5 font-medium text-(--reader-ui-muted)">
                  {seriesTitle}
                  {nextInSeries.series_index ? ` #${nextInSeries.series_index}` : ""}
                </p>
              )}
            </div>
          </div>
          <button
            type="button"
            onClick={onGoToNextInSeries}
            className="btn btn-xs w-full gap-1.5 rounded-xl mt-3 text-[11px] font-bold shadow-xs cursor-pointer bg-(--reader-ui-accent) text-(--reader-ui-accent-text) border-0 hover:brightness-105"
          >
            <BookOpen className="w-3 h-3" />
            {t("reader.read_next_volume", "Read Next Volume")}
            <ChevronRight className="w-3 h-3" />
          </button>
        </div>
      );
    }

    return null;
  };

  return (
    <div className="drawer-side z-50">
      <label
        htmlFor="reader-drawer"
        aria-label={t("reader.close_toc", "Close table of contents")}
        className="drawer-overlay"
      />
      <aside
        ref={sidebarRef}
        className={`reader-sidebar ${singleChapter ? "reader-sidebar-single" : ""} flex flex-col h-full max-h-screen min-h-0 w-80 sm:w-88 overflow-hidden border-r shadow-2xl transition-colors duration-300 ${sidebarBg}`}
      >
        {/* Modern Sidebar Header with Book Cover Thumbnail */}
        <div className="flex items-start gap-3 p-4 border-b border-(--reader-ui-border) shrink-0 bg-(--reader-ui-soft)">
          <div className="relative shrink-0 overflow-hidden rounded-lg shadow-sm border border-(--reader-ui-border) bg-(--reader-ui-surface-strong) w-11 h-15">
            {book.cover_url ? (
              <img
                src={getMediaUrl(book.cover_url, book.id, book.updated_at)}
                alt={book.title}
                className="w-full h-full object-cover"
              />
            ) : (
              <div className="w-full h-full flex items-center justify-center text-(--reader-ui-muted)">
                <BookOpen className="w-5 h-5" />
              </div>
            )}
          </div>

          <div className="flex-1 min-w-0 pr-1">
            <h2 className="text-sm font-bold leading-snug line-clamp-2 text-(--reader-ui-text)">
              {book.title}
            </h2>
            {book.author_name && (
              <p className="text-xs text-(--reader-ui-muted) truncate mt-0.5 font-medium">
                {book.author_name}
              </p>
            )}
            <div className="flex items-center gap-2 mt-1">
              <span className="inline-flex items-center text-[10px] font-semibold text-(--reader-ui-muted) uppercase tracking-wider">
                {!singleChapter
                  ? `${chapters.length} ${t("reader.chapters_count", "chapters")}`
                  : (book.status ? t(`book.status_${book.status.toLowerCase()}`, book.status) : t("reader.current_reading", "Reading now"))}
              </span>
            </div>
          </div>

          <button
            type="button"
            onClick={onClose}
            className="btn btn-ghost btn-xs btn-circle shrink-0 text-(--reader-ui-text) opacity-70 hover:opacity-100 hover:bg-(--reader-ui-hover) transition-all cursor-pointer"
            title={t("reader.close_toc", "Close table of contents")}
            aria-label={t("reader.close_toc", "Close table of contents")}
          >
            <PanelLeftClose className="h-4 w-4" />
          </button>
        </div>

        {/* Segmented Tab Bar (iOS / Modern eReader pill control) */}
        <div className="px-3 pt-3 pb-2 shrink-0">
          <div className="grid grid-cols-2 gap-1 p-1 bg-(--reader-ui-soft) rounded-xl border border-(--reader-ui-border)">
            <button
              type="button"
              onClick={() => setActiveTab("toc")}
              className={`flex items-center justify-center gap-1.5 py-1.5 px-3 rounded-lg text-xs font-semibold transition-all cursor-pointer ${
                activeTab === "toc"
                  ? "bg-(--reader-ui-accent) text-(--reader-ui-accent-text) shadow-xs font-bold"
                  : "text-(--reader-ui-text) opacity-75 hover:opacity-100 hover:bg-(--reader-ui-hover)"
              }`}
            >
              <ListTree className="w-3.5 h-3.5" />
              <span>{t("reader.toc_tab", "Contents")}</span>
              {!singleChapter && chapters.length > 0 && (
                <span
                  className={`text-[10px] px-1.5 py-0.5 rounded-full font-mono font-bold ${
                    activeTab === "toc"
                      ? "bg-(--reader-ui-accent-text)/20 text-(--reader-ui-accent-text)"
                      : "bg-(--reader-ui-surface-strong) text-(--reader-ui-text)"
                  }`}
                >
                  {chapters.length}
                </span>
              )}
            </button>

            <button
              type="button"
              onClick={() => setActiveTab("highlights")}
              className={`flex items-center justify-center gap-1.5 py-1.5 px-3 rounded-lg text-xs font-semibold transition-all cursor-pointer ${
                activeTab === "highlights"
                  ? "bg-(--reader-ui-accent) text-(--reader-ui-accent-text) shadow-xs font-bold"
                  : "text-(--reader-ui-text) opacity-75 hover:opacity-100 hover:bg-(--reader-ui-hover)"
              }`}
            >
              <Bookmark className="w-3.5 h-3.5" />
              <span>
                {isAudio
                  ? t("reader.audio_bookmarks", "Audio Bookmarks")
                  : t("reader.highlights_tab", "Bookmarks")}
              </span>
              {totalBookmarksCount > 0 && (
                <span
                  className={`text-[10px] px-1.5 py-0.5 rounded-full font-mono font-bold ${
                    activeTab === "highlights"
                      ? "bg-(--reader-ui-accent-text)/20 text-(--reader-ui-accent-text)"
                      : "bg-(--reader-ui-surface-strong) text-(--reader-ui-text)"
                  }`}
                >
                  {totalBookmarksCount}
                </span>
              )}
            </button>
          </div>
        </div>

        {/* Chapter Search Bar (when more than 5 chapters exist) */}
        {activeTab === "toc" && !singleChapter && chapters.length > 5 && (
          <div className="px-3 pb-2 shrink-0">
            <div className="relative flex items-center">
              <Search className="w-3.5 h-3.5 absolute left-3 text-(--reader-ui-muted) pointer-events-none" />
              <input
                type="text"
                value={searchQuery}
                onChange={(e) => setSearchQuery(e.target.value)}
                placeholder={t("reader.search_chapters", "Search chapters...")}
                className="input input-xs w-full pl-8 pr-7 py-3 bg-(--reader-ui-soft) border-(--reader-ui-border) rounded-lg text-xs text-(--reader-ui-text) focus:bg-(--reader-ui-surface-strong) focus:border-(--reader-ui-accent) transition-all placeholder:text-(--reader-ui-muted)"
              />
              {searchQuery && (
                <button
                  type="button"
                  onClick={() => setSearchQuery("")}
                  className="absolute right-2 p-0.5 rounded-full hover:bg-(--reader-ui-hover) text-(--reader-ui-muted) hover:text-(--reader-ui-text) cursor-pointer"
                >
                  <X className="w-3 h-3" />
                </button>
              )}
            </div>
          </div>
        )}

        {/* Tab Content Body */}
        <div className="flex-1 min-h-0 overflow-hidden flex flex-col">
          {activeTab === "toc" ? (
            <div className="flex-1 min-h-0 overflow-y-auto reader-sidebar-body">
              {singleChapter ? (
                <div className="p-4 space-y-4">
                  {/* Modern Single Volume / Current File Card */}
                  <div className="relative overflow-hidden rounded-2xl border border-(--reader-ui-border) bg-(--reader-ui-soft) p-4 shadow-sm transition-all hover:border-(--reader-ui-accent)/40">
                    <div className="flex items-start gap-3.5">
                      <div className="relative h-16 aspect-[3/4.2] shrink-0 overflow-hidden rounded-xl bg-(--reader-ui-surface-strong) shadow-md border border-(--reader-ui-border)">
                        {book.cover_url ? (
                          <img
                            src={getMediaUrl(book.cover_url, book.id, book.updated_at)}
                            alt={book.title}
                            className="h-full w-full object-cover"
                          />
                        ) : (
                          <div className="flex h-full w-full items-center justify-center bg-(--reader-ui-surface-strong)">
                            <BookOpen className="h-6 w-6 opacity-40 text-(--reader-ui-accent)" />
                          </div>
                        )}
                      </div>

                      <div className="min-w-0 flex-1">
                        <div className="flex items-center gap-1.5 mb-1.5">
                          <span className="badge badge-xs font-bold gap-1 py-1.5 px-2 bg-(--reader-ui-accent-soft) text-(--reader-ui-accent) border border-(--reader-ui-border)">
                            <span className="w-1.5 h-1.5 rounded-full bg-current animate-pulse" />
                            {t("reader.current_reading", "Reading now")}
                          </span>
                        </div>

                        <h3 className="text-xs font-bold leading-snug line-clamp-2 text-(--reader-ui-text)">
                          {chapters[0] ? formatChapterTitle(chapters[0], t) : book.title}
                        </h3>
                        {book.author_name && (
                          <p className="text-[11px] text-(--reader-ui-muted) truncate mt-0.5 font-medium">
                            {book.author_name}
                          </p>
                        )}
                      </div>
                    </div>

                    <button
                      type="button"
                      onClick={() => chapters[0] && onSelectChapter(chapters[0])}
                      className="btn btn-xs w-full gap-2 rounded-xl mt-3.5 font-bold shadow-xs text-xs cursor-pointer bg-(--reader-ui-accent) text-(--reader-ui-accent-text) border-0 hover:brightness-105"
                    >
                      <BookOpen className="w-3.5 h-3.5" />
                      <span>{t("reader.read", "Read")}</span>
                    </button>
                  </div>

                  {renderNextVolumeCard()}
                </div>
              ) : (
                <div className="flex flex-col min-h-0 py-1">
                  {filteredChapters.length === 0 ? (
                    <div className="py-12 px-4 text-center space-y-2 opacity-60">
                      <Search className="w-6 h-6 mx-auto opacity-40 text-(--reader-ui-muted)" />
                      <p className="text-xs font-medium text-(--reader-ui-text)">
                        {t("reader.no_chapters_found", "No chapters found")}
                      </p>
                    </div>
                  ) : (
                    <ul className="reader-sidebar-list space-y-0.5 px-2">
                      {filteredChapters.map((chapter) => {
                        const displayTitle = formatChapterTitle(chapter, t);
                        const entryKind = getSidebarEntryKind(displayTitle);
                        const isActive = currentChapter?.id === chapter.id;

                        return (
                          <li
                            key={chapter.id}
                            ref={isActive ? activeChapterRef : undefined}
                          >
                            <button
                              type="button"
                              onClick={() => {
                                onSelectChapter(chapter);
                              }}
                              className={`group relative flex w-full items-center gap-2.5 rounded-xl px-3 py-2.5 text-left text-xs transition-all cursor-pointer ${
                                isActive
                                  ? "bg-(--reader-ui-accent-soft) text-(--reader-ui-accent) font-bold shadow-2xs border-l-3 border-(--reader-ui-accent) pl-2.5"
                                  : "text-(--reader-ui-text) opacity-80 hover:opacity-100 hover:bg-(--reader-ui-hover)"
                              }`}
                            >
                              {entryKind === "chapter" && (
                                <span
                                  className={`font-mono text-[11px] shrink-0 font-semibold px-1.5 py-0.5 rounded-md transition-colors ${
                                    isActive
                                      ? "bg-(--reader-ui-accent) text-(--reader-ui-accent-text) font-bold"
                                      : "bg-(--reader-ui-soft) text-(--reader-ui-muted) group-hover:bg-(--reader-ui-hover)"
                                  }`}
                                >
                                  {String(chapter.chapter_index + 1).padStart(2, "0")}
                                </span>
                              )}

                              {entryKind !== "chapter" && (
                                <span className="badge badge-xs bg-(--reader-ui-soft) text-(--reader-ui-muted) border-(--reader-ui-border) text-[10px] font-semibold shrink-0 py-1 opacity-70">
                                  ★
                                </span>
                              )}

                              <span className="flex-1 line-clamp-2 leading-relaxed min-w-0">
                                {displayTitle}
                              </span>

                              {isActive && (
                                <span className="shrink-0 flex items-center justify-center text-(--reader-ui-accent)">
                                  <BookOpen className="w-3.5 h-3.5 animate-pulse" />
                                </span>
                              )}
                            </button>
                          </li>
                        );
                      })}
                      {(nextInReadList || nextInSeries) && (
                        <li className="px-1 pb-4">
                          {renderNextVolumeCard()}
                        </li>
                      )}
                    </ul>
                  )}
                </div>
              )}
            </div>
          ) : (
            <div className="flex-1 min-h-0 flex flex-col">
              {isAudio ? (
                <div className="flex-1 min-h-0 overflow-y-auto p-4">
                  {!audioBookmarks || audioBookmarks.length === 0 ? (
                    <div className="py-12 text-center space-y-1.5 opacity-60">
                      <Bookmark className="w-6 h-6 mx-auto opacity-40 text-(--reader-ui-accent)" />
                      <p className="text-xs text-(--reader-ui-text)">
                        {t("reader.no_bookmarks", "No bookmarks saved yet.")}
                      </p>
                    </div>
                  ) : (
                    <ul className="flex flex-col gap-2">
                      {audioBookmarks.map((bm) => (
                        <li
                          key={bm.id}
                          className="flex items-center justify-between gap-2 rounded-xl border border-(--reader-ui-border) bg-(--reader-ui-soft) p-2.5 hover:border-(--reader-ui-accent)/40 transition-colors"
                        >
                          <button
                            type="button"
                            onClick={() => {
                              onSelectAudioBookmark?.(bm.time_sec);
                              onClose();
                            }}
                            className="flex flex-1 items-start gap-2 text-left min-w-0 group cursor-pointer"
                          >
                            <span className="px-2 py-0.5 rounded-md bg-(--reader-ui-accent-soft) text-(--reader-ui-accent) font-mono text-[11px] font-bold shrink-0">
                              {formatAudioTime(bm.time_sec)}
                            </span>
                            <div className="min-w-0 flex-1">
                              <p className="text-xs font-semibold text-(--reader-ui-text) line-clamp-2 group-hover:text-(--reader-ui-accent) transition-colors">
                                {bm.note || bm.chapter_title || t("reader.bookmark", "Bookmark")}
                              </p>
                              {bm.note && bm.chapter_title && (
                                <p className="text-[10px] text-(--reader-ui-muted) truncate mt-0.5">
                                  {bm.chapter_title}
                                </p>
                              )}
                            </div>
                          </button>
                          {onDeleteAudioBookmark && (
                            <button
                              type="button"
                              onClick={() => onDeleteAudioBookmark(bm.id)}
                              className="btn btn-ghost btn-xs text-error hover:bg-error/20 cursor-pointer"
                              title={t("common.delete", "Delete")}
                              aria-label={t("common.delete", "Delete")}
                            >
                              <Trash2 className="h-3.5 w-3.5" />
                            </button>
                          )}
                        </li>
                      ))}
                    </ul>
                  )}
                </div>
              ) : (
                <ReaderHighlightsPanel
                  t={t}
                  highlights={highlights || []}
                  imageBookmarks={imageBookmarks || []}
                  chapters={chapters}
                  onUpdate={onUpdateHighlight!}
                  onDelete={onDeleteHighlight!}
                  onSelect={(hl) => {
                    onSelectHighlight?.(hl);
                    onClose();
                  }}
                  onSelectImageBookmark={(bm) => {
                    onSelectImageBookmark?.(bm);
                    onClose();
                  }}
                  onDeleteImageBookmark={onDeleteImageBookmark}
                  onOpenQuoteCard={onOpenQuoteCard}
                />
              )}
            </div>
          )}
        </div>

        {/* Sidebar Footer */}
        <div className="reader-sidebar-footer shrink-0 border-t border-(--reader-ui-border) p-3">
          <button
            type="button"
            onClick={onBack}
            className="btn btn-ghost btn-sm w-full gap-2 rounded-xl text-xs font-semibold text-(--reader-ui-text) hover:bg-(--reader-ui-hover) transition-all cursor-pointer"
          >
            <ArrowLeft className="h-4 w-4" />
            <span>{t("reader.back_to_previous", "Back")}</span>
          </button>
        </div>
      </aside>
    </div>
  );
};

