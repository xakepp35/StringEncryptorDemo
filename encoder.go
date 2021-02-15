package main

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"net/http"
	"os"

	"github.com/gobwas/ws"
	"github.com/gobwas/ws/wsutil"
)

// EncodeJobData payload carrier & reordering foo fighters
type EncodeJobData struct {
	Payload string
	LineNum int
}

// EncodeJob adds result queue to write to
type EncodeJob struct {
	EncodeJobData
	ResultsQueue chan EncodeJobData
}

// microservice-wide queue
var jobsQueue chan *EncodeJob

//func encode_worker(id int, jobsQueue <-chan *EncodeJob) {
func encode_worker(id int) {
	fmt.Printf("Worker %d: started\n", id)
	for j := range jobsQueue {
		// See example at: https://golang.org/pkg/crypto/sha256/
		hashCtx := sha256.New()
		hashCtx.Write([]byte(j.Payload))
		hashSumBlob := hashCtx.Sum(nil)
		hashHexString := fmt.Sprintf("%x", hashSumBlob)
		fmt.Printf("worker %d: %s -> %s\n", id, j.Payload, hashHexString)
		j.ResultsQueue <- EncodeJobData{
			Payload: hashHexString,
			LineNum: j.LineNum,
		}
	}
}

// EncodedJobDataByLineNum implements sort.Interface
// type EncodedJobDataByLineNum []EncodeJobData

// func (a EncodedJobDataByLineNum) Len() int           { return len(a) }
// func (a EncodedJobDataByLineNum) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
// func (a EncodedJobDataByLineNum) Less(i, j int) bool { return a[i].LineNum < a[j].LineNum }

func encoderHTTPServer(w http.ResponseWriter, r *http.Request) {
	// thanks gobwas for simple code like this <3
	conn, _, _, err := ws.UpgradeHTTP(r, w)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	go func() {
		defer conn.Close()

		for {
			// Read from websocket
			msg, op, err := wsutil.ReadClientData(conn)
			if err != nil {
				// recv error -> close connection
				return
			}

			// Decode json
			var strList []string
			err = json.Unmarshal(msg, &strList)
			if err != nil {
				// error parsing json -> close connection
				return
			}

			// Alloc job-wide queue for task completion sync & Enqueue stuff to worker pool
			resultsQueue := make(chan EncodeJobData, len(strList))
			for i := range strList {
				jobsQueue <- &EncodeJob{
					EncodeJobData: EncodeJobData{
						Payload: strList[i],
						LineNum: i,
					},
					ResultsQueue: resultsQueue,
				}
			}

			// Allocate result buffer & wait for appropriate number of results
			jobResults := make([]EncodeJobData, len(strList))
			for i := range strList {
				jobResults[i] = <-resultsQueue
			}

			// Avoid possible memory leak point
			close(resultsQueue)

			// Sort by line number (in case if reordering occured in worker pool) Best case complexity tends to be O(N*Log(N))
			//sort.Sort(EncodedJobDataByLineNum(jobResults))
			// Actually, I dont need Sort(), cuz I have a faster way to reorder in linear time O(N)
			hashList := make([]string, len(jobResults))
			for i := range jobResults {
				hashList[jobResults[i].LineNum] = jobResults[i].Payload
			}

			// Marshal json & send response
			jsonResponseBlob, _ := json.Marshal(hashList)
			err = wsutil.WriteServerMessage(conn, op, jsonResponseBlob)
			if err != nil {
				// send error -> close connection
				return
			}
		}
	}()
}

func main() {
	// getenv
	svcPort := os.Getenv("SERVICE_PORT")
	if svcPort == "" {
		svcPort = "5000"
	}

	// init & start pool
	numWorkers := 4
	jobsQueue = make(chan *EncodeJob)
	for workerID := 0; workerID < numWorkers; workerID++ {
		go encode_worker(workerID)
	}

	// launch http server (which is actually websocket server, for better, faster, stronger inter-ms exchange)
	fmt.Printf("StringEncryptor microservice is listening on port %s.\n", svcPort)
	http.HandleFunc("/", encoderHTTPServer)
	http.ListenAndServe(":"+svcPort, nil)
}
