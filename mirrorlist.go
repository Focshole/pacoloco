package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"strings"
	"time"
)

func setupMirrorlistsTimers() []*time.Ticker {
	tickers := make([]*time.Ticker, 0)
	for name, repo := range config.Repos {
		if repo.MirrorlistCron != "" && repo.Mirrorlist != "" { // if a mirrorlist with duration is specified in its config
			duration, err := getCronDuration(repo.MirrorlistCron, time.Now())
			if err == nil && duration > 0 {
				ticker := time.NewTicker(duration) // set prefetch as specified in config file
				go func() {
					lastTimeInvoked := time.Time{}
					for range ticker.C {
						if time.Since(lastTimeInvoked) > time.Second {
							updateRepoMirrorlist(name)
							lastTimeInvoked = time.Now()
							duration, err := getCronDuration(repo.MirrorlistCron, time.Now())
							if err == nil && duration > 0 {
								ticker.Reset(duration) // update to the new timing
							} else {
								ticker.Stop()
							}
						} // otherwise ignore it. It happened more than once that this function gets invoked twice for no reason
					}
				}()
				tickers = append(tickers, ticker)
			}
		}
	}
	return tickers

}

func updateMirrorlists() {
	for name, repo := range config.Repos {
		if repo.Mirrorlist != "" {
			err := updateRepoMirrorlist(name)
			if err != nil {
				log.Fatalf("Error while updating %v repo mirrorlist: %v\n", name, err)
			}
		}
	}
}

func updateRepoMirrorlist(repoName string) error {
	repo, ok := config.Repos[repoName]
	if !ok {
		return fmt.Errorf("repo %v does not exist in config", repoName)
	}
	file, err := os.Open(repo.Mirrorlist)
	if err != nil {
		return err
	}
	defer file.Close()
	// initialize the urls collection
	repo.URLs = make([]string, 0)
	scanner := bufio.NewScanner(file)
	// resize scanner's capacity if lines are longer than 64K.
	for scanner.Scan() {
		matches := mirrorlistRegex.FindStringSubmatch(scanner.Text())
		if len(matches) > 0 {
			url := matches[1]
			if !strings.Contains(url, "$") {
				repo.URLs = append(repo.URLs, url)
			} else {
				// this can be a regex error or otherwise a very peculiar url
				log.Printf("warning: %v url in repo %v contains suspicious characters, skipping it", url, repoName)
			}

		}
		// skip invalid lines
	}
	if len(repo.URLs) == 0 {
		return fmt.Errorf("mirrorlist for repo %v is either empty or isn't a mirrorlist file", repoName)
	}

	if err := scanner.Err(); err != nil {
		return err
	}
	// update config
	config.Repos[repoName] = repo
	return nil
}
