# MariaDB Last Login

MariaDB Last Login (`mariadb-lastlogin`) is a Go program that monitors and logs database connections from MariaDB's audit log files to a SQLite database. It's designed to run frequently, providing insights into the latest connections per account. This is useful for detecting unused accounts. Note that accounts that have never logged in will be absent from the SQLite database.

When multiple files are present (likely if you use logrotate on your audit files), mariadb-lastlogin compares the file modification date with the last processed date, ensuring already processed files aren't read every time.

To store the last processed time, mariadb-lastlogin updates the last connection date for the `mysql@localhost` user. Keep in mind that this account's `last_seen` date is not related to the actual database access time.

## Features

- Tracks the latest MariaDB database connections per account
- Designed for frequent execution
- Utilizes MariaDB's audit plugin for accurate connection tracking
- Stores the data into a SQLite database

## Rationale

MariaDB doesn't natively store the last connection date of an account (https://jira.mariadb.org/browse/MDEV-27205 request this but is stalled). Creating a stored procedure to do this on every connection could potentially slow down operations. Since we're already using the audit module for various reasons, parsing these logs is the most efficient option.

## Prerequisites

- MariaDB server with the [Audit Plugin](https://mariadb.com/kb/en/mariadb-audit-plugin/) enabled
- Access to MariaDB audit logs
- At a minimum, the audit pluging should be logging 'CONNECT' events: `SET GLOBAL server_audit_events = 'CONNECT';`

## Installation

Choose one of the following.

### Binary

Place `mariadb-lastlogin` in your PATH.

**From source:**

```sh
git clone https://github.com/yourusername/mariadb-lastlogin.git
cd mariadb-lastlogin
go build
sudo cp mariadb-lastlogin /usr/local/bin/
sudo chmod +x /usr/local/bin/mariadb-lastlogin
```

**From release:**

Download the binary from the GitHub [Releases](https://github.com/epfl-si/mariadb-lastlogin/releases) page.

```sh
sudo cp mariadb-lastlogin /usr/local/bin/
sudo chmod +x /usr/local/bin/mariadb-lastlogin
```

### Container

Use the published image:

```sh
podman/docker run --rm \
  --name mariadb-lastlogin \
  --volume ./config.ini:/etc/mariadb-lastlogin/config.ini \
  --volume /var/lib/mysql:/var/lib/mysql \
  ghcr.io/epfl-si/mariadb-lastlogin:latest
```

The container exits after each run if you don't use the internal loop (`LoopEnabled = false`).

## Configuration

1. Ensure the MariaDB [Audit Plugin](https://mariadb.com/kb/en/mariadb-audit-plugin/) is enabled on your database server.
2. Copy `examples/config.ini-dist` to `/etc/mariadb-lastlogin/config.ini`.
3. Edit `/etc/mariadb-lastlogin/config.ini` with your specific configuration details.

## Scheduling

The binary supports two execution modes:

- **One-shot** (default): processes audit logs once and exits. Use this with cron or a container orchestrator.
- **Loop** (`LoopEnabled = true`): runs continuously and sleeps between iterations. Use this with the systemd service or when running as a sidecar container in Kubernetes.

### systemd service (recommended)

Use `examples/mariadb-lastlogin.service` with `LoopEnabled = true` in your config. The unit runs under the `mysql` user's systemd manager:

```sh
sudo mkdir -p /home/mysql/.config/systemd/user
sudo cp examples/mariadb-lastlogin.service /home/mysql/.config/systemd/user/
sudo chown -R mysql:mysql /home/mysql/.config/systemd
sudo loginctl enable-linger mysql
sudo -u mysql systemctl --user daemon-reload
sudo -u mysql systemctl --user enable --now mariadb-lastlogin
```

Logs go to the user's journald, which log shippers such as Grafana Alloy, Promtail, or Fluent Bit can forward to Loki.

### Cron

Run the binary at your chosen interval. Note that cron captures stdout/stderr locally; logs are not forwarded automatically.

```cron
*/15 * * * * /usr/local/bin/mariadb-lastlogin
```

### Container

The container is one-shot. Run it periodically with cron, a Kubernetes CronJob, or your orchestrator.

## Usage

Run the program manually:

```go
./mariadb-lastlogin
```

No output means the script worked. Head to [Read the SQLite database] bellow to retrieve the data.

You can also check the version installed using:
```go
./mariadb-lastlogin version
```

## Read the SQLite database

Assuming you have sqlite3 installed (it's typically shipped with Python3), you can:

```sh
sqlite3 /var/lib/mysql/audit.sqlite

SELECT name, host, last_seen FROM Accounts;

mysql|localhost|2024-10-14 13:22:57+02:00  <- account used to stored the last processing date
root|localhost|2024-10-14 13:23:57+02:00
user1|example.com|2024-10-13 09:11:57+02:00
```

## Performance Considerations

On busy servers, audit log files are rotated frequently. To ensure no data is missed, calculate the minimum interval between script executions based on your server's log rotation frequency.

Our testing on a 4-core computer with 10 files of 100MB each yielded the following results:

- Initial run (parsing all logs): ~1.4 seconds
- Subsequent runs (parsing only the latest 100MB file): <1 second
- Less than 200MB of memory was necessary

Memory usage spikes were not significant in our tests. However, we strongly recommend thorough testing in your specific environment before deploying to production, especially if:

1. The script runs on the same server as MariaDB
2. Most of the server's memory is allocated to InnoDB cache
3. You have more than 5,000 unique accounts

In such scenarios, there's a potential risk of memory exhaustion. Adjust the cron job frequency and/or log rotation settings to balance between timely data collection and system resource consumption.

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

## Authors

- laurent.indermuehle@epfl.ch
