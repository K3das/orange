# Orange

Orange is a Discord utility bot that currently transcribes voice messages.

### Quickstart

Orange is designed to run alongside PostgreSQL inside a container. A compose file is provided in `compose.yaml`. 

An example `.env` is provided in `.env.sample` with all the required environment variables. Then, to build and run the bot:

```shell
compose up --build
```

### Development

#### Database

We're using [sqlc](https://sqlc.dev/) to generate query functions and types. If you change `store/queries.sql`, you need to update the generated code:

```shell
sqlc generate
```

For migrations, we're using [golang-migrate](https://github.com/golang-migrate/migrate) embedded in the program. To create a new migration pair: 

```shell
migrate create -ext sql -dir store/migrations -seq example_migration
```

#### Message Templates

In order to separate out the logic that builds messages from the rest of the code, all messages sent by the bot are defined using [Jsonnet](https://jsonnet.org/) in [`messages/jsonnet/`](./messages/jsonnet/).
