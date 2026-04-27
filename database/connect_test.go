package database

import (
	"io"
	"log/slog"
	"path/filepath"
	"strings"
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestEnableSQLiteWALSuccess(t *testing.T) {
	t.Parallel()

	dbPath := filepath.Join(t.TempDir(), "wal_test.db")
	db, err := gorm.Open(sqlite.Open(dbPath), &gorm.Config{})
	if err != nil {
		t.Fatalf("打开测试数据库失败：%v", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("获取底层数据库连接失败：%v", err)
	}
	t.Cleanup(func() {
		_ = sqlDB.Close()
	})

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	if err := enableSQLiteWAL(db, logger); err != nil {
		t.Fatalf("启用 SQLite WAL 失败：%v", err)
	}

	var journalMode string
	if err := db.Raw("PRAGMA journal_mode;").Scan(&journalMode).Error; err != nil {
		t.Fatalf("读取 SQLite journal_mode 失败：%v", err)
	}

	if !strings.EqualFold(journalMode, sqliteJournalModeWAL) {
		t.Fatalf("期望 journal_mode 为 %s，实际为 %s", sqliteJournalModeWAL, journalMode)
	}
}

func TestEnableSQLiteWALWithNilDB(t *testing.T) {
	t.Parallel()

	err := enableSQLiteWAL(nil, nil)
	if err == nil {
		t.Fatal("期望返回错误，但实际为 nil")
	}
}
