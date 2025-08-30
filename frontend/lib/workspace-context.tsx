"use client";

import React, { createContext, useContext, useEffect, useMemo, useState } from "react";
import { listWorkspaces, createWorkspace as apiCreate, deleteWorkspace as apiDelete } from "./workspaces";

export type CurrentWorkspace = { id: number; name: string } | null;

type Ctx = {
  current: CurrentWorkspace;
  setCurrentId: (id: number | null) => void;
  all: { id: number; name: string }[];
  refresh: () => Promise<void>;
  createWorkspace: (name: string) => Promise<void>;
  deleteWorkspace: (id: number) => Promise<void>;
};

const WorkspaceCtx = createContext<Ctx | undefined>(undefined);

export function WorkspaceProvider({ children }: { children: React.ReactNode }) {
  const [all, setAll] = useState<{ id: number; name: string }[]>([]);
  const [currentId, setCurrentId] = useState<number | null>(null);

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
    const mapped = items.map((w) => ({ id: w.id, name: w.name }));
    setAll(mapped);
    if (!currentId && mapped.length) setCurrentId(mapped[0].id);
    if (currentId && !mapped.find((w) => w.id === currentId)) setCurrentId(mapped[0]?.id ?? null);
  }

  useEffect(() => {
    (async () => {
      try {
        await refresh();
      } catch {
        // handled upstream by auth guards
      }
    })();
  }, []);

  async function createWorkspace(name: string) {
    const created = await apiCreate(name);
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
