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
	"context"
	"fmt"
	"math"
	"strconv"
	"time"

	"github.com/kndrad/wcrack/config"
	"github.com/kndrad/wcrack/internal/textproc"
	"github.com/kndrad/wcrack/internal/textproc/database"
	"github.com/kndrad/wcrack/pkg/retry"
	"github.com/spf13/cobra"
)

// wordsCmd represents the words command.
var wordsCmd = &cobra.Command{
	Use:     "words",
	Short:   "Lists words from a database",
	Example: "wcrack words [OPTIONAL args: limit[int32]]",
	RunE: func(cmd *cobra.Command, args []string) error {
		logger := DefaultLogger(Verbose)

		cfg, err := config.Load("config/development.yaml")
		if err != nil {
			logger.Error("Loading onfig", "err", err.Error())

			return fmt.Errorf("load config: %w", err)
		}

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
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

		q := database.New(conn)

		limit := math.MaxInt32
		if len(args) > 0 {
			limitArg, err := strconv.ParseInt(args[0], 10, 32)
			if err != nil {
				return fmt.Errorf("parse uint: %w", err)
			}
			if limitArg < math.MaxInt32 {
				limit = int(limitArg)
			}
		}
		params := database.ListWordsParams{Limit: int32(limit)}

		if len(args) > 0 {
			limit, err := strconv.ParseInt(args[0], 10, 32)
			if err != nil {
				logger.Error("Failed to strconv", "err", err.Error())
			}
			params.Limit = int32(limit)
		}
		words, err := q.ListWords(ctx, params)
		if err != nil {
			logger.Error("Connecting to database", "err", err.Error())

			return fmt.Errorf("database connection: %w", err)
		}
		logger.Info("Listing words from a database", "len_words", len(words))
		for _, word := range words {
			fmt.Printf("%v\n", word)
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(wordsCmd)
}
