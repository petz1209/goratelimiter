package client

import (
	"encoding/json"
	"fmt"
	"log"
	"net"
)

const (
	UNKNOWN_STATE                 = 0
	OK                            = 1
	MAX_USER_CONCURRENCY_REACHED  = 2
	MAX_TOTAL_CONCURRENCY_REACHED = 3
	MAX_VOLUME_REACHED            = 4
)

func NewRateLimitClient(addr string) *RateLimitClient {

	conn, err := net.Dial("tcp", addr)
	if err != nil {
		log.Fatal(err)
	}
	return &RateLimitClient{
		conn: conn,
	}

}

type RateState struct {
	GroupActive int  `json:"group_active"`
	TotalActive int  `json:"total_active"`
	Volume      int  `json:"volume"`
	Aquired     bool `json:"aquired"`
}

type RateLimitClient struct {
	conn net.Conn
}

func (c *RateLimitClient) Close() {
	c.conn.Close()
}

func (c *RateLimitClient) Aquire(key string, maxConcurrency, maxVolume int) (int, error) {

	var statuscode int

	buff := make([]byte, 1024)
	_, err := c.conn.Write([]byte(fmt.Sprintf("AQUIRE %s %d %d", key, maxConcurrency, maxVolume)))
	if err != nil {
		return UNKNOWN_STATE, err
	}
	n, err := c.conn.Read(buff)
	if err != nil {
		return UNKNOWN_STATE, err
	}

	if err := json.Unmarshal(buff[:n], &statuscode); err != nil {
		return UNKNOWN_STATE, err
	}
	return statuscode, nil

}

func (c *RateLimitClient) Return(key string, volume int) (int, error) {
	buff := make([]byte, 1024)

	msg := fmt.Sprintf("RETURN %s %d", key, volume)
	_, err := c.conn.Write([]byte(msg))
	if err != nil {
		return UNKNOWN_STATE, err
	}

	var response int
	n, err := c.conn.Read(buff)
	if err != nil {
		return UNKNOWN_STATE, err
	}
	if err = json.Unmarshal(buff[:n], &response); err != nil {
		return UNKNOWN_STATE, err
	}
	return response, nil
}
