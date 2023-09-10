/*
Copyright 2023 The Knative Authors

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
	"bytes"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"os"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Printf("usage: %s <URL>\n", os.Args[0])
		os.Exit(1)
	}
	for i := 0; i < 100000; i++ {
		if i%100 == 0 {
			fmt.Printf("%d\n", i)
		}
		sendRandom(os.Args[1])
	}

}

func sendRandom(requestURL string) {
	jsonBody := randomKeyVal(rand.Intn(10) + 1)
	bodyReader := bytes.NewReader(jsonBody)

	req, err := http.NewRequest(http.MethodPost, requestURL, bodyReader)
	if err != nil {
		fmt.Printf("client: could not create request: %s\n", err)
		os.Exit(1)
	}
	req.Header.Set("X-Custom-Header", "myvalue")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Content-Type", "application/json; charset=UTF-8")

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		fmt.Printf("client: error making http request: %s\n", err)
		os.Exit(1)
	}

	_, err = io.ReadAll(res.Body)
	if err != nil {
		fmt.Printf("client: could not read response body: %s\n", err)
		os.Exit(1)
	}
}

func randomKeyVal(depth int) []byte {
	if depth <= 0 {
		parts := [][]byte{
			[]byte(`"`),
			randomVal(rand.Intn(100)),
			[]byte(`"`),
		}
		return bytes.Join(parts, []byte(``))
	}
	parts := [][]byte{
		[]byte(`{"`),
		randomVal(rand.Intn(100)),
		[]byte(`":`),
		randomKeyVal(depth - 1),
		[]byte(`}`),
	}
	return bytes.Join(parts, []byte(``))
}

func randomVal(length int) []byte {
	result := make([]byte, length)
	for i := 0; i < length; i++ {
		result[i] = byte(rand.Intn(26) + 65)
	}
	return result
}
