import {
  createContext,
  useCallback,
  useContext,
  useEffect,
  useState,
  type ReactNode,
} from "react";
import { authApi, type Credentials } from "../features/auth/api";
import { ApiError, type CurrentUser, type Session } from "../lib/apiClient";
import { authStorage } from "../lib/authStorage";
type Auth = {
  loading: boolean;
  initialized: boolean;
  user: CurrentUser | null;
  error: string;
  reload: () => Promise<void>;
  login: (v: Credentials) => Promise<CurrentUser>;
  setup: (v: Credentials) => Promise<CurrentUser>;
  logout: () => void;
  can: (code: string) => boolean;
};
const Context = createContext<Auth>(null!);
export function AuthProvider({ children }: { children: ReactNode }) {
  const [loading, setLoading] = useState(true),
    [initialized, setInitialized] = useState(false),
    [user, setUser] = useState<CurrentUser | null>(null),
    [error, setError] = useState("");
  const reload = useCallback(async () => {
    setLoading(true);
    setError("");
    try {
      const status = await authApi.status();
      setInitialized(status.initialized);
      if (status.initialized && authStorage.get()) {
        try {
          setUser(await authApi.me());
        } catch (e) {
          if (e instanceof ApiError && e.status === 401) {
            setUser(null);
          } else {
            throw e;
          }
        }
      } else {
        setUser(null);
      }
    } catch (e) {
      setError((e as Error).message);
    } finally {
      setLoading(false);
    }
  }, []);
  useEffect(() => {
    void reload();
    const expired = () => setUser(null);
    window.addEventListener("auth-expired", expired);
    return () => window.removeEventListener("auth-expired", expired);
  }, [reload]);
  const accept = (s: Session) => {
    authStorage.set(s.token);
    setUser(s.user);
    setInitialized(true);
    return s.user;
  };
  return (
    <Context.Provider
      value={{
        loading,
        initialized,
        user,
        error,
        reload,
        login: async (v) => accept(await authApi.login(v)),
        setup: async (v) => accept(await authApi.bootstrap(v)),
        logout: () => {
          authStorage.clear();
          setUser(null);
        },
        can: (code) => !!user?.permissions.includes(code),
      }}
    >
      {children}
    </Context.Provider>
  );
}
export const useAuth = () => useContext(Context);
