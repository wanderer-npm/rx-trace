import { MapPin, Phone, Clock } from "lucide-react";
import { Badge } from "@/components/ui/badge";
import { cn } from "@/lib/utils";
import type { FindRow } from "@/lib/api";

function freshness(min: number) {
  if (min < 1) return "just now";
  if (min < 60) return `${min}m ago`;
  const h = Math.floor(min / 60);
  if (h < 24) return `${h}h ago`;
  return `${Math.floor(h / 24)}d ago`;
}

const variant = {
  in_stock: "success",
  limited: "warning",
  out: "destructive",
} as const;

const label = {
  in_stock: "in stock",
  limited: "limited",
  out: "out",
} as const;

export function StockCard({ row }: { row: FindRow }) {
  const p = row.pharmacy;
  return (
    <div className="group rounded-lg border bg-card p-4 transition-colors hover:bg-accent/30">
      <div className="flex items-start justify-between gap-3">
        <div className="min-w-0">
          <div className="flex items-center gap-2">
            <span className="font-medium truncate">{p.name}</span>
            {p.verified && (
              <Badge variant="outline" className="h-5 text-[10px]">
                verified
              </Badge>
            )}
          </div>
          <div className="mt-1 flex items-center gap-2 text-xs text-muted-foreground">
            <MapPin className="h-3 w-3 shrink-0" />
            <span className="truncate">
              {[p.address, p.city].filter(Boolean).join(", ") || "no address on file"}
            </span>
          </div>
        </div>
        <Badge variant={variant[row.status]}>{label[row.status]}</Badge>
      </div>

      <div className="mt-3 flex items-center gap-4 text-xs text-muted-foreground">
        {p.distance_m != null && (
          <span>{Math.round(p.distance_m)} m away</span>
        )}
        <span className="flex items-center gap-1">
          <Clock className="h-3 w-3" />
          {freshness(row.freshness_min)}
        </span>
        {p.phone && (
          <a
            href={`tel:${p.phone}`}
            className="ml-auto flex items-center gap-1 text-foreground hover:underline"
          >
            <Phone className="h-3 w-3" />
            call
          </a>
        )}
      </div>
    </div>
  );
}
