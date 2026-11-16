import { useEffect, useState } from "react";
import { rolesApi, type RoleOption } from "./api";
import SearchField from "../../components/SearchField";
import Pagination from "../../components/Pagination";
import { CheckboxItem } from "../../components/FormControls";
import { ErrorBox, Loading, Empty } from "../../components/Shared";
import Button from "../../components/Button";
import Badge from "../../components/Badge";

export default function RolePicker({
  value,
  onChange,
  initialOptions = [],
  disabled = false,
}: {
  value: number[];
  onChange: (ids: number[]) => void;
  initialOptions?: RoleOption[];
  disabled?: boolean;
}) {
  const [text, setText] = useState("");
  const [query, setQuery] = useState("");
  const [page, setPage] = useState(1);
  const [items, setItems] = useState<RoleOption[]>([]);
  const [known, setKnown] = useState<Record<number, RoleOption>>({});
  const [total, setTotal] = useState(0);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");
  const [retry, setRetry] = useState(0);
  useEffect(() => {
    let active = true;
    setLoading(true);
    setError("");
    rolesApi
      .options(query, page)
      .then((result) => {
        if (!active) return;
        setItems(result.items);
        setTotal(result.total);
        setKnown((previous) => ({
          ...previous,
          ...Object.fromEntries(result.items.map((role) => [role.id, role])),
        }));
      })
      .catch((err) => {
        if (active) setError(err.message);
      })
      .finally(() => {
        if (active) setLoading(false);
      });
    return () => {
      active = false;
    };
  }, [query, page, retry]);
  const names = new Map(
    [...initialOptions, ...Object.values(known)].map((role) => [
      role.id,
      role.name,
    ]),
  );
  return (
    <div className="mt-4">
      <SearchField
        label="搜索可分配角色"
        placeholder="搜索角色名称或说明…"
        value={text}
        disabled={disabled}
        onChange={setText}
        onSearch={() => {
          setPage(1);
          setQuery(text.trim());
          setRetry((v) => v + 1);
        }}
      />
      {value.length > 0 && (
        <div className="mt-3 flex flex-wrap gap-2" aria-label="已选角色">
          {value.map((id) => (
            <Badge
              key={id}
              disabled={disabled}
              removeLabel={`移除角色${names.get(id) ?? id}`}
              onRemove={() =>
                onChange(value.filter((selected) => selected !== id))
              }
            >
              {names.get(id) ?? `角色 #${id}`}
            </Badge>
          ))}
        </div>
      )}
      <ErrorBox message={error} />
      {loading ? (
        <Loading />
      ) : error ? (
        <Button variant="text" onClick={() => setRetry((v) => v + 1)}>
          重新加载角色
        </Button>
      ) : (
        <>
          <div className="mt-3 grid">
            {items.map((role) => (
              <CheckboxItem
                key={role.id}
                title={role.name}
                description={role.description}
                disabled={disabled}
                checked={value.includes(role.id)}
                onChange={(event) =>
                  onChange(
                    event.target.checked
                      ? [...value, role.id]
                      : value.filter((id) => id !== role.id),
                  )
                }
              />
            ))}
          </div>
          {!items.length && <Empty text="没有找到可分配角色" />}
        </>
      )}
      <Pagination
        page={page}
        total={total}
        disabled={loading || disabled}
        onChange={setPage}
      />
    </div>
  );
}
