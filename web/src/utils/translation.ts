export interface TranslationLanguage {
  code: string;
  name: string;
  nativeName: string;
  flag: string;
}

export const SUPPORTED_LANGUAGES: TranslationLanguage[] = [
  { code: "vi", name: "Vietnamese", nativeName: "Tiếng Việt", flag: "🇻🇳" },
  { code: "en", name: "English", nativeName: "English", flag: "🇬🇧" },
  { code: "zh-CN", name: "Chinese (Simplified)", nativeName: "简体中文", flag: "🇨🇳" },
  { code: "zh-TW", name: "Chinese (Traditional)", nativeName: "繁體中文", flag: "🇹🇼" },
  { code: "ja", name: "Japanese", nativeName: "日本語", flag: "🇯🇵" },
  { code: "ko", name: "Korean", nativeName: "한국어", flag: "🇰🇷" },
  { code: "fr", name: "French", nativeName: "Français", flag: "🇫🇷" },
  { code: "es", name: "Spanish", nativeName: "Español", flag: "🇪🇸" },
  { code: "de", name: "German", nativeName: "Deutsch", flag: "🇩🇪" },
  { code: "ru", name: "Russian", nativeName: "Русский", flag: "🇷🇺" },
  { code: "pt", name: "Portuguese", nativeName: "Português", flag: "🇵🇹" },
  { code: "it", name: "Italian", nativeName: "Italiano", flag: "🇮🇹" },
  { code: "id", name: "Indonesian", nativeName: "Bahasa Indonesia", flag: "🇮🇩" },
  { code: "th", name: "Thai", nativeName: "ภาษาไทย", flag: "🇹🇭" },
  { code: "hi", name: "Hindi", nativeName: "हिन्दी", flag: "🇮🇳" },
  { code: "ar", name: "Arabic", nativeName: "العربية", flag: "🇸🇦" },
  { code: "tr", name: "Turkish", nativeName: "Türkçe", flag: "🇹🇷" },
  { code: "nl", name: "Dutch", nativeName: "Nederlands", flag: "🇳🇱" },
  { code: "pl", name: "Polish", nativeName: "Polski", flag: "🇵🇱" },
  { code: "uk", name: "Ukrainian", nativeName: "Українська", flag: "🇺🇦" },
  { code: "sv", name: "Swedish", nativeName: "Svenska", flag: "🇸🇪" },
  { code: "cs", name: "Czech", nativeName: "Čeština", flag: "🇨🇿" },
  { code: "el", name: "Greek", nativeName: "Ελληνικά", flag: "🇬🇷" },
  { code: "he", name: "Hebrew", nativeName: "עברית", flag: "🇮🇱" },
  { code: "ro", name: "Romanian", nativeName: "Română", flag: "🇷🇴" },
  { code: "hu", name: "Hungarian", nativeName: "Magyar", flag: "🇭🇺" },
  { code: "da", name: "Danish", nativeName: "Dansk", flag: "🇩🇰" },
  { code: "fi", name: "Finnish", nativeName: "Suomi", flag: "🇫🇮" },
  { code: "no", name: "Norwegian", nativeName: "Norsk", flag: "🇳🇴" },
  { code: "ms", name: "Malay", nativeName: "Bahasa Melayu", flag: "🇲🇾" },
  { code: "tl", name: "Filipino", nativeName: "Tagalog", flag: "🇵🇭" },
];

const PREF_KEY = "novelhub_reader_target_lang";

export function getDefaultTargetLanguage(currentLocale?: string): string {
  try {
    const saved = localStorage.getItem(PREF_KEY);
    if (saved && SUPPORTED_LANGUAGES.some((l) => l.code === saved)) {
      return saved;
    }
  } catch {
    // Ignore localStorage errors
  }

  if (currentLocale) {
    const matched = SUPPORTED_LANGUAGES.find(
      (l) => l.code === currentLocale || l.code === currentLocale.split("-")[0]
    );
    if (matched) return matched.code;
  }

  return "vi";
}

export function saveTargetLanguagePreference(langCode: string): void {
  try {
    localStorage.setItem(PREF_KEY, langCode);
  } catch {
    // Ignore localStorage errors
  }
}

export interface TranslationResult {
  text: string;
  detectedSourceLang?: string;
  engine: "google" | "mymemory" | "lingva";
}

/**
 * Multi-engine translator with automatic fallback:
 * 1. Google Translate GTX
 * 2. MyMemory Translated API
 * 3. Lingva Translate Mirror
 */
export async function translateText(
  rawText: string,
  targetLang: string,
  sourceLang = "auto"
): Promise<TranslationResult> {
  const text = rawText.trim();
  if (!text) {
    return { text: "", engine: "google" };
  }

  // 1. Try Google Translate Engine (GTX)
  try {
    const googleLang = targetLang === "zh-CN" ? "zh-CN" : targetLang === "zh-TW" ? "zh-TW" : targetLang.split("-")[0];
    const url = `https://translate.googleapis.com/translate_a/single?client=gtx&sl=${encodeURIComponent(
      sourceLang
    )}&tl=${encodeURIComponent(googleLang)}&dt=t&q=${encodeURIComponent(text)}`;

    const res = await fetch(url);
    if (res.ok) {
      const data = await res.json();
      if (Array.isArray(data) && Array.isArray(data[0])) {
        const fullTranslation = data[0]
          .map((item: any) => (item && item[0] ? item[0] : ""))
          .join("");
        if (fullTranslation.trim()) {
          const detected = data[2] ? String(data[2]) : undefined;
          return { text: fullTranslation, detectedSourceLang: detected, engine: "google" };
        }
      }
    }
  } catch (err) {
    console.warn("[Translation] Google engine failed, falling back to MyMemory", err);
  }

  // 2. Fallback to MyMemory Translated API
  try {
    const myMemoryLang = targetLang === "zh-CN" || targetLang === "zh-TW" ? "zh" : targetLang.split("-")[0];
    const langPair = sourceLang === "auto" ? `autodetect|${myMemoryLang}` : `${sourceLang}|${myMemoryLang}`;
    const url = `https://api.mymemory.translated.net/get?q=${encodeURIComponent(text)}&langpair=${encodeURIComponent(
      langPair
    )}`;

    const res = await fetch(url);
    if (res.ok) {
      const data = await res.json();
      if (data?.responseData?.translatedText) {
        return { text: data.responseData.translatedText, engine: "mymemory" };
      }
    }
  } catch (err) {
    console.warn("[Translation] MyMemory engine failed, falling back to Lingva", err);
  }

  // 3. Fallback to Lingva Translate API
  try {
    const lingvaLang = targetLang === "zh-CN" ? "zh" : targetLang.split("-")[0];
    const url = `https://lingva.ml/api/v1/${encodeURIComponent(sourceLang)}/${encodeURIComponent(
      lingvaLang
    )}/${encodeURIComponent(text)}`;

    const res = await fetch(url);
    if (res.ok) {
      const data = await res.json();
      if (data?.translation) {
        return { text: data.translation, engine: "lingva" };
      }
    }
  } catch (err) {
    console.error("[Translation] All translation engines failed", err);
  }

  throw new Error("Unable to translate with available services");
}
