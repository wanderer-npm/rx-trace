# rx-trace

crowd-sourced map of which pharmacies near you actually have a given medicine in stock.

the problem: you get a prescription, walk to a pharmacy, they're out. you walk
to the next one. same thing. happens to millions of people in india, pakistan,
brazil, most of africa, anywhere outside rich-country pharmacy chains with
their own apps. nobody's built a public tool that tracks this.

so: crowd-source it. someone finds dolo 650 in stock, taps "in stock".
anyone else searching for dolo 650 nearby sees a green pin on that shop with a
timestamp. that's it.

the data model is country-agnostic. we ship with an indian drug seed because
that's what we tested on. scroll to "using it outside india" below.

## stack

- go backend (chi router, pgx)
- postgres 16 with postgis
- next.js 14 frontend, shadcn/ui, maplibre
- resend for email alerts

no orm, no sqlc, no framework magic. handlers + sql.

## running

```
cp .env.example .env
docker compose up -d
open http://localhost:3000
```

first boot creates the schema, seeds ~100 indian drug brands and 15 sample
pharmacies. `.env` is optional, email alerts just won't work without a
resend key.

## loading real pharmacy data

we pull from openstreetmap via overpass api. no key needed.

```
docker compose exec api rx-trace scrape --source overpass --city delhi
docker compose exec api rx-trace scrape --source overpass --city kolkata
```

built-in cities: delhi, mumbai, bengaluru, hyderabad, chennai, kolkata, pune,
ahmedabad, jaipur, lucknow.

for anywhere else, pass a bbox as `south,west,north,east`:

```
docker compose exec api rx-trace scrape --source overpass --bbox 40.5,-74.3,40.9,-73.7
```

that's new york. find a bbox for any city at https://boundingbox.klokantech.com.

overpass rate-limits hard. if you get 403, wait a few minutes, try again.

## loading shortage data

us fda publishes a drug shortages feed. india's cdsco doesn't. pull fda with:

```
docker compose exec api rx-trace scrape --source fda
```

## email alerts

people subscribe to "tell me if <drug> shows up within 5km of <coords>".
nothing sends automatically, you run the dispatcher on a cron:

```
docker compose exec api rx-trace dispatch-alerts
```

good cadence is every 15 minutes. the dispatcher only considers reports newer
than each alert's `last_fired_at`, so a single run won't spam people twice.

## the endpoint that matters

```
GET /v1/find?drug_id=3&lat=22.564&lng=88.369&radius_m=5000
```

returns the most recent report per pharmacy within the radius, sorted
in_stock > limited > out, then by distance. this is what the ui calls on
every search.

## full api

```
GET    /healthz
GET    /v1/drugs?q=dolo
GET    /v1/drugs/:id

GET    /v1/pharmacies/nearby?lat=...&lng=...&r=5000
POST   /v1/pharmacies

POST   /v1/reports           { drug_id, pharmacy_id, status, note? }
GET    /v1/reports/recent?drug_id=...

GET    /v1/find?drug_id=...&lat=...&lng=...&radius_m=5000

GET    /v1/shortages

POST   /v1/alerts            { drug_id, lat, lng, radius_m, channel, target }
GET    /v1/alerts?target=you@example.com
DELETE /v1/alerts/:id?target=you@example.com
```

statuses: `in_stock`, `limited`, `out`.
channels: `email`, `push`, `webhook` (only `email` is implemented today).

## using it outside india

three things are india-flavoured. none are structural.

1. the seed drug list (`db/seed.sql`) has ~100 indian brand names. you can
   either keep it and add your own, or replace it. for the us, import rxnorm.
   for the uk, bnf. for the eu, the ema product database.
2. `scrapers/overpass.go` has bboxes only for indian cities. add more, one
   line each.
3. default map center on geolocation-denied is delhi. change it in
   `web/src/app/page.tsx`.

the schema, api, frontend, alerts, scrapers work globally. postgis doesn't
care what continent your points are on.

## rate limits

each report is keyed by sha256(ip + user-agent). limits:

- 5 per minute
- 20 per hour
- 100 per day

same reporter posting the same (drug, pharmacy) pair inside 2 minutes returns 409.

## contributing

prs welcome, keep them small. there's lots to do:

- more city bboxes, especially outside india
- rxnorm / bnf / ema ingest
- anti-spam (phone otp instead of ip-based buckets)
- push notifications (firebase or web push)
- a proper pwa manifest so it installs on phones
- scraping 1mg / apollo store-locators for verified chain pharmacies

no cla. open an issue first if it's a big change.

## license

mit. see `LICENSE`.
