package timescale

import (
	"context"

	"github.com/jackc/pgconn"
	"github.com/jackc/pgerrcode"
	"github.com/jmoiron/sqlx"
	"github.com/mainflux/callhome/callhome"
	"github.com/mainflux/callhome/callhome/repository"
	"github.com/mainflux/mainflux/readers"
	"github.com/pkg/errors"
)

var _ callhome.TelemetryRepo = (*repo)(nil)

type repo struct {
	db *sqlx.DB
}

// New returns new TimescaleSQL writer.
func New(db *sqlx.DB) callhome.TelemetryRepo {
	return &repo{db: db}
}

// RetrieveAll gets all records from repo.
func (r repo) RetrieveAll(ctx context.Context, pm callhome.PageMetadata) (callhome.TelemetryPage, error) {
	q := `SELECT * FROM telemetry LIMIT :limit OFFSET :offset;`

	params := map[string]interface{}{
		"limit":  pm.Limit,
		"offset": pm.Offset,
	}

	rows, err := r.db.NamedQuery(q, params)
	if err != nil {
		return callhome.TelemetryPage{}, errors.Wrap(readers.ErrReadMessages, err.Error())
	}
	defer rows.Close()

	var results callhome.TelemetryPage

	for rows.Next() {
		var result callhome.Telemetry
		if err := rows.StructScan(&result); err != nil {
			return callhome.TelemetryPage{}, errors.Wrap(readers.ErrReadMessages, err.Error())
		}

		results.Telemetry = append(results.Telemetry, result)
	}

	q = `SELECT COUNT(*) FROM telemetry;`
	rows, err = r.db.NamedQuery(q, params)
	if err != nil {
		return callhome.TelemetryPage{}, errors.Wrap(readers.ErrReadMessages, err.Error())
	}
	defer rows.Close()

	total := uint64(0)
	if rows.Next() {
		if err := rows.Scan(&total); err != nil {
			return results, err
		}
	}
	results.Total = total

	return results, nil
}

// RetrieveByIP get record given an ip address.
func (repo) RetrieveByIP(ctx context.Context, email string) (callhome.Telemetry, error) {
	return callhome.Telemetry{}, repository.ErrRecordNotFound
}

// Save creates record in repo.
func (r repo) Save(ctx context.Context, t callhome.Telemetry) error {
	q := `INSERT INTO telemetry (id, ip_address, longitude, latitude,
		mf_version, service, last_seen, country, city)
		VALUES (:id, :ip_address, :longitude, :latitude,
			:mf_version, :service, :last_seen, :country, :city);`

	tx, err := r.db.BeginTxx(context.Background(), nil)
	if err != nil {
		return errors.Wrap(repository.ErrSaveEvent, err.Error())
	}
	defer func() {
		if err != nil {
			if txErr := tx.Rollback(); txErr != nil {
				err = errors.Wrap(err, errors.Wrap(repository.ErrTransRollback, txErr.Error()).Error())
			}
			return
		}

		if err = tx.Commit(); err != nil {
			err = errors.Wrap(repository.ErrSaveEvent, err.Error())
		}
	}()

	if _, err := tx.NamedExec(q, t); err != nil {
		if pgErr, ok := err.(*pgconn.PgError); ok {
			if pgErr.Code == pgerrcode.InvalidTextRepresentation {
				return errors.Wrap(repository.ErrSaveEvent, repository.ErrInvalidEvent.Error())
			}
		}
		return errors.Wrap(repository.ErrSaveEvent, err.Error())
	}
	return nil

}

// UpdateTelemetry updates record to repo.
func (repo) Update(ctx context.Context, u callhome.Telemetry) error {
	return nil
}