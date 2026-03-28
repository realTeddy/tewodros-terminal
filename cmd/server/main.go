package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	gb "tewodros-terminal/internal/guestbook"
	sshserver "tewodros-terminal/internal/ssh"
	webserver "tewodros-terminal/internal/web"
)

func main() {
	sshHost := envOr("SSH_HOST", "0.0.0.0")
	sshPort := envOr("SSH_PORT", "22")
	httpHost := envOr("HTTP_HOST", "0.0.0.0")
	httpPort := envOr("HTTP_PORT", "8080")
	hostKeyDir := envOr("HOST_KEY_DIR", ".ssh")
	dbPath := envOr("DB_PATH", "guestbook.db")

	os.MkdirAll(hostKeyDir, 0700)

	guestbook, err := gb.New(dbPath)
	if err != nil {
		log.Fatalf("failed to init guestbook: %v", err)
	}
	defer guestbook.Close()

	sshSrv, err := sshserver.NewServer(sshserver.Config{
		Host:       sshHost,
		Port:       sshPort,
		HostKeyDir: hostKeyDir,
		Guestbook:  guestbook,
	})
	if err != nil {
		log.Fatalf("failed to create ssh server: %v", err)
	}

	go func() {
		log.Printf("SSH server listening on %s:%s", sshHost, sshPort)
		if err := sshSrv.ListenAndServe(); err != nil {
			log.Fatalf("ssh server error: %v", err)
		}
	}()

	httpSrv := webserver.NewServer(webserver.Config{
		Host:      httpHost,
		Port:      httpPort,
		Guestbook: guestbook,
	})

	go func() {
		log.Printf("HTTP server listening on %s:%s", httpHost, httpPort)
		if err := httpSrv.ListenAndServe(); err != nil {
			log.Printf("http server error: %v", err)
		}
	}()

	fmt.Println("tewodros-terminal is running")
	fmt.Printf("  SSH:  ssh -p %s %s\n", sshPort, sshHost)
	fmt.Printf("  HTTP: http://%s:%s\n", httpHost, httpPort)

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	<-ctx.Done()

	log.Println("shutting down...")
	sshSrv.Close()
	httpSrv.Shutdown(context.Background())
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
