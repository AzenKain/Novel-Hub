import { describe, expect, it } from "vitest";
import { getToolbarPosition } from "./useReaderSelection";

describe("getToolbarPosition", () => {
  it("clamps the left edge to viewport margins at every width", () => {
    const narrow = getToolbarPosition({ left: 200, width: 20, top: 50 }, 360);
    expect(narrow).toEqual({ top: 10, left: 8 });
    expect(narrow.left).toBeGreaterThanOrEqual(8);
    expect(narrow.left + (360 - 16)).toBeLessThanOrEqual(360);

    const edge = getToolbarPosition({ left: 790, width: 20, top: 100 }, 800);
    expect(edge).toEqual({ top: 60, left: 792 });
    expect(edge.left).toBeLessThanOrEqual(792);
  });
});
