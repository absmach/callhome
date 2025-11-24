// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package timescale

import (
	_ "github.com/jackc/pgx/v5/stdlib" // required for SQL access
	migrate "github.com/rubenv/sql-migrate"
)

// Migration of Telemetry service.
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
						mg_version		TEXT,
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
			{
				Id: "telemetry_3",
				Up: []string{
					`ALTER TABLE telemetry add mac_address TEXT;`,
				},
				Down: []string{`ALTER TABLE telemetry DROP COLUMN mac_address;`},
			},
			{
				Id: "telemetry_4",
				Up: []string{
					`CREATE INDEX IF NOT EXISTS idx_telemetry_ip_address ON telemetry (ip_address);`,
					`CREATE INDEX IF NOT EXISTS idx_telemetry_country ON telemetry (country);`,
					`CREATE INDEX IF NOT EXISTS idx_telemetry_city ON telemetry (city);`,
					`CREATE INDEX IF NOT EXISTS idx_telemetry_service ON telemetry (service);`,
					`CREATE INDEX IF NOT EXISTS idx_telemetry_mg_version ON telemetry (mg_version);`,
					`CREATE INDEX IF NOT EXISTS idx_telemetry_time ON telemetry (time DESC);`,
					`CREATE INDEX IF NOT EXISTS idx_telemetry_country_time ON telemetry (country, time DESC);`,
				},
				Down: []string{
					`DROP INDEX IF EXISTS idx_telemetry_ip_address;`,
					`DROP INDEX IF EXISTS idx_telemetry_country;`,
					`DROP INDEX IF EXISTS idx_telemetry_city;`,
					`DROP INDEX IF EXISTS idx_telemetry_service;`,
					`DROP INDEX IF EXISTS idx_telemetry_mg_version;`,
					`DROP INDEX IF EXISTS idx_telemetry_time;`,
					`DROP INDEX IF EXISTS idx_telemetry_country_time;`,
				},
			},
			{
				Id: "telemetry_5",
				Up: []string{
					`CREATE INDEX IF NOT EXISTS idx_telemetry_ip_time ON telemetry (ip_address, time DESC);`,
					`CREATE INDEX IF NOT EXISTS idx_telemetry_service_ip ON telemetry (ip_address, service);`,
				},
				Down: []string{
					`DROP INDEX IF EXISTS idx_telemetry_ip_time;`,
					`DROP INDEX IF EXISTS idx_telemetry_service_ip;`,
				},
			},
			{
				Id: "telemetry_6",
				Up: []string{
					`CREATE INDEX IF NOT EXISTS idx_telemetry_time_ip ON telemetry (time DESC, ip_address);`,
					`CREATE INDEX IF NOT EXISTS idx_telemetry_ip_service_incl ON telemetry (ip_address) INCLUDE (service);`,
				},
				Down: []string{
					`DROP INDEX IF EXISTS idx_telemetry_time_ip;`,
					`DROP INDEX IF EXISTS idx_telemetry_ip_service_incl;`,
				},
			},
			{
				Id: "telemetry_7",
				Up: []string{
					`SELECT remove_retention_policy('telemetry');`,
					`SELECT add_retention_policy('telemetry', INTERVAL '1 year');`,
				},
				Down: []string{
					`SELECT remove_retention_policy('telemetry');`,
					`SELECT add_retention_policy('telemetry', INTERVAL '90 days');`,
				},
			},
			{
				Id: "telemetry_8",
				Up: []string{
					`ALTER TABLE telemetry ADD deployment_id TEXT;`,
					`CREATE INDEX IF NOT EXISTS idx_telemetry_deployment_id ON telemetry (deployment_id);`,
				},
				Down: []string{
					`DROP INDEX IF EXISTS idx_telemetry_deployment_id;`,
					`ALTER TABLE telemetry DROP COLUMN deployment_id;`,
				},
			},
		},
	}
}
