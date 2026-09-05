/** Tests useTTS against simulated speech synthesis. */
import { describe, expect, it, beforeEach, afterEach } from "vitest";

type Utterance = SpeechSynthesisUtterance;

let synth: {
  speakCount: number;
  cancelCount: number;
  paused: boolean;
  queue: Utterance[];
  getVoices: () => any[];
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
  constructor(text: string) {
    this.text = text;
  }
}

function installFakeSpeechSynthesis() {
  (globalThis as any).SpeechSynthesisUtterance = FakeUtterance;
  const queue: Utterance[] = [];

  const fake = {
    queue,
    paused: false,
    speakCount: 0,
    cancelCount: 0,
    getVoices: () => [
      { name: "Mock Voice", lang: "vi-VN", default: true } as any,
    ],
    speak: (u: Utterance) => {
      fake.speakCount++;
      queue.push(u);
      u.onstart?.({ utterance: u } as any);
    },
    cancel: () => {
      fake.cancelCount++;
    },
    pause: () => {
      fake.paused = true;
    },
    resume: () => {
      fake.paused = false;
    },
    emitBoundary: (u: Utterance, charIndex: number, charLength = 1) => {
      u.onboundary?.({
        charIndex,
        charLength,
        name: "word",
        elapsedTime: 100,
        utterance: u,
      } as any);
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
  const boundaries: { charIndex: number; charLength: number; word: string }[] =
    [];

  const Harness = () => {
    api = mod.useTTS({
      onBoundary: (e) => {
        const t = (e.utterance as SpeechSynthesisUtterance).text || "";
        boundaries.push({
          charIndex: e.charIndex,
          charLength: e.charLength,
          word: t.slice(e.charIndex, e.charIndex + e.charLength),
        });
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

    api.pause();
    await new Promise((r) => setTimeout(r, 0));
    api.resume();
    await new Promise((r) => setTimeout(r, 10));

    const u1 = synth.queue[1];
    expect(u1, "resume speaks again").toBeTruthy();
    expect(
      u1.text.includes("Chương"),
      "first word still present after resume",
    ).toBe(true);
    expect(
      u1.text.startsWith("Chương"),
      "resume starts at the beginning, nothing skipped",
    ).toBe(true);

    synth.emitBoundary(u1, 0, "Chương".length);
    await new Promise((r) => setTimeout(r, 0));
    const last = h.boundaries[h.boundaries.length - 1];
    expect(last.charIndex, "first word boundary maps to raw 0").toBe(0);

    await h.cleanup();
  });

  it("pause at a mid-chunk word then resume: resumed boundary lands on the next raw word", async () => {
    const raw =
      "Anh ấy nói rằng: “tôi không biết”, rồi bước đi. Cô đứng đó, im lặng.";
    const words = extractWords(raw);
    const h = await mountHook();
    const api = h.api();

    api.speak(raw);
    await new Promise((r) => setTimeout(r, 10));

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
    expect(
      resumeWords[0].word,
      "first resumed word is after the paused word",
    ).toBe("không");

    const firstLocal = u1.text.indexOf(resumeWords[0].word);
    synth.emitBoundary(u1, firstLocal, resumeWords[0].word.length);
    await new Promise((r) => setTimeout(r, 0));

    const last = h.boundaries[h.boundaries.length - 1];
    const expectedRawPos = raw.indexOf("không");
    expect(last.charIndex, "global charIndex of resumed first word").toBe(
      expectedRawPos,
    );
    expect(
      raw.slice(last.charIndex, last.charIndex + last.charLength),
      "raw slice is the resumed word",
    ).toBe("không");

    await h.cleanup();
  });

  it("interrupt then continue: no offset on the next world", async () => {
    const raw =
      "Sáng hôm đó, cô gái thức dậy. Trời mưa rất to, mọi thứ đều ẩm ướt.";
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
    synth.emitError(u0, "interrupted", 1000);
    await new Promise((r) => setTimeout(r, 70));

    const u1 = synth.queue[1];
    expect(u1).toBeTruthy();
    const resumeWords = extractWords(u1.text);
    expect(resumeWords[0].word).toBe(words[8].word);

    const firstLocal = u1.text.indexOf(resumeWords[0].word);
    synth.emitBoundary(u1, firstLocal, resumeWords[0].word.length);
    await new Promise((r) => setTimeout(r, 0));

    const last = h.boundaries[h.boundaries.length - 1];
    expect(raw.slice(last.charIndex, last.charIndex + last.charLength)).toBe(
      resumeWords[0].word,
    );

    await h.cleanup();
  });

  it("plays using system default voice when getVoices returns empty list", async () => {
    synth.getVoices = () => [];
    const mod = await import("./useTTS");
    const React = await import("react");
    const { createRoot } = await import("react-dom/client");

    let api: ReturnType<typeof mod.useTTS> | null = null;
    const Harness = () => {
      api = mod.useTTS({});
      return null;
    };

    const container = document.createElement("div");
    document.body.appendChild(container);
    const root = createRoot(container);
    await new Promise<void>((r) => {
      root.render(React.createElement(Harness));
      setTimeout(r, 10);
    });

    api!.speak("Testing speak with system default voice");
    await new Promise((r) => setTimeout(r, 10));

    expect(synth.queue.length).toBeGreaterThan(0);

    root.unmount();
    container.remove();
  });
});

describe("detectLanguage", () => {
  it("detects Vietnamese text correctly", async () => {
    const { detectLanguage } = await import("./useTTS");
    expect(detectLanguage("Đây là cuốn sách rất hay.")).toBe("vi-VN");
    expect(detectLanguage("Tuuka nhìn tôi và cười nhẹ nhàng.")).toBe("vi-VN");
  });

  it("detects Japanese, Chinese, Korean, Cyrillic, Arabic correctly", async () => {
    const { detectLanguage } = await import("./useTTS");
    expect(detectLanguage("吾輩は猫である。名前はまだ無い。")).toBe("ja-JP");
    expect(detectLanguage("你好世界，今天天气真好。")).toBe("zh-CN");
    expect(detectLanguage("안녕하세요, 반갑습니다.")).toBe("ko-KR");
    expect(detectLanguage("Привет мир, как дела?")).toBe("ru-RU");
    expect(detectLanguage("مرحبا بكم في عالم جديد")).toBe("ar-SA");
  });

  it("falls back to en-US for plain ASCII english text", async () => {
    const { detectLanguage } = await import("./useTTS");
    expect(detectLanguage("Once upon a time in a faraway kingdom.")).toBe(
      "en-US",
    );
  });
});
