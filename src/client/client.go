package client

import (
	"encoding/json"
	"fmt"
	"net"
	"strings"
)

const (
	UNKNOWN_STATE                 = 0
	OK                            = 1
	MAX_USER_CONCURRENCY_REACHED  = 2
	MAX_TOTAL_CONCURRENCY_REACHED = 3
	MAX_VOLUME_REACHED            = 4
)

func NewRateLimitClient(addr string) *RateLimitClient {

	cl := &RateLimitClient{Addr: addr, MaxReconnectAttempts: 3}
	cl.Connect()
	return cl

}

type RateState struct {
	GroupActive int  `json:"group_active"`
	TotalActive int  `json:"total_active"`
	Volume      int  `json:"volume"`
	Aquired     bool `json:"aquired"`
}

type RateLimitClient struct {
	ReconnectAttempts    int
	MaxReconnectAttempts int
	Addr                 string
	conn                 net.Conn
}

func (c *RateLimitClient) SetMaxReconnectAttemps(v int) *RateLimitClient {
	c.MaxReconnectAttempts = v
	return c

}

func (c *RateLimitClient) Close() {
	c.conn.Close()
}

func (c *RateLimitClient) Connect() error {
	conn, err := net.Dial("tcp", c.Addr)
	if err != nil {
		c.ReconnectAttempts++
		return err
	}
	c.ReconnectAttempts = 0
	c.conn = conn
	return nil

}
func (c *RateLimitClient) Reconnect() error {
	return c.Connect()
}

func (c *RateLimitClient) Aquire(key string, maxConcurrency, maxVolume int) (int, error) {

	var statuscode int

	buff := make([]byte, 1024)
	_, err := c.conn.Write([]byte(fmt.Sprintf("AQUIRE %s %d %d", key, maxConcurrency, maxVolume)))
	if err != nil {
		if strings.HasSuffix(err.Error(), "broken pipe") {
			if c.ReconnectAttempts < c.MaxReconnectAttempts {
				c.Reconnect()
				return c.Aquire(key, maxConcurrency, maxVolume)
			}

		}

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
		if strings.HasSuffix(err.Error(), "broken pipe") {
			if c.ReconnectAttempts < c.MaxReconnectAttempts {
				c.Reconnect()
				return c.Return(key, volume)
			}

		}
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
