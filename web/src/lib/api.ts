const BASE = process.env.NEXT_PUBLIC_API_URL ?? "http://localhost:8080";

export type Drug = {
  id: number;
  name: string;
  generic?: string;
  form?: string;
  strength?: string;
  tags?: string[];
};

export type Pharmacy = {
  id: number;
  name: string;
  address?: string;
  city?: string;
  phone?: string;
  lat: number;
  lng: number;
  distance_m?: number;
  verified: boolean;
};

export type FindRow = {
  pharmacy: Pharmacy;
  status: "in_stock" | "out" | "limited";
  reported_at: string;
  freshness_min: number;
};

async function json<T>(r: Response): Promise<T> {
  if (!r.ok) throw new Error(`${r.status} ${await r.text()}`);
  return r.json();
}

export async function searchDrugs(q: string) {
  return json<Drug[]>(await fetch(`${BASE}/v1/drugs?q=${encodeURIComponent(q)}`));
}

export async function nearbyPharmacies(lat: number, lng: number, r = 5000) {
  return json<Pharmacy[]>(await fetch(`${BASE}/v1/pharmacies/nearby?lat=${lat}&lng=${lng}&r=${r}`));
}

export async function findInStock(args: {
  drug_id: number;
  lat: number;
  lng: number;
  radius_m?: number;
  include_stale?: boolean;
}) {
  const p = new URLSearchParams({
    drug_id: String(args.drug_id),
    lat: String(args.lat),
    lng: String(args.lng),
    radius_m: String(args.radius_m ?? 5000),
    include_stale: String(args.include_stale ?? false),
  });
  return json<FindRow[]>(await fetch(`${BASE}/v1/find?${p}`));
}

export async function submitReport(body: {
  drug_id: number;
  pharmacy_id: number;
  status: "in_stock" | "out" | "limited";
  note?: string;
}) {
  return json<{ id: number }>(
    await fetch(`${BASE}/v1/reports`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(body),
    }),
  );
}

export async function createAlert(body: {
  drug_id: number;
  lat: number;
  lng: number;
  radius_m: number;
  channel: "email" | "push" | "webhook";
  target: string;
}) {
  return json<{ id: number }>(
    await fetch(`${BASE}/v1/alerts`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(body),
    }),
  );
}
