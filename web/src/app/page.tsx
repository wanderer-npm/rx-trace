"use client";

import { useCallback, useEffect, useMemo, useState } from "react";
import dynamic from "next/dynamic";
import { Loader2, MapPin, Pill, Sparkles } from "lucide-react";
import { toast } from "sonner";

import { SearchBar } from "@/components/SearchBar";
import { StockCard } from "@/components/StockCard";
import { ReportPanel } from "@/components/ReportPanel";
import { AlertDialog } from "@/components/AlertDialog";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { Switch } from "@/components/ui/switch";
import { Label } from "@/components/ui/label";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { Separator } from "@/components/ui/separator";
import {
  findInStock,
  nearbyPharmacies,
  type Drug,
  type FindRow,
  type Pharmacy,
} from "@/lib/api";

const MapView = dynamic(() => import("@/components/MapView"), { ssr: false });

const radiusOptions = [
  { v: 1000, label: "1 km" },
  { v: 3000, label: "3 km" },
  { v: 5000, label: "5 km" },
  { v: 10000, label: "10 km" },
  { v: 25000, label: "25 km" },
];

export default function Home() {
  const [loc, setLoc] = useState<{ lat: number; lng: number } | null>(null);
  const [gettingLoc, setGettingLoc] = useState(true);

  const [picked, setPicked] = useState<Drug | null>(null);
  const [pharmacies, setPharmacies] = useState<Pharmacy[]>([]);
  const [rows, setRows] = useState<FindRow[] | null>(null);

  const [radius, setRadius] = useState(5000);
  const [includeStale, setIncludeStale] = useState(false);
  const [loading, setLoading] = useState(false);

  useEffect(() => {
    if (!navigator.geolocation) {
      setLoc({ lat: 28.6139, lng: 77.209 });
      setGettingLoc(false);
      return;
    }
    navigator.geolocation.getCurrentPosition(
      (p) => {
        setLoc({ lat: p.coords.latitude, lng: p.coords.longitude });
        setGettingLoc(false);
      },
      () => {
        setLoc({ lat: 28.6139, lng: 77.209 });
        setGettingLoc(false);
      },
      { enableHighAccuracy: false, timeout: 5000 },
    );
  }, []);

  useEffect(() => {
    if (!loc) return;
    nearbyPharmacies(loc.lat, loc.lng, radius)
      .then(setPharmacies)
      .catch(() => setPharmacies([]));
  }, [loc, radius]);

  const reload = useCallback(() => {
    if (!loc || !picked) {
      setRows(null);
      return;
    }
    setLoading(true);
    findInStock({
      drug_id: picked.id,
      lat: loc.lat,
      lng: loc.lng,
      radius_m: radius,
      include_stale: includeStale,
    })
      .then(setRows)
      .catch(() => setRows([]))
      .finally(() => setLoading(false));
  }, [loc, picked, radius, includeStale]);

  useEffect(() => {
    reload();
  }, [reload]);

  const counts = useMemo(() => {
    if (!rows) return null;
    return {
      inStock: rows.filter((r) => r.status === "in_stock").length,
      limited: rows.filter((r) => r.status === "limited").length,
      out: rows.filter((r) => r.status === "out").length,
    };
  }, [rows]);

  return (
    <main className="grid h-dvh grid-cols-1 md:grid-cols-[min(440px,40vw)_1fr]">
      <aside className="flex h-full min-h-0 flex-col border-r bg-card/50 backdrop-blur">
        <header className="flex items-center justify-between gap-3 border-b px-5 py-4">
          <div className="flex items-center gap-2">
            <div className="flex h-7 w-7 items-center justify-center rounded-md bg-primary text-primary-foreground">
              <Pill className="h-4 w-4" />
            </div>
            <div className="leading-tight">
              <div className="font-semibold tracking-tight">rx-trace</div>
              <div className="text-[11px] text-muted-foreground">in-stock, near you</div>
            </div>
          </div>
          {gettingLoc ? (
            <Badge variant="outline" className="gap-1.5 text-xs">
              <Loader2 className="h-3 w-3 animate-spin" /> locating
            </Badge>
          ) : (
            loc && (
              <Badge variant="outline" className="gap-1.5 text-xs">
                <MapPin className="h-3 w-3" />
                {loc.lat.toFixed(3)}, {loc.lng.toFixed(3)}
              </Badge>
            )
          )}
        </header>

        <div className="space-y-4 p-5">
          <SearchBar value={picked} onPick={setPicked} />

          <div className="flex flex-wrap items-center gap-3">
            <div className="flex items-center gap-2">
              <Label htmlFor="radius" className="text-xs text-muted-foreground">
                radius
              </Label>
              <Select value={String(radius)} onValueChange={(v) => setRadius(Number(v))}>
                <SelectTrigger id="radius" className="h-8 w-[96px] text-xs">
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  {radiusOptions.map((o) => (
                    <SelectItem key={o.v} value={String(o.v)}>
                      {o.label}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>

            <div className="flex items-center gap-2">
              <Switch id="stale" checked={includeStale} onCheckedChange={setIncludeStale} />
              <Label htmlFor="stale" className="text-xs text-muted-foreground">
                show out of stock
              </Label>
            </div>

            {picked && loc && (
              <div className="ml-auto">
                <AlertDialog
                  drugId={picked.id}
                  drugName={picked.name}
                  lat={loc.lat}
                  lng={loc.lng}
                  radiusM={radius}
                />
              </div>
            )}
          </div>

          {picked && counts && (
            <div className="flex flex-wrap gap-2 text-xs">
              <Badge variant="success">{counts.inStock} in stock</Badge>
              <Badge variant="warning">{counts.limited} limited</Badge>
              {includeStale && <Badge variant="destructive">{counts.out} out</Badge>}
            </div>
          )}

          <Separator />
        </div>

        <div className="flex-1 overflow-y-auto px-5 pb-5">
          {!picked && (
            <EmptyState pharmacies={pharmacies} />
          )}

          {picked && loading && (
            <div className="flex items-center justify-center py-12 text-sm text-muted-foreground">
              <Loader2 className="mr-2 h-4 w-4 animate-spin" />
              checking stock…
            </div>
          )}

          {picked && !loading && rows && rows.length > 0 && (
            <ul className="space-y-2">
              {rows.map((r) => (
                <li key={r.pharmacy.id}>
                  <StockCard row={r} />
                </li>
              ))}
            </ul>
          )}

          {picked && !loading && rows && rows.length === 0 && (
            <div className="rounded-lg border border-dashed p-6 text-center">
              <Sparkles className="mx-auto mb-2 h-5 w-5 text-muted-foreground" />
              <div className="text-sm font-medium">no recent reports for {picked.name}</div>
              <div className="mt-1 text-xs text-muted-foreground">
                be the first. scroll down and tap a pharmacy.
              </div>
            </div>
          )}

          {picked && pharmacies.length > 0 && (
            <div className="mt-6">
              <ReportPanel
                drugId={picked.id}
                pharmacies={pharmacies}
                onSubmitted={() => {
                  toast.success("thanks, report saved");
                  reload();
                }}
              />
            </div>
          )}
        </div>
      </aside>

      <section className="relative hidden md:block">
        {loc && <MapView center={loc} pharmacies={pharmacies} rows={rows ?? []} />}
      </section>
    </main>
  );
}

function EmptyState({ pharmacies }: { pharmacies: Pharmacy[] }) {
  return (
    <div>
      <div className="rounded-lg border border-dashed p-6">
        <div className="mb-1 text-sm font-medium">start by picking a medicine</div>
        <p className="text-xs text-muted-foreground text-balance">
          type a brand or generic name. we'll show you every pharmacy nearby that reported having
          it recently, freshest first.
        </p>
      </div>
      {pharmacies.length > 0 && (
        <div className="mt-4">
          <div className="mb-2 text-xs uppercase tracking-wide text-muted-foreground">
            pharmacies nearby
          </div>
          <ul className="space-y-2">
            {pharmacies.slice(0, 15).map((p) => (
              <li key={p.id} className="rounded-lg border p-3">
                <div className="flex items-center gap-2">
                  <span className="font-medium truncate">{p.name}</span>
                  {p.verified && (
                    <Badge variant="outline" className="h-5 text-[10px]">
                      verified
                    </Badge>
                  )}
                </div>
                <div className="text-xs text-muted-foreground truncate">
                  {[p.address, p.city].filter(Boolean).join(", ")}
                </div>
                {p.distance_m != null && (
                  <div className="text-[11px] text-muted-foreground mt-1">
                    {Math.round(p.distance_m)} m
                  </div>
                )}
              </li>
            ))}
          </ul>
        </div>
      )}
    </div>
  );
}
