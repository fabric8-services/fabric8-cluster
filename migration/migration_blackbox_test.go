package migration_test

import (
	"database/sql"
	"fmt"
	"os"
	"testing"

	"github.com/fabric8-services/fabric8-cluster/migration"
	"github.com/fabric8-services/fabric8-common/gormsupport"
	migrationsupport "github.com/fabric8-services/fabric8-common/migration"
	"github.com/fabric8-services/fabric8-common/resource"

	"github.com/jinzhu/gorm"
	_ "github.com/lib/pq"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

const (
	dbName      = "test"
	defaultHost = "localhost"
	defaultPort = "5434"
)

type MigrationTestSuite struct {
	suite.Suite
}

const (
	databaseName = "test"
)

var (
	sqlDB *sql.DB
	host  string
	port  string
)

func TestMigration(t *testing.T) {
	suite.Run(t, new(MigrationTestSuite))
}

func (s *MigrationTestSuite) SetupTest() {
	resource.Require(s.T(), resource.Database)
	host = os.Getenv("F8_POSTGRES_HOST")
	if host == "" {
		host = defaultHost
	}
	port = os.Getenv("F8_POSTGRES_PORT")
	if port == "" {
		port = defaultPort
	}
	dbConfig := fmt.Sprintf("host=%s port=%s user=postgres password=mysecretpassword sslmode=disable connect_timeout=5", host, port)
	db, err := sql.Open("postgres", dbConfig)
	require.NoError(s.T(), err, "cannot connect to database: %s", dbName)
	defer db.Close()
	_, err = db.Exec("DROP DATABASE " + dbName)
	if err != nil && !gormsupport.IsInvalidCatalogName(err) {
		require.NoError(s.T(), err, "failed to drop database '%s'", dbName)
	}
	_, err = db.Exec("CREATE DATABASE " + dbName)
	require.NoError(s.T(), err, "failed to create database '%s'", dbName)
}

func (s *MigrationTestSuite) TestMigrate() {
	dbConfig := fmt.Sprintf("host=%s port=%s user=postgres password=mysecretpassword dbname=%s sslmode=disable connect_timeout=5",
		host, port, dbName)
	var err error
	sqlDB, err = sql.Open("postgres", dbConfig)
	require.NoError(s.T(), err, "cannot connect to DB '%s'", dbName)
	defer sqlDB.Close()
	gormDB, err := gorm.Open("postgres", dbConfig)
	require.NoError(s.T(), err, "cannot connect to DB '%s'", dbName)
	defer gormDB.Close()
	s.T().Run("testMigration001Cluster", testMigration001Cluster)
	s.T().Run("testMigration002ClusterOnDeleteCascade", testMigration002ClusterOnDeleteCascade)
}

func testMigration001Cluster(t *testing.T) {
	err := migrationsupport.Migrate(sqlDB, databaseName, migration.Steps()[:2])
	require.NoError(t, err)
	t.Run("insert cluster", func(t *testing.T) {
		_, err := sqlDB.Exec(`INSERT INTO cluster (cluster_id, created_at, updated_at, name, url, console_url,
                     metrics_url, logging_url, app_dns, sa_token, sa_username, token_provider_id, 
                     auth_client_id, auth_client_secret, auth_default_scope, type)
			VALUES ('0b3d3751-69a7-4981-bf6f-63cd08b723af', now(), now(), 'stage', 'https://api.cluster.com', 'https://console.cluster.com',
			        'https://metrics.cluster.com', 'https://login.cluster.com', 'https://app.cluster.com', 'sometoken', 'dssas-sre', 'pr-id',
			        'client-id', 'cleint-scr', 'somescope', 'OSD')`)
		require.NoError(t, err)
	})
	t.Run("insert identity cluster", func(t *testing.T) {
		_, err := sqlDB.Exec(`INSERT INTO identity_cluster (identity_id, cluster_id, created_at, updated_at)
			VALUES (uuid_generate_v4(), '0b3d3751-69a7-4981-bf6f-63cd08b723af', now(), now())`)
		require.NoError(t, err)
	})
	t.Run("insert identity cluster fail for unknown cluster ID", func(t *testing.T) {
		_, err := sqlDB.Exec(`INSERT INTO identity_cluster (identity_id, cluster_id, created_at, updated_at)
			VALUES (uuid_generate_v4(), uuid_generate_v4(), now(), now())`)
		require.Error(t, err)
	})
}

func testMigration002ClusterOnDeleteCascade(t *testing.T) {
	err := migrationsupport.Migrate(sqlDB, databaseName, migration.Steps()[:3])
	require.NoError(t, err)
	t.Run("insert cluster", func(t *testing.T) {
		_, err := sqlDB.Exec(`INSERT INTO cluster (cluster_id, created_at, updated_at, name, url, console_url,
                     metrics_url, logging_url, app_dns, sa_token, sa_username, token_provider_id, 
                     auth_client_id, auth_client_secret, auth_default_scope, type)
			VALUES ('c55a6344-95d5-455e-ad8f-92c6783dbd4d', now(), now(), 'stage', 'https://api.cluster.com', 'https://console.cluster.com',
			        'https://metrics.cluster.com', 'https://login.cluster.com', 'https://app.cluster.com', 'sometoken', 'dssas-sre', 'pr-id',
			        'client-id', 'cleint-scr', 'somescope', 'OSD')`)
		require.NoError(t, err)
	})
	t.Run("insert identity cluster", func(t *testing.T) {
		_, err := sqlDB.Exec(`INSERT INTO identity_cluster (identity_id, cluster_id, created_at, updated_at)
			VALUES (uuid_generate_v4(), 'c55a6344-95d5-455e-ad8f-92c6783dbd4d', now(), now())`)
		require.NoError(t, err)
	})
	t.Run("insert identity cluster fail for unknown cluster ID", func(t *testing.T) {
		_, err := sqlDB.Exec(`INSERT INTO identity_cluster (identity_id, cluster_id, created_at, updated_at)
			VALUES (uuid_generate_v4(), uuid_generate_v4(), now(), now())`)
		require.Error(t, err)
	})
	t.Run("identity cluster on delete cascade", func(t *testing.T) {
		// Identity cluster available
		r, err := sqlDB.Exec(`SELECT * FROM identity_cluster WHERE cluster_id = 'c55a6344-95d5-455e-ad8f-92c6783dbd4d'`)
		require.NoError(t, err)
		rows, err := r.RowsAffected()
		require.NoError(t, err)
		require.Equal(t, int64(1), rows)
		// Delete cluster to make sure cascade delete works
		_, err = sqlDB.Exec(`DELETE FROM cluster WHERE cluster_id = 'c55a6344-95d5-455e-ad8f-92c6783dbd4d'`)
		require.NoError(t, err)
		// Identity cluster is gone
		r, err = sqlDB.Exec(`SELECT * FROM identity_cluster WHERE cluster_id = 'c55a6344-95d5-455e-ad8f-92c6783dbd4d'`)
		require.NoError(t, err)
		rows, err = r.RowsAffected()
		require.NoError(t, err)
		require.Equal(t, int64(0), rows)
	})
}
