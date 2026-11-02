package httpserver

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func Run(address string, handler http.Handler) error {
	return RunWithWorkers(address, handler)
}

func RunWithWorkers(address string, handler http.Handler, workers ...func(context.Context) error) error {
	server := &http.Server{
		Addr:              address,
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       60 * time.Second,
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	serveErrors := make(chan error, 1)
	go func() {
		serveErrors <- server.ListenAndServe()
	}()
	workerErrors := make(chan error, len(workers))
	for _, worker := range workers {
		go func(run func(context.Context) error) {
			err := run(ctx)
			if err == nil && ctx.Err() == nil {
				err = errors.New("worker returned without cancellation")
			}
			workerErrors <- err
		}(worker)
	}

	select {
	case err := <-serveErrors:
		cancel()
		if shutdownErr := shutdownServer(server); shutdownErr != nil {
			return fmt.Errorf("HTTP server stopped (%v); shutdown failed: %w", err, shutdownErr)
		}
		if workerErr := waitWorkers(workerErrors, len(workers)); workerErr != nil {
			return workerErr
		}
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	case err := <-workerErrors:
		cancel()
		if shutdownErr := shutdownServer(server); shutdownErr != nil {
			return fmt.Errorf("background worker stopped (%v); HTTP shutdown failed: %w", err, shutdownErr)
		}
		serverErr := <-serveErrors
		workerErr := waitWorkers(workerErrors, len(workers)-1)
		if serverErr != nil && !errors.Is(serverErr, http.ErrServerClosed) {
			return fmt.Errorf("HTTP server stopped: %w", serverErr)
		}
		if err != nil {
			if workerErr != nil {
				return fmt.Errorf("background worker stopped: %v; %w", err, workerErr)
			}
			return fmt.Errorf("background worker stopped: %w", err)
		}
		if workerErr != nil {
			return workerErr
		}
		return errors.New("background worker stopped unexpectedly")
	case <-ctx.Done():
		if err := shutdownServer(server); err != nil {
			return err
		}
		err := <-serveErrors
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			return fmt.Errorf("HTTP server stopped: %w", err)
		}
		return waitWorkers(workerErrors, len(workers))
	}
}

func shutdownServer(server *http.Server) error {
	shutdownContext, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := server.Shutdown(shutdownContext); err != nil {
		return fmt.Errorf("shut down HTTP server: %w", err)
	}
	return nil
}

func waitWorkers(workerErrors <-chan error, count int) error {
	var firstErr error
	for range count {
		if err := <-workerErrors; err != nil && firstErr == nil {
			firstErr = err
		}
	}
	if firstErr != nil {
		return fmt.Errorf("background worker stopped: %w", firstErr)
	}
	return nil
}
