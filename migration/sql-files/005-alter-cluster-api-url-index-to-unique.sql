DROP INDEX idx_cluster_url;
CREATE UNIQUE INDEX idx_cluster_url ON cluster (url);