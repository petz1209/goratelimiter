package main

import (
	"fmt"
	"math/rand/v2"
	"net/http"
	"sync"
	"time"
)

/**********************************************************************
	runs increasing scapes on the http server
	and documents the status codes
**********************************************************************/

const host = "http://localhost:3000"

func main() {

	tr := http.Transport{MaxConnsPerHost: 200,
		MaxIdleConnsPerHost: 200}
	client := &http.Client{Transport: &tr}
	defer client.CloseIdleConnections()

	//RunRequest(client, 1)

	for i := 10; i < 100; i += 10 {

		RunRequests(client, i)

	}

}

func RunRequests2(count int) {

}

func RunRequests(client *http.Client, count int) {
	var wg sync.WaitGroup
	wg.Add(count)
	for i := range count {
		go func() {
			defer wg.Done()
			RunRequest(client, i)
		}()
	}

	wg.Wait()
	fmt.Println("----------------------------------------------")
	fmt.Println("finished run with count:", count)
	fmt.Println("----------------------------------------------")
	time.Sleep(1 * time.Second)
}

func RunRequest(client *http.Client, id int) {
	users := []string{"patrick", "janina", "anna", "maria", "jos", "timo", "jessie", "papa"}

	user := users[rand.IntN(len(users))]
	req, err := http.NewRequest("GET", host+"/"+user, nil)
	if err != nil {
		fmt.Println("invalid http request with error: " + err.Error())
		return
	}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Println("http request with error: " + err.Error())
		return
	}
	defer resp.Body.Close()
	fmt.Printf("[%d] RequestId: %d Endpoint: /%s\n", resp.StatusCode, id, user)

}
