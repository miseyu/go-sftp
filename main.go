package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/miseyu/go-sftp/pkg/sftp"
	"github.com/miseyu/go-sftp/pkg/sftp/gcs"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM, syscall.SIGQUIT)
	defer stop()
	handler, err := gcs.NewGoogleCloudStorageHandler(ctx, "")
	if err != nil {
		log.Printf("create gcs handler error with %v", err)
		os.Exit(1)
	}
	// user password
	s := sftp.NewServer(10022, "", "", handler)

	go func() {
		<-ctx.Done()
		s.Close()
		os.Exit(0)
	}()
	s.ListenAndServe(ctx)
}
