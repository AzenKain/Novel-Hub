import { act } from "react";
import { createRoot } from "react-dom/client";
import { describe, expect, it } from "vitest";
import { PasswordStrength } from "./PasswordStrength";

describe("PasswordStrength", () => {
  it("renders null when password is empty", () => {
    const container = document.createElement("div");
    const root = createRoot(container);
    act(() => {
      root.render(<PasswordStrength password="" />);
    });
    expect(container.children.length).toBe(0);
  });

  it("renders weak strength for short simple password", () => {
    const container = document.createElement("div");
    const root = createRoot(container);
    act(() => {
      root.render(<PasswordStrength password="abc" />);
    });
    expect(container.textContent).toContain("auth.password_strength");
    expect(container.textContent).toContain("auth.strength_weak");
    expect(container.querySelectorAll(".grid-cols-5 div").length).toBe(5);
  });

  it("renders strong strength when all requirements are met", () => {
    const container = document.createElement("div");
    const root = createRoot(container);
    act(() => {
      root.render(<PasswordStrength password="P@ssw0rd123" />);
    });
    expect(container.textContent).toContain("auth.strength_strong");
    const successTexts = container.querySelectorAll(".text-success");
    expect(successTexts.length).toBeGreaterThanOrEqual(5);
  });
});
