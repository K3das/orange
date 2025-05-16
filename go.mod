module github.com/K3das/orange

go 1.23.7

require (
	github.com/bwmarrin/discordgo v0.28.1
	github.com/caarlos0/env/v9 v9.0.0
	github.com/golang-migrate/migrate/v4 v4.18.3
	github.com/jackc/pgx/v5 v5.7.4
	go.uber.org/zap v1.27.0
	golang.org/x/sync v0.12.0
)

// https://github.com/bwmarrin/discordgo/pull/1618
replace github.com/bwmarrin/discordgo => github.com/K3das/discordgo v0.28.2-0.20250511041712-dc5eaecbcfe8

require (
	github.com/google/go-jsonnet v0.21.0 // indirect
	github.com/gorilla/websocket v1.4.2 // indirect
	github.com/hashicorp/errwrap v1.1.0 // indirect
	github.com/hashicorp/go-multierror v1.1.1 // indirect
	github.com/jackc/pgerrcode v0.0.0-20220416144525-469b46aa5efa // indirect
	github.com/jackc/pgpassfile v1.0.0 // indirect
	github.com/jackc/pgservicefile v0.0.0-20240606120523-5a60cdf6a761 // indirect
	github.com/jackc/puddle/v2 v2.2.2 // indirect
	go.uber.org/atomic v1.7.0 // indirect
	go.uber.org/multierr v1.10.0 // indirect
	golang.org/x/crypto v0.36.0 // indirect
	golang.org/x/sys v0.31.0 // indirect
	golang.org/x/text v0.23.0 // indirect
	sigs.k8s.io/yaml v1.4.0 // indirect
)
