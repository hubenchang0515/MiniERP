import { useEffect, useState } from "react";
import { aiApi, type Resource } from "./api";
import { Select } from "../../components/FormControls";
import SearchField from "../../components/SearchField";
import Pagination from "../../components/Pagination";
import { ErrorBox } from "../../components/Shared";

export type ResourceOption = Pick<Resource, "id" | "name" | "enabled">;

export default function ResourceChoice({
  value,
  onChange,
  disabled,
}: {
  value: ResourceOption | null;
  onChange: (value: ResourceOption) => void;
  disabled: boolean;
}) {
  const [text, setText] = useState("");
  const [query, setQuery] = useState("");
  const [page, setPage] = useState(1);
  const [revision, setRevision] = useState(0);
  const [items, setItems] = useState<ResourceOption[]>([]);
  const [total, setTotal] = useState(0);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");
  useEffect(() => {
    let active = true;
    setLoading(true);
    setError("");
    aiApi
      .resourceOptions(query, page)
      .then((r) => {
        if (active) {
          setItems(r.items);
          setTotal(r.total);
        }
      })
      .catch((e) => {
        if (active) setError(e.message);
      })
      .finally(() => {
        if (active) setLoading(false);
      });
    return () => {
      active = false;
    };
  }, [query, page, revision]);
  return (
    <div className="grid gap-2">
      <strong className="text-[13px] font-medium">供应商</strong>
      <SearchField
        label="搜索供应商"
        value={text}
        onChange={setText}
        disabled={disabled}
        onSearch={() => {
          setQuery(text.trim());
          setPage(1);
          setRevision((v) => v + 1);
        }}
      />
      <ErrorBox message={error} />
      <Select
        required
        aria-label="供应商"
        value={value?.id ?? ""}
        disabled={disabled || loading || !!error}
        onChange={(e) => {
          const choice = items.find((r) => r.id === Number(e.target.value));
          if (choice) onChange(choice);
        }}
      >
        <option value="">{loading ? "正在加载…" : "请选择供应商"}</option>
        {value && !items.some((r) => r.id === value.id) && (
          <option value={value.id}>
            {value.name}
            {!value.enabled ? "（已停用）" : ""}
          </option>
        )}
        {items.map((r) => (
          <option key={r.id} value={r.id}>
            {r.name}
            {!r.enabled ? "（已停用）" : ""}
          </option>
        ))}
      </Select>
      <Pagination
        page={page}
        total={total}
        disabled={disabled || loading || !!error}
        onChange={setPage}
      />
    </div>
  );
}
