"use client";

import { useState } from "react";
import { Check, CircleMinus, CircleX, Loader2 } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { submitReport, type Pharmacy } from "@/lib/api";
import { cn } from "@/lib/utils";

type Status = "in_stock" | "out" | "limited";

type Props = {
  drugId: number;
  pharmacies: Pharmacy[];
  onSubmitted?: () => void;
};

export function ReportPanel({ drugId, pharmacies, onSubmitted }: Props) {
  const [query, setQuery] = useState("");
  const [busy, setBusy] = useState<{ [k: string]: Status | undefined }>({});

  const filtered = pharmacies.filter((p) =>
    (p.name + (p.city ?? "") + (p.address ?? "")).toLowerCase().includes(query.toLowerCase()),
  );

  async function send(pharmacyId: number, status: Status) {
    setBusy((s) => ({ ...s, [pharmacyId]: status }));
    try {
      await submitReport({ drug_id: drugId, pharmacy_id: pharmacyId, status });
      onSubmitted?.();
    } catch (e: any) {
      alert(e.message);
    } finally {
      setBusy((s) => ({ ...s, [pharmacyId]: undefined }));
    }
  }

  return (
    <div className="rounded-lg border bg-muted/30 p-3">
      <div className="mb-2 flex items-center justify-between">
        <div className="text-xs font-medium text-muted-foreground">
          report stock at a pharmacy nearby
        </div>
      </div>
      <Input
        value={query}
        onChange={(e) => setQuery(e.target.value)}
        placeholder="filter pharmacy…"
        className="mb-2 h-8 text-xs"
      />
      <ul className="max-h-64 space-y-1 overflow-y-auto text-sm">
        {filtered.slice(0, 50).map((p) => {
          const b = busy[p.id];
          return (
            <li
              key={p.id}
              className="flex items-center justify-between gap-2 rounded-md px-2 py-1.5 hover:bg-background"
            >
              <span className="min-w-0 flex-1 truncate" title={p.name}>
                {p.name}
              </span>
              <div className="flex shrink-0 gap-1">
                <IconButton
                  onClick={() => send(p.id, "in_stock")}
                  busy={b === "in_stock"}
                  label="in"
                  className="text-[hsl(var(--success))] hover:bg-[hsl(var(--success)/0.1)]"
                  icon={<Check className="h-3.5 w-3.5" />}
                />
                <IconButton
                  onClick={() => send(p.id, "limited")}
                  busy={b === "limited"}
                  label="low"
                  className="text-[hsl(var(--warning))] hover:bg-[hsl(var(--warning)/0.1)]"
                  icon={<CircleMinus className="h-3.5 w-3.5" />}
                />
                <IconButton
                  onClick={() => send(p.id, "out")}
                  busy={b === "out"}
                  label="out"
                  className="text-destructive hover:bg-destructive/10"
                  icon={<CircleX className="h-3.5 w-3.5" />}
                />
              </div>
            </li>
          );
        })}
        {filtered.length === 0 && (
          <li className="py-4 text-center text-xs text-muted-foreground">no pharmacies in view</li>
        )}
      </ul>
    </div>
  );
}

function IconButton({
  onClick,
  busy,
  label,
  className,
  icon,
}: {
  onClick: () => void;
  busy: boolean;
  label: string;
  className: string;
  icon: React.ReactNode;
}) {
  return (
    <button
      onClick={onClick}
      disabled={busy}
      className={cn(
        "flex h-7 items-center gap-1 rounded-md border border-transparent bg-background px-2 text-[10px] font-medium transition-colors disabled:opacity-50",
        className,
      )}
    >
      {busy ? <Loader2 className="h-3 w-3 animate-spin" /> : icon}
      {label}
    </button>
  );
}
