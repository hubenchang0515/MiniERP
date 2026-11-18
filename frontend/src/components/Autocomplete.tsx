import {
  useEffect,
  useId,
  useState,
  type KeyboardEvent,
  type ReactNode,
} from "react";
import { FormField, Input } from "./FormControls";
import Button from "./Button";

type OptionPage<T> = {
  items: T[];
  hasMore: boolean;
};

type AutocompleteProps<T> = {
  label: string;
  placeholder?: string;
  required?: boolean;
  disabled?: boolean;
  maxLength?: number;
  value: T | null;
  onChange: (value: T | null) => void;
  createOption?: (text: string) => T;
  loadOptions?: (query: string, page: number) => Promise<OptionPage<T>>;
  getOptionKey: (option: T) => string | number;
  getOptionLabel: (option: T) => string;
  renderOption?: (option: T) => ReactNode;
  emptyMessage?: string;
  loadErrorMessage?: string;
};

export default function Autocomplete<T>({
  label,
  placeholder,
  required,
  maxLength,
  value,
  onChange,
  disabled,
  createOption,
  loadOptions,
  getOptionKey,
  getOptionLabel,
  renderOption,
  emptyMessage = "没有匹配的选项",
  loadErrorMessage = "无法加载选项",
}: AutocompleteProps<T>) {
  const id = useId();
  const listId = `${id}-options`;
  const [text, setText] = useState("");
  const [open, setOpen] = useState(false);
  const [composing, setComposing] = useState(false);
  const [page, setPage] = useState(1);
  const [suggestions, setSuggestions] = useState<
    (OptionPage<T> & { page: number }) | null
  >(null);
  const [activeIndex, setActiveIndex] = useState(-1);
  const [error, setError] = useState("");
  const [retry, setRetry] = useState(0);
  const query = value === null || createOption ? text.trim() : "";
  const selectedKey = value === null ? null : getOptionKey(value);
  const expanded = open && !disabled && Boolean(loadOptions);
  const items = suggestions?.items ?? [];
  const loading =
    expanded && !composing && !error && suggestions?.page !== page;

  useEffect(() => {
    if (!expanded || composing || !loadOptions) return;
    let current = true;
    const timer = window.setTimeout(
      () => {
        loadOptions(query, page)
          .then((result) => {
            if (!current) return;
            setSuggestions((previous) => ({
              items:
                page === 1
                  ? result.items
                  : [...(previous?.items ?? []), ...result.items],
              hasMore: result.hasMore,
              page,
            }));
          })
          .catch((e: unknown) => {
            if (current) {
              setError(
                e instanceof Error && e.message ? e.message : loadErrorMessage,
              );
            }
          });
      },
      page === 1 && query ? 200 : 0,
    );
    return () => {
      current = false;
      window.clearTimeout(timer);
    };
  }, [composing, expanded, page, query, retry, loadOptions, loadErrorMessage]);

  function resetSuggestions() {
    setPage(1);
    setSuggestions(null);
    setActiveIndex(-1);
    setError("");
  }

  function close() {
    setOpen(false);
    if (createOption) setText("");
    resetSuggestions();
  }

  function choose(option: T) {
    setText("");
    onChange(option);
    close();
  }

  function edit(next: string) {
    if (next.trim() !== query) resetSuggestions();
    setText(next);
    if (createOption) onChange(createOption(next));
    else if (value !== null) onChange(null);
    setOpen(true);
  }

  function handleKeyDown(event: KeyboardEvent<HTMLInputElement>) {
    if (composing || event.nativeEvent.isComposing) {
      if (event.key === "Escape") event.stopPropagation();
      return;
    }

    if (
      loadOptions &&
      (event.key === "ArrowDown" || (expanded && event.key === "ArrowUp"))
    ) {
      event.preventDefault();
      setOpen(true);
      if (items.length) {
        const index =
          event.key === "ArrowDown"
            ? Math.min(activeIndex + 1, items.length - 1)
            : activeIndex < 0
              ? items.length - 1
              : Math.max(activeIndex - 1, 0);
        setActiveIndex(index);
        document
          .getElementById(`${listId}-${index}`)
          ?.scrollIntoView?.({ block: "nearest" });
      }
    } else if (expanded && event.key === "Enter") {
      if (activeIndex >= 0 && activeIndex < items.length) {
        event.preventDefault();
        choose(items[activeIndex]);
      } else if (!createOption) {
        event.preventDefault();
      }
    }
  }

  return (
    <div
      className="relative"
      onBlur={(event) => {
        if (!event.currentTarget.contains(event.relatedTarget)) close();
      }}
      onKeyDown={(event) => {
        if (expanded && event.key === "Escape") {
          event.preventDefault();
          event.stopPropagation();
          if (event.target !== document.getElementById(id)) {
            document.getElementById(id)?.focus();
          }
          close();
        }
      }}
    >
      <FormField label={label} required={required}>
        <Input
          id={id}
          role="combobox"
          aria-autocomplete={loadOptions ? "list" : "none"}
          aria-expanded={expanded}
          aria-controls={expanded ? listId : undefined}
          aria-activedescendant={
            expanded && activeIndex >= 0 && activeIndex < items.length
              ? `${listId}-${activeIndex}`
              : undefined
          }
          autoComplete="off"
          placeholder={placeholder}
          required={required}
          maxLength={maxLength}
          disabled={disabled}
          value={value === null ? text : getOptionLabel(value)}
          onFocus={() => setOpen(true)}
          onClick={() => setOpen(true)}
          onChange={(event) => edit(event.target.value)}
          onCompositionStart={() => {
            resetSuggestions();
            setComposing(true);
          }}
          onCompositionEnd={(event) => {
            edit(event.currentTarget.value);
            setComposing(false);
          }}
          onKeyDown={handleKeyDown}
        />
      </FormField>
      {expanded && (
        <div className="absolute inset-x-0 top-full z-10 mt-1 overflow-hidden rounded-md border border-line bg-surface shadow-lg">
          <ul
            id={listId}
            role="listbox"
            aria-label={`${label}选项`}
            className="max-h-48 overflow-y-auto"
          >
            {!composing &&
              items.map((option, index) => (
                <li
                  key={getOptionKey(option)}
                  id={`${listId}-${index}`}
                  role="option"
                  aria-selected={
                    index === activeIndex ||
                    (activeIndex < 0 && getOptionKey(option) === selectedKey)
                  }
                  className={`cursor-pointer wrap-break-word px-3 py-2 text-[13px] hover:bg-subtle ${index === activeIndex ? "bg-subtle" : ""}`}
                  onMouseDown={(event) => event.preventDefault()}
                  onClick={() => choose(option)}
                >
                  {renderOption ? renderOption(option) : getOptionLabel(option)}
                </li>
              ))}
          </ul>
          {!composing && (loading || error || !items.length) && (
            <p role="status" className="px-3 py-2 text-[12px] text-muted">
              {error || (loading ? "正在加载…" : emptyMessage)}
            </p>
          )}
          {!composing && (error || suggestions?.hasMore) && (
            <div className="border-t border-line px-3 py-1">
              <Button
                variant="text"
                disabled={loading}
                onClick={() => {
                  document.getElementById(id)?.focus();
                  if (error) {
                    setError("");
                    setRetry((attempt) => attempt + 1);
                  } else {
                    setPage((currentPage) => currentPage + 1);
                  }
                }}
              >
                {error ? "重新加载" : "加载更多"}
              </Button>
            </div>
          )}
        </div>
      )}
    </div>
  );
}
