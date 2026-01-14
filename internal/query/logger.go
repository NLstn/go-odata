package query

import (
	"log/slog"

	"gorm.io/gorm"
)

const loggerKey = "_odata_logger"

func setLoggerInDB(db *gorm.DB, logger *slog.Logger) *gorm.DB {
	if db == nil {
		return db
	}
	if logger == nil {
		logger = slog.Default()
	}
	return db.Set(loggerKey, logger)
}

func getLoggerFromDB(db *gorm.DB) *slog.Logger {
	if db == nil {
		return slog.Default()
	}
	if value, ok := db.Get(loggerKey); ok {
		if logger, ok := value.(*slog.Logger); ok && logger != nil {
			return logger
		}
	}
	return slog.Default()
}
