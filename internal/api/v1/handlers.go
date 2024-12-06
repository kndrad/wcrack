package v1

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"math"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/kndrad/wcrack/internal/textproc/database"
	"github.com/kndrad/wcrack/pkg/ocr"
)

func healthCheckHandler(logger *slog.Logger) http.HandlerFunc {
	type Response struct {
		Status int `json:"status"`
	}

	return func(w http.ResponseWriter, r *http.Request) {
		logger.Info("Received health check request",
			slog.String("url", r.URL.String()),
		)

		resp := Response{
			Status: http.StatusOK,
		}
		if err := encode(w, r, http.StatusOK, &resp); err != nil {
			writeJSONErr(w, "Failed to check health", err, http.StatusInternalServerError)
		}
	}
}

func listWordsHandler(svc WordService, logger *slog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		logger.Info("Received request",
			slog.String("url", r.URL.String()),
		)
		limit, err := limitValue(r.URL.Query())
		if err != nil {
			http.Error(w, "Failed get limit param from query", http.StatusBadRequest)

			return
		}
		offset, err := offsetValue(r.URL.Query())
		if err != nil {
			http.Error(w, "Failed get limit param from query", http.StatusBadRequest)

			return
		}
		rows, err := svc.ListWords(r.Context(), limit, offset)
		if err != nil {
			http.Error(w, "Failed to fetch all words from a database", http.StatusInternalServerError)

			return
		}
		if err := encode(w, r, http.StatusOK, rows); err != nil {
			http.Error(w, "Failed to encode rows", http.StatusInternalServerError)

			return
		}
	}
}

func createWordHandler(svc WordService, logger *slog.Logger) http.HandlerFunc {
	type Request struct {
		Value string `json:"value"`
	}

	return func(w http.ResponseWriter, r *http.Request) {
		logger.Info("Received request",
			slog.String("url", r.URL.String()),
		)

		request, err := decode[Request](r)
		if err != nil {
			http.Error(w, "Failed to decode request", http.StatusBadRequest)

			return
		}
		row, err := svc.CreateWord(r.Context(), request.Value)
		if err != nil {
			http.Error(w, "Failed to insert word", http.StatusInternalServerError)

			return
		}
		logger.Info("Inserted word",
			slog.Int64("id", row.ID),
			slog.String("value", row.Value),
		)
		if err := encode(w, r, http.StatusOK, row); err != nil {
			http.Error(w, "Failed to encode insert word row", http.StatusInternalServerError)

			return
		}
	}
}

func uploadWordsHandler(svc WordService, logger *slog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		logger.Info("Received request",
			slog.String("url", r.URL.String()),
		)

		// Parse file from form
		const MaxSize = 1024 * 1024 * 20 // 20 MB
		r.Body = http.MaxBytesReader(w, r.Body, MaxSize)

		if err := r.ParseMultipartForm(MaxSize); err != nil {
			writeJSONErr(w, "File too big", err, http.StatusBadRequest)
		}
		f, fheader, err := r.FormFile("file")
		if err != nil {
			writeJSONErr(w, "Failed to get file", err, http.StatusBadRequest)
		}
		defer f.Close()

		// Detect content type of file and check if it's a text
		data := make([]byte, 512)
		if _, err := f.Read(data); err != nil {
			writeJSONErr(w, "Failed to read file into buffer", err, http.StatusInternalServerError)
		}
		contentType := http.DetectContentType(data)
		allowed := func() bool {
			types := map[string]bool{
				"text/plain":                true,
				"text/plain; charset=utf-8": true,
				"application/txt":           true,
				"text/x-plain":              true,
			}

			return types[contentType]
		}
		if !allowed() {
			writeJSONErr(w,
				fmt.Sprintf("Content type %s not allowed. Upload text file", contentType),
				nil,
				http.StatusBadRequest,
			)
		}
		// Return pointer back to the start of the file after content type detection
		if _, err := f.Seek(0, io.SeekStart); err != nil {
			writeJSONErr(w, "Failed to seek to start of the file", err, http.StatusInternalServerError)
		}
		logger.Info("Received form",
			slog.String("filename", fheader.Filename),
		)

		// Read many words one by one from file
		scanner := bufio.NewScanner(f)
		scanner.Split(bufio.ScanWords)

		var words []string
		for scanner.Scan() {
			words = append(words, scanner.Text())
		}
		if err := scanner.Err(); err != nil {
			writeJSONErr(w, "Scanner returned an error", err, http.StatusInternalServerError)
		}
		count := 0
		for _, word := range words {
			_, err := svc.CreateWord(r.Context(), word)
			if err != nil {
				writeJSONErr(w, "Failed to insert row", err, http.StatusInternalServerError)
			}
			count++
		}
		type Response struct {
			Count int `json:"count"`
		}
		resp := Response{
			Count: count,
		}
		if err := encode(w, r, http.StatusOK, resp); err != nil {
			writeJSONErr(w, "Failed to insert row", err, http.StatusInternalServerError)
		}
		logger.Info("Inserted words", slog.Int("count", count))
	}
}

func uploadImageWordsHandler(svc WordService, logger *slog.Logger) http.HandlerFunc {
	var maxSize int64 = 1024 * 1024 * 50 // 50 MB

	return func(w http.ResponseWriter, r *http.Request) {
		r.Body = http.MaxBytesReader(w, r.Body, maxSize)

		if err := r.ParseMultipartForm(maxSize); err != nil {
			writeJSONErr(w, "Image file too big", err, http.StatusBadRequest)
		}
		f, fh, err := r.FormFile("image")
		if err != nil {
			writeJSONErr(w, "Failed to get image file", err, http.StatusBadRequest)
		}
		defer f.Close()

		// Detect content type of file and check if it's a text
		data := make([]byte, 512)
		if _, err := f.Read(data); err != nil {
			writeJSONErr(w, "Failed to read image file into buffer", err, http.StatusInternalServerError)
		}
		contentType := http.DetectContentType(data)

		var allowed bool

		switch contentType {
		case "image/jpeg", "image/png":
			allowed = true
		default:
			allowed = false
		}
		// Return pointer back to the start of the file after content type detection
		if _, err := f.Seek(0, io.SeekStart); err != nil {
			writeJSONErr(w, "Failed to seek to start of the file", err, http.StatusInternalServerError)
		}
		if !allowed {
			writeJSONErr(w,
				fmt.Sprintf("Content type %s not allowed. Upload text file", contentType),
				nil,
				http.StatusBadRequest,
			)
		}
		logger.Info("Received form", slog.String("header_filename", fh.Filename))

		// Get OCR result from uploaded image
		content := make([]byte, ocr.MaxImageSize)
		if _, err := f.Read(content); err != nil {
			writeJSONErr(w,
				"Failed to read file content for image words recognition.",
				err,
				http.StatusInternalServerError,
			)
		}
		c, err := ocr.NewClient()
		if err != nil {
			writeJSONErr(w,
				"Failed to init tesseract client",
				err,
				http.StatusInternalServerError,
			)
		}
		result, err := ocr.Run(c, ocr.NewImage(content))
		if err != nil {
			writeJSONErr(w,
				"Failed to recognize words from an image",
				err,
				http.StatusInternalServerError,
			)
		}

		var builder strings.Builder
		for w := range result.Words() {
			builder.WriteString(w.String() + " ")
		}
		if _, err := w.Write([]byte(builder.String())); err != nil {
			writeJSONErr(w,
				"Failed to write ",
				err,
				http.StatusInternalServerError,
			)
		}
	}
}

func listWordBatchesHandler(svc WordService, logger *slog.Logger) http.HandlerFunc {
	type response struct {
		Results []database.ListWordBatchesRow `json:"word_batches"`
	}

	return func(w http.ResponseWriter, r *http.Request) {
		limit, err := limitValue(r.URL.Query())
		if err != nil {
			writeJSONErr(w, "Failed to get limit query value", err, http.StatusBadRequest)
		}
		offset, err := offsetValue(r.URL.Query())
		if err != nil {
			writeJSONErr(w, "Failed to get offset query value", err, http.StatusBadRequest)
		}

		rows, err := svc.ListWordBatches(r.Context(), limit, offset)
		if err != nil {
			writeJSONErr(w, "Failed to list word batches via word service", err, http.StatusInternalServerError)
		}

		resp := response{
			Results: rows,
		}
		logger.Info("Got word batches", "total", len(rows))

		if err := encode(w, r, http.StatusOK, resp); err != nil {
			writeJSONErr(w, "Failed to serve response", err, http.StatusInternalServerError)
		}
	}
}

func limitValue(values url.Values) (int32, error) {
	var param string
	const defaultLimitParam = "1000"

	param = values.Get("limit")
	if param == "" {
		param = defaultLimitParam
	}
	limit, err := strconv.ParseUint(param, 10, 32)
	if err != nil {
		return 0, fmt.Errorf("parse uint: %w", err)
	}
	if limit > math.MaxInt32 {
		limit = 1000
	}

	return int32(limit), nil
}

func offsetValue(values url.Values) (int32, error) {
	var param string
	const defaultOffsetParam = "0"

	param = values.Get("offset")
	if param == "" {
		param = defaultOffsetParam
	}
	offset, err := strconv.ParseUint(param, 10, 32)
	if err != nil {
		return 0, fmt.Errorf("parse uint: %w", err)
	}
	if offset > math.MaxInt32 {
		offset = 0
	}

	return int32(offset), nil
}

func encode[T any](w http.ResponseWriter, _ *http.Request, status int, v T) error {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	if err := json.NewEncoder(w).Encode(v); err != nil {
		return fmt.Errorf("encode json: %w", err)
	}

	return nil
}

func decode[T any](r *http.Request) (T, error) {
	var v T

	if err := json.NewDecoder(r.Body).Decode(&v); err != nil {
		return v, fmt.Errorf("decode json: %w", err)
	}

	return v, nil
}

func writeJSONErr(w http.ResponseWriter, msg string, err error, code int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)

	response := struct {
		Message string `json:"message"`
		Error   string `json:"error,omitempty"`
	}{
		Message: msg,
	}
	if err != nil {
		response.Error = err.Error()
	}

	if err := json.NewEncoder(w).Encode(response); err != nil {
		fmt.Fprintf(w, "Internal Server Error: failed to marshal error response")
	}
}
