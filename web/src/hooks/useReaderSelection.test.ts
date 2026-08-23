import { describe, expect, it } from "vitest";
import { getToolbarPosition } from "./useReaderSelection";

describe("getToolbarPosition", () => {
  it("positions toolbar above when selection has enough clearance", () => {
    const pos = getToolbarPosition({ left: 390, width: 20, top: 400, height: 20 }, 1000);
    expect(pos).toEqual({
      top: 217,
      left: 210,
      placement: "above",
    });
  });

  it("positions toolbar below when selection is near top of viewport", () => {
    const pos = getToolbarPosition({ left: 200, width: 20, top: 70, height: 20 }, 360);
    expect(pos).toEqual({
      top: 98,
      left: 8,
      placement: "below",
    });
    expect(pos.left).toBeGreaterThanOrEqual(8);
    expect(pos.left + (360 - 16)).toBeLessThanOrEqual(360);
  });

  it("clamps toolbar edges on wide viewports near screen boundaries", () => {
    const edge = getToolbarPosition({ left: 790, width: 20, top: 400, height: 20 }, 800);
    expect(edge).toEqual({
      top: 217,
      left: 412,
      placement: "above",
    });
    expect(edge.left).toBeGreaterThanOrEqual(8);
    expect(edge.left + 380).toBeLessThanOrEqual(792);
  });

  it("safely positions toolbar at top under header when entire page or large block is selected", () => {
    const pos = getToolbarPosition({ left: 100, width: 800, top: 70, bottom: 780, height: 710 }, 1024, 800);
    expect(pos).toEqual({
      top: 72,
      left: 310,
      placement: "below",
    });
  });
});
