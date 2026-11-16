import type { ReactNode } from "react";
import { Empty, ErrorBox, Loading } from "./Shared";
import Pagination from "./Pagination";
import Card from "./Card";

export default function ListPanel({
  children,
  loading,
  error = "",
  empty,
  emptyText = "暂无数据",
  page,
  total,
  pageSize = 10,
  disabled,
  onPageChange,
}: {
  children: ReactNode;
  loading: boolean;
  error?: string;
  empty: boolean;
  emptyText?: string;
  page: number;
  total: number;
  pageSize?: number;
  disabled?: boolean;
  onPageChange: (page: number) => void;
}) {
  return (
    <Card
      as="section"
      padding="none"
      className="overflow-hidden"
      aria-busy={loading || undefined}
    >
      <ErrorBox message={error} />
      {loading ? (
        <Loading />
      ) : error ? (
        <Empty text="加载失败，请重新搜索后重试。" />
      ) : empty ? (
        <Empty text={emptyText} />
      ) : (
        children
      )}
      <Pagination
        page={page}
        total={total}
        pageSize={pageSize}
        disabled={loading || Boolean(error) || disabled}
        onChange={onPageChange}
      />
    </Card>
  );
}
