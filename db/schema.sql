create extension if not exists postgis;
create extension if not exists pg_trgm;
create extension if not exists unaccent;

create table if not exists drugs (
    id bigserial primary key,
    name text not null,
    generic text,
    form text,
    strength text,
    atc text,
    tags text[] default '{}',
    created_at timestamptz default now(),
    unique(name, form, strength)
);

create index if not exists drugs_name_trgm on drugs using gin (name gin_trgm_ops);
create index if not exists drugs_generic_trgm on drugs using gin (generic gin_trgm_ops) where generic is not null;

create table if not exists pharmacies (
    id bigserial primary key,
    osm_id bigint unique,
    name text not null,
    phone text,
    address text,
    city text,
    state text,
    country text not null default 'IN',
    location geography(point, 4326) not null,
    hours jsonb,
    verified boolean default false,
    added_at timestamptz default now()
);

create index if not exists pharmacies_loc on pharmacies using gist (location);
create index if not exists pharmacies_city on pharmacies (lower(city));
create index if not exists pharmacies_name_trgm on pharmacies using gin (name gin_trgm_ops);

create table if not exists reports (
    id bigserial primary key,
    drug_id bigint not null references drugs(id) on delete cascade,
    pharmacy_id bigint not null references pharmacies(id) on delete cascade,
    status text not null check (status in ('in_stock','out','limited')),
    price_paise int,
    reporter_hash text,
    note text,
    reported_at timestamptz default now()
);

create index if not exists reports_pharmacy_drug on reports (pharmacy_id, drug_id, reported_at desc);
create index if not exists reports_drug_recent on reports (drug_id, reported_at desc);
create index if not exists reports_hash_time on reports (reporter_hash, reported_at desc);

create table if not exists shortages (
    id bigserial primary key,
    drug_id bigint references drugs(id) on delete cascade,
    source text not null,
    severity text,
    started_at date,
    expected_end date,
    url text,
    raw jsonb,
    ingested_at timestamptz default now(),
    unique(drug_id, source, started_at)
);

create index if not exists shortages_drug on shortages (drug_id);

create table if not exists alternatives (
    drug_id bigint references drugs(id) on delete cascade,
    alt_drug_id bigint references drugs(id) on delete cascade,
    relationship text,
    confidence real,
    primary key (drug_id, alt_drug_id)
);

create table if not exists sources (
    key text primary key,
    last_run_at timestamptz,
    last_status text,
    records_seen int default 0
);

create table if not exists alerts (
    id bigserial primary key,
    drug_id bigint not null references drugs(id) on delete cascade,
    lat double precision not null,
    lng double precision not null,
    radius_m int not null default 5000,
    channel text not null check (channel in ('email','push','webhook')),
    target text not null,
    created_at timestamptz default now(),
    last_fired_at timestamptz,
    active boolean default true
);

create index if not exists alerts_drug_active on alerts (drug_id) where active;

create or replace view current_stock as
select distinct on (r.pharmacy_id, r.drug_id)
    r.drug_id,
    r.pharmacy_id,
    r.status,
    r.reported_at,
    p.name as pharmacy_name,
    p.location
from reports r
join pharmacies p on p.id = r.pharmacy_id
order by r.pharmacy_id, r.drug_id, r.reported_at desc;
