-- Store sa_token_encrypted for cluster
ALTER TABLE cluster ADD COLUMN sa_token_encrypted boolean NOT NULL DEFAULT FALSE;