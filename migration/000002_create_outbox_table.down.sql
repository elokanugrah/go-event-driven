-- migration/000002_create_outbox_table.down.sql
DROP TABLE IF EXISTS "outbox_events";
