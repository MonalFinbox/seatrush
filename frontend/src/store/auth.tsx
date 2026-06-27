import {
  createContext,
  useContext,
  useEffect,
  useMemo,
  useState,
  type ReactNode,
} from "react";
import { authApi } from "@/lib/services";
import { setLogoutHandler, tokenStore } from "@/lib/api";
import type { TokenPair, User } from "@/types/schemas";

interface AuthContextValue {
  user: User | null;
  loading: boolean;
  setSession: (pair: TokenPair) => void;
  logout: () => void;
  refreshUser: () => Promise<void>;
}

const AuthContext = createContext<AuthContextValue | null>(null);

export function AuthProvider({ children }: { children: ReactNode }) {
  const [user, setUser] = useState<User | null>(null);
  const [loading, setLoading] = useState(true);

  const setSession = (pair: TokenPair) => {
    tokenStore.set(pair.accessToken, pair.refreshToken);
    setUser(pair.user);
  };

  const logout = () => {
    tokenStore.clear();
    setUser(null);
  };

  const refreshUser = async () => {
    try {
      setUser(await authApi.me());
    } catch {
      logout();
    }
  };

  // Let the axios interceptor end the session when a refresh fails.
  useEffect(() => {
    setLogoutHandler(logout);
  }, []);

  // Hydrate the user on first load if a token is already present.
  useEffect(() => {
    if (!tokenStore.access()) {
      setLoading(false);
      return;
    }
    authApi
      .me()
      .then(setUser)
      .catch(() => tokenStore.clear())
      .finally(() => setLoading(false));
  }, []);

  const value = useMemo(
    () => ({ user, loading, setSession, logout, refreshUser }),
    [user, loading]
  );

  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>;
}

// eslint-disable-next-line react-refresh/only-export-components
export function useAuth() {
  const ctx = useContext(AuthContext);
  if (!ctx) throw new Error("useAuth must be used within AuthProvider");
  return ctx;
}
