import { describe, expect, it } from "vitest";
import { getToolbarPosition } from "./useReaderSelection";

describe("getToolbarPosition", () => {
  it("keeps the transformed desktop toolbar inside viewport margins", () => {
    expect(getToolbarPosition({ left: 2, width: 4, top: 100 }, 800)).toEqual({
      top: 60,
      left: 228,
    });
    expect(getToolbarPosition({ left: 790, width: 20, top: 100 }, 800)).toEqual({
      top: 60,
      left: 572,
    });
  });

  it("anchors narrow toolbars to the viewport margin", () => {
    expect(getToolbarPosition({ left: 200, width: 20, top: 50 }, 360)).toEqual({
      top: 10,
      left: 8,
    });
  });
});
