import { ConfirmDialog } from "../../components/Dialog";
import Badge from "../../components/Badge";
import {
  TableHeader,
  TableHead,
  TableCell,
  TableBody,
  Table,
} from "../../components/Table";
import Button from "../../components/Button";
import { useEffect, useState } from "react";
import { Plus, Pencil, Trash2 } from "lucide-react";
import { rolesApi } from "./api";
import type { Role } from "../../lib/apiClient";
import { useAuth } from "../../app/AuthContext";
import { PageTitle, ErrorBox } from "../../components/Shared";
import SearchField from "../../components/SearchField";
import ListPanel from "../../components/ListPanel";
export default function RolesPage() {
  const a = useAuth(),
    [roles, setRoles] = useState<Role[]>([]),
    [error, setError] = useState(""),
    [loading, setLoading] = useState(true),
    [target, setTarget] = useState<Role | null>(null),
    [busy, setBusy] = useState(false);
  const [text, setText] = useState("");
  const [query, setQuery] = useState("");
  const [page, setPage] = useState(1);
  const [total, setTotal] = useState(0);
  const [revision, setRevision] = useState(0);
  const [loadFailed, setLoadFailed] = useState(false);
  useEffect(() => {
    let active = true;
    setLoading(true);
    setError("");
    setLoadFailed(false);
    rolesApi
      .list(query, page)
      .then((v) => {
        if (active) {
          setRoles(v.items);
          setTotal(v.total);
        }
      })
      .catch((e) => {
        if (active) {
          setError(e.message);
          setLoadFailed(true);
        }
      })
      .finally(() => {
        if (active) setLoading(false);
      });
    return () => {
      active = false;
    };
  }, [query, page, revision]);
  async function remove() {
    if (!target) return;
    setBusy(true);
    setError("");
    try {
      await rolesApi.remove(target.id);
      setTarget(null);
      if (roles.length === 1 && page > 1) setPage(page - 1);
      else setRevision((v) => v + 1);
    } catch (e) {
      setError((e as Error).message);
    } finally {
      setBusy(false);
    }
  }
  return (
    <>
      <PageTitle
        title="角色与权限"
        action={
          a.can("roles:manage") && (
            <Button to="/roles/new" variant="primary">
              <Plus size={17} /> 新建角色
            </Button>
          )
        }
      />
      <div className="mb-4">
        <SearchField
          label="搜索角色"
          placeholder="搜索角色名称或说明…"
          value={text}
          onChange={setText}
          onSearch={() => {
            setPage(1);
            setQuery(text.trim());
            setRevision((v) => v + 1);
          }}
        />
      </div>
      <ErrorBox message={loadFailed ? "" : error} />
      <ListPanel
        loading={loading}
        error={loadFailed ? error : ""}
        empty={!roles.length}
        emptyText="暂无角色"
        page={page}
        total={total}
        disabled={busy}
        onPageChange={setPage}
      >
        <Table>
          <TableHead>
            <tr>
              <TableHeader>角色</TableHeader>
              <TableHeader>说明</TableHeader>
              <TableHeader>权限</TableHeader>
              <TableHeader>成员数</TableHeader>
              {a.can("roles:manage") && (
                <TableHeader className="text-right">操作</TableHeader>
              )}
            </tr>
          </TableHead>
          <TableBody>
            {roles.map((r) => (
              <tr key={r.id}>
                <TableCell>
                  <strong>{r.name}</strong>
                  {r.system && (
                    <span className="mt-1 block text-[11px] text-muted">
                      系统内置
                    </span>
                  )}
                </TableCell>
                <TableCell className="min-w-[160px] max-w-[300px] text-[13px] leading-[1.8] whitespace-normal text-muted [overflow-wrap:anywhere]">
                  {r.description || "—"}
                </TableCell>
                <TableCell>
                  <div className="flex flex-wrap gap-[6px]">
                    {r.permissions.map((p) => (
                      <Badge key={p.code}>{p.name}</Badge>
                    ))}
                    {!r.permissions.length && (
                      <span className="text-muted">未配置</span>
                    )}
                  </div>
                </TableCell>
                <TableCell>{r.userCount}</TableCell>
                {a.can("roles:manage") && (
                  <TableCell className="text-right">
                    {r.system ? (
                      <span className="text-muted">不可修改</span>
                    ) : (
                      <div className="flex justify-end gap-4">
                        <Button to={`/roles/${r.id}/edit`} variant="text">
                          <Pencil size={14} />
                          编辑
                        </Button>
                        <Button
                          aria-label={`删除${r.name}`}
                          title={
                            r.userCount ? "已被分配的角色不可删除" : "删除角色"
                          }
                          disabled={r.userCount > 0}
                          onClick={() => {
                            setTarget(r);
                            setError("");
                          }}
                          variant="dangerText"
                        >
                          <Trash2 size={14} />
                          删除
                        </Button>
                      </div>
                    )}
                  </TableCell>
                )}
              </tr>
            ))}
          </TableBody>
        </Table>
      </ListPanel>
      {target && (
        <ConfirmDialog
          title={<>删除角色「{target.name}」？</>}
          busy={busy}
          description="删除后无法恢复。只有未分配给任何成员的角色可以删除。"
          error={error}
          confirmLabel="确认删除"
          onConfirm={() => void remove()}
          onClose={() => setTarget(null)}
        />
      )}
    </>
  );
}
