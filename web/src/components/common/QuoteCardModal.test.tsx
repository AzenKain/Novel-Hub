import { act } from "react";
import { createRoot } from "react-dom/client";
import { describe, expect, it } from "vitest";
import { QuoteCardModal } from "./QuoteCardModal";

describe("QuoteCardModal", () => {
  it("does not render when open is false", () => {
    const container = document.createElement("div");
    const root = createRoot(container);
    act(() => {
      root.render(
        <QuoteCardModal
          open={false}
          onClose={() => {}}
          quote="Sample text"
        />,
      );
    });
    expect(container.querySelector("dialog")).toBeNull();
  });

  it("renders modal dialog and canvas when open is true with multi-paragraph text", () => {
    const container = document.createElement("div");
    const root = createRoot(container);
    const multiPara =
      "Paragraph 1 line.\n\nParagraph 2 line.\n\nParagraph 3 line.";
    act(() => {
      root.render(
        <QuoteCardModal
          open={true}
          onClose={() => {}}
          quote={multiPara}
          bookTitle="Test Book"
          bookAuthor="Author Name"
        />,
      );
    });
    expect(container.querySelector("dialog")).not.toBeNull();
    expect(container.querySelector("canvas")).not.toBeNull();
  });
});
