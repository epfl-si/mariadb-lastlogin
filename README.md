# MariaDB Last Login

MariaDB Last Login (`mariadb-lastlogin`) is a Go program that monitors and logs database connections from MariaDB's audit log files to a SQLite database. It's designed to run frequently via a cron job, providing insights into the latest connections per account. This is useful for detecting unused accounts. Note that accounts that have never logged in will be absent from the SQLite database.

When multiple files are present (likely if you use logrotate on your audit files), mariadb-lastlogin compares the file modification date with the last processed date, ensuring already processed files aren't read every time.

To store the last processed time, mariadb-lastlogin updates the last connection date for the `mysql@localhost` user. Keep in mind that this account's `last_seen` date is not related to the actual database access time.

## Features

- Tracks the latest MariaDB database connections per account
- Designed for frequent execution via cron jobs
- Utilizes MariaDB's audit plugin for accurate connection tracking
- Stores the data into a SQLite database

## Rationale

MariaDB doesn't natively store the last connection date of an account. Creating a stored procedure to do this on every connection could potentially slow down operations. Since we're already using the audit module for various reasons, parsing these logs is the most efficient option.

## Prerequisites

- MariaDB server with the [Audit Plugin](https://mariadb.com/kb/en/mariadb-audit-plugin/) enabled
- Access to MariaDB audit logs
- At a minimum, the audit pluging should be logging 'CONNECT' events: `SET GLOBAL server_audit_events = 'CONNECT';`

## Installation

1. Clone the repository:
git clone https://github.com/yourusername/mariadb-lastlogin.git

2. Navigate to the project directory:
cd mariadb-lastlogin

3. Build the project:
go build

4. Copy the binary and make it executable:
sudo cp mariadb-lastlogin /usr/bin
sudo chmod +x /usr/bin/mariadb-lastlogin

## Configuration

1. Ensure the MariaDB [Audit Plugin](https://mariadb.com/kb/en/mariadb-audit-plugin/) is enabled on your database server.
2. Copy the `config.ini-dist` to `/etc/mariadb-lastlogin/config.ini`
3. Edit `/etc/mariadb-lastlogin/config.ini` with your specific configuration details.

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

## Cron Job Setup

To set up a cron job:

1. Open your crontab file:
crontab -e

2. Add a line like this (adjust the timing and path as needed). Here we start with a run every 15 minutes:
```
*/15 * * * * /path/to/mariadb-lastlogin
```

3. Save and exit the editor.

## Use the container
Requirements:
- Configuration file at /etc/mariadb-lastlogin/config.ini
- Access to the audit log files
- A storage for the Sqlite database

Copy the `config.ini-dist` and name it `config.ini`.
Then start the container. It will exit when finished, that's intended:
```sh
podman/docker run --detach --interactive --tty \
--name mariadb-lastlogin \
--volume ./config.ini:/etc/mariadb-lastlogin/config.ini \
--volume /var/lib/mysql:/var/lib/mysql \
ghcr.io/epfl-si/mariadb-lastlogin:latest
```

Display the logs:
```sh
podman/docker logs mariadb-lastlogin
```

## Performance Considerations

On busy servers, audit log files are rotated frequently. To ensure no data is missed, calculate the minimum interval between script executions based on your server's log rotation frequency.

Our testing on a 4-core computer with 10 files of 100MB each yielded the following results:

- Initial run (parsing all logs): ~1.4 seconds
- Subsequent runs (parsing only the latest 100MB file): <1 second

Memory usage spikes were not significant in our tests. However, we strongly recommend thorough testing in your specific environment before deploying to production, especially if:

1. The script runs on the same server as MariaDB
2. Most of the server's memory is allocated to InnoDB cache

In such scenarios, there's a potential risk of memory exhaustion. Adjust the cron job frequency and/or log rotation settings to balance between timely data collection and system resource consumption.

## Read the SQLite database

Assuming you have sqlite3 installed (it's typically shipped with Python3), you can:

```sh
sqlite3 /var/lib/mysql/audit.sqlite

SELECT name, host, last_seen FROM Accounts;

mysql|localhost|2024-10-14 13:22:57+02:00  <- account used to stored the last processing date
root|localhost|2024-10-14 13:23:57+02:00
user1|example.com|2024-10-13 09:11:57+02:00
```

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

## Authors

- laurent.indermuehle@epfl.ch
