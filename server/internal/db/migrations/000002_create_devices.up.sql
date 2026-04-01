CREATE TABLE devices (
  id         UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id    UUID        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  name       TEXT        NOT NULL,
  type       TEXT        NOT NULL,
  last_seen  TIMESTAMPTZ,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_devices_user_id ON devices(user_id);
