# `censys-takehome`

Hello! This is my solution for the Censys take-home assignment. The original README can be found at [README-prompt.md](./README-prompt.md).

This project is probably a little more "in-depth" than expected. This is because I took the opportunity to not only demonstrate that I can put together a script connecting Pubsub to a Postgres database, but also to show off some other stuff. Of note:

* The project is divided into bite-sized, separated-concern "modules", which get their own directories and namespaces.

* These modules are assembled together into a running process by using [Google Wire](https://github.com/google/wire), a compile-time dependency injection tool.

* The main executable (`censys-takehome-processor`) is created by the script at [./cmd/censys-takehome-processor/app.go](./cmd/censys-takehome-processor/app.go), which uses the [urfave/cli](https://cli.urfave.org/) CLI, which helps with making everything configurable via _environment variables_.

* The database module is made to be as resilient as possible, and to support massively parallel ingestion from the processor module. As such, it features a channel-synchronized setup that keeps a single instance of `*sql.DB`, useable by whatever code requests it, which is built on-demand (or cached as long as the connection is valid).

* `compose.yml` has a fully configured development setup. This includes a container with a Postgres database, which is the destination of the records processed.

* The logging module uses the Zerolog library to create JSON-format logs, which are very thorough throughout the application. Configuration options to enable debug logs, and to print logs in pretty text (rather than JSON) is also available.

* I have kept the original "scanner" code as intact as I could (though its binary has been renamed to `censys-takehome-scanner` for consistency).

## Local Building/Usage

To build the project locally, the `build.sh` convenience script is included. It will create the two binaries in the `bin/` directory.

The reason for the script (rather than a simple `go build`) is because the `wire` CLI tool is needed to link everything together at the compile-time dependency injection. If you need to install it, run:

```bash
go install github.com/google/wire/cmd/wire@0.6.0
```

It can then be run directly:

```text
$ bin/censys-takehome-processor 
NAME:
   censys-takehome-processor - A new cli application

USAGE:
   censys-takehome-processor [global options] command [command options]

COMMANDS:
   server   
   schema   
   help, h  Shows a list of commands or help for one command

GLOBAL OPTIONS:
   --pghost value                  host of the output database server [$POSTGRES_HOST]
   --pgport value                  port of the output database server (default: 5432) [$POSTGRES_PORT]
   --pguser value                  user for connecting to the output database server [$POSTGRES_USER]
   --pgpass value                  password for connecting to the output database server; you should use the POSTGRES_PASSWORD env var to specify this [$POSTGRES_PASSWORD]
   --pgdb value                    database name to use [$POSTGRES_DB]
   --project value, -P value       what Pubsub project to receive data from (default: "test-project") [$PUBSUB_PROJECT_ID]
   --subscription value, -S value  what Pubsub topic to receive data from (default: "scan-sub") [$PUBSUB_SUBSCRIPTION_ID]
   --debug, -D                     enable more thorough debugging (default: false) [$DEBUG]
   --pretty                        enable pretty logging (default: false) [$PRETTY_LOGS]
   --help, -h                      show help
```

The CLI features two subcommands: `schema` and `server`. The former is a one-shot script which initializes the database schema in the supplied Postgres database. The latter runs the actual processor.

## Container Building/Usage

> The names of container-related files were generalized. I lean towards not using Docker itself (due to concerns around its licensing, security, isolation, and lack of true rootless operation). My development environment is instead based on Podman.

Just as the starter presented in the original repository, this project is fully compatible with Docker Compose. You can build it by using:

```bash
docker-compose build
```

Then, stand up the stack by using:

```bash
docker-compose up -d
```

You can monitor the logs of the takehome-relevant code by using:

```bash
docker-compose logs -f schema-init processor
```

## Testing and Verification

I verified the functionality of the code by monitoring logs (especially with `DEBUG=1`), and tracking message IDs and database transaction IDs, verifying that data makes it end-to-end properly.

Additionally, I also queried the Postgres database directly to inspect and verify the data that makes it in. To do so, use this command to launch a standard psql console:

```bash
docker-compose exec scandb psql -U scan-ingest scandb
```

### Concurrency/Parallelization

Horizontal scalability is baked in to the project. Multiple instances of the processor can function in tandem seamlessly. To observe this, take the working stack (see above) and run this:

```bash
docker-compose up -d --scale processor=4 processor
```

This will replace the single processor container with four that have identical configuration. Checking the logs and the output database in the same manner validates that they are working properly.

### Clearing the Environment

The Postgres database uses a volume to store its data. Thus, to fully reset the Docker Compose environment, you should use:

```bash
docker-compose down -v
```

## Unit Testing (TODO)

I wanted to add unit tests (using `go test`) to all the code, but that would extend the time this mini-project would take a bit too much. However, I do have a plan for how to do them.

The biggest challenge with implementing unit tests for a project like this (which relies on external connections) is the mocking/stubbing. The modularized approach I took makes this relatively easy: the encapsulation can be defined as an interface around the injected struct. This interface can then receive a substituted mock implementation when it is not relevant to the current test.

For example, the Processor module depends on the `ScanEntryDAO`. Right now that DAO is a struct, but it could be substituted with something like:

```go
type ScanEntryDAO interface {
    AddEntry(ScanEntry) error
}
```

This would allow a unit test to initialize a processor as:

```go
type MockScanEntryDAO struct {
    ReceivedEntries []ScanEntry
}

func (mock *MockScanEntryDAO) AddEntry(e ScanEntry) error {
    mock.ReceivedEntries = append(mock.ReceivedEntries, e)
    return nil
}

// ...

mockDAO := MockScanEntryDAO{}
myProc := processor.Processor {
    // ...
    ScanEntryDAO: &mockDAO,
}
```

The test could then trigger the processor as normal, but would be able to check that the processor utilized the DAO correctly by consulting `mockDAO.ReceivedEntries`.
