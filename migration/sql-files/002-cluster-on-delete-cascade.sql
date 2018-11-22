-- Cluster contains sensitive information. No soft deletes.
ALTER TABLE cluster DROP COLUMN deleted_at;

-- Drop the cluster_id foreign key on the identity_cluster that references the cluster.
ALTER TABLE identity_cluster DROP CONSTRAINT identity_cluster_cluster_id_fkey;

-- Add the foreign key back in to add ON DELETE CASCADE
ALTER TABLE identity_cluster
  ADD CONSTRAINT "identity_cluster_cluster_id_fkey"
  FOREIGN KEY (cluster_id)
  REFERENCES cluster(cluster_id)
  ON DELETE CASCADE;
