# ICMP Measurements

## Description

Parse fping output and store results in timescaledb.

## Install

### TSDB

```sql
CREATE ROLE darkping WITH login password '{random password}';
CREATE DATABASE darkping WITH owner darkping;
\c darkping
CREATE EXTENSION IF NOT EXISTS timescaledb CASCADE;

CREATE TABLE measurement_icmp (
  time TIMESTAMPTZ NOT NULL,
  target TEXT NOT NULL,
  sent integer NOT NULL,
  recv integer NOT NULL,
  loss integer NOT NULL,
  min NUMERIC(8,3),
  avg NUMERIC(8,3),
  max NUMERIC(8,3)
);

ALTER TABLE measurement_icmp owner TO darkping;
CREATE INDEX measurement_icmp_target_idx ON measurement_icmp USING btree(target);

SELECT create_hypertable(
  'measurement_icmp', 'time',
  chunk_time_interval => interval '1 day',
  migrate_data => true
);
```
