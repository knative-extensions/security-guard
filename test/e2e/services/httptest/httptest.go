/*
Copyright 2022 The Knative Authors

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package main

import (
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"
)

func ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var sleep time.Duration
	var sleepStep time.Duration
	var sleepNumSteps int

	sleep, _ = time.ParseDuration("100ms")
	sleepStep = 0
	sleepNumSteps = 100

	sleepStr := r.Header.Get("X-Sleep")
	sleepStepStr := r.Header.Get("X-Sleep-Step")
	sleepNumStepsStr := r.Header.Get("X-Sleep-Num-Steps")

	if sleepStr != "" {
		if t, err := time.ParseDuration(sleepStr); err == nil {
			fmt.Println("SampleServer received request with X-Sleep header", sleepStr)
			sleep = t
		} else {
			fmt.Println("SampleServer received request with errored X-Sleep header", err)
		}
	}

	if sleepStepStr != "" {
		if t, err := time.ParseDuration(sleepStepStr); err == nil {
			fmt.Println("SampleServer received request with X-Sleep-Step header", sleepStepStr)
			sleepStep = t
		} else {
			fmt.Println("SampleServer received request with errored X-Sleep-Step header", err)
		}
	}

	if sleepNumStepsStr != "" {
		if n, err := strconv.Atoi(sleepNumStepsStr); err == nil {
			fmt.Println("SampleServer received request with X-Num-Steps header", sleepNumStepsStr)
			sleepNumSteps = n
		} else {
			fmt.Println("SampleServer received request with errored X-Num-Steps header", err)
		}
	}
	fmt.Printf("SampleServer was asked to sleep %0.2f secs and do %d steps of %0.2f secs\n", float32(sleep)/1e9, sleepNumSteps, float32(sleepStep)/1e9)

	time.Sleep(sleep)
	fmt.Fprintf(w, "<H1>SampleServer</H1>\n")
	fmt.Fprintf(w, "<p>Shalom, this request will take %0.2f seconds to respond</p>\n", float32(int(sleep)+int(sleepStep)*sleepNumSteps)/1e9)

	ctx := r.Context()

	if int(sleepStep)*sleepNumSteps > 0 {
	Exit:
		for i := 0; i < sleepNumSteps; i++ {
			select {
			case <-ctx.Done():
				fmt.Printf("Request Conext is Done %s\n", ctx.Err())
				break Exit
			case <-time.After(sleepStep):
				fmt.Fprintf(w, "<p>Elapsed time: %0.2f secs</p>\n", float32(int(sleep)+int(sleepStep)*i)/1e9)
			}
		}
	}
	fmt.Fprintf(w, "<p>Elapsed time: %0.2f secs</p>\n", float32(int(sleep)+int(sleepStep)*sleepNumSteps)/1e9)
	fmt.Fprintf(w, "<p>SampleServer is now done sending!</p>\n")
	fmt.Fprintf(w, "<p>See ya!</p>\n")
	fmt.Printf("\nSampleServer finished processing the received request\n\n")

}

func main() {
	http.HandleFunc("/", ServeHTTP)
	fmt.Printf("Starting SampleServer at  127.0.0.1:8080\n")
	if err := http.ListenAndServe("127.0.0.1:8080", nil); err != nil {
		log.Fatal(err)
	}
}
