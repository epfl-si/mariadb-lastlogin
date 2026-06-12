package main

import (
	"flag"
	"fmt"
	"log"
	"log/slog"
	"os"
	"time"

	"github.com/epfl-si/mariadb-lastlogin/internal/lastlogin"
)

var version = "dev"

func main() {

	versionFlag := flag.Bool("version", false, "Print version information and exit")
	flag.Parse()

	if *versionFlag || (len(os.Args) > 1 && os.Args[1] == "version") {
		fmt.Printf("lastlogin version %s\n", version)
		os.Exit(0)
	}

	cfg, err := lastlogin.InitConfig()
	if err != nil {
		log.Fatalf("Failed to initialize config: %v", err)
	}

	var programLevel = new(slog.LevelVar)
	h := slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: programLevel})
	slog.SetDefault(slog.New(h))
	programLevel.Set(cfg.LogLevel)

	if cfg.LoopEnabled {
		// loop mode
		for {
			slog.Info("loop tick", "intervalMinutes", cfg.IntervalMinutes)
			if err := run(cfg); err != nil {
				slog.Error("run failed, will retry", "error_msg", err)
			}
			time.Sleep(time.Duration(cfg.IntervalMinutes) * time.Minute)
		}
	} else {
		// oneshot mode
		if err := run(cfg); err != nil {
			log.Fatal(err)
		}
	}
}

func run(cfg lastlogin.Config) error {

	db, err := lastlogin.OpenOrCreateDB(cfg)
	if err != nil {
		slog.Error("failed to open or create database.", "error_msg", err)
		return err
	}
	defer db.Close()

	lastProcessedTime, err := lastlogin.GetLastProcessedTime(cfg, db)
	if err != nil {
		slog.Error("failed to get last processed time", "error_msg", err)
		return err
	}

	filePaths, newLastProcessedTime, err := lastlogin.FilterAndSortNewFiles(cfg, lastProcessedTime)
	if err != nil {
		slog.Error("error getting files to process", "error_msg", err)
		return err
	}

	// Round to the nearest second (filesystem returns 2024-10-15 13:43:57.109984656 +0200 CEST)
	roundedNewLastProcessedTime := newLastProcessedTime.Round(time.Second)

	if lastProcessedTime.Compare(roundedNewLastProcessedTime) >= 0 {
		slog.Info("no connections found since last processed time", "lastProcessedTime", lastProcessedTime)
		return nil
	}

	accounts, err := lastlogin.ProcessFilesParallel(cfg, filePaths)
	if err != nil {
		slog.Error("error processing files", "error_msg", err)
		return err
	}
	slog.Debug("debug accounts found accross all files", "accounts_parsed", len(accounts))

	inserted, updated, err := lastlogin.InsertOrUpdateAccounts(cfg, db, accounts)
	if err != nil {
		slog.Error("failed to insert or update accounts", "error_msg", err)
		return err
	}

	if err := lastlogin.UpdateLastProcessedTime(cfg, db, newLastProcessedTime); err != nil {
		slog.Error("error updating the date of the last parsing.", "error_msg", err)
		return err
	}

	slog.Info("processing run completed", "inserted", inserted, "updated", updated, "files", len(filePaths))

	return nil
}
