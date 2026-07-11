-- migration/000002_create_outbox_table.up.sql
CREATE TABLE "outbox_events" (
  "id" bigserial PRIMARY KEY,
  "event_type" varchar NOT NULL,
  "payload" jsonb NOT NULL,
  "status" varchar NOT NULL DEFAULT 'pending',
  "created_at" timestamptz NOT NULL DEFAULT (now()),
  "processed_at" timestamptz
);

CREATE INDEX "idx_outbox_events_status_id" ON "outbox_events" ("status", "id");
