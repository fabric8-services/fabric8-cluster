CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

CREATE TABLE cluster (
    cluster_id uuid primary key DEFAULT uuid_generate_v4() NOT NULL,
    created_at timestamp with time zone,
    updated_at timestamp with time zone,
    deleted_at timestamp with time zone,
    name text,
    url text NOT NULL CHECK (url <> ''),
    console_url text,
    metrics_url text,
    logging_url text,
    app_dns text NOT NULL CHECK (app_dns <> ''),
    sa_token text,
    sa_username text,
    token_provider_id text,
    auth_client_id text,
    auth_client_secret text,
    auth_default_scope text,
    type text
);

CREATE TABLE identity_cluster (
    identity_id uuid NOT NULL,
    cluster_id uuid references cluster(cluster_id),
    created_at timestamp with time zone,
    updated_at timestamp with time zone,
    deleted_at timestamp with time zone,
    PRIMARY KEY(identity_id, cluster_id)
);


CREATE INDEX identity_cluster_identity_id_idx ON identity_cluster USING BTREE (identity_id);