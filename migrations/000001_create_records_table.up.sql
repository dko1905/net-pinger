CREATE TABLE records (
  id TEXT PRIMARY KEY NOT NULL,
  ts TEXT NOT NULL,
  failure INTEGER NOT NULL, -- 1=true 0=false
  description TEXT NOT NULL
);
