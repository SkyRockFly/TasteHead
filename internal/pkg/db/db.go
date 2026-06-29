package db

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog/log"
)

const (
	defaultDBConnTimeout    = 20 * time.Second
	defaultPingRetryTimeout = 3 * time.Second
)

func InitDB(ctx context.Context, url string) (*pgxpool.Pool, error) {
	childCtx, cancel := context.WithTimeout(ctx, defaultDBConnTimeout)
	defer cancel()

	pgxpool, err := pgxpool.New(childCtx, url)
	if err != nil {
		return nil, fmt.Errorf("establish pgxpool connection: %w", err)
	}

	tryNum := 0
	for err := pgxpool.Ping(childCtx); err != nil; tryNum++ {
		if tryNum == 3 {
			return nil, fmt.Errorf("ping database: %w", err)
		}
		time.Sleep(defaultPingRetryTimeout)
		log.Warn().Int("try", tryNum+1).Int("retry after", int(defaultDBConnTimeout)).Msg("ping database")
	}

	log.Info().Msg("Successfully established connection")

	return pgxpool, nil
}
