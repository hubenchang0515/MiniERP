import "@testing-library/jest-dom/vitest";
import { cleanup } from "@testing-library/react";
import { afterEach, vi } from "vitest";
// Node 26 exposes its own optional localStorage; always use browser storage in tests.
const values = new Map<string, string>();
vi.stubGlobal("localStorage", {
  getItem: (key: string) => values.get(key) ?? null,
  setItem: (key: string, value: string) => {
    values.set(key, String(value));
  },
  removeItem: (key: string) => {
    values.delete(key);
  },
  clear: () => values.clear(),
  key: (index: number) => [...values.keys()][index] ?? null,
  get length() {
    return values.size;
  },
} satisfies Storage);
afterEach(() => {
  cleanup();
  localStorage.clear();
});
