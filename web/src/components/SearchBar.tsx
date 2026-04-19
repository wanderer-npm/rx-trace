"use client";

import { useEffect, useRef, useState } from "react";
import { Search, X } from "lucide-react";
import { Input } from "@/components/ui/input";
import {
  Command,
  CommandEmpty,
  CommandGroup,
  CommandItem,
  CommandList,
} from "@/components/ui/command";
import { searchDrugs, type Drug } from "@/lib/api";
import { cn } from "@/lib/utils";

type Props = {
  value: Drug | null;
  onPick: (d: Drug | null) => void;
};

export function SearchBar({ value, onPick }: Props) {
  const [q, setQ] = useState("");
  const [open, setOpen] = useState(false);
  const [drugs, setDrugs] = useState<Drug[]>([]);
  const ref = useRef<HTMLDivElement>(null);

  useEffect(() => {
    if (q.trim().length < 2) {
      setDrugs([]);
      return;
    }
    const t = setTimeout(() => {
      searchDrugs(q)
        .then((r) => {
          setDrugs(r);
          setOpen(r.length > 0);
        })
        .catch(() => setDrugs([]));
    }, 150);
    return () => clearTimeout(t);
  }, [q]);

  useEffect(() => {
    function handle(e: MouseEvent) {
      if (ref.current && !ref.current.contains(e.target as Node)) setOpen(false);
    }
    document.addEventListener("mousedown", handle);
    return () => document.removeEventListener("mousedown", handle);
  }, []);

  if (value) {
    return (
      <div className="flex items-center justify-between rounded-lg border bg-card p-3">
        <div className="min-w-0">
          <div className="font-medium truncate">{value.name}</div>
          {value.generic && (
            <div className="text-xs text-muted-foreground truncate">
              {[value.generic, value.strength, value.form].filter(Boolean).join(", ")}
            </div>
          )}
        </div>
        <button
          onClick={() => {
            onPick(null);
            setQ("");
          }}
          className="shrink-0 rounded-md p-1.5 text-muted-foreground hover:bg-accent hover:text-foreground"
          aria-label="clear"
        >
          <X className="h-4 w-4" />
        </button>
      </div>
    );
  }

  return (
    <div ref={ref} className="relative">
      <Search className="pointer-events-none absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-muted-foreground" />
      <Input
        value={q}
        onChange={(e) => {
          setQ(e.target.value);
          setOpen(true);
        }}
        onFocus={() => drugs.length > 0 && setOpen(true)}
        placeholder="dolo 650, azithral, insulin…"
        className="h-11 pl-9 text-sm"
      />
      {open && drugs.length > 0 && (
        <div className="absolute left-0 right-0 top-full z-30 mt-1 rounded-lg border bg-popover shadow-lg animate-in fade-in-0 zoom-in-95">
          <Command shouldFilter={false}>
            <CommandList>
              <CommandEmpty>no matches</CommandEmpty>
              <CommandGroup>
                {drugs.map((d) => (
                  <CommandItem
                    key={d.id}
                    value={d.name}
                    onSelect={() => {
                      onPick(d);
                      setQ("");
                      setOpen(false);
                    }}
                    className="flex items-start justify-between gap-3"
                  >
                    <div className="min-w-0">
                      <div className="font-medium truncate">{d.name}</div>
                      {d.generic && (
                        <div className="text-xs text-muted-foreground truncate">{d.generic}</div>
                      )}
                    </div>
                    <div className="shrink-0 text-xs text-muted-foreground">
                      {[d.strength, d.form].filter(Boolean).join(", ")}
                    </div>
                  </CommandItem>
                ))}
              </CommandGroup>
            </CommandList>
          </Command>
        </div>
      )}
    </div>
  );
}
