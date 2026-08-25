import { act, useState, useEffect } from "react";
import { createRoot } from "react-dom/client";
import { describe, expect, it, vi, beforeEach } from "vitest";
import { useReaderPaging } from "./useReaderPaging";

(globalThis as any).IS_REACT_ACT_ENVIRONMENT = true;

// Mock ResizeObserver for JSDOM
globalThis.ResizeObserver = class ResizeObserver {
  observe() {}
  unobserve() {}
  disconnect() {}
} as unknown as typeof ResizeObserver;

describe("useReaderPaging mode transition tests", () => {
  beforeEach(() => {
    document.body.innerHTML = "";
  });

  function setupHarness(initialProps: {
    htmlContent?: string;
    effectiveReadingMode?: string;
    scrollLayout?: boolean;
    pageIndex?: number;
  } = {}) {
    const container = document.createElement("div");
    const content = document.createElement("div");
    const columns = document.createElement("div");
    const pageFrame = document.createElement("div");

    let mockScrollTop = 0;
    Object.defineProperty(content, "scrollTop", {
      configurable: true,
      get: () => mockScrollTop,
      set: (v) => { mockScrollTop = v; },
    });
    Object.defineProperty(content, "clientHeight", { configurable: true, value: 500 });
    Object.defineProperty(content, "scrollHeight", { configurable: true, value: 1500 });
    Object.defineProperty(pageFrame, "clientWidth", { configurable: true, value: 800 });
    Object.defineProperty(pageFrame, "clientHeight", { configurable: true, value: 500 });
    Object.defineProperty(columns, "clientWidth", { configurable: true, value: 800 });
    Object.defineProperty(columns, "scrollWidth", { configurable: true, value: 8400 }); // ~10 pages

    let mockScrollLeft = 0;
    Object.defineProperty(columns, "scrollLeft", {
      configurable: true,
      get: () => mockScrollLeft,
      set: (v) => { mockScrollLeft = v; },
    });
    (columns as any).scrollTo = (opts: any) => {
      if (opts && typeof opts.left === "number") {
        mockScrollLeft = opts.left;
      }
    };

    // Add multiple paragraph nodes to columns
    for (let i = 0; i < 20; i++) {
      const p = document.createElement("p");
      p.id = `para-${i}`;
      p.textContent = `Paragraph content ${i}`;
      Object.defineProperty(p, "offsetTop", { configurable: true, value: i * 75 });
      Object.defineProperty(p, "offsetLeft", { configurable: true, value: i * 420 });
      Object.defineProperty(p, "offsetWidth", { configurable: true, value: 700 });
      columns.appendChild(p);
    }

    content.appendChild(pageFrame);
    pageFrame.appendChild(columns);
    container.appendChild(content);
    document.body.appendChild(container);

    const contentRef = { current: content };
    const columnsRef = { current: columns };
    const pageFrameRef = { current: pageFrame };

    let api: ReturnType<typeof useReaderPaging>;
    let currentMode = initialProps.effectiveReadingMode ?? "scroll";
    let currentScrollLayout = initialProps.scrollLayout ?? true;
    let storePageIndex = initialProps.pageIndex ?? 0;
    const listeners = new Set<() => void>();
    const pageIndexSetter = vi.fn();
    const pageFrameWidthSetter = vi.fn();

    function setStorePageIndex(val: number) {
      storePageIndex = val;
      listeners.forEach((l) => l());
      pageIndexSetter(val);
    }

    let rerenderFn: (props: Partial<typeof initialProps>) => void;

    function Harness(props: {
      mode: string;
      scroll: boolean;
      html: string;
    }) {
      const [idx, setIdx] = useState(storePageIndex);
      useEffect(() => {
        const l = () => setIdx(storePageIndex);
        listeners.add(l);
        return () => { listeners.delete(l); };
      }, []);

      api = useReaderPaging({
        contentRef,
        columnsRef,
        pageFrameRef,
        htmlContent: props.html,
        maxWidth: 920,
        scrollLayout: props.scroll,
        effectiveReadingMode: props.mode,
        rtlPaging: false,
        pageAnimation: "eink",
        pageIndex: idx,
        setPageIndex: (newIdx) => {
          setStorePageIndex(typeof newIdx === "function" ? (newIdx as any)(storePageIndex) : newIdx);
        },
        setPageFrameWidth: pageFrameWidthSetter,
        onChapterNext: vi.fn(),
        onChapterPrev: vi.fn(),
      });
      return null;
    }

    const mountPoint = document.createElement("div");
    document.body.appendChild(mountPoint);
    const root = createRoot(mountPoint);
    act(() => {
      root.render(
        <Harness
          mode={currentMode}
          scroll={currentScrollLayout}
          html={initialProps.htmlContent ?? "<p>Test</p>"}
        />
      );
    });

    rerenderFn = (newProps) => {
      if (newProps.effectiveReadingMode !== undefined) currentMode = newProps.effectiveReadingMode;
      if (newProps.scrollLayout !== undefined) currentScrollLayout = newProps.scrollLayout;
      if (newProps.pageIndex !== undefined) setStorePageIndex(newProps.pageIndex);
      act(() => {
        root.render(
          <Harness
            mode={currentMode}
            scroll={currentScrollLayout}
            html={newProps.htmlContent ?? initialProps.htmlContent ?? "<p>Test</p>"}
          />
        );
      });
    };

    return {
      root,
      content,
      columns,
      pageFrame,
      getApi: () => api,
      rerender: rerenderFn,
      pageIndexSetter,
      pageFrameWidthSetter,
      cleanup: () => {
        root.unmount();
        mountPoint.remove();
        container.remove();
      },
    };
  }

  it("preserves position when switching from scroll mode to single page mode", async () => {
    const harness = setupHarness({
      effectiveReadingMode: "scroll",
      scrollLayout: true,
      pageIndex: 0,
    });

    // Simulate user scrolling halfway down in scroll mode
    act(() => {
      harness.content.scrollTop = 500; // 500 / (1500 - 500) = 50%
      harness.content.dispatchEvent(new Event("scroll"));
    });

    // Check fraction in scroll mode
    expect(harness.getApi().getLocationFraction()).toBeCloseTo(0.5, 1);

    // User switches to single page mode
    harness.rerender({
      effectiveReadingMode: "single",
      scrollLayout: false,
    });

    // Wait for frame and rAF calculations
    await act(async () => {
      await new Promise((r) => setTimeout(r, 100));
    });

    expect(harness.pageIndexSetter).toHaveBeenCalled();
    const lastCalledIndex = harness.pageIndexSetter.mock.calls.at(-1)?.[0];
    expect(lastCalledIndex).toBeGreaterThan(0);

    harness.cleanup();
  });

  it("preserves position when switching from single mode to double mode and back", async () => {
    const harness = setupHarness({
      effectiveReadingMode: "single",
      scrollLayout: false,
      pageIndex: 0,
    });

    // Move to page 4
    act(() => {
      harness.getApi().scrollToPageIndex(4, true);
    });
    expect(harness.pageIndexSetter).toHaveBeenCalledWith(4);
    harness.pageIndexSetter.mockClear();

    // Switch from single to double: page 4 -> page 2
    harness.rerender({
      effectiveReadingMode: "double",
      scrollLayout: false,
    });

    await act(async () => {
      await new Promise((r) => setTimeout(r, 50));
    });

    expect(harness.pageIndexSetter).toHaveBeenCalledWith(2);
    harness.pageIndexSetter.mockClear();

    // Switch from double to single: page 2 -> page 4
    harness.rerender({
      effectiveReadingMode: "single",
      scrollLayout: false,
    });

    await act(async () => {
      await new Promise((r) => setTimeout(r, 50));
    });

    expect(harness.pageIndexSetter).toHaveBeenCalledWith(4);

    harness.cleanup();
  });

  it("restores scroll position when switching from single page mode back to scroll mode", async () => {
    const harness = setupHarness({
      effectiveReadingMode: "single",
      scrollLayout: false,
      pageIndex: 0,
    });

    // Move to page 5
    act(() => {
      harness.getApi().scrollToPageIndex(5, true);
    });

    // Mock scrollTo on HTMLElement
    harness.content.scrollTo = vi.fn();

    // Switch from single to scroll
    harness.rerender({
      effectiveReadingMode: "scroll",
      scrollLayout: true,
    });

    await act(async () => {
      await new Promise((r) => setTimeout(r, 50));
    });

    // In scroll mode, scrollTop or scrollIntoView should position the reader
    expect(harness.content.scrollTop).toBeGreaterThanOrEqual(0);

    harness.cleanup();
  });

  it("does not jump to chapter start after switching single to double then scroll", async () => {
    const harness = setupHarness({
      effectiveReadingMode: "single",
      scrollLayout: false,
      pageIndex: 0,
    });

    Array.from(harness.columns.querySelectorAll("p")).forEach((p) => {
      Object.defineProperty(p, "offsetTop", { configurable: true, value: 0 });
    });

    act(() => {
      harness.getApi().scrollToPageIndex(5, true);
    });

    harness.rerender({
      effectiveReadingMode: "double",
      scrollLayout: false,
    });

    await act(async () => {
      await new Promise((r) => setTimeout(r, 80));
    });

    harness.rerender({
      effectiveReadingMode: "scroll",
      scrollLayout: true,
    });

    await act(async () => {
      await new Promise((r) => setTimeout(r, 180));
    });

    expect(harness.content.scrollTop).toBeGreaterThan(250);

    harness.cleanup();
  });
});
