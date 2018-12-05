-- Store sa_token_encrypted for cluster
ALTER TABLE cluster ADD COLUMN sa_token_encrypted boolean DEFAULT FALSE;