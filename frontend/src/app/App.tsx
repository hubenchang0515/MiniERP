import Button from "../components/Button";
import ResultState from "../components/ResultState";
import { BrowserRouter } from "react-router-dom";
import { AuthProvider, useAuth } from "./AuthContext";
import AppRoutes from "./router";
import { Loading, ErrorBox } from "../components/Shared";
import { ThemeProvider } from "./ThemeContext";
function Ready() {
  const a = useAuth();
  if (a.loading) return <Loading />;
  if (a.error)
    return (
      <ResultState
        description={<ErrorBox message={a.error} />}
        actions={
          <Button onClick={() => void a.reload()} variant="primary">
            重新连接
          </Button>
        }
      />
    );
  return <AppRoutes />;
}
export default function App() {
  return (
    <ThemeProvider>
      <BrowserRouter>
        <AuthProvider>
          <Ready />
        </AuthProvider>
      </BrowserRouter>
    </ThemeProvider>
  );
}
