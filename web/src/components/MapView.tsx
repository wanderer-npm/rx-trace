"use client";

import "maplibre-gl/dist/maplibre-gl.css";

import { useMemo } from "react";
import MapGL, { Marker, NavigationControl, type ViewState } from "react-map-gl/maplibre";

import type { FindRow, Pharmacy } from "@/lib/api";

type Props = {
  center: { lat: number; lng: number };
  pharmacies?: Pharmacy[];
  rows?: FindRow[];
};

const STYLE = "https://tiles.openfreemap.org/styles/dark";

const statusFill: Record<string, string> = {
  in_stock: "hsl(142 71% 45%)",
  limited: "hsl(38 92% 50%)",
  out: "hsl(0 70% 55%)",
};

export default function MapView({ center, pharmacies = [], rows = [] }: Props) {
  const initial: Partial<ViewState> = useMemo(
    () => ({ latitude: center.lat, longitude: center.lng, zoom: 13 }),
    [center.lat, center.lng],
  );

  const statusById = new Map(rows.map((r) => [r.pharmacy.id, r.status]));
  const markers = rows.length > 0 ? rows.map((r) => r.pharmacy) : pharmacies;

  return (
    <MapGL initialViewState={initial} mapStyle={STYLE} attributionControl={false}>
      <NavigationControl position="top-right" showCompass={false} />
      <Marker latitude={center.lat} longitude={center.lng}>
        <div className="relative">
          <div className="absolute inset-0 h-3.5 w-3.5 animate-ping rounded-full bg-blue-400/40" />
          <div className="relative h-3.5 w-3.5 rounded-full bg-blue-500 ring-2 ring-background" />
        </div>
      </Marker>
      {markers.map((p) => {
        const status = statusById.get(p.id);
        return (
          <Marker key={p.id} latitude={p.lat} longitude={p.lng}>
            <div
              className="h-3 w-3 rounded-full ring-2 ring-background shadow-lg"
              style={{ background: status ? statusFill[status] : "hsl(var(--muted-foreground))" }}
            />
          </Marker>
        );
      })}
    </MapGL>
  );
}
