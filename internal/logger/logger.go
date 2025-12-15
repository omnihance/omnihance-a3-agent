package logger

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"time"

	"github.com/rs/zerolog"
)

type Field struct {
	Key   string
	Value any
}

type Logger interface {
	Debug(msg string, fields ...Field)
	Info(msg string, fields ...Field)
	Warn(msg string, fields ...Field)
	Error(msg string, fields ...Field)
	With(fields ...Field) Logger
	GetLoggerInstance() interface{}
	Close() error
}

type zerologLogger struct {
	logger         zerolog.Logger
	fileWriter     *DailyFileWriter
	ownsFileWriter bool
}

func NewZerologLogger(l zerolog.Logger, serviceName string, level zerolog.Level) Logger {
	return &zerologLogger{
		logger:         l.With().Str("service", serviceName).Timestamp().Logger().Level(level),
		ownsFileWriter: false,
	}
}

func NewZerologFileLogger(serviceName string, logDir string, level zerolog.Level) Logger {
	err := os.MkdirAll(logDir, 0755)
	if err != nil {
		panic(fmt.Errorf("failed to create log directory: %w", err))
	}

	fileWriter, err := NewDailyFileWriter(serviceName, logDir)
	if err != nil {
		panic(fmt.Errorf("failed to create file writer: %w", err))
	}

	multi := io.MultiWriter(os.Stdout, fileWriter)
	return &zerologLogger{
		logger:         zerolog.New(multi).With().Str("service", serviceName).Timestamp().Logger().Level(level),
		fileWriter:     fileWriter,
		ownsFileWriter: true,
	}
}

func (z *zerologLogger) Debug(msg string, fields ...Field) {
	z.logger.Debug().Fields(toMap(fields)).Msg(msg)
}

func (z *zerologLogger) Info(msg string, fields ...Field) {
	z.logger.Info().Fields(toMap(fields)).Msg(msg)
}

func (z *zerologLogger) Warn(msg string, fields ...Field) {
	z.logger.Warn().Fields(toMap(fields)).Msg(msg)
}

func (z *zerologLogger) Error(msg string, fields ...Field) {
	z.logger.Error().Fields(toMap(fields)).Msg(msg)
}

func (z *zerologLogger) With(fields ...Field) Logger {
	return &zerologLogger{
		logger:         z.logger.With().Fields(toMap(fields)).Logger(),
		fileWriter:     z.fileWriter,
		ownsFileWriter: false,
	}
}

func (z *zerologLogger) GetLoggerInstance() interface{} {
	return z.logger
}

func toMap(fields []Field) map[string]any {
	if len(fields) == 0 {
		return nil
	}

	m := make(map[string]any, len(fields))
	for _, f := range fields {
		m[f.Key] = f.Value
	}

	return m
}

func (z *zerologLogger) Close() error {
	if z.fileWriter != nil && z.ownsFileWriter {
		return z.fileWriter.Close()
	}

	return nil
}

type DailyFileWriter struct {
	service    string
	dir        string
	mu         sync.RWMutex
	file       *os.File
	currDate   string
	ctx        context.Context
	cancel     context.CancelFunc
	wg         sync.WaitGroup
	closed     int32
	lastRotate time.Time
}

func NewDailyFileWriter(service string, logDir string) (*DailyFileWriter, error) {
	ctx, cancel := context.WithCancel(context.Background())
	w := &DailyFileWriter{
		service: service,
		dir:     logDir,
		ctx:     ctx,
		cancel:  cancel,
	}

	if err := w.rotate(); err != nil {
		cancel()
		return nil, fmt.Errorf("initial rotation failed: %w", err)
	}

	w.wg.Add(1)
	go w.autoRotate()
	return w, nil
}

func (w *DailyFileWriter) Close() error {
	if !atomic.CompareAndSwapInt32(&w.closed, 0, 1) {
		return nil
	}

	w.cancel()
	w.wg.Wait()

	w.mu.Lock()
	defer w.mu.Unlock()

	if w.file != nil {
		err := w.file.Close()
		w.file = nil
		return err
	}

	return nil
}

func (w *DailyFileWriter) autoRotate() {
	defer w.wg.Done()

	ticker := time.NewTicker(time.Hour)
	defer ticker.Stop()

	for {
		select {
		case <-w.ctx.Done():
			return
		case <-ticker.C:
			if atomic.LoadInt32(&w.closed) == 1 {
				return
			}

			w.mu.Lock()
			_ = w.rotateInternal()
			w.mu.Unlock()
		}
	}
}

func (w *DailyFileWriter) rotate() error {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.rotateInternal()
}

func (w *DailyFileWriter) rotateInternal() error {
	if atomic.LoadInt32(&w.closed) == 1 {
		return fmt.Errorf("writer is closed")
	}

	now := time.Now()
	date := now.Format("2006-01-02")

	if date == w.currDate && w.file != nil &&
		now.Sub(w.lastRotate) < time.Minute {
		return nil
	}

	if w.file != nil {
		_ = w.file.Close()
		w.file = nil
	}

	filename := filepath.Join(w.dir, fmt.Sprintf("%s_%s.log", w.service, date))
	file, err := os.OpenFile(filename, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("failed to open log file %s: %w", filename, err)
	}

	w.file = file
	w.currDate = date
	w.lastRotate = now
	return nil
}

func (w *DailyFileWriter) Write(p []byte) (int, error) {
	if atomic.LoadInt32(&w.closed) == 1 {
		return 0, fmt.Errorf("writer is closed")
	}

	w.mu.RLock()
	needsRotation := w.needsRotation()
	currentFile := w.file
	w.mu.RUnlock()

	if needsRotation {
		w.mu.Lock()
		if w.needsRotation() {
			if err := w.rotateInternal(); err != nil {
				w.mu.Unlock()
				return 0, fmt.Errorf("rotation failed: %w", err)
			}
		}

		currentFile = w.file
		w.mu.Unlock()
	}

	w.mu.RLock()
	defer w.mu.RUnlock()

	if w.file == nil {
		return 0, fmt.Errorf("log file is not open")
	}

	if w.file != currentFile {
		currentFile = w.file
	}

	return currentFile.Write(p)
}

func (w *DailyFileWriter) needsRotation() bool {
	if w.file == nil {
		return true
	}

	date := time.Now().Format("2006-01-02")
	return date != w.currDate
}

func (w *DailyFileWriter) ForceRotate() error {
	return w.rotate()
}

func (w *DailyFileWriter) CurrentLogFile() string {
	w.mu.RLock()
	defer w.mu.RUnlock()

	if w.file == nil {
		return ""
	}

	return filepath.Join(w.dir, fmt.Sprintf("%s_%s.log", w.service, w.currDate))
}
