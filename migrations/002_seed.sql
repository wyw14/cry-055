INSERT INTO schema_migrations(version) VALUES ('001_init') ON CONFLICT DO NOTHING;
INSERT INTO schema_migrations(version) VALUES ('002_seed') ON CONFLICT DO NOTHING;
