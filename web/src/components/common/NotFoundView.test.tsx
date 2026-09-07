import { act } from "react";
import { createRoot } from "react-dom/client";
import { describe, expect, it, vi, beforeEach } from "vitest";
import { BrowserRouter } from "react-router-dom";
import { NotFoundView } from "./NotFoundView";
import { useSettingsStore } from "@/stores/settingsStore";

describe("NotFoundView", () => {
  beforeEach(() => {
    useSettingsStore.setState({
      publicSettings: {
        site: {
          title: "Custom NovelHub Test",
          logo: "/custom-test-logo.png",
          favicon: "/favicon.ico",
        },
      } as any,
    });
  });

  it("renders site logo and title from stores", () => {
    const container = document.createElement("div");
    const root = createRoot(container);
    act(() => {
      root.render(
        <BrowserRouter>
          <NotFoundView type="book" />
        </BrowserRouter>,
      );
    });

    const img = container.querySelector("img");
    expect(img).not.toBeNull();
    expect(img?.getAttribute("src")).toBe("/custom-test-logo.png");
    expect(container.textContent).toContain("Custom NovelHub Test");
  });

  it("renders 404 badge and action buttons", () => {
    const container = document.createElement("div");
    const root = createRoot(container);
    act(() => {
      root.render(
        <BrowserRouter>
          <NotFoundView type="book" />
        </BrowserRouter>,
      );
    });

    expect(container.textContent).toContain("404");
    const buttons = container.querySelectorAll("button");
    expect(buttons.length).toBeGreaterThanOrEqual(2);
  });

  it("renders custom title and description when passed", () => {
    const container = document.createElement("div");
    const root = createRoot(container);
    act(() => {
      root.render(
        <BrowserRouter>
          <NotFoundView
            type="page"
            title="Custom Missing Title"
            description="Custom Missing Description Details"
          />
        </BrowserRouter>,
      );
    });

    expect(container.textContent).toContain("Custom Missing Title");
    expect(container.textContent).toContain("Custom Missing Description Details");
  });

  it("invokes onGoBack callback when back button clicked", () => {
    const container = document.createElement("div");
    const root = createRoot(container);
    const onGoBack = vi.fn();
    act(() => {
      root.render(
        <BrowserRouter>
          <NotFoundView type="book" onGoBack={onGoBack} />
        </BrowserRouter>,
      );
    });

    const backButton = container.querySelector("button");
    expect(backButton).not.toBeNull();
    act(() => {
      backButton?.click();
    });
    expect(onGoBack).toHaveBeenCalledTimes(1);
  });
});
