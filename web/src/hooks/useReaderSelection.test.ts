import { describe, expect, it } from "vitest";
import { getToolbarPosition } from "./useReaderSelection";

describe("getToolbarPosition", () => {
  it("centers the known-width toolbar around desktop selections", () => {
    expect(getToolbarPosition({ left: 390, width: 20, top: 100 }, 1000)).toEqual({
      top: 60,
      left: 180,
    });
  });

  it("clamps toolbar edges and keeps narrow layouts within margins", () => {
    const narrow = getToolbarPosition({ left: 200, width: 20, top: 50 }, 360);
    expect(narrow).toEqual({ top: 10, left: 8 });
    expect(narrow.left).toBeGreaterThanOrEqual(8);
    expect(narrow.left + (360 - 16)).toBeLessThanOrEqual(360);

    const edge = getToolbarPosition({ left: 790, width: 20, top: 100 }, 800);
    expect(edge).toEqual({ top: 60, left: 352 });
    expect(edge.left).toBeGreaterThanOrEqual(8);
    expect(edge.left + 440).toBeLessThanOrEqual(792);
  });
});
