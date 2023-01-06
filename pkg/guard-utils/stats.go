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

package guardutils

import (
	"fmt"
	"sync"
)

type Stat struct {
	statistics map[string]uint32
	mutex      *sync.Mutex
}

func (s *Stat) Init() {
	s.mutex = new(sync.Mutex)
	s.statistics = make(map[string]uint32, 8)
}

func (s *Stat) Add(key string) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.statistics[key]++
}

func (s *Stat) Log() string {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	str := fmt.Sprintf("%v", s.statistics)
	return str
}
