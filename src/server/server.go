package server

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net"
	"strconv"
	"strings"
	"sync"
)

const (
	UNKNOWN_STATE                 = 0
	OK                            = 1
	MAX_USER_CONCURRENCY_REACHED  = 2
	MAX_TOTAL_CONCURRENCY_REACHED = 3
	MAX_VOLUME_REACHED            = 4
)

func NewDB(MaxConcurrency int) *IMDB {
	return &IMDB{
		MaxConcurrency: MaxConcurrency,
		requestsDB:     make(map[string]int),
		volumeDB:       make(map[string]int),
	}

}

type Overview struct {
	Volume      int `json:"volume"`
	Concurrency int `json:"concurrency"`
}

// IMB (In.Memory.Data.Base)
type IMDB struct {
	MaxConcurrency int
	mu             sync.Mutex
	requestsDB     map[string]int // stores how many requests are currently active of a given client
	volumeDB       map[string]int // stores the total query volume of a client
}

// *IMDB.GetStatus
//
//	returns the total number of current request
func (s *IMDB) GetStatus(key string) RateState {

	var groupActive int
	var totalActive int
	var vol int

	s.mu.Lock()
	defer s.mu.Unlock()
	groupActive, ok := s.requestsDB[key]
	if !ok {
		s.requestsDB[key] = 0
		groupActive = 0

	}

	totalActive, ok = s.requestsDB["total"]
	if !ok {
		s.requestsDB["total"] = 0
		totalActive = 0

	}

	vol, ok = s.volumeDB[key]
	if !ok {
		s.volumeDB[key] = 0
		vol = 0

	}

	return RateState{GroupActive: groupActive, TotalActive: totalActive, Volume: vol}

}

func (s *IMDB) IncreaseConcurrency(key string) bool {

	s.mu.Lock()
	defer s.mu.Unlock()

	s.requestsDB[key]++
	s.requestsDB["total"]++
	//fmt.Println("s.reqestDB: ", s.requestsDB[key])
	return true

}

func (s *IMDB) Return(key string, vol int) int {
	s.mu.Lock()
	defer s.mu.Unlock()

	_, ok := s.volumeDB[key]
	if !ok {
		s.volumeDB[key] = vol
	} else {
		s.volumeDB[key] += vol
	}

	s.requestsDB[key]--
	s.requestsDB["total"]--
	return OK

}

func (s *IMDB) IncreaseVolume(key string, v int) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.volumeDB[key] += v
	return true

}

func (s *IMDB) Aquire(key string, userMaxConc, userMaxVol int) int {
	var groupActive int
	var totalActive int
	var vol int

	s.mu.Lock()
	defer s.mu.Unlock()

	groupActive = s.requestsDB[key]
	totalActive = s.requestsDB["total"]
	vol = s.volumeDB[key]

	//fmt.Println(key, "volume:", vol)

	rs := RateState{GroupActive: groupActive, TotalActive: totalActive, Volume: vol}
	if rs.GroupActive < userMaxConc && rs.TotalActive < s.MaxConcurrency && rs.Volume < userMaxVol {
		s.requestsDB[key]++
		s.requestsDB["total"]++
		return OK
	}
	if rs.GroupActive >= userMaxConc {
		return MAX_USER_CONCURRENCY_REACHED
	}
	if rs.TotalActive >= s.MaxConcurrency {
		return MAX_TOTAL_CONCURRENCY_REACHED
	}
	if rs.Volume >= userMaxVol {
		return MAX_VOLUME_REACHED
	}

	return UNKNOWN_STATE

}

func (s *IMDB) ResetAll() (int, error) {

	s.mu.Lock()
	defer s.mu.Unlock()
	s.requestsDB = make(map[string]int)
	s.volumeDB = make(map[string]int)
	return OK, nil

}

func (s *IMDB) AdjustMaxConcurrency(v int) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.MaxConcurrency = v

	return OK, nil

}

func (s *IMDB) Overview() map[string]Overview {

	vdb := make(map[string]int)
	concdb := make(map[string]int)
	for k, v := range s.volumeDB {
		vdb[k] = v
	}
	for k, v := range s.requestsDB {
		concdb[k] = v
	}

	response := make(map[string]Overview)
	total := Overview{}
	for k, v := range vdb {
		response[k] = Overview{Volume: v, Concurrency: concdb[k]}
		total.Volume += v
		total.Concurrency += concdb[k]
	}
	response["total"] = total

	return response

}

type RateState struct {
	GroupActive int  `json:"group_active"`
	TotalActive int  `json:"total_active"`
	Volume      int  `json:"volume"`
	Aquired     bool `json:"aquired"`
}

// HandleClient
//
//	handles interaction with a client connection
func HandleClient(ctx context.Context, conn net.Conn, db *IMDB) error {
	defer conn.Close()
	// json.NewEncoder(conn).Encode(map[string]string{"msg": "connected"})
	SERVER_DISCONNECT := map[string]string{"msg": "server disconnected"}

	buff := make([]byte, 1024)

	for {
		select {
		case <-ctx.Done():
			json.NewEncoder(conn).Encode(SERVER_DISCONNECT)
			return nil

		default:
			n, err := conn.Read(buff)
			if err != nil {
				if err == io.EOF {
					slog.InfoContext(ctx, "client disconnected")
					return nil
				}
				slog.ErrorContext(ctx, err.Error())
				return err
			}

			rec := strings.ToUpper(string(buff[:n]))
			switch {

			// Public APIs
			case strings.HasPrefix(rec, "QUIT"):
				slog.InfoContext(ctx, "client disconnected")
				json.NewEncoder(conn).Encode(SERVER_DISCONNECT)
				return nil

			case strings.HasPrefix(rec, "AQUIRE"):
				//fmt.Println("Suffix AQUIRE")
				handleAquire(conn, rec, db)

			case strings.HasPrefix(rec, "RETURN"):
				handleReturn(conn, rec, db)

			// Admin APIs
			case strings.HasPrefix(rec, "RESET ALL"):
				//slog.InfoContext(ctx, "RESET ALL WAS CALLED")
				handleResetAll(conn, rec, db)

			case strings.HasPrefix(rec, "CONCURRENCY ADJUST"):
				//slog.InfoContext(ctx, "RESET ALL WAS CALLED")
				handleConcurrencyAdjust(conn, rec, db)
			}
			clear(buff)

		}

	}

	return nil

}

func handleAquire(conn net.Conn, rec string, db *IMDB) {
	tokens := strings.Split(rec, " ")
	if len(tokens) != 4 {
		return
	}
	key := tokens[1]
	conc, err := strconv.Atoi(tokens[2])
	if err != nil {
		return
	}

	vol, err := strconv.Atoi(tokens[3])
	if err != nil {
		return
	}

	statusCode := db.Aquire(key, conc, vol)
	json.NewEncoder(conn).Encode(statusCode)
	switch statusCode {
	case 0:
		slog.Error("[0] error AQUIRE operation with status UNKNOWN_STATE")
	case 1:
		slog.Info("[1] successfull AQUIRE operation with status OK")
	case 2:
		slog.Info(fmt.Sprintf("[2] successfull AQUIRE operation with status MAX_USER_CONCURRENCY_REACHED, user: %s", key))
	case 3:
		slog.Info("[3] successfull AQUIRE operation with status MAX_TOTAL_CONCURRENCY_REACHED")
	case 4:
		slog.Info(fmt.Sprintf("[4] successfull AQUIRE operation with status MAX_VOLUME_REACHED, user: %s", key))
	}

}

// handleReturn
//
//	handles the RETURN command.
//	When a user gives back a slot, they return it with their userkey and the data volume consumed.
//	handleReturn takes care of correctly persiting the changes in the database.
func handleReturn(conn net.Conn, rec string, db *IMDB) error {

	var (
		key string
		vol int
	)
	//fmt.Println(rec)
	rec = strings.TrimPrefix(rec, "RETURN ")
	tokens := strings.Split(rec, " ")

	switch len(tokens) {
	case 1:
		key = tokens[0]
	case 2:
		key = tokens[0]
		vol, _ = strconv.Atoi(tokens[1])
	}

	status := db.Return(key, vol)

	switch status {
	case 0:
		slog.Error("[0] error RETURN operation with status UNKNOWN_STATE")
	case 1:
		slog.Info("[1] successfull RETURN operation with status OK")
	}

	json.NewEncoder(conn).Encode(status)

	return nil
}

// handleResetAll
//
//	resets all concurrency settings and also resets all volume settings
func handleResetAll(conn net.Conn, rec string, db *IMDB) error {
	statuscode, err := db.ResetAll()
	if err != nil {
		slog.Error(err.Error())
	}
	switch statuscode {
	case 0:
		slog.Error("[0] error RESET ALL operation with status UNKNOWN_STATE")
	case 1:
		slog.Info("[1] successfull RESET ALL  operation with status OK")
	}

	json.NewEncoder(conn).Encode(statuscode)
	return nil
}

// handleConcurrencyAjdust
//
//	adjusts the max concurrency across all users
func handleConcurrencyAdjust(conn net.Conn, rec string, db *IMDB) error {

	rec = strings.TrimPrefix(rec, "CONCURRENCY ADJUST ")
	newMaxConcurrency, err := strconv.Atoi(rec)
	if err != nil {

	}
	statuscode, err := db.AdjustMaxConcurrency(newMaxConcurrency)
	if err != nil {

	}
	switch statuscode {
	case 0:
		slog.Error("[0] error CONCURRENCY ADJUST operation with status UNKNOWN_STATE")
	case 1:
		slog.Info("[1] successfull CONCURRENCY ADJUST operation with status OK")
	}

	json.NewEncoder(conn).Encode(statuscode)
	return nil
}
