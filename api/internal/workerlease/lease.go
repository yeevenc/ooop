package workerlease

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type Lease struct {
	Name      string    `gorm:"primaryKey;size:100"`
	OwnerID   string    `gorm:"size:128;not null"`
	ExpiresAt time.Time `gorm:"not null;index"`
	CreatedAt time.Time
	UpdatedAt time.Time
}

func (Lease) TableName() string {
	return "worker_leases"
}

type Repository interface {
	TryAcquire(
		ctx context.Context,
		name string,
		ownerID string,
		ttl time.Duration,
	) (bool, error)
	Release(ctx context.Context, name string, ownerID string) error
}

type GormRepository struct {
	db *gorm.DB
}

func NewGormRepository(db *gorm.DB) *GormRepository {
	return &GormRepository{db: db}
}

func (r *GormRepository) TryAcquire(
	ctx context.Context,
	name string,
	ownerID string,
	ttl time.Duration,
) (bool, error) {
	name = strings.TrimSpace(name)
	ownerID = strings.TrimSpace(ownerID)
	if name == "" || ownerID == "" {
		return false, errors.New("Worker 租约名称或持有者不能为空")
	}
	if ttl <= 0 {
		return false, errors.New("Worker 租约有效期必须大于零")
	}

	ttlMicroseconds := ttl.Microseconds()
	result := r.db.WithContext(ctx).
		Model(&Lease{}).
		Where(
			`name = ? AND (
				owner_id = ?
				OR expires_at <= CURRENT_TIMESTAMP(3)
				OR expires_at > TIMESTAMPADD(MICROSECOND, ?, CURRENT_TIMESTAMP(3))
			)`,
			name,
			ownerID,
			ttlMicroseconds*2,
		).
		Updates(map[string]interface{}{
			"owner_id": ownerID,
			"expires_at": gorm.Expr(
				"TIMESTAMPADD(MICROSECOND, ?, CURRENT_TIMESTAMP(3))",
				ttlMicroseconds,
			),
			"updated_at": gorm.Expr("CURRENT_TIMESTAMP(3)"),
		})
	if result.Error != nil {
		return false, result.Error
	}
	if result.RowsAffected == 1 {
		return true, nil
	}

	result = r.db.WithContext(ctx).
		Table((Lease{}).TableName()).
		Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "name"}},
			DoNothing: true,
		}).
		Create(map[string]interface{}{
			"name":     name,
			"owner_id": ownerID,
			"expires_at": gorm.Expr(
				"TIMESTAMPADD(MICROSECOND, ?, CURRENT_TIMESTAMP(3))",
				ttlMicroseconds,
			),
			"created_at": gorm.Expr("CURRENT_TIMESTAMP(3)"),
			"updated_at": gorm.Expr("CURRENT_TIMESTAMP(3)"),
		})
	if result.Error != nil {
		return false, result.Error
	}
	return result.RowsAffected == 1, nil
}

func (r *GormRepository) Release(
	ctx context.Context,
	name string,
	ownerID string,
) error {
	return r.db.WithContext(ctx).
		Model(&Lease{}).
		Where("name = ? AND owner_id = ?", name, ownerID).
		Updates(map[string]interface{}{
			"owner_id":   "",
			"expires_at": gorm.Expr("CURRENT_TIMESTAMP(3)"),
			"updated_at": gorm.Expr("CURRENT_TIMESTAMP(3)"),
		}).Error
}

type Guard struct {
	repository Repository
	name       string
	ownerID    string
}

func NewGuard(repository Repository, name string) *Guard {
	return &Guard{
		repository: repository,
		name:       strings.TrimSpace(name),
		ownerID:    newOwnerID(),
	}
}

func (g *Guard) TryAcquire(
	ctx context.Context,
	ttl time.Duration,
) (bool, error) {
	if g == nil || g.repository == nil {
		return false, errors.New("Worker 租约仓储未初始化")
	}
	return g.repository.TryAcquire(ctx, g.name, g.ownerID, ttl)
}

func (g *Guard) Release(ctx context.Context) error {
	if g == nil || g.repository == nil {
		return nil
	}
	return g.repository.Release(ctx, g.name, g.ownerID)
}

func newOwnerID() string {
	random := make([]byte, 12)
	if _, err := rand.Read(random); err != nil {
		return fmt.Sprintf("%d-%d", os.Getpid(), time.Now().UnixNano())
	}
	hostname, _ := os.Hostname()
	value := fmt.Sprintf(
		"%s-%d-%s",
		strings.TrimSpace(hostname),
		os.Getpid(),
		base64.RawURLEncoding.EncodeToString(random),
	)
	if len(value) > 128 {
		return value[len(value)-128:]
	}
	return value
}
