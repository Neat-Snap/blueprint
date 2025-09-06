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
  const [currentId, setCurrentId] = useState<number | null>(() => {
    if (typeof window !== "undefined") {
      const saved = window.localStorage.getItem("currentTeamId");
      if (saved) {
        const num = Number(saved);
        return Number.isFinite(num) ? num : null;
      }
    }
    return null;
  });
  const [switching, setSwitching] = useState(false);

  // removed: we now read from localStorage in the useState initializer above

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
    if (mapped.length === 0) {
      // No teams available
      if (currentId !== null) setCurrentId(null);
      return;
    }
    // If we have a currentId and it's present in the list, keep it
    if (currentId && mapped.some((w) => w.id === currentId)) return;
    // Otherwise, try to use saved localStorage selection if valid
    let nextId: number | null = null;
    if (typeof window !== "undefined") {
      const saved = window.localStorage.getItem("currentTeamId");
      if (saved) {
        const s = Number(saved);
        if (Number.isFinite(s) && mapped.some((w) => w.id === s)) {
          nextId = s;
        }
      }
    }
    // Fallback to first team if nothing else
    if (nextId == null) nextId = mapped[0]?.id ?? null;
    setCurrentId(nextId);
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
