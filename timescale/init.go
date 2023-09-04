package timescale

import (
	_ "github.com/jackc/pgx/v5/stdlib" // required for SQL access
	migrate "github.com/rubenv/sql-migrate"
)

// Migration of telemetry service
func Migration() migrate.MemoryMigrationSource {
	return migrate.MemoryMigrationSource{
		Migrations: []*migrate.Migration{
			{
				Id: "telemetry_1",
				Up: []string{
					`CREATE TABLE IF NOT EXISTS telemetry (
						time			TIMESTAMPTZ,
						service_time	TIMESTAMPTZ,
						ip_address		TEXT	NOT	NULL,
					 	longitude 		FLOAT	NOT	NULL,
						latitude		FLOAT	NOT NULL,
						mf_version		TEXT,
						service			TEXT,
						country 		TEXT,
						city 			TEXT,
						PRIMARY KEY (time)
					);
					SELECT create_hypertable('telemetry', 'time', chunk_time_interval => INTERVAL '1 day');`,
				},
				Down: []string{"DROP TABLE telemetry;"},
			},
			{
				Id: "telemetry_2",
				Up: []string{
					`SELECT add_retention_policy('telemetry', INTERVAL '90 days');`,
				},
				Down: []string{`SELECT remove_retention_policy('telemetry');`},
			},
		},
	}
}
