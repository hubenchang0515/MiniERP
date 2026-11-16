import { act, fireEvent, render, screen } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { ThemeProvider, THEME_KEY } from "./ThemeContext";
import ThemeSelect from "../components/ThemeSelect";

const originalMatchMedia = window.matchMedia;
let dark = true;
let media: MediaQueryList;
let listeners: Set<() => void>;
const mount = () =>
  render(
    <ThemeProvider>
      <ThemeSelect />
    </ThemeProvider>,
  );
const changeSystem = (next: boolean) =>
  act(() => {
    dark = next;
    listeners.forEach((listener) => listener());
  });

describe("theme preference", () => {
  beforeEach(() => {
    dark = true;
    listeners = new Set();
    media = {
      get matches() {
        return dark;
      },
      addEventListener: (_type: string, listener: () => void) =>
        listeners.add(listener),
      removeEventListener: (_type: string, listener: () => void) =>
        listeners.delete(listener),
    } as unknown as MediaQueryList;
    window.matchMedia = vi.fn(() => media);
  });
  afterEach(() => {
    window.matchMedia = originalMatchMedia;
    document.documentElement.removeAttribute("data-theme");
    document.documentElement.style.colorScheme = "";
    vi.restoreAllMocks();
  });
  it("defaults to the system and follows changes immediately", () => {
    mount();
    expect(
      screen.getByRole("button", { name: /当前主题：跟随系统/ }),
    ).toBeInTheDocument();
    expect(document.documentElement).toHaveAttribute("data-theme", "dark");
    expect(localStorage.getItem(THEME_KEY)).toBeNull();
    changeSystem(false);
    expect(document.documentElement).toHaveAttribute("data-theme", "light");
    expect(document.documentElement.style.colorScheme).toBe("light");
  });
  it("cycles system, dark and light while remembering the selected mode", () => {
    const first = mount();
    const click = () => fireEvent.click(screen.getByRole("button"));
    expect(screen.getByRole("button").textContent).toBe("");
    click();
    expect(localStorage.getItem(THEME_KEY)).toBe("dark");
    changeSystem(false);
    expect(document.documentElement).toHaveAttribute("data-theme", "dark");
    first.unmount();
    mount();
    expect(
      screen.getByRole("button", { name: /当前主题：深色模式/ }),
    ).toBeInTheDocument();
    click();
    expect(document.documentElement).toHaveAttribute("data-theme", "light");
    expect(localStorage.getItem(THEME_KEY)).toBe("light");
    click();
    expect(localStorage.getItem(THEME_KEY)).toBe("system");
    changeSystem(true);
    expect(document.documentElement).toHaveAttribute("data-theme", "dark");
  });
  it("falls back to the system for invalid stored preferences", () => {
    localStorage.setItem(THEME_KEY, "unknown");
    mount();
    expect(
      screen.getByRole("button", { name: /当前主题：跟随系统/ }),
    ).toBeInTheDocument();
    expect(document.documentElement).toHaveAttribute("data-theme", "dark");
  });
  it("synchronizes choices from other tabs", () => {
    mount();
    act(() =>
      window.dispatchEvent(
        new StorageEvent("storage", { key: THEME_KEY, newValue: "light" }),
      ),
    );
    expect(
      screen.getByRole("button", { name: /当前主题：浅色模式/ }),
    ).toBeInTheDocument();
    expect(document.documentElement).toHaveAttribute("data-theme", "light");
    act(() =>
      window.dispatchEvent(
        new StorageEvent("storage", { key: THEME_KEY, newValue: null }),
      ),
    );
    expect(document.documentElement).toHaveAttribute("data-theme", "dark");
  });
  it("still switches themes when storage is unavailable", () => {
    vi.spyOn(localStorage, "getItem").mockImplementation(() => {
      throw new Error("blocked");
    });
    vi.spyOn(localStorage, "setItem").mockImplementation(() => {
      throw new Error("blocked");
    });
    mount();
    fireEvent.click(screen.getByRole("button", { name: /当前主题：跟随系统/ }));
    fireEvent.click(screen.getByRole("button", { name: /当前主题：深色模式/ }));
    expect(document.documentElement).toHaveAttribute("data-theme", "light");
  });
});
