-- URL shortener core table
CREATE TABLE IF NOT EXISTS url_records (
  id          UUID PRIMARY KEY,
  code        TEXT NOT NULL UNIQUE,
  long_url    TEXT NOT NULL UNIQUE,
  short_url   TEXT NOT NULL,
  created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);
