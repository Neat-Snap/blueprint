import React from "react";
import {
  Briefcase,
  Building2,
  Bolt,
  FlaskConical,
  Book,
  Calendar,
  BarChart2,
  Code2,
  Compass,
  Cpu,
  Database,
} from "lucide-react";

export const ALLOWED_TEAM_ICONS = [
  "briefcase",
  "building",
  "bolt",
  "beaker",
  "book",
  "calendar",
  "chart",
  "code",
  "compass",
  "cpu",
  "database",
] as const;

const map: Record<(typeof ALLOWED_TEAM_ICONS)[number], React.ComponentType<{ className?: string }>> = {
  briefcase: Briefcase,
  building: Building2,
  bolt: Bolt,
  beaker: FlaskConical,
  book: Book,
  calendar: Calendar,
  chart: BarChart2,
  code: Code2,
  compass: Compass,
  cpu: Cpu,
  database: Database,
};

export function renderTeamIcon(key?: string, className?: string): React.ReactNode {
  if (!key) return null;
  const k = key.toLowerCase() as (typeof ALLOWED_TEAM_ICONS)[number];
  const Cmp = map[k];
  if (!Cmp) return null;
  return React.createElement(Cmp, { className });
}
