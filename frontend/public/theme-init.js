// Apply the preference before the app and styles load to avoid a light flash.
(() => {
  let mode = "system";
  let accent = "green";
  try {
    const saved = localStorage.getItem("minioa.theme");
    if (saved === "light" || saved === "dark") mode = saved;
    const savedAccent = localStorage.getItem("minioa.accent");
    if (["green", "blue", "purple", "orange", "rose"].includes(savedAccent))
      accent = savedAccent;
  } catch {}
  const theme =
    mode === "system"
      ? window.matchMedia?.("(prefers-color-scheme: dark)").matches
        ? "dark"
        : "light"
      : mode;
  document.documentElement.dataset.theme = theme;
  document.documentElement.dataset.accent = accent;
  document.documentElement.style.colorScheme = theme;
  document
    .querySelector('meta[name="theme-color"]')
    ?.setAttribute("content", theme === "dark" ? "#101722" : "#f6f7f9");
})();
