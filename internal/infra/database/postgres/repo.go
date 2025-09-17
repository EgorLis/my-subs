package postgres

import (
	"context"
	"database/sql"
	"embed"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/EgorLis/my-subs/internal/domain"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	_ "github.com/jackc/pgx/v5/stdlib"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/golang-migrate/migrate/v4/source/iofs"
)

// ---- Postgres репозиторий (pgxpool) + golang-migrate ----

type PGRepo struct {
	logger *log.Logger
	pool   *pgxpool.Pool
	schema string
}

func NewPGRepo(ctx context.Context, logger *log.Logger, dsn, schema string) (*PGRepo, error) {
	// Запускаем golang-migrate используя pgx/stdlib
	if err := runMigrations(dsn, logger); err != nil {
		return nil, fmt.Errorf("migrations: %w", err)
	}

	// Создаем pgxpool
	logger.Println("initializing pgxpool...")
	cfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, fmt.Errorf("parse dsn: %w", err)
	}
	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("open pool: %w", err)
	}
	logger.Println("pgxpool initialized")

	r := &PGRepo{pool: pool, schema: schema, logger: logger}
	return r, nil
}

func (r *PGRepo) Close() {
	r.logger.Println("closing pgxpool...")
	r.pool.Close()
	r.logger.Println("pgxpool closed")
}

// ---- Миграции через golang-migrate ----

//go:embed migrations/*.sql
var EmbeddedMigrations embed.FS

func runMigrations(dsn string, logger *log.Logger) error {
	// Открываем *sql.DB с помощью pgx stdlib. Важно: это отдельный экземпляр от pgxpool.
	sqldb, err := sql.Open("pgx", dsn)
	if err != nil {
		return fmt.Errorf("sql.Open pgx: %w", err)
	}
	defer sqldb.Close()

	driver, err := postgres.WithInstance(sqldb, &postgres.Config{})
	if err != nil {
		return fmt.Errorf("postgres driver: %w", err)
	}

	src, err := iofs.New(EmbeddedMigrations, "migrations")
	if err != nil {
		return fmt.Errorf("iofs source: %w", err)
	}

	m, err := migrate.NewWithInstance("iofs", src, "postgres", driver)
	if err != nil {
		return fmt.Errorf("migrate.New: %w", err)
	}
	defer m.Close()

	logger.Println("applying migrations...")
	if err := m.Up(); err != nil {
		if errors.Is(err, migrate.ErrNoChange) {
			logger.Println("no new migrations to apply")
			return nil
		}
		return fmt.Errorf("apply migrations: %w", err)
	}
	logger.Println("migrations applied successfully")
	return nil
}

// ---- Реализация репозитория ----

func (r *PGRepo) Ping(ctx context.Context) error {
	r.logger.Println("pinging database...")
	if err := r.pool.Ping(ctx); err != nil {
		r.logger.Printf("ping failed: %v", err)
		return err
	}
	r.logger.Println("ping successful")
	return nil
}

func (r *PGRepo) AddSub(ctx context.Context, s domain.Subscription) (domain.Subscription, error) {
	id := uuid.NewString()
	r.logger.Printf("adding subscription user=%s service=%s price=%d from %s to %s",
		s.UserID, s.ServiceName, s.Price, s.StartDate.Format("01-2006"), s.EndDate.Format("01-2006"))
	q := fmt.Sprintf(`
		INSERT INTO %s.subscriptions (id, service_name, price, user_id, start_date, end_date)
		VALUES ($1,$2,$3,$4,$5,$6)
		RETURNING id, service_name, price, user_id, start_date, end_date`, r.schema)
	var out domain.Subscription
	err := r.pool.QueryRow(ctx, q, id, s.ServiceName, s.Price, s.UserID, s.StartDate, s.EndDate).
		Scan(&out.ID, &out.ServiceName, &out.Price, &out.UserID, &out.StartDate, &out.EndDate)
	if err != nil {
		r.logger.Printf("add subscription failed: %v", err)
		return out, err
	}
	r.logger.Printf("subscription added id=%s", out.ID)
	return out, nil
}

func (r *PGRepo) UpdateSub(ctx context.Context, s domain.Subscription) error {
	r.logger.Printf("updating subscription id=%s", s.ID)
	q := fmt.Sprintf(`
		UPDATE %s.subscriptions
		SET service_name=$2, price=$3, user_id=$4, start_date=$5, end_date=$6
		WHERE id=$1`, r.schema)
	ct, err := r.pool.Exec(ctx, q, s.ID, s.ServiceName, s.Price, s.UserID, s.StartDate, s.EndDate)
	if err != nil {
		r.logger.Printf("update failed for id=%s: %v", s.ID, err)
		return err
	}
	if ct.RowsAffected() == 0 {
		r.logger.Printf("update: subscription not found id=%s", s.ID)
		return domain.ErrNotFound
	}
	r.logger.Printf("subscription updated id=%s", s.ID)
	return nil
}

func (r *PGRepo) DeleteSub(ctx context.Context, id string) error {
	r.logger.Printf("deleting subscription id=%s", id)
	q := fmt.Sprintf(`DELETE FROM %s.subscriptions WHERE id=$1`, r.schema)
	ct, err := r.pool.Exec(ctx, q, id)
	if err != nil {
		r.logger.Printf("delete failed id=%s: %v", id, err)
		return err
	}
	if ct.RowsAffected() == 0 {
		r.logger.Printf("delete: subscription not found id=%s", id)
		return domain.ErrNotFound
	}
	r.logger.Printf("subscription deleted id=%s", id)
	return nil
}

func (r *PGRepo) GetSub(ctx context.Context, id string) (domain.Subscription, error) {
	r.logger.Printf("getting subscription id=%s", id)
	q := fmt.Sprintf(`
        SELECT id, service_name, price, user_id, start_date, end_date
        FROM %s.subscriptions WHERE id=$1`, r.schema)
	var s domain.Subscription
	err := r.pool.QueryRow(ctx, q, id).Scan(&s.ID, &s.ServiceName, &s.Price, &s.UserID, &s.StartDate, &s.EndDate)
	if errors.Is(err, pgx.ErrNoRows) {
		r.logger.Printf("get: subscription not found id=%s", id)
		return domain.Subscription{}, domain.ErrNotFound
	}
	if err != nil {
		r.logger.Printf("get failed id=%s: %v", id, err)
		return domain.Subscription{}, err
	}
	r.logger.Printf("subscription retrieved id=%s", id)
	return s, nil
}

func (r *PGRepo) ListSubs(ctx context.Context) ([]domain.Subscription, error) {
	r.logger.Println("listing subscriptions...")
	q := fmt.Sprintf(`
        SELECT id, service_name, price, user_id, start_date, end_date
        FROM %s.subscriptions
        ORDER BY created_at NULLS LAST, id`, r.schema)
	rows, err := r.pool.Query(ctx, q)
	if err != nil {
		r.logger.Printf("list failed: %v", err)
		return nil, err
	}
	defer rows.Close()
	var out []domain.Subscription
	for rows.Next() {
		var s domain.Subscription
		if err := rows.Scan(&s.ID, &s.ServiceName, &s.Price, &s.UserID, &s.StartDate, &s.EndDate); err != nil {
			r.logger.Printf("scan row failed: %v", err)
			return nil, err
		}
		out = append(out, s)
	}
	if err := rows.Err(); err != nil {
		r.logger.Printf("list rows error: %v", err)
		return nil, err
	}
	r.logger.Printf("list complete, count=%d", len(out))
	return out, nil
}

// TotalCost суммирует поле Price для подписок, которые пересекают период [start,end] включительно.
// Необязательные фильтры serviceName и userID применяются, если они не пустые.
func (r *PGRepo) TotalCost(ctx context.Context, serviceName, userID string, start, end time.Time) (int, error) {
	r.logger.Printf("calculating total cost service=%s user=%s period=%s..%s",
		serviceName, userID, start.Format(time.RFC3339), end.Format(time.RFC3339))
	if end.Before(start) {
		return 0, fmt.Errorf("invalid period: end before start")
	}
	base := fmt.Sprintf(`
        SELECT COALESCE(SUM(price),0)
        FROM %s.subscriptions
        WHERE start_date <= $1 AND end_date >= $2`, r.schema)
	args := []any{end, start}
	idx := 3
	if serviceName != "" {
		base += fmt.Sprintf(" AND service_name = $%d", idx)
		args = append(args, serviceName)
		idx++
	}
	if userID != "" {
		base += fmt.Sprintf(" AND user_id = $%d", idx)
		args = append(args, userID)
		idx++
	}
	var total int
	if err := r.pool.QueryRow(ctx, base, args...).Scan(&total); err != nil {
		r.logger.Printf("total cost query failed: %v", err)
		return 0, err
	}
	r.logger.Printf("total cost calculated: %d", total)
	return total, nil
}
