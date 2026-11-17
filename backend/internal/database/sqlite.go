package database

import (
	"github.com/libtnb/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"log"
	"os"
	"path/filepath"
	"time"
)

func Open(dir string) (*gorm.DB, error) {
	if err := os.MkdirAll(dir, 0700); err != nil {
		return nil, err
	}
	db, err := gorm.Open(sqlite.Open(filepath.Join(dir, "mini-erp.db")+"?_txlock=immediate&_pragma=foreign_keys(1)&_pragma=journal_mode(WAL)&_pragma=busy_timeout(5000)"), dbConfig())
	if err != nil {
		return nil, err
	}
	sqlDB, err := db.DB()
	if err != nil {
		return nil, err
	}
	sqlDB.SetMaxOpenConns(1)
	sqlDB.SetMaxIdleConns(1)
	if err = Migrate(db); err != nil {
		sqlDB.Close()
		return nil, err
	}
	return db, nil
}

func dbConfig() *gorm.Config {
	return &gorm.Config{TranslateError: true, Logger: logger.New(log.New(os.Stderr, "sql ", log.LstdFlags), logger.Config{SlowThreshold: 200 * time.Millisecond, LogLevel: logger.Warn, IgnoreRecordNotFoundError: true, ParameterizedQueries: true})}
}

// Readers use separate, query-only connections so a queued writer cannot occupy
// every connection used by ordinary API reads. WAL exposes committed changes immediately.
func OpenReader(dir string) (*gorm.DB, error) {
	db, err := gorm.Open(sqlite.Open(filepath.Join(dir, "mini-erp.db")+"?mode=ro&_pragma=query_only(1)&_pragma=busy_timeout(5000)"), dbConfig())
	if err != nil {
		return nil, err
	}
	pool, err := db.DB()
	if err != nil {
		return nil, err
	}
	pool.SetMaxOpenConns(8)
	pool.SetMaxIdleConns(8)
	return db, nil
}
