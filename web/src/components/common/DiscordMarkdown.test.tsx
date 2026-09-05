import { act } from "react";
import { createRoot } from "react-dom/client";
import { describe, expect, it } from "vitest";
import { DiscordMarkdown } from "./DiscordMarkdown";

describe("DiscordMarkdown", () => {
  it("renders plain text correctly", () => {
    const container = document.createElement("div");
    const root = createRoot(container);
    act(() => {
      root.render(<DiscordMarkdown content="Hello world" />);
    });
    expect(container.textContent).toContain("Hello world");
  });

  it("renders bold, italic, underline, strikethrough, code, and links", () => {
    const container = document.createElement("div");
    const root = createRoot(container);
    act(() => {
      root.render(
        <DiscordMarkdown
          content={
            "**Bold Text** and *Italic Text* and __Underline__ and ~~Strike~~ and `code block` and https://example.com"
          }
        />,
      );
    });

    const bold = container.querySelector("strong");
    expect(bold?.textContent).toBe("Bold Text");

    const italic = container.querySelector("em");
    expect(italic?.textContent).toBe("Italic Text");

    const underline = container.querySelector("u");
    expect(underline?.textContent).toBe("Underline");

    const del = container.querySelector("del");
    expect(del?.textContent).toBe("Strike");

    const code = container.querySelector("code");
    expect(code?.textContent).toBe("code block");

    const link = container.querySelector("a");
    expect(link?.getAttribute("href")).toBe("https://example.com");
  });

  it("renders spoilers and toggles reveal on click", () => {
    const container = document.createElement("div");
    const root = createRoot(container);
    act(() => {
      root.render(
        <DiscordMarkdown content="The killer is ||John Doe|| at the end." />,
      );
    });

    const spoilerBtn = container.querySelector(
      '[role="button"]',
    ) as HTMLElement;
    expect(spoilerBtn).not.toBeNull();
    expect(spoilerBtn.title).toBe("Click to reveal spoiler");

    act(() => {
      spoilerBtn.click();
    });

    expect(spoilerBtn.title).toBe("Click to hide spoiler");
    expect(spoilerBtn.textContent).toContain("John Doe");

    act(() => {
      spoilerBtn.click();
    });

    expect(spoilerBtn.title).toBe("Click to reveal spoiler");
  });

  it("renders blockquotes correctly", () => {
    const container = document.createElement("div");
    const root = createRoot(container);
    act(() => {
      root.render(<DiscordMarkdown content={"> This is a quoted review"} />);
    });

    const blockquote = container.querySelector("blockquote");
    expect(blockquote).not.toBeNull();
    expect(blockquote?.textContent).toBe("This is a quoted review");
  });
});
