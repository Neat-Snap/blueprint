"use client";

import React, { createContext, useCallback, useContext, useEffect, useMemo, useRef, useState } from "react";
import { listTeams, createTeam as apiCreate, deleteTeam as apiDelete } from "./teams";

export type CurrentTeam = { id: number; name: string; icon?: string } | null;

type Ctx = {
  current: CurrentTeam;
  setCurrentId: (id: number | null) => void;
  switchTo: (id: number) => Promise<void>;
  switching: boolean;
  all: { id: number; name: string; icon?: string }[];
  refresh: () => Promise<void>;
  createTeam: (name: string, icon?: string) => Promise<void>;
  deleteTeam: (id: number) => Promise<void>;
};

const TeamCtx = createContext<Ctx | undefined>(undefined);

export function TeamProvider({ children }: { children: React.ReactNode }) {
  const [all, setAll] = useState<{ id: number; name: string; icon?: string }[]>([]);
  const [currentId, setCurrentId] = useState<number | null>(null);
  const [switching, setSwitching] = useState(false);

  useEffect(() => {
    const saved = typeof window !== "undefined" ? window.localStorage.getItem("currentTeamId") : null;
    if (saved) setCurrentId(Number(saved));
  }, []);

  useEffect(() => {
    if (typeof window !== "undefined") {
      if (currentId) window.localStorage.setItem("currentTeamId", String(currentId));
      else window.localStorage.removeItem("currentTeamId");
    }
  }, [currentId]);

  const refresh = useCallback(async () => {
    const items = await listTeams();
    const mapped = items.map((w: { id: number; name: string; icon?: string }) => ({ id: w.id, name: w.name, icon: w.icon }));
    setAll(mapped);
    if (!currentId && mapped.length) setCurrentId(mapped[0].id);
    if (currentId && !mapped.find((w: { id: number }) => w.id === currentId)) setCurrentId(mapped[0]?.id ?? null);
  }, [currentId]);

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
  }, [refresh]);

  async function createTeam(name: string, icon?: string) {
    const created = await apiCreate(name, icon);
    await refresh();
    setCurrentId(created.id);
  }

  async function deleteTeam(id: number) {
    await apiDelete(id);
    await refresh();
  }

  const current = useMemo<CurrentTeam>(() => {
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
    createTeam,
    deleteTeam,
  };

  return <TeamCtx.Provider value={value}>{children}</TeamCtx.Provider>;
}

export function useTeam() {
  const ctx = useContext(TeamCtx);
  if (!ctx) throw new Error("useTeam must be used within TeamProvider");
  return ctx;
}
