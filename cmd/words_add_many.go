/*
Copyright © 2024 Konrad Nowara

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in
all copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
THE SOFTWARE.
*/
package cmd

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/kndrad/wcrack/config"
	"github.com/kndrad/wcrack/internal/textproc"
	"github.com/kndrad/wcrack/internal/textproc/database"
	"github.com/kndrad/wcrack/pkg/retry"
	"github.com/spf13/cobra"
)

var addManyWordsCmd = &cobra.Command{
	Use:     "many",
	Short:   "Adds many words to a database.",
	Example: "wcrack words add many [FILE PATH <name>.txt | <name>.json]",
	RunE: func(cmd *cobra.Command, args []string) error {
		logger := DefaultLogger(Verbose)

		cfg, err := config.Load("config/development.yaml")
		if err != nil {
			logger.Error("Loading database config", "err", err.Error())

			return fmt.Errorf("config load: %w", err)
		}
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()

		pool, err := textproc.DatabasePool(ctx, cfg.Database)
		if err != nil {
			logger.Error("Loading database pool", "err", err.Error())

			return fmt.Errorf("database pool: %w", err)
		}
		defer pool.Close()

		if err := retry.Ping(ctx, pool, retry.MaxRetries); err != nil {
			logger.Error("Pinging database", "err", err.Error())

			return fmt.Errorf("database ping: %w", err)
		}

		conn, err := textproc.DatabaseConnection(ctx, pool)
		if err != nil {
			logger.Error("Connecting to database", "err", err.Error())

			return fmt.Errorf("database connection: %w", err)
		}
		defer conn.Close(ctx)

		// Read words from a json file
		analysis := new(textproc.TextAnalysis)
		path := filepath.Clean(args[0])

		switch filepath.Ext(path) {
		case ".json":
			data, err := os.ReadFile(args[0])
			if err != nil {
				logger.Error("Failed to read file",
					slog.String("path", path),
				)

				return fmt.Errorf("read file: %w", err)
			}
			if err := json.Unmarshal(data, &analysis); err != nil {
				logger.Error("Failed to unmarshal json into analysis")

				return fmt.Errorf("unmarshal json: %w", err)
			}
		case ".txt":
			data, err := os.ReadFile(args[0])
			if err != nil {
				logger.Error("Failed to read file",
					slog.String("path", path),
				)

				return fmt.Errorf("read file: %w", err)
			}
			scanner := bufio.NewScanner(bytes.NewReader(data))
			scanner.Split(bufio.ScanWords)

			for scanner.Scan() {
				word := strings.Trim(scanner.Text(), " ")
				analysis.IncWordCount(word)
			}

			if err := scanner.Err(); err != nil {
				logger.Error("Scanner returned an error", "err", err.Error())

				return fmt.Errorf("scanner err: %w", err)
			}
		}
		if Verbose {
			printWords(analysis)
		}

		// Query db to insert each word
		q := database.New(conn)
		for word := range analysis.WordFrequency {
			row, err := q.CreateWord(ctx, word)
			if err != nil {
				logger.Error("Failed to insert word",
					slog.String("word", word),
				)

				return fmt.Errorf("word insert: %w", err)
			}
			logger.Info("Inserted row to a database",
				slog.Int64("id", row.ID),
				slog.String("word", row.Value),
			)
		}

		logger.Info("Program completed successfully.")

		return nil
	},
}

func init() {
	addWordCmd.AddCommand(addManyWordsCmd)
}

func printWords(analysis *textproc.TextAnalysis) {
	for word, frequency := range analysis.WordFrequency {
		fmt.Printf("WORD: %s, FREQUENCY: %s\n", word, strconv.Itoa(frequency))
	}
}
