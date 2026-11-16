import { Palette } from "lucide-react";
import { accentColors, useTheme } from "../app/ThemeContext";
import IconButton from "./IconButton";

export default function AccentSelect() {
  const { accent, setAccent } = useTheme();
  const index = accentColors.findIndex((color) => color.value === accent);
  const current = accentColors[index];
  const next = accentColors[(index + 1) % accentColors.length];
  const label = `当前主题色：${current.label}，切换为${next.label}`;
  return (
    <IconButton
      icon={Palette}
      label={label}
      onClick={() => setAccent(next.value)}
    />
  );
}
