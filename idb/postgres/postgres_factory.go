package postgres

import (
	log "github.com/sirupsen/logrus"

	"github.com/geoff-phan/indexer/idb"
)

type postgresFactory struct {
}

func (df postgresFactory) Name() string {
	return "postgres"
}

func (df postgresFactory) Build(arg string, opts idb.IndexerDbOptions, log *log.Logger) (idb.IndexerDb, error) {
	return OpenPostgres(arg, opts, log)
}

func init() {
	idb.RegisterFactory("postgres", &postgresFactory{})
}
