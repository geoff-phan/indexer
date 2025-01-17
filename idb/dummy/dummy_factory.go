package dummy

import (
	log "github.com/sirupsen/logrus"

	"github.com/geoff-phanindexer/idb"
)

type dummyFactory struct {
}

// Name is part of the IndexerFactory interface.
func (df dummyFactory) Name() string {
	return "dummy"
}

// Build is part of the IndexerFactory interface.
func (df dummyFactory) Build(arg string, opts idb.IndexerDbOptions, log *log.Logger) (idb.IndexerDb, error) {
	return &dummyIndexerDb{log: log}, nil
}

func init() {
	idb.RegisterFactory("dummy", &dummyFactory{})
}
