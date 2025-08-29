"use client";

import React, { createContext, useCallback, useContext, useEffect, useMemo, useState } from "react";
import jwtDecode from "jwt-decode";
import { getToken, setToken } from "./token";

type Decoded = { sub?: string; email?: string; [k: string]: any };

interface AuthState {
  token: string | null;
  email: string | null;
  setAuthToken: (t: string | null) => void;
  logout: () => void;
}

const AuthContext = createContext<AuthState | undefined>(undefined);

export function AuthProvider({ children }: { children: React.ReactNode }) {
  const [token, _setToken] = useState<string | null>(null);
  const [email, setEmail] = useState<string | null>(null);

  const setAuthToken = useCallback((t: string | null) => {
    _setToken(t);
    setToken(t);
    if (t) {
      try {
        const decoded = jwtDecode<Decoded>(t);
        setEmail((decoded.email as string) || (decoded.sub as string) || null);
      } catch {
        setEmail(null);
      }
    } else {
      setEmail(null);
    }
  }, []);

  useEffect(() => {
    const existing = getToken();
    if (existing) setAuthToken(existing);
  }, [setAuthToken]);

  const logout = useCallback(() => setAuthToken(null), [setAuthToken]);

  const value = useMemo(() => ({ token, email, setAuthToken, logout }), [token, email, setAuthToken, logout]);

  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>;
}

export function useAuth() {
  const ctx = useContext(AuthContext);
  if (!ctx) throw new Error("useAuth must be used within AuthProvider");
  return ctx;
}
