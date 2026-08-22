import { useState, useEffect, useCallback, useRef } from 'react';

export interface UseTTSOptions {
  onEnd?: () => void;
  onError?: (event: any) => void;
  onBoundary?: (event: any) => void;
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

  useEffect(() => {
    if (typeof window !== 'undefined') {
      setIsSupported(true);
      
      if (window.speechSynthesis) {
        const updateVoices = () => {
          const availableVoices = window.speechSynthesis.getVoices();
          setVoices(availableVoices);
          if (availableVoices.length > 0 && !selectedVoice) {
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
      if (typeof window !== 'undefined' && window.speechSynthesis) {
        try {
          window.speechSynthesis.cancel();
        } catch {}
      }
    };
  }, []);

  const speak = useCallback((text: string) => {
    if (typeof window === 'undefined' || !text.trim() || !window.speechSynthesis) return;

    if (activeUtteranceRef.current) {
      activeUtteranceRef.current.onstart = null;
      activeUtteranceRef.current.onend = null;
      activeUtteranceRef.current.onerror = null;
      activeUtteranceRef.current.onpause = null;
      activeUtteranceRef.current.onresume = null;
      activeUtteranceRef.current.onboundary = null;
      activeUtteranceRef.current = null;
    }

    try {
      window.speechSynthesis.cancel();
      if (window.speechSynthesis.paused) {
        window.speechSynthesis.resume();
      }
    } catch {}

    const utterance = new SpeechSynthesisUtterance(text);
    activeUtteranceRef.current = utterance;

    if (selectedVoice) {
      utterance.voice = selectedVoice;
    }
    utterance.rate = rate;
    utterance.pitch = pitch;
    utterance.volume = volume;

    utterance.onstart = () => {
      setIsPlaying(true);
      setIsPaused(false);
    };

    utterance.onend = () => {
      activeUtteranceRef.current = null;
      setIsPlaying(false);
      setIsPaused(false);
      optionsRef.current?.onEnd?.();
    };

    utterance.onerror = (e) => {
      if (e.error === "canceled" || e.error === "interrupted") {
        setIsPlaying(false);
        setIsPaused(false);
        return;
      }
      activeUtteranceRef.current = null;
      setIsPlaying(false);
      setIsPaused(false);
      optionsRef.current?.onError?.(e);
    };

    utterance.onpause = () => {
      setIsPaused(true);
      setIsPlaying(false);
    };
    
    utterance.onresume = () => {
      setIsPaused(false);
      setIsPlaying(true);
    };

    utterance.onboundary = (e) => {
      optionsRef.current?.onBoundary?.(e);
    };

    try {
      window.speechSynthesis.speak(utterance);
      setIsPlaying(true);
      setIsPaused(false);
    } catch {
      setIsPlaying(false);
      setIsPaused(false);
    }
  }, [selectedVoice, rate, pitch, volume]);

  const pause = useCallback(() => {
    if (typeof window === 'undefined' || !window.speechSynthesis) return;
    try {
      window.speechSynthesis.pause();
    } catch {}
    setIsPaused(true);
    setIsPlaying(false);
  }, []);

  const resume = useCallback(() => {
    if (typeof window === 'undefined' || !window.speechSynthesis) return;
    try {
      window.speechSynthesis.resume();
    } catch {}
    setIsPaused(false);
    setIsPlaying(true);
  }, []);

  const stop = useCallback(() => {
    if (typeof window === 'undefined' || !window.speechSynthesis) return;
    try {
      window.speechSynthesis.cancel();
    } catch {}
    if (activeUtteranceRef.current) {
      activeUtteranceRef.current.onstart = null;
      activeUtteranceRef.current.onend = null;
      activeUtteranceRef.current.onerror = null;
      activeUtteranceRef.current.onpause = null;
      activeUtteranceRef.current.onresume = null;
      activeUtteranceRef.current.onboundary = null;
      activeUtteranceRef.current = null;
    }
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
