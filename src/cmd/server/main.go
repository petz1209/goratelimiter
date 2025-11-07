package main

import (
	"context"
	"log/slog"
	"net"

	"github.com/petz1209/goratelimiter/server"
)

/***************************************************************************
RUNNING THE ACTUAL RATELIMITING SERVER

***************************************************************************/

const MaxConcurrency = 10

func main() {
	ctx := context.Background()

	db := server.NewDB(MaxConcurrency)

	svr, err := net.Listen("tcp", ":8000")
	if err != nil {
		panic(err)
	}

	for {

		conn, err := svr.Accept()
		if err != nil {
			continue
		}
		go func(c net.Conn) {
			if err := server.HandleClient(ctx, c, db); err != nil {
				slog.ErrorContext(ctx, err.Error())
			}
		}(conn)

	}

}
