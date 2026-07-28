package workerlease

import (
	"bytes"
	"context"
	"database/sql"
	"log"
	"strings"
	"testing"
	"time"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type leaseTestRepository struct {
	acquireName    string
	acquireOwnerID string
	acquireTTL     time.Duration
	releaseName    string
	releaseOwnerID string
}

func (r *leaseTestRepository) TryAcquire(
	_ context.Context,
	name string,
	ownerID string,
	ttl time.Duration,
) (bool, error) {
	r.acquireName = name
	r.acquireOwnerID = ownerID
	r.acquireTTL = ttl
	return true, nil
}

func (r *leaseTestRepository) Release(
	_ context.Context,
	name string,
	ownerID string,
) error {
	r.releaseName = name
	r.releaseOwnerID = ownerID
	return nil
}

func TestGuardKeepsStableOwnerAcrossAcquireAndRelease(t *testing.T) {
	repository := &leaseTestRepository{}
	guard := NewGuard(repository, " activity_image_audit ")

	acquired, err := guard.TryAcquire(context.Background(), 45*time.Second)
	if err != nil {
		t.Fatalf("TryAcquire() error = %v", err)
	}
	if !acquired {
		t.Fatal("TryAcquire() acquired = false, want true")
	}
	if err := guard.Release(context.Background()); err != nil {
		t.Fatalf("Release() error = %v", err)
	}
	if repository.acquireName != "activity_image_audit" {
		t.Fatalf("acquire name = %s", repository.acquireName)
	}
	if repository.acquireOwnerID == "" {
		t.Fatal("acquire owner id is empty")
	}
	if repository.acquireOwnerID != repository.releaseOwnerID {
		t.Fatalf(
			"owner id changed: acquire=%s release=%s",
			repository.acquireOwnerID,
			repository.releaseOwnerID,
		)
	}
	if repository.releaseName != repository.acquireName {
		t.Fatalf(
			"lease name changed: acquire=%s release=%s",
			repository.acquireName,
			repository.releaseName,
		)
	}
	if repository.acquireTTL != 45*time.Second {
		t.Fatalf("acquire ttl = %s", repository.acquireTTL)
	}
}

func TestTryAcquireGeneratesValidIdempotentInsert(t *testing.T) {
	sqlDB, err := sql.Open("mysql", "")
	if err != nil {
		t.Fatalf("sql.Open() error = %v", err)
	}
	defer sqlDB.Close()

	var output bytes.Buffer
	gormLogger := logger.New(
		log.New(&output, "", 0),
		logger.Config{LogLevel: logger.Info},
	)
	db, err := gorm.Open(
		mysql.New(mysql.Config{
			Conn:                      sqlDB,
			SkipInitializeWithVersion: true,
		}),
		&gorm.Config{
			DryRun:                 true,
			DisableAutomaticPing:   true,
			SkipDefaultTransaction: true,
			Logger:                 gormLogger,
		},
	)
	if err != nil {
		t.Fatalf("gorm.Open() error = %v", err)
	}

	repository := NewGormRepository(db)
	if _, err := repository.TryAcquire(
		context.Background(),
		"activity_image_audit",
		"worker-test",
		45*time.Second,
	); err != nil {
		t.Fatalf("TryAcquire() error = %v", err)
	}

	generatedSQL := output.String()
	if !strings.Contains(generatedSQL, "INSERT IGNORE INTO worker_leases") {
		t.Fatalf("generated SQL does not contain INSERT IGNORE: %s", generatedSQL)
	}
	if strings.Contains(generatedSQL, "ON DUPLICATE KEY UPDATE") {
		t.Fatalf("generated SQL contains incomplete duplicate clause: %s", generatedSQL)
	}
}
