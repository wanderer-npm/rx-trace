"use client";

import { useState } from "react";
import { Bell, Loader2 } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Popover, PopoverContent, PopoverTrigger } from "@/components/ui/popover";
import { Label } from "@/components/ui/label";
import { createAlert } from "@/lib/api";

type Props = {
  drugId: number;
  drugName: string;
  lat: number;
  lng: number;
  radiusM: number;
};

export function AlertDialog({ drugId, drugName, lat, lng, radiusM }: Props) {
  const [email, setEmail] = useState("");
  const [busy, setBusy] = useState(false);
  const [open, setOpen] = useState(false);
  const [saved, setSaved] = useState(false);

  async function submit() {
    if (!email) return;
    setBusy(true);
    try {
      await createAlert({
        drug_id: drugId,
        lat,
        lng,
        radius_m: radiusM,
        channel: "email",
        target: email,
      });
      setSaved(true);
      setTimeout(() => setOpen(false), 900);
    } catch (e: any) {
      alert(e.message);
    } finally {
      setBusy(false);
    }
  }

  return (
    <Popover open={open} onOpenChange={setOpen}>
      <PopoverTrigger asChild>
        <Button variant="outline" size="sm" className="gap-1.5">
          <Bell className="h-3.5 w-3.5" />
          alert me
        </Button>
      </PopoverTrigger>
      <PopoverContent className="w-80" align="end">
        {saved ? (
          <div className="py-4 text-center text-sm">
            <div className="mb-1 font-medium">saved.</div>
            <div className="text-muted-foreground">
              you'll get an email when {drugName} shows up within {Math.round(radiusM / 1000)} km.
            </div>
          </div>
        ) : (
          <div className="space-y-3">
            <div>
              <div className="font-medium text-sm">alert for {drugName}</div>
              <div className="text-xs text-muted-foreground">
                {Math.round(radiusM / 1000)} km radius from this spot
              </div>
            </div>
            <div className="space-y-1.5">
              <Label htmlFor="alert-email" className="text-xs">
                email
              </Label>
              <Input
                id="alert-email"
                type="email"
                value={email}
                onChange={(e) => setEmail(e.target.value)}
                placeholder="you@example.com"
                onKeyDown={(e) => e.key === "Enter" && submit()}
              />
            </div>
            <Button onClick={submit} disabled={!email || busy} className="w-full">
              {busy && <Loader2 className="mr-2 h-3.5 w-3.5 animate-spin" />}
              subscribe
            </Button>
          </div>
        )}
      </PopoverContent>
    </Popover>
  );
}
