import { Monitor, Moon, Sun } from "lucide-react";
import { useTheme, type ThemeMode } from "../app/ThemeContext";
import IconButton from "./IconButton";

const modes = {
  system: { icon: Monitor, label: "跟随系统", next: "dark" },
  dark: { icon: Moon, label: "深色模式", next: "light" },
  light: { icon: Sun, label: "浅色模式", next: "system" },
} satisfies Record<
  ThemeMode,
  { icon: typeof Sun; label: string; next: ThemeMode }
>;

export default function ThemeSelect() {
  const { mode, setMode } = useTheme();
  const { icon: Icon, label, next } = modes[mode];
  return (
    <IconButton
      icon={Icon}
      label={`当前主题：${label}，切换为${modes[next].label}`}
      onClick={() => setMode(next)}
    />
  );
}
