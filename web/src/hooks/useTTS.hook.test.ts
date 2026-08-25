/**
 * Drive the REAL useTTS hook (mounted with react-dom) against a fake
 * speechSynthesis that emits realistic word boundaries, and assert the
 * onBoundary global charIndex stream maps to the right raw words after
 * pause/resume and interrupt cycles.
 */
import { describe, expect, it, beforeEach, afterEach } from "vitest";

type Utterance = SpeechSynthesisUtterance;

let synth: {
  speakCount: number;
  cancelCount: number;
  paused: boolean;
  queue: Utterance[];
  emitBoundary: (u: Utterance, charIndex: number, charLength?: number) => void;
  emitEnd: (u: Utterance) => void;
  emitError: (u: Utterance, error: string, elapsedTime?: number) => void;
};

class FakeUtterance {
  text: string;
  voice: any = null;
  rate = 1;
  pitch = 1;
  volume = 1;
  onstart: any = null;
  onend: any = null;
  onerror: any = null;
  onpause: any = null;
  onresume: any = null;
  onboundary: any = null;
  constructor(text: string) { this.text = text; }
}

function installFakeSpeechSynthesis() {
  (globalThis as any).SpeechSynthesisUtterance = FakeUtterance;
  const queue: Utterance[] = [];

  const fake = {
    queue,
    paused: false,
    speakCount: 0,
    cancelCount: 0,
    getVoices: () => [],
    speak: (u: Utterance) => {
      fake.speakCount++;
      queue.push(u);
      u.onstart?.({ utterance: u } as any);
    },
    cancel: () => {
      fake.cancelCount++;
    },
    pause: () => { fake.paused = true; },
    resume: () => { fake.paused = false; },
    emitBoundary: (u: Utterance, charIndex: number, charLength = 1) => {
      u.onboundary?.({ charIndex, charLength, name: "word", elapsedTime: 100, utterance: u } as any);
    },
    emitEnd: (u: Utterance) => {
      u.onend?.({ utterance: u } as any);
    },
    emitError: (u: Utterance, error: string, elapsedTime = 500) => {
      u.onerror?.({ error, elapsedTime, utterance: u } as any);
    },
  };
  return fake;
}

const extractWords = (t: string) => {
  const out: { start: number; word: string }[] = [];
  const re = /[\p{L}\p{M}\p{N}_—–-]+/gu;
  let m: RegExpExecArray | null;
  while ((m = re.exec(t)) !== null) out.push({ start: m.index, word: m[0] });
  return out;
};

const mountHook = async () => {
  const mod = await import("./useTTS");
  const React = await import("react");
  const { createRoot } = await import("react-dom/client");

  let api: ReturnType<typeof mod.useTTS> | null = null;
  const boundaries: { charIndex: number; charLength: number; word: string }[] = [];

  const Harness = () => {
    api = mod.useTTS({
      onBoundary: (e) => {
        const t = (e.utterance as SpeechSynthesisUtterance).text || "";
        boundaries.push({ charIndex: e.charIndex, charLength: e.charLength, word: t.slice(e.charIndex, e.charIndex + e.charLength) });
      },
    });
    return null;
  };

  const container = document.createElement("div");
  document.body.appendChild(container);
  const root = createRoot(container);
  await new Promise<void>((r) => {
    root.render(React.createElement(Harness));
    setTimeout(r, 10);
  });

  return {
    api: () => api!,
    boundaries,
    cleanup: async () => {
      root.unmount();
      container.remove();
    },
  };
};

describe("real useTTS hook, pause/resume offset", () => {
  beforeEach(() => {
    (globalThis as any).CSS = undefined;
    synth = installFakeSpeechSynthesis();
    globalThis.window.speechSynthesis = synth as any;
  });
  afterEach(() => {});

  it("pause BEFORE any word boundary fires keeps the first word for resume", async () => {
    const raw = "Chương một. Đây là câu mở đầu rất quan trọng, xin hãy đọc kỹ.";
    const h = await mountHook();
    const api = h.api();

    api.speak(raw);
    await new Promise((r) => setTimeout(r, 10));
    const u0 = synth.queue[0];
    expect(u0).toBeTruthy();

    // User pauses before the engine has emitted its first word boundary.
    api.pause();
    await new Promise((r) => setTimeout(r, 0));
    api.resume();
    await new Promise((r) => setTimeout(r, 10));

    const u1 = synth.queue[1];
    expect(u1, "resume speaks again").toBeTruthy();
    // The whole utterance must be re-read from its start — the first word is NOT dropped.
    expect(u1.text.includes("Chương"), "first word still present after resume").toBe(true);
    expect(u1.text.startsWith("Chương"), "resume starts at the beginning, nothing skipped").toBe(true);

    // And its first boundary maps to raw index 0.
    synth.emitBoundary(u1, 0, "Chương".length);
    await new Promise((r) => setTimeout(r, 0));
    const last = h.boundaries[h.boundaries.length - 1];
    expect(last.charIndex, "first word boundary maps to raw 0").toBe(0);

    await h.cleanup();
  });

  it("pause at a mid-chunk word then resume: resumed boundary lands on the next raw word", async () => {
    const raw = "Anh ấy nói rằng: “tôi không biết”, rồi bước đi. Cô đứng đó, im lặng.";
    const words = extractWords(raw);
    const h = await mountHook();
    const api = h.api();

    api.speak(raw);
    await new Promise((r) => setTimeout(r, 10));

    // Spoken words 0..4 (through "tôi"); current word = "tôi".
    const u0 = synth.queue[0];
    for (let i = 0; i < 5; i++) {
      synth.emitBoundary(u0, words[i].start, words[i].word.length);
      await new Promise((r) => setTimeout(r, 0));
    }

    api.pause();
    await new Promise((r) => setTimeout(r, 0));
    api.resume();
    await new Promise((r) => setTimeout(r, 10));

    const u1 = synth.queue[1];
    expect(u1, "resume speaks a new utterance").toBeTruthy();

    const resumeWords = extractWords(u1.text);
    // Resume should START at the word right after "tôi" ("không").
    expect(resumeWords[0].word, "first resumed word is after the paused word").toBe("không");

    // Emit the first resumed boundary (local 0).
    const firstLocal = u1.text.indexOf(resumeWords[0].word);
    synth.emitBoundary(u1, firstLocal, resumeWords[0].word.length);
    await new Promise((r) => setTimeout(r, 0));

    const last = h.boundaries[h.boundaries.length - 1];
    const expectedRawPos = raw.indexOf("không");
    expect(last.charIndex, "global charIndex of resumed first word").toBe(expectedRawPos);
    expect(raw.slice(last.charIndex, last.charIndex + last.charLength), "raw slice is the resumed word").toBe("không");

    await h.cleanup();
  });

  it("interrupt then continue: no offset on the next world", async () => {
    const raw = "Sáng hôm đó, cô gái thức dậy. Trời mưa rất to, mọi thứ đều ẩm ướt.";
    const words = extractWords(raw);
    const h = await mountHook();
    const api = h.api();

    api.speak(raw);
    await new Promise((r) => setTimeout(r, 10));

    const u0 = synth.queue[0];
    for (let i = 0; i < 8; i++) {
      synth.emitBoundary(u0, words[i].start, words[i].word.length);
      await new Promise((r) => setTimeout(r, 0));
    }
    // current word = words[7] ("thức"?), interrupt mid-speech.
    synth.emitError(u0, "interrupted", 1000);
    await new Promise((r) => setTimeout(r, 70)); // wait out the 40ms re-play delay

    const u1 = synth.queue[1];
    expect(u1).toBeTruthy();
    const resumeWords = extractWords(u1.text);
    expect(resumeWords[0].word).toBe(words[8].word);

    const firstLocal = u1.text.indexOf(resumeWords[0].word);
    synth.emitBoundary(u1, firstLocal, resumeWords[0].word.length);
    await new Promise((r) => setTimeout(r, 0));

    const last = h.boundaries[h.boundaries.length - 1];
    expect(raw.slice(last.charIndex, last.charIndex + last.charLength)).toBe(resumeWords[0].word);

    await h.cleanup();
  });
});