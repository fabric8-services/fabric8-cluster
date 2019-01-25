package migration

import (
	"database/sql"

	"github.com/fabric8-services/fabric8-common/migration"
)

func Migrate(db *sql.DB, catalog string) error {
	return migration.Migrate(db, catalog, Steps())
}

type Scripts [][]string

func Steps() Scripts {
	return [][]string{
		{"000-bootstrap.sql"},
		{"001-cluster.sql"},
		{"002-cluster-on-delete-cascade.sql"},
		{"003-unique-index-on-cluster-api-url.sql"},
		{"004-add-capacity-exhausted-to-cluster.sql"},
		{"005-alter-cluster-api-url-index-to-unique.sql"},
		{"006-add-sa-token-encrypted-to-cluster.sql"},
		{"007-add-url-trailing-slash.sql"},
	}
}

func (s Scripts) Asset(name string) ([]byte, error) {
	return Asset(name)
}

func (s Scripts) AssetNameWithArgs() [][]string {
	return s
}
