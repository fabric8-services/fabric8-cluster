package migration_test

import (
	"database/sql"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/fabric8-services/fabric8-cluster/migration"
	"github.com/fabric8-services/fabric8-common/gormsupport"
	migrationsupport "github.com/fabric8-services/fabric8-common/migration"
	"github.com/fabric8-services/fabric8-common/resource"

	"github.com/jinzhu/gorm"
	_ "github.com/lib/pq"
	"github.com/stretchr/testify/assert"
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
	sqlDB   *sql.DB
	host    string
	port    string
	dialect gorm.Dialect
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

	dialect = gormDB.Dialect()
	dialect.SetDB(sqlDB)

	s.T().Run("testMigration001Cluster", testMigration001Cluster)
	s.T().Run("testMigration002ClusterOnDeleteCascade", testMigration002ClusterOnDeleteCascade)
	s.T().Run("testMigration003UniqueIndexOnClusterApiUrl", testMigration003UniqueIndexOnClusterApiUrl)
	s.T().Run("testMigration004AddCapacityExhaustedToCluster", testMigration004AddCapacityExhaustedToCluster)
	s.T().Run("testMigration005AlterClusterAPIURLIndexToUnique", testMigration005AlterClusterAPIURLIndexToUnique)
	s.T().Run("testMigration006AddSaTokenEncryptedToCluster", testMigration006AddSaTokenEncryptedToCluster)
	s.T().Run("testMigration007AddTrailingSlash", testMigration007AddTrailingSlash)
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

func testMigration003UniqueIndexOnClusterApiUrl(t *testing.T) {
	err := migrationsupport.Migrate(sqlDB, databaseName, migration.Steps()[:4])
	require.NoError(t, err)

	assert.True(t, dialect.HasIndex("cluster", "idx_cluster_url"))
}

func testMigration004AddCapacityExhaustedToCluster(t *testing.T) {
	err := migrationsupport.Migrate(sqlDB, databaseName, migration.Steps()[:5])
	require.NoError(t, err)

	assert.True(t, dialect.HasColumn("cluster", "capacity_exhausted"))

	_, err = sqlDB.Exec(`INSERT INTO cluster (cluster_id, created_at, updated_at, name, url, console_url,
                     metrics_url, logging_url, app_dns, sa_token, sa_username, token_provider_id,
                     auth_client_id, auth_client_secret, auth_default_scope, type)
			VALUES ('1b3d3751-69a7-4981-bf6f-63cd08b723af', now(), now(), 'exhausted', 'https://exhausted.api.cluster.com', 'https://console.cluster.com',
			        'https://metrics.cluster.com', 'https://login.cluster.com', 'https://app.cluster.com', 'sometoken', 'dssas-sre', 'pr-id',
			        'client-id', 'cleint-scr', 'somescope', 'OSD')`)
	require.NoError(t, err)

	// check if ALL the existing rows & new rows have the default value
	rows, err := sqlDB.Query("SELECT capacity_exhausted FROM cluster")
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	for rows.Next() {
		var capacity_exhausted bool
		err = rows.Scan(&capacity_exhausted)
		require.NoError(t, err)
		assert.False(t, capacity_exhausted)
	}
}

func testMigration005AlterClusterAPIURLIndexToUnique(t *testing.T) {
	err := migrationsupport.Migrate(sqlDB, databaseName, migration.Steps()[:6])
	require.NoError(t, err)

	assert.True(t, dialect.HasIndex("cluster", "idx_cluster_url"))

	_, err = sqlDB.Exec(`INSERT INTO cluster (cluster_id, created_at, updated_at, name, url, console_url,
                     metrics_url, logging_url, app_dns, sa_token, sa_username, token_provider_id,
                     auth_client_id, auth_client_secret, auth_default_scope, type)
			VALUES ('2c4e4852-69a7-4981-bf6f-63cd08b723af', now(), now(), 'exhausted', 'https://unique.api.cluster.com', 'https://console.cluster.com',
			        'https://metrics.cluster.com', 'https://login.cluster.com', 'https://app.cluster.com', 'sometoken', 'dssas-sre', 'pr-id',
			        'client-id', 'cleint-scr', 'somescope', 'OSD')`)
	require.NoError(t, err)

	// add cluster with same url https://unique.api.cluster.com to verify uniqueness
	_, err = sqlDB.Exec(`INSERT INTO cluster (cluster_id, created_at, updated_at, name, url, console_url,
                     metrics_url, logging_url, app_dns, sa_token, sa_username, token_provider_id,
                     auth_client_id, auth_client_secret, auth_default_scope, type)
			VALUES ('2aye4862-69a7-4981-bf6f-63cd08b723af', now(), now(), 'exhausted', 'https://unique.api.cluster.com', 'https://console.cluster.com',
			        'https://metrics.cluster.com', 'https://login.cluster.com', 'https://app.cluster.com', 'sometoken', 'dssas-sre', 'pr-id',
			        'client-id', 'cleint-scr', 'somescope', 'OSD')`)

	require.Error(t, err)
}

func testMigration006AddSaTokenEncryptedToCluster(t *testing.T) {
	err := migrationsupport.Migrate(sqlDB, databaseName, migration.Steps()[:7])
	require.NoError(t, err)

	assert.True(t, dialect.HasColumn("cluster", "sa_token_encrypted"))

	_, err = sqlDB.Exec(`INSERT INTO cluster (cluster_id, created_at, updated_at, name, url, console_url,
                     metrics_url, logging_url, app_dns, sa_token, sa_username, sa_token_encrypted, token_provider_id,
                     auth_client_id, auth_client_secret, auth_default_scope, type)
			VALUES ('3eb0fb9a-b7d3-479f-9ec0-e5f93c7b1e53', now(), now(), 'exhausted', 'https://token-encrypted.api.cluster.com', 'https://console.cluster.com',
			        'https://metrics.cluster.com', 'https://login.cluster.com', 'https://app.cluster.com', 'sometoken', 'dssas-sre', false, 'pr-id',
			        'client-id', 'cleint-scr', 'somescope', 'OSD')`)
	require.NoError(t, err)

	// check if ALL the existing rows & new rows have the default value
	rows, err := sqlDB.Query("SELECT sa_token_encrypted FROM cluster")
	require.NoError(t, err)

	defer rows.Close()
	for rows.Next() {
		var sa_token_encrypted bool
		err = rows.Scan(&sa_token_encrypted)
		require.NoError(t, err)
		assert.False(t, sa_token_encrypted)
	}
}

func testMigration007AddTrailingSlash(t *testing.T) {
	// first, migrate to step 6
	err := migrationsupport.Migrate(sqlDB, databaseName, migration.Steps()[:7])
	require.NoError(t, err)
	// then insert some records with *some* url having no trailing slash
	_, err = sqlDB.Exec(`INSERT INTO cluster (cluster_id, name, url, console_url, metrics_url, logging_url, app_dns)
		VALUES ('00000000-0000-0000-0007-000000000001', 'cluster1', 'https://cluster1.com', 'https://console.cluster1.com',
	   'https://metrics.cluster1.com', 'https://login.cluster1.com', 'cluster1.com')`)
	require.NoError(t, err)
	_, err = sqlDB.Exec(`INSERT INTO cluster (cluster_id, name, url, console_url, metrics_url, logging_url , app_dns)
		VALUES ('00000000-0000-0000-0007-000000000002', 'cluster2', 'https://cluster2.com/', 'https://console.cluster2.com/',
	   'https://metrics.cluster2.com/', 'https://login.cluster2.com/', 'cluster2.com')`)
	require.NoError(t, err)

	// then apply step 7 of migration
	err = migrationsupport.Migrate(sqlDB, databaseName, migration.Steps()[:8])
	require.NoError(t, err)

	// and verify that all URLs have a single trailing slash
	rows, err := sqlDB.Query("SELECT url, console_url, metrics_url, logging_url FROM cluster")
	require.NoError(t, err)

	defer rows.Close()
	for rows.Next() {
		var url, consoleURL, metricsURL, loggingURL string
		err = rows.Scan(&url, &consoleURL, &metricsURL, &loggingURL)
		require.NoError(t, err)
		assert.True(t, strings.HasSuffix(url, "/"))
		assert.False(t, strings.HasSuffix(url, "//")) // make sure there was no extra trailing slash appended if not needed
		assert.True(t, strings.HasSuffix(consoleURL, "/"))
		assert.False(t, strings.HasSuffix(consoleURL, "//"))
		assert.True(t, strings.HasSuffix(loggingURL, "/"))
		assert.False(t, strings.HasSuffix(loggingURL, "//"))
		assert.True(t, strings.HasSuffix(metricsURL, "/"))
		assert.False(t, strings.HasSuffix(metricsURL, "//"))
	}

}
