import {
  createContext,
  useCallback,
  useContext,
  useEffect,
  useLayoutEffect,
  useMemo,
  useState,
  type ReactNode,
} from "react";

export type ThemeMode = "system" | "light" | "dark";
export const THEME_KEY = "minierp.theme";
export const ACCENT_KEY = "minierp.accent";
export const accentColors = [
  { value: "green", label: "绿色" },
  { value: "blue", label: "蓝色" },
  { value: "purple", label: "紫色" },
  { value: "orange", label: "橙色" },
  { value: "rose", label: "玫红色" },
] as const;
export type AccentColor = (typeof accentColors)[number]["value"];
const parseAccent = (value: string | null): AccentColor =>
  accentColors.find((color) => color.value === value)?.value ?? "green";
function readAccent(): AccentColor {
  try {
    return parseAccent(localStorage.getItem(ACCENT_KEY));
  } catch {
    return "green";
  }
}
const SYSTEM_THEME = "(prefers-color-scheme: dark)";
const parseMode = (value: string | null): ThemeMode =>
  value === "light" || value === "dark" ? value : "system";
function readMode(): ThemeMode {
  try {
    return parseMode(localStorage.getItem(THEME_KEY));
  } catch {
    return "system";
  }
}
const ThemeContext = createContext<{
  mode: ThemeMode;
  resolved: "light" | "dark";
  setMode: (mode: ThemeMode) => void;
  accent: AccentColor;
  setAccent: (accent: AccentColor) => void;
} | null>(null);

export function ThemeProvider({ children }: { children: ReactNode }) {
  const [mode, setModeState] = useState<ThemeMode>(readMode);
  const [accent, setAccentState] = useState<AccentColor>(readAccent);
  const [systemDark, setSystemDark] = useState(
    () => window.matchMedia?.(SYSTEM_THEME).matches ?? false,
  );

  useEffect(() => {
    const media = window.matchMedia?.(SYSTEM_THEME);
    const syncSystem = () => setSystemDark(media?.matches ?? false);
    const syncStorage = (event: StorageEvent) => {
      if (event.key === THEME_KEY || event.key === null)
        setModeState(parseMode(event.newValue));
      if (event.key === ACCENT_KEY || event.key === null)
        setAccentState(parseAccent(event.newValue));
    };
    syncSystem();
    media?.addEventListener("change", syncSystem);
    window.addEventListener("storage", syncStorage);
    return () => {
      media?.removeEventListener("change", syncSystem);
      window.removeEventListener("storage", syncStorage);
    };
  }, []);

  const resolved = mode === "system" ? (systemDark ? "dark" : "light") : mode;
  useLayoutEffect(() => {
    document.documentElement.dataset.theme = resolved;
    document.documentElement.style.colorScheme = resolved;
    document
      .querySelector('meta[name="theme-color"]')
      ?.setAttribute("content", resolved === "dark" ? "#101722" : "#f6f7f9");
  }, [resolved]);

  useLayoutEffect(() => {
    document.documentElement.dataset.accent = accent;
  }, [accent]);

  const setMode = useCallback((next: ThemeMode) => {
    setModeState(next);
    try {
      localStorage.setItem(THEME_KEY, next);
    } catch {
      /* The selected theme still works when browser storage is unavailable. */
    }
  }, []);
  const setAccent = useCallback((next: AccentColor) => {
    setAccentState(next);
    try {
      localStorage.setItem(ACCENT_KEY, next);
    } catch {
      /* Color selection remains available without browser storage. */
    }
  }, []);
  const value = useMemo(
    () => ({ mode, resolved, setMode, accent, setAccent }),
    [mode, resolved, setMode, accent, setAccent],
  );
  return (
    <ThemeContext.Provider value={value}>{children}</ThemeContext.Provider>
  );
}

export function useTheme() {
  const theme = useContext(ThemeContext);
  if (!theme) throw new Error("ThemeProvider is required");
  return theme;
}
