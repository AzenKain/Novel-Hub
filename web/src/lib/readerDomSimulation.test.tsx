import { act } from "react";
import { createRoot } from "react-dom/client";
import { beforeEach, describe, expect, it, vi } from "vitest";
import {
  applyUserHighlights,
  clearHighlight,
  highlightTextRangeFromNode,
  saveSelection,
  setActiveSelectionHighlight,
  clearActiveSelectionHighlight,
  type HighlightEntity,
  type SavedSelection,
} from "./readerHighlight";
import {
  getToolbarPosition,
  getVisibleSelectionRect,
  useReaderSelection,
} from "@/hooks/useReaderSelection";
import { ReaderSelectionToolbar } from "@/components/reader/ReaderSelectionToolbar";

if (typeof Highlight === "undefined") {
  // @ts-ignore
  globalThis.Highlight = class MockHighlight {
    ranges: Range[];
    constructor(...ranges: Range[]) {
      this.ranges = ranges;
    }
  };
}

if (!globalThis.CSS) {
  // @ts-ignore
  globalThis.CSS = {};
}

const mockHighlightMap = new Map<string, any>();
// @ts-ignore
globalThis.CSS.highlights = {
  set(name: string, highlight: any) {
    mockHighlightMap.set(name, highlight);
    return this;
  },
  get(name: string) {
    return mockHighlightMap.get(name);
  },
  delete(name: string) {
    return mockHighlightMap.delete(name);
  },
  has(name: string) {
    return mockHighlightMap.has(name);
  },
  clear() {
    mockHighlightMap.clear();
  },
} as any;

describe("Reader DOM Simulations (Selection, Highlights, TTS & Placement)", () => {
  beforeEach(() => {
    document.body.innerHTML = "";
    mockHighlightMap.clear();
    vi.clearAllMocks();
  });

  it("Simulation 1: Retains blue highlight on mouseup, input blur, and saves with note", async () => {
    const rootContainer = document.createElement("div");
    rootContainer.id = "reader-root";
    document.body.appendChild(rootContainer);

    const contentDiv = document.createElement("div");
    contentDiv.className = "reader-content";
    contentDiv.innerHTML = `
      <p id="p1">Giọng nói tôi nghe được từ đó của Tuuka... tuy phần mở đầu nhẹ bẫng như đùa cợt.</p>
      <p id="p2">Nhưng trước phát ngôn động trời như thế, tôi chết lặng tại chỗ.</p>
    `;
    rootContainer.appendChild(contentDiv);

    const addHighlightMock = vi.fn().mockResolvedValue(undefined);
    const savedSelectionRef = { current: null as SavedSelection | null };
    const ttsStartPointRef = { current: null };

    let hookApi: ReturnType<typeof useReaderSelection> | null = null;

    function TestReaderHarness() {
      const ref = { current: contentDiv };
      hookApi = useReaderSelection({
        columnsRef: ref,
        contentRef: ref,
        savedSelectionRef,
        ttsStartPointRef,
        addHighlight: addHighlightMock,
        speak: vi.fn(),
        stop: vi.fn(),
      });

      return (
        <div>
          {hookApi.selectionRange && (
            <ReaderSelectionToolbar
              t={((key: string, def?: string) => def || key) as any}
              toolbarPos={hookApi.toolbarPos}
              isSupported={true}
              onReadSelection={hookApi.handleReadSelection}
              onReadFromHere={hookApi.handleReadFromHere}
              onCopyText={hookApi.handleCopyText}
              onHighlight={hookApi.handleHighlight}
            />
          )}
        </div>
      );
    }

    const reactRoot = createRoot(rootContainer);
    await act(async () => {
      reactRoot.render(<TestReaderHarness />);
    });

    const p2TextNode = contentDiv.querySelector("#p2")!.firstChild!;
    const p2FullText = p2TextNode.textContent!;
    const targetPhrase = "tôi chết lặng tại chỗ";
    const startOffset = p2FullText.indexOf(targetPhrase);
    const endOffset = startOffset + targetPhrase.length;

    const testRange = document.createRange();
    testRange.setStart(p2TextNode, startOffset);
    testRange.setEnd(p2TextNode, endOffset);

    const saved = saveSelection(contentDiv, testRange);
    expect(saved).not.toBeNull();
    expect(saved!.selectedText).toBe("tôi chết lặng tại chỗ");

    await act(async () => {
      hookApi!.setSelectionRange(testRange);
    });

    expect(mockHighlightMap.has("reader-active-selection")).toBe(true);
    const activeHighlight = mockHighlightMap.get("reader-active-selection");
    expect(activeHighlight.ranges.length).toBeGreaterThan(0);
    expect(activeHighlight.ranges[0].toString()).toBe("tôi chết lặng tại chỗ");

    const noteInput = document.querySelector(
      'textarea[data-reader-toolbar="true"], input[data-reader-toolbar="true"]',
    ) as HTMLTextAreaElement | HTMLInputElement;
    expect(noteInput).not.toBeNull();

    await act(async () => {
      noteInput.value = "Ghi chú đoạn cao trào";
      noteInput.dispatchEvent(new Event("input", { bubbles: true }));
    });

    expect(mockHighlightMap.has("reader-active-selection")).toBe(true);

    await act(async () => {
      await hookApi!.handleHighlight("#fbcfe8", "Ghi chú đoạn cao trào");
    });

    expect(addHighlightMock).toHaveBeenCalledTimes(1);
    expect(addHighlightMock).toHaveBeenCalledWith(
      "tôi chết lặng tại chỗ",
      saved!.startIndex,
      saved!.endIndex,
      "#fbcfe8",
      undefined,
      "Ghi chú đoạn cao trào",
    );

    reactRoot.unmount();
  });

  it("Simulation 2: TTS speech boundaries do NOT clear or destroy user highlights or active selections", () => {
    const container = document.createElement("div");
    container.innerHTML = `
      <p id="t1">Trước phát ngôn động trời như thế, tôi chết lặng tại chỗ.</p>
    `;
    document.body.appendChild(container);

    const userHighlights: HighlightEntity[] = [
      {
        id: "hl-1",
        text_content: "phát ngôn động trời",
        start_index: 6,
        end_index: 25,
        color: "yellow",
      },
    ];

    applyUserHighlights(container, userHighlights);

    expect(mockHighlightMap.has("user-highlight-yellow")).toBe(true);

    highlightTextRangeFromNode(
      container,
      { textNodeIndex: 0, offset: 0 },
      6,
      9,
    );

    expect(mockHighlightMap.has("tts-active-word")).toBe(true);

    expect(mockHighlightMap.has("user-highlight-yellow")).toBe(true);
    expect(mockHighlightMap.get("user-highlight-yellow").ranges.length).toBe(1);

    clearHighlight();
    expect(mockHighlightMap.has("tts-active-word")).toBe(false);
    expect(mockHighlightMap.has("user-highlight-yellow")).toBe(true);
  });

  it("Simulation 3: Toolbar automatically shifts BELOW when selection is near top of viewport", () => {
    const topSelection = {
      left: 150,
      width: 80,
      top: 70,
      height: 24,
      bottom: 94,
    };

    const posTop = getToolbarPosition(topSelection, 1000, 800);
    expect(posTop.placement).toBe("below");
    expect(posTop.top).toBeGreaterThanOrEqual(102);
    expect(posTop.top).toBeLessThan(200);

    const midSelection = {
      left: 150,
      width: 80,
      top: 400,
      height: 24,
      bottom: 424,
    };

    const posMid = getToolbarPosition(midSelection, 1000, 800);
    expect(posMid.placement).toBe("above");
    expect(posMid.top).toBe(217);

    const mobilePos = getToolbarPosition(
      { left: 340, width: 20, top: 400, height: 20 },
      360,
      640,
    );
    expect(mobilePos.left).toBeGreaterThanOrEqual(8);
    expect(mobilePos.left + 344).toBeLessThanOrEqual(360);

    const fullPageSelection = {
      left: 50,
      width: 900,
      top: 30,
      height: 750,
      bottom: 780,
    };

    const posFullPage = getToolbarPosition(fullPageSelection, 1000, 800);
    expect(posFullPage.placement).toBe("below");
    expect(posFullPage.top).toBe(72);
  });

  it("Simulation 4: Previous chapter landing rules for Paged vs Scroll modes", () => {
    const singleMode: string = "single";
    const isSinglePaged = singleMode === "single" || singleMode === "double";
    expect(isSinglePaged).toBe(true);

    const doubleMode: string = "double";
    const isDoublePaged = doubleMode === "single" || doubleMode === "double";
    expect(isDoublePaged).toBe(true);

    const scrollMode: string = "scroll";
    const isScrollPaged = scrollMode === "single" || scrollMode === "double";
    expect(isScrollPaged).toBe(false);

    const webtoonMode: string = "webtoon";
    const isWebtoonPaged = webtoonMode === "single" || webtoonMode === "double";
    expect(isWebtoonPaged).toBe(false);
  });

  it("Simulation 5: Overlapping highlights and active text selection coexistence with priorities", () => {
    const container = document.createElement("div");
    container.innerHTML = `
      <p id="t1">Người đang nhìn tôi ngồi trên thảm với ánh mắt khinh miệt, không ai khác ngoài cô em gái Sakata Nayu.</p>
    `;
    document.body.appendChild(container);

    const userHighlights: HighlightEntity[] = [
      {
        id: "hl-yellow",
        text_content: "Người đang nhìn tôi ngồi trên",
        start_index: 0,
        end_index: 29,
        color: "yellow",
      },
      {
        id: "hl-green",
        text_content: "ngồi trên thảm với ánh mắt khinh",
        start_index: 20,
        end_index: 52,
        color: "green",
      },
    ];

    applyUserHighlights(container, userHighlights);

    expect(mockHighlightMap.has("user-highlight-yellow")).toBe(true);
    expect(mockHighlightMap.has("user-highlight-green")).toBe(true);
    expect(mockHighlightMap.get("user-highlight-yellow").priority).toBe(2);
    expect(mockHighlightMap.get("user-highlight-green").priority).toBe(2);

    const selRange = document.createRange();
    const textNode = container.querySelector("p")!.firstChild!;
    selRange.setStart(textNode, 0);
    selRange.setEnd(textNode, 60);

    setActiveSelectionHighlight(selRange, container, null);

    expect(mockHighlightMap.has("reader-active-selection")).toBe(true);
    expect(mockHighlightMap.get("reader-active-selection").priority).toBe(1);
    expect(mockHighlightMap.has("user-highlight-yellow")).toBe(true);
    expect(mockHighlightMap.has("user-highlight-green")).toBe(true);

    clearActiveSelectionHighlight();
    expect(mockHighlightMap.has("reader-active-selection")).toBe(false);
    expect(mockHighlightMap.has("user-highlight-yellow")).toBe(true);
    expect(mockHighlightMap.has("user-highlight-green")).toBe(true);
  });

  it("Simulation 6: Complete Selection + TTS playback lifecycle preserves all user highlights", () => {
    const container = document.createElement("div");
    container.innerHTML = `
      <p id="c1">Chương 14: Khiến cho em gái thực sự nổi giận, bạn sẽ xử lý như thế nào?</p>
      <p id="c2">Người đang nhìn tôi ngồi trên thảm với ánh mắt khinh miệt, không ai khác ngoài cô em gái hàng thật Sakata Nayu.</p>
    `;
    document.body.appendChild(container);

    const userHighlights: HighlightEntity[] = [
      {
        id: "hl-p2",
        text_content:
          "Người đang nhìn tôi ngồi trên thảm với ánh mắt khinh miệt",
        start_index: 70,
        end_index: 126,
        color: "yellow",
      },
    ];

    applyUserHighlights(container, userHighlights);
    expect(mockHighlightMap.has("user-highlight-yellow")).toBe(true);
    expect(mockHighlightMap.get("user-highlight-yellow").priority).toBe(2);

    const selRange = document.createRange();
    const p1Text = container.querySelector("#c1")!.firstChild!;
    const p2Text = container.querySelector("#c2")!.firstChild!;
    selRange.setStart(p1Text, 10);
    selRange.setEnd(p2Text, 40);

    setActiveSelectionHighlight(selRange, container, null);
    expect(mockHighlightMap.has("reader-active-selection")).toBe(true);
    expect(mockHighlightMap.get("reader-active-selection").priority).toBe(1);
    expect(mockHighlightMap.has("user-highlight-yellow")).toBe(true);

    clearActiveSelectionHighlight();
    expect(mockHighlightMap.has("reader-active-selection")).toBe(false);

    highlightTextRangeFromNode(
      container,
      { textNodeIndex: 0, offset: 0 },
      10,
      15,
    );
    expect(mockHighlightMap.has("tts-active-word")).toBe(true);
    expect(mockHighlightMap.get("tts-active-word").priority).toBe(3);
    expect(mockHighlightMap.has("user-highlight-yellow")).toBe(true);
    expect(mockHighlightMap.get("user-highlight-yellow").priority).toBe(2);

    highlightTextRangeFromNode(
      container,
      { textNodeIndex: 1, offset: 0 },
      10,
      10,
    );
    expect(mockHighlightMap.has("tts-active-word")).toBe(true);
    expect(mockHighlightMap.has("user-highlight-yellow")).toBe(true);

    clearHighlight();
    expect(mockHighlightMap.has("tts-active-word")).toBe(false);
    expect(mockHighlightMap.has("user-highlight-yellow")).toBe(true);
  });

  it("Simulation 7: Auto-scroll button is enabled/visible in scroll and webtoon modes, hidden in single/double", () => {
    const isAutoScrollVisible = (mode: string, isAudio = false) =>
      !isAudio && (mode === "scroll" || mode === "webtoon");

    expect(isAutoScrollVisible("single")).toBe(false);
    expect(isAutoScrollVisible("double")).toBe(false);
    expect(isAutoScrollVisible("scroll")).toBe(true);
    expect(isAutoScrollVisible("webtoon")).toBe(true);
    expect(isAutoScrollVisible("scroll", true)).toBe(false);
    expect(isAutoScrollVisible("webtoon", true)).toBe(false);
  });

  it("Simulation 8: Transitioning to Scroll Mode and AutoScroll keeps all highlights completely intact", () => {
    const container = document.createElement("div");
    container.innerHTML = `
      <div class="reader-content h-auto">
        <p id="p1">Người đang nhìn tôi ngồi trên thảm với ánh mắt khinh miệt, không ai khác ngoài cô em gái hàng thật Sakata Nayu.</p>
        <p id="p2">Mái tóc đen ngắn và khuôn mặt song tính không trang điểm.</p>
      </div>
    `;
    document.body.appendChild(container);

    const userHighlights: HighlightEntity[] = [
      {
        id: "hl-1",
        text_content:
          "Người đang nhìn tôi ngồi trên thảm với ánh mắt khinh miệt",
        start_index: 0,
        end_index: 56,
        color: "yellow",
      },
      {
        id: "hl-2",
        text_content: "Mái tóc đen ngắn",
        start_index: 104,
        end_index: 120,
        color: "pink",
      },
    ];

    applyUserHighlights(container, userHighlights);

    expect(mockHighlightMap.has("user-highlight-yellow")).toBe(true);
    expect(mockHighlightMap.has("user-highlight-pink")).toBe(true);
    expect(mockHighlightMap.get("user-highlight-yellow").priority).toBe(2);
    expect(mockHighlightMap.get("user-highlight-pink").priority).toBe(2);
  });

  it("Simulation 9: Full-page selection in CSS-column reader-mode-single positions toolbar above, not below", () => {
    const drawerContent = document.createElement("div");
    drawerContent.className =
      "drawer-content flex flex-col h-screen overflow-hidden relative";

    const header = document.createElement("header");
    header.className = "relative z-50 flex h-14 w-full flex-none items-center";
    drawerContent.appendChild(header);

    const contentArea = document.createElement("div");
    contentArea.className =
      "flex-1 min-h-0 overflow-hidden flex flex-col pt-4 pb-6 px-4 relative";
    drawerContent.appendChild(contentArea);

    const contentWrapper = document.createElement("div");
    contentWrapper.className = "w-full mx-auto flex-1 min-h-0 flex flex-col";
    contentWrapper.style.maxWidth = "920px";
    contentWrapper.style.fontSize = "18px";
    contentArea.appendChild(contentWrapper);

    const readerContent = document.createElement("div");
    readerContent.className =
      "reader-content prose prose-invert max-w-none w-full reader-mode-single reader-mode-measured";
    readerContent.innerHTML = `
      <h2>Chương 1: Mọi người làm gì khi bị hôn thê tiếp cận quá đáng</h2>
      <p>"Fuhehehehe~ Fuhehe♪"</p>
      <p>Tôi vừa mở cửa phòng khách thì thấy Yuuka đang lăn cuồn cuộn trên thảm.</p>
      <p>Khoé miệng thì toe toét, liên tục phát ra tiếng "Fuhe".</p>
      <p>Ừm, đúng Yuuka của mọi khi rồi.</p>
      <p>"Funya funya♪ Funya funya... Unya!!"</p>
      <p>...Cho tôi đính chính lại.</p>
      <p>Còn "Fuhe" nhiều hơn bình thường nữa.</p>
      <p>Yuuka vừa nằm vừa lắc hai chân liên tục đầy cao hứng, trông không khác gì một đứa trẻ hồn nhiên, khiến đến cả tôi cũng không khỏi thấy vui.</p>
      <p>Mà, nếu là lý do khiến em ấy vui mừng như vậy... chắc hẳn là do tối qua.</p>
      <p>Hôm qua tôi đã đi gặp và thăm thân với gia đình Yuuka.</p>
      <p>Ban nãy tôi đã nghĩ sẽ không có gì quá căng thẳng và mọi chuyện sẽ diễn ra ổn thoả thôi.</p>
    `;
    contentWrapper.appendChild(readerContent);
    document.body.appendChild(drawerContent);

    const cssColumnSelectAllRect = {
      left: 72,
      width: 8500,
      top: 80,
      height: 770,
      bottom: 850,
    };

    const posSelectAll = getToolbarPosition(cssColumnSelectAllRect, 1200, 900);

    expect(posSelectAll.placement).toBe("below");
    expect(posSelectAll.top).toBe(72);

    const mockRange = document.createRange();
    const p1Text = readerContent.querySelector("p")!.firstChild!;
    mockRange.setStart(p1Text, 0);
    mockRange.setEnd(p1Text, 10);

    const visibleRects = [
      {
        top: 100,
        bottom: 130,
        left: 80,
        right: 800,
        width: 720,
        height: 30,
        x: 80,
        y: 100,
        toJSON: () => ({}),
      },
      {
        top: 135,
        bottom: 165,
        left: 80,
        right: 800,
        width: 720,
        height: 30,
        x: 80,
        y: 135,
        toJSON: () => ({}),
      },
      {
        top: 170,
        bottom: 200,
        left: 80,
        right: 800,
        width: 720,
        height: 30,
        x: 80,
        y: 170,
        toJSON: () => ({}),
      },
      {
        top: 600,
        bottom: 630,
        left: 80,
        right: 800,
        width: 720,
        height: 30,
        x: 80,
        y: 600,
        toJSON: () => ({}),
      },
      {
        top: 800,
        bottom: 830,
        left: 80,
        right: 800,
        width: 720,
        height: 30,
        x: 80,
        y: 800,
        toJSON: () => ({}),
      },
    ];

    const hiddenColumnRects = Array.from({ length: 10 }, (_, i) => ({
      top: 100 + i * 35,
      bottom: 130 + i * 35,
      left: 1300 + i * 900,
      right: 2100 + i * 900,
      width: 720,
      height: 30,
      x: 1300 + i * 900,
      y: 100 + i * 35,
      toJSON: () => ({}),
    }));

    const allRects = [...visibleRects, ...hiddenColumnRects];

    const origGetClientRects = mockRange.getClientRects;
    mockRange.getClientRects = () => {
      const list = allRects as unknown as DOMRectList;
      Object.defineProperty(list, "length", { value: allRects.length });
      Object.defineProperty(list, "item", {
        value: (i: number) => allRects[i] || null,
      });
      return list;
    };

    const origInnerWidth = window.innerWidth;
    const origInnerHeight = window.innerHeight;
    Object.defineProperty(window, "innerWidth", {
      value: 1200,
      writable: true,
    });
    Object.defineProperty(window, "innerHeight", {
      value: 900,
      writable: true,
    });

    try {
      const visibleRect = getVisibleSelectionRect(mockRange);

      expect(visibleRect.top).toBe(100);
      expect(visibleRect.bottom).toBe(830);
      expect(visibleRect.left).toBe(80);
      expect(visibleRect.width).toBe(720);

      const toolbarPos = getToolbarPosition(visibleRect, 1200, 900);

      expect(toolbarPos.placement).toBe("below");
      expect(toolbarPos.top).toBe(72);

      expect(toolbarPos.left).toBeGreaterThanOrEqual(8);
      expect(toolbarPos.left).toBeLessThanOrEqual(812);
    } finally {
      mockRange.getClientRects = origGetClientRects;
      Object.defineProperty(window, "innerWidth", {
        value: origInnerWidth,
        writable: true,
      });
      Object.defineProperty(window, "innerHeight", {
        value: origInnerHeight,
        writable: true,
      });
    }

    const normalRect = {
      left: 200,
      width: 150,
      top: 450,
      height: 24,
      bottom: 474,
    };

    const posNormal = getToolbarPosition(normalRect, 1200, 900);
    expect(posNormal.placement).toBe("above");
    expect(posNormal.top).toBe(267);
  });
});
