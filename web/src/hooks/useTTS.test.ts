import { describe, expect, it } from "vitest";
import {
  cleanTextForTTSWithMap,
  getNextWordOffset,
  splitTextIntoChunks,
} from "./useTTS";

/** Tests TTS pause/resume character offset tracking. */
function resumeState(text: string, map: number[], lastIdx: number) {
  const nextWordPos = getNextWordOffset(text, lastIdx);
  const sliceFrom = text.slice(nextWordPos);
  const remaining = sliceFrom.trim();
  const cleanStart =
    nextWordPos + (sliceFrom.length - sliceFrom.trimStart().length);
  const rawStart = map[cleanStart] ?? cleanStart;
  return {
    remaining,
    cleanStart,
    rawStart,
    offsetShift: rawStart,
    map: map
      .slice(cleanStart)
      .map((rawIndex) => Math.max(0, rawIndex - rawStart)),
  };
}

function wordStarts(text: string): number[] {
  const starts: number[] = [];
  const re = /[\p{L}\p{M}\p{N}_—–-]+/gu;
  let m: RegExpExecArray | null;
  while ((m = re.exec(text)) !== null) starts.push(m.index);
  return starts;
}

function assertResumeInvariant(
  text: string,
  map: number[],
  lastIdx: number,
  label: string,
) {
  const {
    remaining,
    cleanStart,
    map: rebased,
    offsetShift,
  } = resumeState(text, map, lastIdx);
  expect(
    text.slice(cleanStart, cleanStart + remaining.length),
    `${label} remaining is an exact substring`,
  ).toBe(remaining);
  // Every word in the resumed text must map to the same raw position it had in the chunk.
  for (const s of wordStarts(remaining)) {
    const localRaw = rebased[s] ?? s;
    const absRaw = map[cleanStart + s] ?? cleanStart + s;
    expect(
      localRaw + offsetShift,
      `${label} resumed word @ ${s} matches chunk mapping`,
    ).toBe(absRaw);
  }
}

describe("resume offset invariants", () => {
  const samples = [
    "Phần 1. Lời mở đầu: xin chào bạn đọc.",
    "Cô ấy nói: “tôi đến rồi”, rồi quay đi. Chúng ta — hai người — gặp lại sau nhé!",
    "Đây là câu đầu. Câu thứ hai cũng vậy. Câu thứ ba kết thúc.",
    "これは最初の文章です。次は二番目の文章です。三番目で終わり。",
  ];

  it("holds for every word boundary in every sample", () => {
    for (const raw of samples) {
      const chunk = cleanTextForTTSWithMap(raw);
      for (const s of wordStarts(chunk.text)) {
        assertResumeInvariant(
          chunk.text,
          chunk.cleanToRawMap,
          s,
          `raw="${raw}" lastIdx=${s}`,
        );
      }
    }
  });

  it("holds even when the chunk text ends with whitespace (trailing trim must not shift the map)", () => {
    // Simulates a sub-split chunk boundary that leaves trailing spaces at the end
    // of the current utterance text (splitPos lands 1 char past a whitespace run).
    const trailingWsChunk = cleanTextForTTSWithMap(
      "Trong khi đó, có một khoảng trắng dài ở cuối.   ",
    );
    const clean = trailingWsChunk.text;
    const expectedTrailing = "   ";
    expect(
      clean.slice(-expectedTrailing.length),
      "sanity: chunk indeed ends with whitespace",
    ).toBe(expectedTrailing);

    for (const s of wordStarts(clean)) {
      assertResumeInvariant(
        clean,
        trailingWsChunk.cleanToRawMap,
        s,
        "trailing-ws lastIdx=" + s,
      );
    }
  });
});

describe("TTS text offset mapping", () => {
  it("keeps Vietnamese word offsets aligned after punctuation cleanup", () => {
    const raw = "“lời mở đầu”";
    const cleaned = cleanTextForTTSWithMap(raw);
    const cleanStart = cleaned.text.indexOf("lời");
    const cleanEnd = cleaned.text.indexOf("đầu") + "đầu".length - 1;

    expect(cleaned.text).toBe('"lời mở đầu"');
    expect(
      raw.slice(
        cleaned.cleanToRawMap[cleanStart],
        cleaned.cleanToRawMap[cleanEnd] + 1,
      ),
    ).toBe("lời mở đầu");
  });

  it("returns raw character positions for chunked text spoken from the reader DOM", () => {
    const chunks = splitTextIntoChunks("Phần 1. Lời mở đầu: xin chào bạn đọc.");
    const chunk = chunks.find((item) => item.text.includes("Lời mở đầu"));

    expect(chunk).toBeTruthy();
    const localStart = chunk!.text.indexOf("Lời");
    const localEnd = chunk!.text.indexOf("đầu") + "đầu".length - 1;
    const rawStart = chunk!.charOffset + chunk!.cleanToRawMap[localStart];
    const rawEnd = chunk!.charOffset + chunk!.cleanToRawMap[localEnd] + 1;

    expect(
      "Phần 1. Lời mở đầu: xin chào bạn đọc.".slice(rawStart, rawEnd),
    ).toBe("Lời mở đầu");
  });
});
