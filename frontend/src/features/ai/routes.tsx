import { Navigate, Route } from "react-router-dom";
import { useAuth } from "../../app/AuthContext";
import AIPage from "./AIPage";
import { aiPages, type AISection } from "./paths";

function AIHomeRedirect() {
  const admin = useAuth().user?.roles.some((role) => role.system) ?? false;
  return <Navigate replace to={admin ? "/ai/providers" : "/ai/allocations"} />;
}

function AISectionPage({ section }: { section: AISection }) {
  const admin = useAuth().user?.roles.some((role) => role.system) ?? false;
  if (section !== "keys" && !admin) return <Navigate replace to="/403" />;
  return <AIPage section={section} />;
}

export const aiRoutes = (
  <Route path="/ai">
    <Route index element={<AIHomeRedirect />} />
    {aiPages.map((page) => (
      <Route
        key={page.section}
        path={page.path.slice("/ai/".length)}
        element={<AISectionPage section={page.section} />}
      />
    ))}
  </Route>
);
