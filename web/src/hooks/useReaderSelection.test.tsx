import { act } from "react";
import { createRoot } from "react-dom/client";
import { describe, expect, it } from "vitest";
import { useReaderSelection } from "./useReaderSelection";

function setup(addHighlight: ReturnType<typeof vi.fn>) {
  const container = document.createElement("div");
  const content = document.createElement("div");
  content.textContent = "hello world";
  container.append(content);
  document.body.append(container);
  const ref = { current: content };
  let api: ReturnType<typeof useReaderSelection>;
  function Harness() {
    api = useReaderSelection({ columnsRef: ref, contentRef: ref, savedSelectionRef: { current: null }, ttsStartPointRef: { current: null }, addHighlight, speak: vi.fn(), stop: vi.fn() });
    return null;
  }
  const root = createRoot(container);
  act(() => root.render(<Harness />));
  return { root, content, getApi: () => api! };
}

describe("useReaderSelection highlight validation", () => {
  it("does not submit empty or equal selections", async () => {
    const add = vi.fn().mockResolvedValue(undefined);
    const { root, content, getApi } = setup(add);
    const range = document.createRange();
    range.setStart(content.firstChild!, 2);
    range.setEnd(content.firstChild!, 2);
    act(() => getApi().setSelectionRange(range));
    await act(async () => getApi().handleHighlight("yellow"));
    expect(add).not.toHaveBeenCalled();
    root.unmount();
  });

  it("submits trimmed text with document-relative offsets", async () => {
    const add = vi.fn().mockResolvedValue(undefined);
    const { root, content, getApi } = setup(add);
    const range = document.createRange();
    range.setStart(content.firstChild!, 1);
    range.setEnd(content.firstChild!, 6);
    act(() => getApi().setSelectionRange(range));
    await act(async () => getApi().handleHighlight("blue"));
    expect(add).toHaveBeenCalledWith("hello", 1, 6, "blue");
    root.unmount();
  });
});
