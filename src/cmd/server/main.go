package main

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"log"
	"log/slog"
	"net"
	"net/http"
	"time"

	"github.com/petz1209/goratelimiter/server"
)

/***************************************************************************
RUNNING THE ACTUAL RATELIMITING SERVER
***************************************************************************/

//go:embed overview.html
var overviewHtml []byte

const MaxConcurrency = 10

func main() {
	ctx := context.Background()

	db := server.NewDB(MaxConcurrency)
	go func() {

		if err := TcpServerInit(ctx, "8000", db); err != nil {
			log.Fatal(err)

		}
	}()

	go func() {
		if err := HttpServerInit(ctx, "8001", db); err != nil {
			log.Fatal(err)
		}
	}()

	<-ctx.Done()

}

func TcpServerInit(ctx context.Context, port string, db *server.IMDB) error {
	fmt.Println("starting tcp server on port", port)
	svr, err := net.Listen("tcp", ":"+port)
	if err != nil {
		return err
	}

	for {

		conn, err := svr.Accept()
		if err != nil {
			continue
		}
		// make it a long living connection be configuring a heartbeat
		if tcpConn, ok := conn.(*net.TCPConn); ok {
			tcpConn.SetKeepAlive(true)
			tcpConn.SetKeepAlivePeriod(30 * time.Second)
		}
		go func(c net.Conn) {
			if err := server.HandleClient(ctx, c, db); err != nil {
				slog.ErrorContext(ctx, err.Error())
			}
		}(conn)

	}

}

func HttpServerInit(ctx context.Context, port string, db *server.IMDB) error {

	mux := http.NewServeMux()

	mux.HandleFunc("GET /overview", func(w http.ResponseWriter, r *http.Request) {
		ov := db.Overview()
		w.WriteHeader(200)
		json.NewEncoder(w).Encode(ov)
	})

	mux.HandleFunc("GET /page", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		// Schreiben Sie einfach die statischen Bytes in den ResponseWriter
		w.Write(overviewHtml)
	})

	srv := http.Server{
		Addr:    ":" + port,
		Handler: mux,
	}

	fmt.Println("starting http server on port", port)
	if err := srv.ListenAndServe(); err != nil {
		return err
	}
	return nil

}
