package db

import (
	"database/sql"
	"sync"
)

var ConfigDb *sql.DB
var RwLock = sync.RWMutex{}
