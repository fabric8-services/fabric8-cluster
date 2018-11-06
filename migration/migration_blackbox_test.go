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
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

const (
	dbName      = "test"
	defaultHost = "localhost"
	defaultPort = "5436"
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
}

func testMigration001Cluster(t *testing.T) {
	err := migrationsupport.Migrate(sqlDB, databaseName, migration.Steps()[:2])
	require.NoError(t, err)
	//t.Run("insert ok", func(t *testing.T) {
	//	_, err := sqlDB.Exec(`INSERT INTO cluster (cluster_id, url, type, identity_id)
	//		VALUES (uuid_generate_v4(),'osio-stage', 'stage', uuid_generate_v4(),'', 'cluster1.com')`)
	//	require.NoError(t, err)
	//})
}

//INSERT INTO
//users(created_at, updated_at, id, email, full_name, image_url, bio, url, context_information)
//VALUES
//(
//now(), now(), 'f03f023b-0427-4cdb-924b-fb2369018ab7', 'test2@example.com', 'test1', 'https://www.gravatar.com/avatar/testtwo2', 'my test bio one', 'http://example.com/001', '{"key": "value"}'
//),
//(
//now(), now(), 'f03f023b-0427-4cdb-924b-fb2369018ab6', 'test3@example.com', 'test2', 'http://https://www.gravatar.com/avatar/testtwo3', 'my test bio two', 'http://example.com/002', '{"key": "value"}'
//)
//;
//-- identities
//INSERT INTO
//identities(created_at, updated_at, id, username, provider_type, user_id, profile_url)
//VALUES
//(
//now(), now(), '2a808366-9525-4646-9c80-ed704b2eebbe', 'test1', 'github', 'f03f023b-0427-4cdb-924b-fb2369018ab7', 'http://example-github.com/001'
//),
//(
//now(), now(), '2a808366-9525-4646-9c80-ed704b2eebbb', 'test2', 'rhhd', 'f03f023b-0427-4cdb-924b-fb2369018ab6', 'http://example-rhd.com/002'
//)
//;
