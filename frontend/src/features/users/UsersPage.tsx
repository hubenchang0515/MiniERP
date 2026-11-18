import ListPanel from "../../components/ListPanel";
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
import { Plus, Pencil } from "lucide-react";
import SearchField from "../../components/SearchField";
import { useAuth } from "../../app/AuthContext";
import { usersApi } from "./api";
import type { User } from "../../lib/apiClient";
import { PageTitle } from "../../components/Shared";
export default function UsersPage() {
  const a = useAuth(),
    [items, setItems] = useState<User[]>([]),
    [total, setTotal] = useState(0),
    [q, setQ] = useState(""),
    [search, setSearch] = useState(""),
    [page, setPage] = useState(1),
    [loading, setLoading] = useState(true),
    [error, setError] = useState("");
  const [revision, setRevision] = useState(0);
  useEffect(() => {
    let active = true;
    setLoading(true);
    setError("");
    usersApi
      .list(search, page)
      .then((d) => {
        if (active) {
          setItems(d.items);
          setTotal(d.total);
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
  }, [search, page, revision]);
  return (
    <>
      <PageTitle
        title="用户管理"
        action={
          a.can("users:manage") && (
            <Button to="/users/new" variant="primary">
              <Plus size={17} /> 新建用户
            </Button>
          )
        }
      />
      <div className="mb-4">
        <SearchField
          label="搜索用户"
          placeholder="搜索账号或姓名…"
          value={q}
          onChange={setQ}
          onSearch={() => {
            setPage(1);
            setSearch(q.trim());
            setRevision((value) => value + 1);
          }}
        />
      </div>
      <ListPanel
        loading={loading}
        error={error}
        empty={!items.length}
        emptyText="没有找到匹配的成员"
        page={page}
        total={total}
        onPageChange={setPage}
      >
        <Table>
          <TableHead>
            <tr>
              <TableHeader>成员</TableHeader>
              <TableHeader>账号</TableHeader>
              <TableHeader>角色</TableHeader>
              <TableHeader>状态</TableHeader>
              <TableHeader>创建时间</TableHeader>
              {a.can("users:manage") && (
                <TableHeader className="text-right">操作</TableHeader>
              )}
            </tr>
          </TableHead>
          <TableBody>
            {items.map((u) => (
              <tr key={u.id}>
                <TableCell>
                  <div className="flex items-center gap-[11px] [&_strong]:text-[12px] [&_strong]:font-medium">
                    <strong>{u.displayName || u.username}</strong>
                  </div>
                </TableCell>
                <TableCell className="text-muted">{u.username}</TableCell>
                <TableCell>
                  <div className="flex flex-wrap gap-[6px]">
                    {u.roles.map((r) => (
                      <Badge key={r.id}>{r.name}</Badge>
                    ))}
                    {!u.roles.length && (
                      <span className="text-muted">未分配</span>
                    )}
                  </div>
                </TableCell>
                <TableCell>
                  <span
                    className={`inline-flex items-center gap-[6px] rounded-[5px] px-2 py-1 text-[12px] ${u.enabled ? "bg-success-bg text-success-ink" : "bg-subtle text-muted"}`}
                  >
                    <span className="inline-block size-[6px] shrink-0 rounded-full bg-current" />
                    {u.enabled ? "已启用" : "已停用"}
                  </span>
                </TableCell>
                <TableCell className="text-muted">
                  {new Date(u.createdAt).toLocaleDateString("zh-CN")}
                </TableCell>
                {a.can("users:manage") && (
                  <TableCell className="text-right">
                    <Button to={`/users/${u.id}/edit`} variant="text">
                      <Pencil size={14} /> 编辑
                    </Button>
                  </TableCell>
                )}
              </tr>
            ))}
          </TableBody>
        </Table>
      </ListPanel>
    </>
  );
}
