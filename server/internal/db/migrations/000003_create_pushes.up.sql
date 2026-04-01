CREATE TABLE pushes (
  id               UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id          UUID        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  source_device_id UUID        REFERENCES devices(id) ON DELETE SET NULL,
  target_device_id UUID        REFERENCES devices(id) ON DELETE SET NULL,
  type             TEXT        NOT NULL CHECK (type IN ('note', 'link', 'file')),
  title            TEXT,
  body             TEXT,
  url              TEXT,
  file_name        TEXT,
  file_type        TEXT,
  file_s3_key      TEXT,
  file_size        BIGINT,
  delivered        BOOLEAN     NOT NULL DEFAULT false,
  created_at       TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_pushes_user_id  ON pushes(user_id, created_at DESC);
CREATE INDEX idx_pushes_delivery ON pushes(target_device_id, delivered);
