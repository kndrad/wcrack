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
	"bytes"
	"context"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"

	"github.com/kndrad/wcrack/config"
	"github.com/spf13/cobra"
)

var apiHealthzCmd = &cobra.Command{
	Use:   "healthz",
	Short: "Checks health http API server",
	RunE: func(cmd *cobra.Command, args []string) error {
		logger := DefaultLogger(Verbose)

		cfg, err := config.Load(os.Getenv("CONFIG_PATH"))
		if err != nil {
			logger.Error("Failed to load config", "err", err)

			return fmt.Errorf("loading config err: %w", err)
		}
		url := net.JoinHostPort(cfg.HTTP.Host, cfg.HTTP.Port) + "/api/v1/healthz"
		buf := new(bytes.Buffer)
		req, err := http.NewRequestWithContext(
			context.TODO(),
			http.MethodGet,
			url,
			buf,
		)
		if err != nil {
			logger.Error("Failed to create request", "err", err)

			return fmt.Errorf("new request err: %w", err)
		}
		logger.Info("Sending request",
			slog.String("url", url),
		)

		c := &http.Client{}
		defer c.CloseIdleConnections()

		resp, err := c.Do(req)
		if err != nil {
			logger.Error("Failed to do request with a client", "err", err)

			return fmt.Errorf("client do request err: %w", err)
		}
		defer resp.Body.Close()

		switch resp.StatusCode {
		case http.StatusOK:
			logger.Info("Received response and server OK", "statusCode", resp.StatusCode)

			return nil
		case http.StatusNotFound:
			logger.Info("Received not found", "statusCode", resp.StatusCode)
		default:
			logger.Info("Received response", "statusCode", resp.StatusCode)
		}

		logger.Info("Program completed successfully.")

		return nil
	},
}

func init() {
	apiCmd.AddCommand(apiHealthzCmd)
}
