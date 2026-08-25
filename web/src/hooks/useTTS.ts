import { useState, useEffect, useCallback, useRef } from 'react';

export interface UseTTSOptions {
  onEnd?: () => void;
  onError?: (event: any) => void;
  onBoundary?: (event: any) => void;
}



export interface TextChunk {
  text: string;
  charOffset: number;
  cleanToRawMap: number[];
}

export function getNextWordOffset(text: string, currentWordStart: number): number {
  if (currentWordStart < 0 || currentWordStart >= text.length) return currentWordStart;
  const rest = text.slice(currentWordStart);
  // Match word characters (letters, digits, marks) plus trailing spaces and punctuation
  const match = rest.match(/^[\p{L}\p{N}\u0300-\u036f_—–-]+[\s,.;:!?"“”'»«)\]]*/u);
  if (match && match[0].length > 0) {
    return currentWordStart + match[0].length;
  }
  return currentWordStart + 1;
}

export function cleanTextForTTS(text: string): string {
  if (!text) return "";
  return text
    .normalize("NFC")
    .replace(/——/g, ", ")
    .replace(/[—–―]/g, "-")
    .replace(/[“”«»]/g, '"')
    .replace(/[‘’]/g, "'")
    .replace(/[\u00A0\u1680\u180E\u2000-\u200B\u202F\u205F\u3000\uFEFF]/g, " ")
    .replace(/[\x00-\x08\x0B\x0C\x0E-\x1F\x7F]/g, " ");
}

export function cleanTextForTTSWithMap(text: string): TextChunk {
  let cleaned = "";
  const cleanToRawMap: number[] = [];

  for (let i = 0; i < text.length; i++) {
    const ch = text[i];
    let replacement = ch;

    if (ch === "—" || ch === "–" || ch === "―") {
      replacement = "-";
    } else if (ch === "“" || ch === "”" || ch === "«" || ch === "»") {
      replacement = '"';
    } else if (ch === "‘" || ch === "’") {
      replacement = "'";
    } else if (/[\u00A0\u1680\u180E\u2000-\u200B\u202F\u205F\u3000\uFEFF]/u.test(ch)) {
      replacement = " ";
    } else if (/[\x00-\x08\x0B\x0C\x0E-\x1F\x7F]/u.test(ch)) {
      replacement = " ";
    }

    cleaned += replacement;
    for (let j = 0; j < replacement.length; j++) {
      cleanToRawMap.push(i);
    }
  }

  const normalized = cleaned.normalize("NFC");
  if (normalized !== cleaned) {
    return { text: cleanTextForTTS(text), charOffset: 0, cleanToRawMap };
  }

  return { text: cleaned, charOffset: 0, cleanToRawMap };
}

function subSplitLongChunk(fullText: string, start: number, end: number, maxChunkLen = 380): TextChunk[] {
  const result: TextChunk[] = [];
  let curStart = start;

  while (curStart < end) {
    if (end - curStart <= maxChunkLen) {
      const raw = fullText.slice(curStart, end);
      const cleaned = cleanTextForTTSWithMap(raw);
      if (cleaned.text.trim()) {
        result.push({ ...cleaned, charOffset: curStart });
      }
      break;
    }

    const windowText = fullText.slice(curStart, Math.min(end, curStart + maxChunkLen));
    const match = windowText.slice(150).search(/[,，、؛\u060C;:\-]\s*|\s+/u);
    let splitPos = end;
    if (match !== -1) {
      splitPos = curStart + 150 + match + 1;
    } else {
      splitPos = curStart + maxChunkLen;
    }

    const raw = fullText.slice(curStart, splitPos);
    const cleaned = cleanTextForTTSWithMap(raw);
    if (cleaned.text.trim()) {
      result.push({ ...cleaned, charOffset: curStart });
    }
    curStart = splitPos;
  }

  return result;
}

export function splitTextIntoChunks(fullText: string): TextChunk[] {
  const chunks: TextChunk[] = [];
  if (!fullText) return chunks;

  const rawSegments: { text: string; index: number }[] = [];

  // Universal text segmentation using Unicode Standard Annex #29
  if (typeof Intl !== "undefined" && Intl.Segmenter) {
    const segmenter = new Intl.Segmenter(undefined, { granularity: "sentence" });
    for (const seg of segmenter.segment(fullText)) {
      rawSegments.push({
        text: seg.segment,
        index: seg.index,
      });
    }
  } else {
    // Universal multilingual regex fallback (Latin, CJK, Arabic, Cyrillic, Indic, etc.)
    const regex = /([^.!?…;:\u061F\u061B\u0964\u0965\n]+[.!?…;:\u061F\u061B\u0964\u0965]*|\s*\n+\s*|[^.!?…;:\u061F\u061B\u0964\u0965\n]+$)/gu;
    let match: RegExpExecArray | null;
    while ((match = regex.exec(fullText)) !== null) {
      if (match[0].trim()) {
        rawSegments.push({ text: match[0], index: match.index });
      }
    }
  }

  // Merge sentences into optimal natural chunks (target: 250 - 400 characters)
  let currentStart = -1;
  let currentEnd = -1;

  for (let i = 0; i < rawSegments.length; i++) {
    const seg = rawSegments[i];
    const segText = seg.text;
    const segStart = seg.index;
    const segEnd = segStart + segText.length;

    // Skip empty whitespace-only segments before a chunk starts
    if (!segText.trim() && currentStart === -1) {
      continue;
    }

    if (currentStart === -1) {
      currentStart = segStart;
      currentEnd = segEnd;
    } else {
      const prospectiveLength = segEnd - currentStart;
      const isParagraphBreak = segText.includes("\n\n") || (fullText.slice(currentEnd, segStart).includes("\n\n"));

      if (prospectiveLength <= 350 && !isParagraphBreak) {
        currentEnd = segEnd;
      } else {
        const chunkLen = currentEnd - currentStart;
        if (chunkLen > 420) {
          chunks.push(...subSplitLongChunk(fullText, currentStart, currentEnd));
        } else {
          const rawChunk = fullText.slice(currentStart, currentEnd);
          const cleaned = cleanTextForTTSWithMap(rawChunk);
          if (cleaned.text.trim()) {
            chunks.push({ ...cleaned, charOffset: currentStart });
          }
        }
        currentStart = segStart;
        currentEnd = segEnd;
      }
    }
  }

  if (currentStart !== -1 && currentEnd > currentStart) {
    const chunkLen = currentEnd - currentStart;
    if (chunkLen > 420) {
      chunks.push(...subSplitLongChunk(fullText, currentStart, currentEnd));
    } else {
      const rawChunk = fullText.slice(currentStart, currentEnd);
      const cleaned = cleanTextForTTSWithMap(rawChunk);
      if (cleaned.text.trim()) {
        chunks.push({ ...cleaned, charOffset: currentStart });
      }
    }
  }

  if (chunks.length === 0 && fullText.trim()) {
    chunks.push(cleanTextForTTSWithMap(fullText));
  }

  return chunks;
}

export function useTTS(options?: UseTTSOptions) {
  const [isSupported, setIsSupported] = useState(false);
  const [voices, setVoices] = useState<SpeechSynthesisVoice[]>([]);
  const [selectedVoice, setSelectedVoice] = useState<SpeechSynthesisVoice | null>(null);
  const [isPlaying, setIsPlaying] = useState(false);
  const [isPaused, setIsPaused] = useState(false);
  const [rate, setRate] = useState(1);
  const [pitch, setPitch] = useState(1);
  const [volume, setVolume] = useState(1);

  const activeUtteranceRef = useRef<SpeechSynthesisUtterance | null>(null);
  const optionsRef = useRef(options);
  optionsRef.current = options;

  const chunksRef = useRef<TextChunk[]>([]);
  const currentChunkIndexRef = useRef<number>(0);
  const savedChunkIndexRef = useRef<number>(0);
  const savedRemainingTextRef = useRef<string | null>(null);
  const savedRemainingOffsetRef = useRef<number>(0);
  const savedRemainingMapRef = useRef<number[]>([]);
  const currentChunkTextRef = useRef<string>("");
  const currentBaseOffsetRef = useRef<number>(0);
  const currentCleanToRawMapRef = useRef<number[]>([]);
  const lastBoundaryLocalIndexRef = useRef<number>(0);
  const lastBoundaryWordLenRef = useRef<number>(0);
  const hasBoundaryFiredRef = useRef<boolean>(false);
  const isCanceledRef = useRef<boolean>(false);
  const isPausedRef = useRef<boolean>(false);

  const rateRef = useRef(rate);
  rateRef.current = rate;
  const pitchRef = useRef(pitch);
  pitchRef.current = pitch;
  const volumeRef = useRef(volume);
  volumeRef.current = volume;
  const selectedVoiceRef = useRef(selectedVoice);
  selectedVoiceRef.current = selectedVoice;

  useEffect(() => {
    if (typeof window !== 'undefined') {
      setIsSupported(true);
      
      if (window.speechSynthesis) {
        const updateVoices = () => {
          const availableVoices = window.speechSynthesis.getVoices();
          setVoices(availableVoices);
          if (availableVoices.length > 0 && !selectedVoiceRef.current) {
            setSelectedVoice(availableVoices.find(v => v.default) || availableVoices[0]);
          }
        };

        updateVoices();
        window.speechSynthesis.onvoiceschanged = updateVoices;

        return () => {
          window.speechSynthesis.onvoiceschanged = null;
        };
      }
    }
  }, []);

  useEffect(() => {
    return () => {
      isCanceledRef.current = true;
      isPausedRef.current = false;
      if (typeof window !== 'undefined' && window.speechSynthesis) {
        try {
          (window as any).__novelhub_active_utterance = null;
          window.speechSynthesis.cancel();
        } catch {}
      }
    };
  }, []);

  const playChunkDirect = useCallback((textToSpeak: string, baseOffset: number, chunkIndex: number, cleanToRawMap?: number[]) => {
    if (isCanceledRef.current || isPausedRef.current || typeof window === 'undefined' || !window.speechSynthesis) {
      return;
    }

    currentChunkTextRef.current = textToSpeak;
    currentBaseOffsetRef.current = baseOffset;
    currentCleanToRawMapRef.current = cleanToRawMap || [];
    lastBoundaryLocalIndexRef.current = 0;
    lastBoundaryWordLenRef.current = 0;
    hasBoundaryFiredRef.current = false;

    if (activeUtteranceRef.current) {
      activeUtteranceRef.current.onstart = null;
      activeUtteranceRef.current.onend = null;
      activeUtteranceRef.current.onerror = null;
      activeUtteranceRef.current.onpause = null;
      activeUtteranceRef.current.onresume = null;
      activeUtteranceRef.current.onboundary = null;
      activeUtteranceRef.current = null;
    }

    const utterance = new SpeechSynthesisUtterance(textToSpeak);
    activeUtteranceRef.current = utterance;
    (window as any).__novelhub_active_utterance = utterance;

    if (selectedVoiceRef.current) {
      utterance.voice = selectedVoiceRef.current;
    }
    utterance.rate = rateRef.current;
    utterance.pitch = pitchRef.current;
    utterance.volume = volumeRef.current;

    utterance.onstart = () => {
      if (isCanceledRef.current || isPausedRef.current) return;
      setIsPlaying(true);
      setIsPaused(false);
    };

    utterance.onend = () => {
      if (isCanceledRef.current || isPausedRef.current) return;
      savedRemainingTextRef.current = null;
      playChunkAtIndex(chunkIndex + 1);
    };

    utterance.onerror = (e) => {
      if (isCanceledRef.current || isPausedRef.current) return;

      // If interrupted mid-sentence (e.g. user clicked on DOM), continue seamlessly from the next unread word
      if (e.error === "interrupted" || e.error === "canceled") {
        const lastIdx = lastBoundaryLocalIndexRef.current;
        // If no boundary has fired yet for this utterance, nothing was read yet:
        // resume the current utterance from its start instead of skipping its first word.
        const nextWordPos = hasBoundaryFiredRef.current ? getNextWordOffset(textToSpeak, lastIdx) : 0;

        const sliceFrom = textToSpeak.slice(nextWordPos);
        const remainingText = sliceFrom.trim();
        if (remainingText.length > 2) {
          // index of the first non-space char (leading trim only; trailing trim must not shift the map)
          const cleanStart = nextWordPos + (sliceFrom.length - sliceFrom.trimStart().length);
          const rawStart = currentCleanToRawMapRef.current[cleanStart] ?? cleanStart;
          const remainingMap = currentCleanToRawMapRef.current
            .slice(cleanStart)
            .map((rawIndex) => Math.max(0, rawIndex - rawStart));
          const newOffset = baseOffset + rawStart;
          setTimeout(() => {
            if (!isCanceledRef.current && !isPausedRef.current) {
              playChunkDirect(remainingText, newOffset, chunkIndex, remainingMap);
            }
          }, 40);
          return;
        }
      }

      savedRemainingTextRef.current = null;
      playChunkAtIndex(chunkIndex + 1);
    };

    utterance.onboundary = (e) => {
      if (isCanceledRef.current || isPausedRef.current) return;
      const rawLocalIdx = e.charIndex || 0;
      lastBoundaryLocalIndexRef.current = rawLocalIdx;
      hasBoundaryFiredRef.current = true;

      // Extract accurate word boundary even when browser does not provide charLength
      let wordStart = rawLocalIdx;
      while (wordStart < textToSpeak.length && /[\s"“«'‘(\[<{]/.test(textToSpeak[wordStart])) {
        wordStart++;
      }

      let wordLen = e.charLength || 0;
      if (wordLen <= 1 && wordStart < textToSpeak.length) {
        const slice = textToSpeak.slice(wordStart);
        const match = slice.match(/^[\p{L}\p{M}\p{N}_—–-]+/u);
        wordLen = match?.[0]?.length || 1;
      } else if (wordLen <= 1) {
        wordLen = 1;
      }
      lastBoundaryWordLenRef.current = wordLen;

      const map = currentCleanToRawMapRef.current;
      const rawWordStart = map[wordStart] ?? wordStart;
      const rawWordEnd = map[Math.min(textToSpeak.length - 1, wordStart + wordLen - 1)];
      const globalCharIndex = baseOffset + rawWordStart;
      optionsRef.current?.onBoundary?.({
        ...e,
        charIndex: globalCharIndex,
        charLength: rawWordEnd !== undefined ? Math.max(1, rawWordEnd - rawWordStart + 1) : wordLen,
        utterance,
      });
    };

    try {
      if (window.speechSynthesis.paused) {
        window.speechSynthesis.resume();
      }
      window.speechSynthesis.speak(utterance);
      setIsPlaying(true);
      setIsPaused(false);
    } catch (err) {
      console.error(`[TTS] speak(chunk ${chunkIndex + 1}) threw:`, err);
      playChunkAtIndex(chunkIndex + 1);
    }
  }, []);

  const playChunkAtIndex = useCallback((index: number) => {
    if (isCanceledRef.current || isPausedRef.current || typeof window === 'undefined' || !window.speechSynthesis) {
      return;
    }

    const chunks = chunksRef.current;
    if (index >= chunks.length) {
      activeUtteranceRef.current = null;
      (window as any).__novelhub_active_utterance = null;
      savedRemainingTextRef.current = null;
      setIsPlaying(false);
      setIsPaused(false);
      optionsRef.current?.onEnd?.();
      return;
    }

    currentChunkIndexRef.current = index;
    const chunk = chunks[index];

    playChunkDirect(chunk.text, chunk.charOffset, index, chunk.cleanToRawMap);
  }, [playChunkDirect]);

  const speak = useCallback((text: string) => {
    if (typeof window === 'undefined' || !text.trim() || !window.speechSynthesis) return;

    const chunks = splitTextIntoChunks(text);
    if (chunks.length === 0) return;

    isCanceledRef.current = false;
    isPausedRef.current = false;
    chunksRef.current = chunks;
    currentChunkIndexRef.current = 0;
    savedChunkIndexRef.current = 0;
    savedRemainingTextRef.current = null;
    savedRemainingMapRef.current = [];

    playChunkAtIndex(0);
  }, [playChunkAtIndex]);

  const pause = useCallback(() => {
    isPausedRef.current = true;
    savedChunkIndexRef.current = currentChunkIndexRef.current;

    // Save exact next word text of current chunk if speaking
    const text = currentChunkTextRef.current;
    const lastIdx = lastBoundaryLocalIndexRef.current;
    if (text) {
      if (!hasBoundaryFiredRef.current) {
        // No word started yet — resume from the start of this utterance unchanged.
        savedRemainingTextRef.current = text;
        savedRemainingOffsetRef.current = currentBaseOffsetRef.current;
        savedRemainingMapRef.current = currentCleanToRawMapRef.current;
      } else if (lastIdx >= 0) {
        const nextWordPos = getNextWordOffset(text, lastIdx);
        if (nextWordPos < text.length) {
          // leading trim count, not length arithmetic: trailing trim must not shift the map
          const sliceFrom = text.slice(nextWordPos);
          const remaining = sliceFrom.trim();
          if (remaining.length > 2) {
            savedRemainingTextRef.current = remaining;
            // index of the first non-space char (leading trim only; trailing trim must not shift the map)
            const cleanStart = nextWordPos + (sliceFrom.length - sliceFrom.trimStart().length);
            const rawStart = currentCleanToRawMapRef.current[cleanStart] ?? cleanStart;
            savedRemainingOffsetRef.current = currentBaseOffsetRef.current + rawStart;
            savedRemainingMapRef.current = currentCleanToRawMapRef.current
              .slice(cleanStart)
              .map((rawIndex) => Math.max(0, rawIndex - rawStart));
          }
        }
      }
    }

    if (typeof window === 'undefined' || !window.speechSynthesis) return;
    try {
      window.speechSynthesis.cancel();
    } catch {}
    if (activeUtteranceRef.current) {
      activeUtteranceRef.current.onstart = null;
      activeUtteranceRef.current.onend = null;
      activeUtteranceRef.current.onerror = null;
      activeUtteranceRef.current.onboundary = null;
      activeUtteranceRef.current = null;
    }
    (window as any).__novelhub_active_utterance = null;
    setIsPaused(true);
    setIsPlaying(false);
  }, []);

  const resume = useCallback(() => {
    if (typeof window === 'undefined' || !window.speechSynthesis) return;
    isPausedRef.current = false;
    isCanceledRef.current = false;
    const resumeIndex = savedChunkIndexRef.current;

    if (savedRemainingTextRef.current) {
      const remaining = savedRemainingTextRef.current;
      const offset = savedRemainingOffsetRef.current;
      const map = savedRemainingMapRef.current;
      savedRemainingTextRef.current = null;
      setIsPaused(false);
      setIsPlaying(true);
      playChunkDirect(remaining, offset, resumeIndex, map);
      return;
    }

    setIsPaused(false);
    setIsPlaying(true);
    playChunkAtIndex(resumeIndex);
  }, [playChunkAtIndex, playChunkDirect]);

  const stop = useCallback(() => {
    isCanceledRef.current = true;
    isPausedRef.current = false;
    chunksRef.current = [];
    currentChunkIndexRef.current = 0;
    savedChunkIndexRef.current = 0;
    savedRemainingTextRef.current = null;
    savedRemainingOffsetRef.current = 0;
    savedRemainingMapRef.current = [];
    if (typeof window === 'undefined' || !window.speechSynthesis) return;
    try {
      window.speechSynthesis.cancel();
    } catch {}
    if (activeUtteranceRef.current) {
      activeUtteranceRef.current.onstart = null;
      activeUtteranceRef.current.onend = null;
      activeUtteranceRef.current.onerror = null;
      activeUtteranceRef.current.onboundary = null;
      activeUtteranceRef.current = null;
    }
    (window as any).__novelhub_active_utterance = null;
    setIsPlaying(false);
    setIsPaused(false);
  }, []);

  return {
    isSupported,
    voices,
    selectedVoice,
    setSelectedVoice,
    isPlaying,
    isPaused,
    rate,
    setRate,
    pitch,
    setPitch,
    volume,
    setVolume,
    speak,
    pause,
    resume,
    stop,
  };
}
