import type { ReactNode } from "react";
import { Link } from "react-router-dom";
import { ArrowLeft, LoaderCircle } from "lucide-react";
import Alert from "./Alert";
export function Loading() {
  return (
    <div
      className="flex items-center justify-center gap-[10px] p-[70px] text-muted"
      role="status"
    >
      <LoaderCircle className="animate-spin" size={22} /> 正在加载…
    </div>
  );
}
export function ErrorBox({ message }: { message: string }) {
  return message ? <Alert tone="error">{message}</Alert> : null;
}
export function PageTitle({
  title,
  action,
}: {
  title: string;
  action?: ReactNode;
}) {
  return (
    <div className="mb-[22px] flex items-center justify-between gap-5 max-[760px]:items-start">
      <div>
        <h1 className="text-[24px] leading-[1.5] font-semibold tracking-[-0.8px]">
          {title}
        </h1>
      </div>
      {action}
    </div>
  );
}
export function Back({ to }: { to: string }) {
  return (
    <Link
      className="mb-5 inline-flex items-center gap-[7px] text-[13px] text-muted"
      to={to}
    >
      <ArrowLeft size={16} /> 返回列表
    </Link>
  );
}
export function Empty({ text }: { text: string }) {
  return <div className="px-5 py-[65px] text-center text-muted">{text}</div>;
}
