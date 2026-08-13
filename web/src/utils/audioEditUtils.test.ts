import { describe, it, expect } from "vitest";
import { formatTime, getActiveRange, getPeaks, FileRange } from "./audioEditUtils";

// Mock helper to create a mock AudioBuffer
function createMockAudioBuffer(channels: number, length: number, sampleRate: number): AudioBuffer {
  const channelData = Array.from({ length: channels }, () => new Float32Array(length));
  return {
    numberOfChannels: channels,
    length,
    sampleRate,
    duration: length / sampleRate,
    getChannelData: (c: number) => channelData[c],
    copyToChannel: (src: Float32Array, c: number) => {
      channelData[c].set(src);
    }
  } as unknown as AudioBuffer;
}

describe("audioEditUtils tests", () => {
  describe("formatTime", () => {
    it("formats 0 seconds correctly", () => {
      expect(formatTime(0)).toBe("00:00.0");
    });

    it("formats sub-minute seconds correctly", () => {
      expect(formatTime(12.34)).toBe("00:12.3");
    });

    it("formats multi-minute seconds correctly", () => {
      expect(formatTime(75.5)).toBe("01:15.5");
    });
  });

  describe("getActiveRange", () => {
    const mockRanges: FileRange<string>[] = [
      { file: "track1.mp3", start: 0, end: 10, duration: 10 },
      { file: "track2.mp3", start: 10, end: 25, duration: 15 },
      { file: "track3.mp3", start: 25, end: 30, duration: 5 }
    ];

    it("returns track1 when time is at 0", () => {
      expect(getActiveRange(0, mockRanges)?.file).toBe("track1.mp3");
    });

    it("returns track1 when time is inside track1", () => {
      expect(getActiveRange(5.5, mockRanges)?.file).toBe("track1.mp3");
    });

    it("returns track2 when time is on the boundary", () => {
      expect(getActiveRange(10, mockRanges)?.file).toBe("track2.mp3");
    });

    it("returns track2 when time is inside track2", () => {
      expect(getActiveRange(15, mockRanges)?.file).toBe("track2.mp3");
    });

    it("returns track3 when time is inside track3", () => {
      expect(getActiveRange(27.8, mockRanges)?.file).toBe("track3.mp3");
    });

    it("defaults to first range if time is out of bounds", () => {
      expect(getActiveRange(50, mockRanges)?.file).toBe("track1.mp3");
    });
  });

  describe("getPeaks", () => {
    it("downsamples a buffer to target width correctly", () => {
      const buffer = createMockAudioBuffer(1, 1000, 100);
      const channel = buffer.getChannelData(0);

      // Seed peak data
      for (let i = 0; i < 1000; i++) {
        channel[i] = Math.sin(i / 10);
      }

      const peaks = getPeaks(buffer, 10);
      expect(peaks).toHaveLength(10);
      peaks.forEach(p => {
        expect(p).toBeGreaterThan(0);
        expect(p).toBeLessThanOrEqual(2.0);
      });
    });
  });
});