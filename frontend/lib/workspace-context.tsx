"use client";

import React, { createContext, useContext, useEffect, useMemo, useRef, useState } from "react";
import { listWorkspaces, createWorkspace as apiCreate, deleteWorkspace as apiDelete } from "./workspaces";

export type CurrentWorkspace = { id: number; name: string; icon?: string } | null;

type Ctx = {
  current: CurrentWorkspace;
  setCurrentId: (id: number | null) => void;
  switchTo: (id: number) => Promise<void>;
  switching: boolean;
  all: { id: number; name: string; icon?: string }[];
  refresh: () => Promise<void>;
  createWorkspace: (name: string, icon?: string) => Promise<void>;
  deleteWorkspace: (id: number) => Promise<void>;
};

const WorkspaceCtx = createContext<Ctx | undefined>(undefined);

export function WorkspaceProvider({ children }: { children: React.ReactNode }) {
  const [all, setAll] = useState<{ id: number; name: string; icon?: string }[]>([]);
  const [currentId, setCurrentId] = useState<number | null>(null);
  const [switching, setSwitching] = useState(false);

  useEffect(() => {
    const saved = typeof window !== "undefined" ? window.localStorage.getItem("currentWorkspaceId") : null;
    if (saved) setCurrentId(Number(saved));
  }, []);

  useEffect(() => {
    if (typeof window !== "undefined") {
      if (currentId) window.localStorage.setItem("currentWorkspaceId", String(currentId));
      else window.localStorage.removeItem("currentWorkspaceId");
    }
  }, [currentId]);

  async function refresh() {
    const items = await listWorkspaces();
    const mapped = items.map((w) => ({ id: w.id, name: w.name, icon: w.icon }));
    setAll(mapped);
    if (!currentId && mapped.length) setCurrentId(mapped[0].id);
    if (currentId && !mapped.find((w) => w.id === currentId)) setCurrentId(mapped[0]?.id ?? null);
  }

  const didInitialRefresh = useRef(false);
  useEffect(() => {
    if (didInitialRefresh.current) return;
    didInitialRefresh.current = true;
    (async () => {
      try {
        await refresh();
      } catch {

      }
    })();
  }, []);

  async function createWorkspace(name: string, icon?: string) {
    const created = await apiCreate(name, icon);
    await refresh();
    setCurrentId(created.id);
  }

  async function deleteWorkspace(id: number) {
    await apiDelete(id);
    await refresh();
  }

  const current = useMemo<CurrentWorkspace>(() => {
    if (!currentId) return null;
    const w = all.find((x) => x.id === currentId) || null;
    return w;
  }, [currentId, all]);

  const value: Ctx = {
    current,
    setCurrentId: (id) => setCurrentId(id),
    switchTo: async (id: number) => {
      if (id === currentId) return;
      setSwitching(true);
      setCurrentId(id);

      await new Promise((r) => setTimeout(r, 450));
      setSwitching(false);
    },
    switching,
    all,
    refresh,
    createWorkspace,
    deleteWorkspace,
  };

  return <WorkspaceCtx.Provider value={value}>{children}</WorkspaceCtx.Provider>;
}

export function useWorkspace() {
  const ctx = useContext(WorkspaceCtx);
  if (!ctx) throw new Error("useWorkspace must be used within WorkspaceProvider");
  return ctx;
}
