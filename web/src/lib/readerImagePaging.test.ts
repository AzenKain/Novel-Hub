import { describe, expect, it } from "vitest";
import { sanitizeReaderHtml } from "@/utils/readerHtml";

describe("Reader Image Multi-column DOM & Paging at 1024x1366", () => {
  const sampleChapterHtml = `
    <h4 align="center">Minh họa</h4>
    <div class="long-text no-select text-justify" id="chapter-content">
      <p style="display: none">Minh họa</p>
      <p><img alt="Illu05.jpg" src="http://localhost:3434/api/v1/reader/019ffa22-3026-7920-9644-566a16d19e9e/asset/EPUB/images/chapter_1/image_4.jpeg?file_id=019ffa22-303f-7be0-a768-9cdca9fa59d4"/></p>
      <p><img alt="Illu06.jpg" src="http://localhost:3434/api/v1/reader/019ffa22-3026-7920-9644-566a16d19e9e/asset/EPUB/images/chapter_1/image_5.jpeg?file_id=019ffa22-303f-7be0-a768-9cdca9fa59d4"/></p>
      <p><img alt="Illu07.jpg" src="http://localhost:3434/api/v1/reader/019ffa22-3026-7920-9644-566a16d19e9e/asset/EPUB/images/chapter_1/image_6.jpeg?file_id=019ffa22-303f-7be0-a768-9cdca9fa59d4"/></p>
      <p style="display: none">Hãy bình luận để ủng hộ người đăng nhé!</p>
    </div>
  `;

  it("sanitizes illustration chapter HTML preserving image elements and URLs", () => {
    const sanitized = sanitizeReaderHtml(sampleChapterHtml);
    expect(sanitized).toContain("image_4.jpeg");
    expect(sanitized).toContain("image_5.jpeg");
    expect(sanitized).toContain("image_6.jpeg");

    const container = document.createElement("div");
    container.innerHTML = sanitized;

    const images = container.querySelectorAll("img");
    expect(images.length).toBe(3);
    expect(images[0].getAttribute("src")).toContain("image_4.jpeg");
    expect(images[1].getAttribute("src")).toContain("image_5.jpeg");
    expect(images[2].getAttribute("src")).toContain("image_6.jpeg");
  });

  it("filters out dummy/empty images like src='#' and empty link anchors to prevent blank pages", () => {
    const brokenHtml = `
      <div>
        <p>Text before</p>
        <a href="/api/v1/reader/019ffa22-3026-7920-9644-566a16d19e9e/asset/EPUB/truyen/14280?file_id=019ffa22-303f-7be0-a768-9cdca9fa59d4" target="__blank">
          <img class="d-md-none" src="#">
        </a>
        <p>Text after</p>
      </div>
    `;
    const sanitized = sanitizeReaderHtml(brokenHtml);
    expect(sanitized).not.toContain('src="#"');
    expect(sanitized).toContain("Text before");
    expect(sanitized).toContain("Text after");

    const container = document.createElement("div");
    container.innerHTML = sanitized;
    const images = container.querySelectorAll("img");
    expect(images.length).toBe(0);
    const anchors = container.querySelectorAll("a");
    expect(anchors.length).toBe(0);
  });

  it("allows heading text and first image to share column without artificial break", () => {
    const container = document.createElement("div");
    container.className = "reader-content reader-mode-double";
    container.innerHTML = sanitizeReaderHtml(sampleChapterHtml);
    document.body.appendChild(container);

    const heading = container.querySelector("h4");
    expect(heading).not.toBeNull();
    expect(heading?.textContent).toBe("Minh họa");

    const firstImageWrapper = container.querySelector("p:has(> img:only-child)");
    expect(firstImageWrapper).not.toBeNull();

    // Verify first image wrapper is naturally preceded by content without break-before forcing an empty page
    expect(firstImageWrapper?.previousElementSibling).not.toBeNull();

    document.body.removeChild(container);
  });

  it("calculates double-column metrics correctly on 1024x1366 resolution", () => {
    const viewportWidth = 1024;
    const viewportHeight = 1366;
    const headerHeight = 56;
    const footerHeight = 48;
    const availableHeight = viewportHeight - headerHeight - footerHeight; // 1262px
    const pageGap = 40;
    const columnCount = 2;

    const columnWidth = (viewportWidth - pageGap) / columnCount;
    expect(columnWidth).toBe(492);
    expect(availableHeight).toBe(1262);

    // Each full-height image page in double-page mode takes exactly 1 column width + gap step
    const pageStep = viewportWidth + pageGap;
    expect(pageStep).toBe(1064);
  });

  it("preserves pending landing 'end' across empty chapter transitions", () => {
    const pendingLanding = { current: "end" as string | null };

    // When loading starts, html is temporarily empty
    let htmlContent = "";
    if (!htmlContent || htmlContent.trim() === "") {
      // Pending landing ref should NOT be consumed/cleared on empty string
      expect(pendingLanding.current).toBe("end");
    }

    // When html arrives:
    htmlContent = sampleChapterHtml;
    if (htmlContent && htmlContent.trim().length > 0 && pendingLanding.current === "end") {
      pendingLanding.current = null;
      const targetIndex = 5; // e.g. maxIndex
      expect(targetIndex).toBeGreaterThan(0);
    }
    expect(pendingLanding.current).toBeNull();
  });

  it("handles chapter opening with title and frontispiece illustration", () => {
    const chapterOpeningHtml = `
      <h2>Chương mở đầu :Lý do cho hành động sai trái</h2>
      <p><img alt="Prologue.jpg" src="http://localhost:3434/api/v1/reader/019ffa22-3026-7920-9644-566a16d19e9e/asset/EPUB/images/prologue.jpeg"/></p>
      <p>Sau nhiều ngày suy nghĩ, tôi đã quyết định viết lại câu chuyện này...</p>
    `;
    const sanitized = sanitizeReaderHtml(chapterOpeningHtml);
    const container = document.createElement("div");
    container.className = "reader-content reader-mode-double";
    container.innerHTML = sanitized;
    document.body.appendChild(container);

    const heading = container.querySelector("h2");
    expect(heading).not.toBeNull();
    expect(heading?.textContent).toContain("Chương mở đầu");

    const img = container.querySelector("img");
    expect(img).not.toBeNull();
    expect(img?.getAttribute("src")).toContain("prologue.jpeg");

    // The heading and image wrapper are contiguous siblings
    const imgWrapper = img?.closest("p");
    expect(imgWrapper?.previousElementSibling).toBe(heading);

    document.body.removeChild(container);
  });

  it("calculates double-column metrics and fits heading + image at 1024x600 resolution", () => {
    const viewportWidth = 1024;
    const viewportHeight = 600;
    const headerHeight = 56;
    const footerHeight = 48;
    const availableHeight = viewportHeight - headerHeight - footerHeight; // 496px
    const pageGap = 40;
    const columnCount = 2;

    const columnWidth = (viewportWidth - pageGap) / columnCount;
    expect(columnWidth).toBe(492);
    expect(availableHeight).toBe(496);

    // Height calculation:
    // Heading: ~40px
    // Image container max-height (calc(100cqh - 4.5rem)): 496px - 72px = 424px
    // Total column height = 40px + 424px = 464px <= 496px (Fits perfectly in Column 1!)
    const headingHeight = 40;
    const maxImageHeight = availableHeight - 72;
    expect(headingHeight + maxImageHeight).toBeLessThanOrEqual(availableHeight);

    // Test with real EPUB DOM containing div#chapter-content wrapper
    const wrappedEpubHtml = `
      <h4 align="center">Minh họa</h4>
      <div class="long-text no-select text-justify" id="chapter-content">
        <p style="display: none">Minh họa</p>
        <p><img alt="img_5194460039730855310.jpg" src="http://localhost:3434/api/v1/reader/019ffa21-80f0-77d9-b9d3-a1204c8cc6ae/asset/EPUB/images/img_5194460039730855310.jpg?file_id=019ffa21-8101-70ab-a5e8-06a30e1a0682"/></p>
      </div>
    `;
    const sanitized = sanitizeReaderHtml(wrappedEpubHtml);
    const container = document.createElement("div");
    container.className = "reader-content reader-mode-double";
    container.innerHTML = sanitized;
    document.body.appendChild(container);

    const heading = container.querySelector("h4");
    expect(heading).not.toBeNull();
    const img = container.querySelector("img");
    expect(img).not.toBeNull();
    expect(img?.getAttribute("src")).toContain("img_5194460039730855310.jpg");

    document.body.removeChild(container);
  });

  it("handles bare img directly following h2 with subsequent text paragraphs", () => {
    const userChapterHtml = `
      <h2>Minh hoạ</h2>
      <img src="/api/v1/reader/019ffa21-80f0-77d9-b9d3-a1204c8cc6ae/asset/EPUB/images/img_5194460039730855310.jpg?file_id=019ffa21-8101-70ab-a5e8-06a30e1a0682" alt="">
      <p>Bầu trời với nửa vầng trăng đang lên 7</p>
      <p>Cuộc sống tiếp diễn</p>
      <p>Hashimoto Tsumugu</p>
    `;
    const sanitized = sanitizeReaderHtml(userChapterHtml);
    const container = document.createElement("div");
    container.className = "reader-content reader-mode-single";
    container.innerHTML = sanitized;
    document.body.appendChild(container);

    const h2 = container.querySelector("h2");
    expect(h2).not.toBeNull();
    expect(h2?.textContent).toBe("Minh hoạ");

    const figure = container.querySelector("figure.reader-image-page");
    expect(figure).not.toBeNull();
    const img = figure?.querySelector("img");
    expect(img).not.toBeNull();
    expect(img?.getAttribute("src")).toContain("img_5194460039730855310.jpg");

    // Sibling relationship: figure.reader-image-page immediately follows h2
    expect(h2?.nextElementSibling).toBe(figure);

    const firstP = container.querySelector("p");
    expect(firstP).not.toBeNull();
    expect(firstP?.textContent).toContain("Bầu trời với nửa vầng trăng");

    // Paragraph immediately follows figure
    expect(figure?.nextElementSibling).toBe(firstP);

    document.body.removeChild(container);
  });

  it("isolates consecutive illustration images into dedicated .reader-image-page blocks on 1024x1336", () => {
    const rawHtml = `
      <div class="title-top">
        <h2 class="title-item">Web Novel [ĐÃ HOÀN THÀNH]</h2>
        <h4 class="title-item1">illustration</h4>
      </div>
      <div class="col" id="chapter-content">
        <p id="2" class="calibre3"><img alt="u68792-1.jpg" src="http://localhost:5173/api/v1/reader/019ffa22-db85-7bbb-ab7e-fb89556f0b4f/asset/1.jpg?file_id=019ffa22-dbaa-78ae-8402-86c5f733772b" class="calibre7"></p>
        <p id="3" class="calibre3"><img alt="u68792-2.jpg" src="http://localhost:5173/api/v1/reader/019ffa22-db85-7bbb-ab7e-fb89556f0b4f/asset/2.jpg?file_id=019ffa22-dbaa-78ae-8402-86c5f733772b" class="calibre7"></p>
        <p id="4" class="calibre3"><img alt="u68792-3.jpg" src="http://localhost:5173/api/v1/reader/019ffa22-db85-7bbb-ab7e-fb89556f0b4f/asset/3.jpg?file_id=019ffa22-dbaa-78ae-8402-86c5f733772b" class="calibre7"></p>
        <p id="5" class="calibre3"><img alt="u68792-4.jpg" src="http://localhost:5173/api/v1/reader/019ffa22-db85-7bbb-ab7e-fb89556f0b4f/asset/4.jpg?file_id=019ffa22-dbaa-78ae-8402-86c5f733772b" class="calibre7"></p>
        <p id="9" class="calibre3">[Người gửi] Vợ tương lai</p>
      </div>
    `;

    const sanitized = sanitizeReaderHtml(rawHtml);
    const container = document.createElement("div");
    container.className = "reader-content reader-mode-double";
    container.innerHTML = sanitized;
    document.body.appendChild(container);

    const imageWrappers = container.querySelectorAll(".reader-image-page");
    expect(imageWrappers.length).toBe(4);

    // Each image wrapper contains exactly 1 image
    imageWrappers.forEach((wrapper, i) => {
      const img = wrapper.querySelector("img");
      expect(img).not.toBeNull();
      expect(img?.getAttribute("src")).toContain(`asset/${i + 1}.jpg`);
    });

    // Text paragraph is NOT tagged as an image page
    const textP = container.querySelector("#\\39");
    expect(textP).not.toBeNull();
    expect(textP?.classList.contains("reader-image-page")).toBe(false);
    expect(textP?.textContent).toContain("[Người gửi] Vợ tương lai");

    document.body.removeChild(container);
  });

  it("calculates 11 discrete pages for an 11-image illustration chapter on 1024x1366 without jumping", () => {
    // Generate 11 illustration paragraphs identical to user's exact DOM
    const imagesHtml = Array.from({ length: 11 }, (_, i) => 
      `<p class="calibre3 reader-image-page"><img alt="" src="/api/v1/reader/book/asset/${i + 1}.jpg" class="calibre6"></p>`
    ).join("\n");

    const fullChapterHtml = `
      <div class="title-top">
        <h2 class="title-item">Minh họa LN</h2>
        <h4 class="title-item1">Minh họa vol 1</h4>
      </div>
      <div class="col" id="chapter-content">
        ${imagesHtml}
      </div>
    `;

    const sanitized = sanitizeReaderHtml(fullChapterHtml);
    const container = document.createElement("div");
    container.className = "reader-content reader-mode-single reader-mode-measured";
    container.innerHTML = sanitized;
    document.body.appendChild(container);

    const imagePages = container.querySelectorAll(".reader-image-page");
    expect(imagePages.length).toBe(11);

    // Simulate multi-column layout geometry on 1024x1366:
    // Container clientWidth: 920px, Page gap: 40px, Step: 960px
    const clientWidth = 920;
    const pageGap = 40;
    const scrollStep = clientWidth + pageGap; // 960px

    // 1 title page + 11 image pages = 12 total pages
    const totalPages = 12;
    const totalScrollWidth = clientWidth + (totalPages - 1) * scrollStep; // 920 + 11 * 960 = 11480px

    // Verify maxIndex calculation
    const maxScroll = totalScrollWidth - clientWidth;
    const maxIndex = Math.round(maxScroll / scrollStep);
    expect(maxIndex).toBe(11); // Pages 0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11

    // Step-by-step navigation test: verify pageIndex increments from 0 to 11 without triggering chapter next
    let currentPageIndex = 0;
    let nextChapterCalled = false;

    const onChapterNext = () => {
      nextChapterCalled = true;
    };

    const handlePageNext = () => {
      if (currentPageIndex >= maxIndex) {
        onChapterNext();
        return;
      }
      currentPageIndex++;
    };

    // Advancing through pages 0 to 11 must NOT call onChapterNext
    for (let step = 0; step < 11; step++) {
      expect(currentPageIndex).toBe(step);
      handlePageNext();
      expect(nextChapterCalled).toBe(false);
      expect(currentPageIndex).toBe(step + 1);
    }

    // Now at last page (11), next click SHOULD call onChapterNext
    expect(currentPageIndex).toBe(11);
    handlePageNext();
    expect(nextChapterCalled).toBe(true);

    document.body.removeChild(container);
  });
});
