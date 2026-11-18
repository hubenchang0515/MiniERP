import Autocomplete from "../../components/Autocomplete";
import { aiApi, type Resource } from "./api";

export type ResourceOption = Pick<Resource, "id" | "name" | "enabled">;

async function loadResourceOptions(query: string, page: number) {
  const result = await aiApi.resourceOptions(query, page);
  return {
    items: result.items,
    hasMore: result.page * result.pageSize < result.total,
  };
}

export default function ResourceChoice({
  value,
  onChange,
  disabled,
}: {
  value: ResourceOption | null;
  onChange: (value: ResourceOption | null) => void;
  disabled: boolean;
}) {
  return (
    <Autocomplete
      label="供应商"
      placeholder="输入供应商名称…"
      required
      disabled={disabled}
      value={value}
      onChange={onChange}
      loadOptions={loadResourceOptions}
      getOptionKey={(option) => option.id}
      getOptionLabel={(option) => option.name}
      renderOption={(option) => (
        <>
          {option.name}
          {!option.enabled && (
            <span className="ml-2 text-muted">（已停用）</span>
          )}
        </>
      )}
      emptyMessage="没有匹配的供应商"
      loadErrorMessage="无法加载供应商"
    />
  );
}
