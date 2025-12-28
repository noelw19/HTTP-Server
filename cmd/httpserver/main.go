package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/noelw19/tcptohttp/internal/server"
)

const port = 42069

func main() {
	server := server.Serve(port)

	server.AddHandler("/test", wakandaHandler).GET()
	server.AddHandler("/wakanda", wakandaHandler).GET()
	server.AddHandler("/wakanda", wakandaPOSTHandler).POST()
	server.AddHandler("/wakanda/{id}/{lala}", wakandaIDHandler).GET()
	server.AddHandler("/query", queryHandler).GET()
	server.AddHandler("/httpbin/stream", streamHandler)
	server.AddHandler("/video", videoHandler)

	log.Println("Server started on port", port)

	server.Listen()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan
	server.Close()
	log.Println("Server gracefully stopped")
}
