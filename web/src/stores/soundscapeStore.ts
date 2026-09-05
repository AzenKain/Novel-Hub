import { create } from "zustand";
import { persist, createJSONStorage } from "zustand/middleware";
import type { Soundscape } from "@/types";

export interface ActiveTrack {
  id: string;
  name: string;
  category: string;
  icon: string;
  volume: number;
  streamUrl: string;
  isSynthesized?: boolean;
  synthType?: "white_noise" | "pink_noise" | "brown_noise" | "rain" | "waves";
}

interface SoundscapeState {
  isPlaying: boolean;
  masterVolume: number;
  activeTracks: Record<string, ActiveTrack>;
  soundscapePanelOpen: boolean;

  setPlaying: (playing: boolean) => void;
  togglePlaying: () => void;
  setMasterVolume: (vol: number) => void;
  setSoundscapePanelOpen: (open: boolean) => void;

  toggleTrack: (track: ActiveTrack) => void;
  setTrackVolume: (trackId: string, volume: number) => void;
  removeTrack: (trackId: string) => void;
  stopAll: () => void;
  applyPreset: (presetName: string) => void;
}

export const BUILTIN_AMBIENT_PRESETS: ActiveTrack[] = [
  {
    id: "synth-white-noise",
    name: "White Noise",
    category: "noise",
    icon: "Wind",
    volume: 0.3,
    streamUrl: "",
    isSynthesized: true,
    synthType: "white_noise",
  },
  {
    id: "synth-pink-noise",
    name: "Pink Noise (Deep Focus)",
    category: "noise",
    icon: "CloudRain",
    volume: 0.35,
    streamUrl: "",
    isSynthesized: true,
    synthType: "pink_noise",
  },
  {
    id: "synth-brown-noise",
    name: "Brown Noise (Cozy Rumble)",
    category: "noise",
    icon: "Waves",
    volume: 0.45,
    streamUrl: "",
    isSynthesized: true,
    synthType: "brown_noise",
  },
  {
    id: "synth-rain",
    name: "Rain & Storm",
    category: "rain",
    icon: "CloudRain",
    volume: 0.4,
    streamUrl: "",
    isSynthesized: true,
    synthType: "rain",
  },
  {
    id: "synth-waves",
    name: "Ocean Waves",
    category: "waves",
    icon: "Waves",
    volume: 0.45,
    streamUrl: "",
    isSynthesized: true,
    synthType: "waves",
  },
];

const audioElements = new Map<string, HTMLAudioElement>();
let audioCtx: AudioContext | null = null;
const synthNodes = new Map<
  string,
  { node: AudioBufferSourceNode; gain: GainNode }
>();

function getOrCreateAudioContext(): AudioContext {
  if (!audioCtx) {
    const AudioCtxClass =
      window.AudioContext ||
      (window as unknown as { webkitAudioContext: typeof AudioContext })
        .webkitAudioContext;
    audioCtx = new AudioCtxClass();
  }
  if (audioCtx.state === "suspended") {
    void audioCtx.resume();
  }
  return audioCtx;
}

function generateNoiseBuffer(
  ctx: AudioContext,
  type: ActiveTrack["synthType"],
): AudioBuffer {
  const duration = 5.0;
  const sampleRate = ctx.sampleRate;
  const bufferSize = Math.floor(sampleRate * duration);
  const buffer = ctx.createBuffer(2, bufferSize, sampleRate);
  const left = buffer.getChannelData(0);
  const right = buffer.getChannelData(1);

  const crossfadeLen = Math.floor(sampleRate * 0.15);

  for (let ch = 0; ch < 2; ch++) {
    const data = ch === 0 ? left : right;

    if (type === "white_noise") {
      for (let i = 0; i < bufferSize; i++) {
        data[i] = (Math.random() * 2 - 1) * 0.35;
      }
    } else if (type === "pink_noise") {
      let b0 = 0,
        b1 = 0,
        b2 = 0,
        b3 = 0,
        b4 = 0,
        b5 = 0,
        b6 = 0;
      for (let i = 0; i < bufferSize; i++) {
        const white = Math.random() * 2 - 1;
        b0 = 0.99886 * b0 + white * 0.0555179;
        b1 = 0.99332 * b1 + white * 0.0750759;
        b2 = 0.969 * b2 + white * 0.153852;
        b3 = 0.8665 * b3 + white * 0.3104856;
        b4 = 0.55 * b4 + white * 0.5329522;
        b5 = -0.7616 * b5 - white * 0.016898;
        data[i] = (b0 + b1 + b2 + b3 + b4 + b5 + b6 + white * 0.5362) * 0.07;
        b6 = white * 0.115926;
      }
    } else if (type === "brown_noise") {
      let lastOut = 0.0;
      for (let i = 0; i < bufferSize; i++) {
        const white = Math.random() * 2 - 1;
        lastOut = (lastOut + 0.025 * white) / 1.015;
        data[i] = lastOut * 1.5;
      }
    } else if (type === "rain") {
      let b0 = 0,
        b1 = 0,
        b2 = 0;
      for (let i = 0; i < bufferSize; i++) {
        const white = Math.random() * 2 - 1;
        b0 = 0.99 * b0 + white * 0.05;
        b1 = 0.95 * b1 + white * 0.08;
        b2 = 0.85 * b2 + white * 0.15;
        const mod =
          0.8 + 0.2 * Math.sin((2 * Math.PI * i) / (sampleRate * 1.8) + ch);
        data[i] = (b0 + b1 + b2 + white * 0.1) * 0.12 * mod;
      }
    } else if (type === "waves") {
      let lastOut = 0.0;
      for (let i = 0; i < bufferSize; i++) {
        const white = Math.random() * 2 - 1;
        lastOut = (lastOut + 0.02 * white) / 1.01;
        const waveMod =
          Math.pow(Math.sin((Math.PI * i) / bufferSize), 2) * 1.8 + 0.2;
        data[i] = lastOut * 1.2 * waveMod;
      }
    }

    for (let i = 0; i < crossfadeLen; i++) {
      const weight = i / crossfadeLen;
      data[i] =
        data[i] * weight + data[bufferSize - crossfadeLen + i] * (1 - weight);
    }
  }

  return buffer;
}

function startSynth(
  id: string,
  type: ActiveTrack["synthType"],
  volume: number,
  masterVolume: number,
) {
  try {
    const ctx = getOrCreateAudioContext();
    stopSynth(id);

    const buffer = generateNoiseBuffer(ctx, type);
    const source = ctx.createBufferSource();
    source.buffer = buffer;
    source.loop = true;

    const gain = ctx.createGain();
    const effectiveVol = Math.max(0, Math.min(1, volume * masterVolume));
    gain.gain.setValueAtTime(effectiveVol, ctx.currentTime);

    source.connect(gain);
    gain.connect(ctx.destination);
    source.start();

    synthNodes.set(id, { node: source, gain });
  } catch (err) {
    console.error("Failed to start synth audio:", err);
  }
}

function stopSynth(id: string) {
  const existing = synthNodes.get(id);
  if (existing) {
    try {
      existing.node.stop();
      existing.node.disconnect();
      existing.gain.disconnect();
    } catch {}
    synthNodes.delete(id);
  }
}

function syncAudioTracks(
  activeTracks: Record<string, ActiveTrack>,
  isPlaying: boolean,
  masterVolume: number,
) {
  if (!isPlaying) {
    audioElements.forEach((audio) => {
      audio.pause();
    });
    synthNodes.forEach((_, id) => {
      stopSynth(id);
    });
    return;
  }

  if (typeof window !== "undefined") {
    getOrCreateAudioContext();
  }

  audioElements.forEach((audio, id) => {
    if (!activeTracks[id]) {
      audio.pause();
      audio.src = "";
      audioElements.delete(id);
    }
  });

  synthNodes.forEach((_, id) => {
    if (!activeTracks[id]) {
      stopSynth(id);
    }
  });

  Object.values(activeTracks).forEach((track) => {
    if (track.isSynthesized && track.synthType) {
      if (!synthNodes.has(track.id)) {
        startSynth(track.id, track.synthType, track.volume, masterVolume);
      } else {
        const entry = synthNodes.get(track.id);
        if (entry && audioCtx) {
          const effectiveVol = Math.max(
            0,
            Math.min(1, track.volume * masterVolume),
          );
          entry.gain.gain.setTargetAtTime(
            effectiveVol,
            audioCtx.currentTime,
            0.05,
          );
        }
      }
    } else if (track.streamUrl) {
      let audio = audioElements.get(track.id);
      if (!audio) {
        audio = new Audio(track.streamUrl);
        audio.crossOrigin = "anonymous";
        audio.loop = true;
        audio.preload = "auto";
        audioElements.set(track.id, audio);
      }
      audio.volume = Math.max(0, Math.min(1, track.volume * masterVolume));
      if (audio.paused) {
        audio.play().catch((err) => {
          console.warn("Autoplay audio error:", err);
        });
      }
    }
  });
}

export const useSoundscapeStore = create<SoundscapeState>()(
  persist(
    (set, get) => ({
      isPlaying: false,
      masterVolume: 0.8,
      activeTracks: {},
      soundscapePanelOpen: false,

      setPlaying: (playing) => {
        const { activeTracks, masterVolume } = get();
        const hasTracks = Object.keys(activeTracks).length > 0;
        if (!hasTracks && playing) {
          const pink =
            BUILTIN_AMBIENT_PRESETS.find((p) => p.id === "synth-pink-noise") ||
            BUILTIN_AMBIENT_PRESETS[0];
          const newTracks = { [pink.id]: pink };
          set({ isPlaying: true, activeTracks: newTracks });
          syncAudioTracks(newTracks, true, masterVolume);
          return;
        }
        const effectivePlaying = hasTracks && playing;
        set({ isPlaying: effectivePlaying });
        syncAudioTracks(activeTracks, effectivePlaying, masterVolume);
      },

      togglePlaying: () => {
        const { isPlaying, activeTracks, masterVolume } = get();
        const hasTracks = Object.keys(activeTracks).length > 0;
        if (!hasTracks) {
          const pink =
            BUILTIN_AMBIENT_PRESETS.find((p) => p.id === "synth-pink-noise") ||
            BUILTIN_AMBIENT_PRESETS[0];
          const newTracks = { [pink.id]: pink };
          set({ isPlaying: true, activeTracks: newTracks });
          syncAudioTracks(newTracks, true, masterVolume);
          return;
        }
        const next = !isPlaying;
        set({ isPlaying: next });
        syncAudioTracks(activeTracks, next, masterVolume);
      },

      setMasterVolume: (vol) => {
        const v = Math.max(0, Math.min(1, vol));
        set({ masterVolume: v });
        const { activeTracks, isPlaying } = get();
        syncAudioTracks(activeTracks, isPlaying, v);
      },

      setSoundscapePanelOpen: (open) => set({ soundscapePanelOpen: open }),

      toggleTrack: (track) => {
        const tracks = { ...get().activeTracks };
        if (tracks[track.id]) {
          delete tracks[track.id];
        } else {
          tracks[track.id] = track;
        }
        const isPlaying = Object.keys(tracks).length > 0;
        set({ activeTracks: tracks, isPlaying });
        syncAudioTracks(tracks, isPlaying, get().masterVolume);
      },

      setTrackVolume: (trackId, volume) => {
        const tracks = { ...get().activeTracks };
        if (tracks[trackId]) {
          tracks[trackId] = {
            ...tracks[trackId],
            volume: Math.max(0, Math.min(1, volume)),
          };
          set({ activeTracks: tracks });
          syncAudioTracks(tracks, get().isPlaying, get().masterVolume);
        }
      },

      removeTrack: (trackId) => {
        const tracks = { ...get().activeTracks };
        delete tracks[trackId];
        set({ activeTracks: tracks });
        syncAudioTracks(tracks, get().isPlaying, get().masterVolume);
      },

      stopAll: () => {
        set({ isPlaying: false, activeTracks: {} });
        syncAudioTracks({}, false, get().masterVolume);
      },

      applyPreset: (presetName) => {
        if (presetName === "pink_focus") {
          const pink = BUILTIN_AMBIENT_PRESETS.find(
            (p) => p.id === "synth-pink-noise",
          )!;
          const tracks = { [pink.id]: pink };
          set({ activeTracks: tracks, isPlaying: true });
          syncAudioTracks(tracks, true, get().masterVolume);
        } else if (presetName === "deep_brown") {
          const brown = BUILTIN_AMBIENT_PRESETS.find(
            (p) => p.id === "synth-brown-noise",
          )!;
          const tracks = { [brown.id]: brown };
          set({ activeTracks: tracks, isPlaying: true });
          syncAudioTracks(tracks, true, get().masterVolume);
        }
      },
    }),
    {
      name: "novelhub-soundscape-settings",
      storage: createJSONStorage(() => localStorage),
      version: 1,
      partialize: (state) => ({
        masterVolume: state.masterVolume,
        activeTracks: state.activeTracks,
      }),
    },
  ),
);
