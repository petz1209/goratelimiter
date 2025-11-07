package main

import (
	"fmt"
	"net"
)

const (
	Host                  = "localhost:8000"
	MAX_TOTAL_CONCURRENCY = 200
)

func main() {

	//setMaxTotalConcurrency()
	resetAll()
	setMaxTotalConcurrency()
}

func setMaxTotalConcurrency() {

	conn, err := net.Dial("tcp", Host)
	if err != nil {
	}
	cmd := fmt.Sprintf("CONCURRENCY ADJUST %d", MAX_TOTAL_CONCURRENCY)
	conn.Write([]byte(cmd))
	conn.Close()

}

func resetAll() {
	conn, err := net.Dial("tcp", Host)
	if err != nil {
		panic(err)
	}

	cmd := "RESET ALL"
	conn.Write([]byte(cmd))
	conn.Close()
}
