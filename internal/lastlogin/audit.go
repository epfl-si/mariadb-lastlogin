package lastlogin

import (
	"bufio"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

func FilterAndSortNewFiles(cfg Config, lastProcessedTime time.Time) ([]string, time.Time, error) {
	var fileInfos []FileInfo
	totalFiles := 0
	slog.Debug("debug", "lastProcessedTime", lastProcessedTime)

	err := filepath.Walk(cfg.AuditLogPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return fmt.Errorf("error accessing path %s: %w", path, err)
		}

		if !info.IsDir() {
			totalFiles++
			if info.ModTime().After(lastProcessedTime) {
				fileInfos = append(fileInfos, FileInfo{
					Path:    path,
					ModTime: info.ModTime(),
				})
			}
		}

		return nil
	})

	if err != nil {
		return nil, lastProcessedTime, fmt.Errorf("error walking through directory: %w", err)
	}

	if totalFiles == 0 {
		return nil, lastProcessedTime, fmt.Errorf("no files found in the directory %s", cfg.AuditLogPath)
	}

	// Sort files by modification time, oldest first
	sort.Slice(fileInfos, func(i, j int) bool {
		return fileInfos[i].ModTime.Before(fileInfos[j].ModTime)
	})

	filePaths := make([]string, len(fileInfos))
	var newLastProcessedTime time.Time
	for i, fileInfo := range fileInfos {
		filePaths[i] = fileInfo.Path
		if fileInfo.ModTime.After(newLastProcessedTime) {
			newLastProcessedTime = fileInfo.ModTime.In(cfg.TimeLocation)
		}
	}

	return filePaths, newLastProcessedTime, nil
}

func ProcessFilesParallel(cfg Config, filenames []string) (map[string]AccountInfo, error) {
	resultChan := make(chan map[string]AccountInfo, len(filenames))
	errorChan := make(chan error, len(filenames))

	// Create a buffered channel to limit the number of concurrent goroutines
	semaphore := make(chan struct{}, cfg.MaxWorkers)

	var wg sync.WaitGroup

	for _, filename := range filenames {
		wg.Add(1)
		go func(file string) {
			semaphore <- struct{}{}
			defer func() {
				<-semaphore
				wg.Done()
			}()

			accounts, err := ParseAuditFile(cfg, file)
			if err != nil {
				errorChan <- fmt.Errorf("error processing %s: %w", file, err)
				return
			}
			resultChan <- accounts
		}(filename)
	}

	go func() {
		wg.Wait()
		close(resultChan)
		close(errorChan)
	}()

	allAccounts := make(map[string]AccountInfo)
	for accounts := range resultChan {
		for key, info := range accounts {
			if existing, ok := allAccounts[key]; !ok || info.LastSeen.After(existing.LastSeen) {
				allAccounts[key] = info
			}
		}
	}

	// Check for any errors
	for err := range errorChan {
		if err != nil {
			return nil, err
		}
	}

	return allAccounts, nil
}

func ParseAuditFile(cfg Config, filename string) (map[string]AccountInfo, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	accounts := make(map[string]AccountInfo)
	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		line := scanner.Text()
		if strings.Contains(line, ",CONNECT") {
			parts := strings.Split(line, ",")
			if len(parts) < 4 {
				continue
			}

			date, err := time.Parse(cfg.TimeFormatAudit, strings.TrimSpace(parts[0]))
			if err != nil {
				continue
			}

			name := strings.TrimSpace(parts[2])
			host := strings.TrimSpace(parts[3])
			key := name + "@" + host

			if existing, ok := accounts[key]; !ok || date.After(existing.LastSeen) {
				accounts[key] = AccountInfo{
					Name:     name,
					Host:     host,
					LastSeen: date,
				}
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return accounts, nil
}
