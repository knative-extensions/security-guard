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

import "fmt"

type jOp struct {
	op   string
	path string
	val  interface{}
}

type jPatch struct {
	jOps  []jOp
	bytes []byte
	empty bool
}

// [{ "op": "add", "path": "/spec/template/metadata/annotations", "value": { "qpoption.knative.dev/guard-activate": "enable" }  }]
func (jp *jPatch) build() {
	jp.empty = true
	first := true
	jp.bytes = make([]byte, 0, 256)
	jp.bytes = append(jp.bytes, `[`...)
	for _, jop := range jp.jOps {
		jp.empty = false
		if !first {
			jp.bytes = append(jp.bytes, `,`...)
		}
		first = false
		jp.bytes = append(jp.bytes, `{"op":"`...)
		jp.bytes = append(jp.bytes, jop.op...)
		jp.bytes = append(jp.bytes, `","path":"`...)
		jp.bytes = append(jp.bytes, jop.path...)
		jp.bytes = append(jp.bytes, `","value":`...)
		jp.buildVal(jop.val)
		jp.bytes = append(jp.bytes, `}`...)
	}
	jp.bytes = append(jp.bytes, `]`...)
}

func (jp *jPatch) buildVal(val interface{}) {
	switch value := val.(type) {
	case string:
		jp.bytes = append(jp.bytes, `"`...)
		jp.bytes = append(jp.bytes, value...)
		jp.bytes = append(jp.bytes, `"`...)
	case map[string]interface{}:
		first := true
		for k, v := range value {
			if !first {
				jp.bytes = append(jp.bytes, `,`...)
			}
			first = false
			jp.bytes = append(jp.bytes, `{"`...)
			jp.bytes = append(jp.bytes, k...)
			jp.bytes = append(jp.bytes, `":`...)
			jp.buildVal(v)
			jp.bytes = append(jp.bytes, `}`...)
		}
	default:
		fmt.Printf(" Unknown type %T", value)
	}
}

//serviceAccountName = []byte(`[{ "op": "add", "path": "/spec/template/spec/serviceaccountname", "value": "guardian-reader"  }]`)
