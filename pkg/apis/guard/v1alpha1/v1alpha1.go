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

package v1alpha1

import (
	"fmt"
	"sort"
	"strings"
	"time"
)

// A Profile describing the Value
type ValueProfile interface {
	// profileI the data provided in args
	profileI(args ...interface{})
}

type Decision struct {
	Children map[string]*Decision `json:"children"`
	Reasons  []string             `json:"reasons"`
	Result   int                  `json:"result"`
}

// Level is "Session" or "Gate"
type Alert struct {
	str      string
	Decision *Decision `json:"decision"`
	Time     int64     `json:"time"`
	Level    string    `json:"level"`
	Count    uint      `json:"count"`
}

type SyncMessageReq struct {
	Pile           *SessionDataPile `json:"pile"`
	Alerts         []Alert          `json:"alerts"`
	IamCompromised bool             `json:"compromised"`
}

type SyncMessageResp struct {
	Guardian *GuardianSpec `json:"guardian"`
}

func AddAlert(alerts []Alert, decision *Decision, level string) []Alert {
	str := decision.SortedString(level)
	for i, alert := range alerts {
		if alert.str == str {
			alerts[i].Count++
			return alerts
		}
	}

	alert := Alert{
		str:      str,
		Decision: decision,
		Time:     time.Now().Unix(),
		Level:    level,
		Count:    1,
	}
	return append(alerts, alert)
}

func DecideInner(current **Decision, result int, format string, a ...any) {
	// found a problem
	d := *current
	if d == nil {
		d = new(Decision)
		d.Children = make(map[string]*Decision, 8)
		*current = d
	}

	reason := fmt.Sprintf(format, a...)
	d.Reasons = append((*current).Reasons, reason)
	d.Result += result
}

func DecideChild(current **Decision, childDecision *Decision, format string, a ...any) {
	if childDecision == nil {
		return
	}

	// child found a problem
	d := *current
	if d == nil {
		d = new(Decision)
		d.Children = make(map[string]*Decision, 8)
		*current = d
	}

	tag := fmt.Sprintf(format, a...)
	d.Children[tag] = childDecision
	d.Result += childDecision.Result
}

func (parent *Decision) Summary() string {
	if parent.Result > 0 {
		return fmt.Sprintf("Fail (%d)", parent.Result)
	}
	return ""
}

func (parent *Decision) SpillOut(sb *strings.Builder) {
	// sprintf("[ %s: %s, %s: %s, ... %s, %s, ... ], ", tag1 , child1.SpillOut(), tag1 , child1.SpillOut(), ..., reason1, reason2, ...  )
	sb.WriteByte('[')
	for tag, child := range parent.Children {
		sb.WriteString(tag)
		sb.WriteByte(':')
		child.SpillOut(sb)
		sb.WriteByte(',')
	}
	for _, reason := range parent.Reasons {
		sb.WriteString(reason)
		sb.WriteByte(',')
	}
	sb.WriteByte(']')
}

func (parent *Decision) String(tag string) string {
	if parent.Result > 0 {
		var sb strings.Builder
		sb.WriteString(tag)
		parent.SpillOut(&sb)
		return sb.String()
	}
	return ""
}

func (parent *Decision) SortedSpillOut(sb *strings.Builder) {
	// sprintf("[ %s: %s, %s: %s, ... %s, %s, ... ], ", tag1 , child1.SpillOut(), tag1 , child1.SpillOut(), ..., reason1, reason2, ...  )

	var tags []string

	sb.WriteByte('[')
	for tag := range parent.Children {
		tags = append(tags, tag)
	}
	sort.Strings(tags)
	sort.Strings(parent.Reasons)

	for _, tag := range tags {
		child := parent.Children[tag]
		sb.WriteString(tag)
		sb.WriteByte(':')
		child.SortedSpillOut(sb)
		sb.WriteByte(',')
	}
	for _, reason := range parent.Reasons {
		sb.WriteString(reason)
		sb.WriteByte(',')
	}
	sb.WriteByte(']')
}

func (parent *Decision) SortedString(tag string) string {
	if parent.Result > 0 {
		var sb strings.Builder

		sb.WriteString(tag)
		parent.SortedSpillOut(&sb)
		return sb.String()
	}
	return ""
}

// A Pile accumulating information from zero or more Values
type ValuePile interface {
	// addI one more profile to pile
	// Profile should not be used after it is added to a pile
	// Pile may absorb some or the profile internal structures
	addI(profile ValueProfile)

	// mergeI otherPile to this pile
	// otherPile should not be used after it is merged to a pile
	// Pile may absorb some or the otherPile internal structures
	mergeI(otherPile ValuePile)

	// Clear the pile from all profiles and free any memory held by pile
	Clear()
}

// A Config defining what Value should adhere to
type ValueConfig interface {
	// learnI from a pile to this config
	// pile should not be used after it is Learned by config
	// Config may absorb some or the pile internal structures
	learnI(pile ValuePile)

	// decideI if profile meets config
	// returns nil if profile is approved by config
	// otherwise, returns *Decision with details about all failures
	// All issues will be reported
	// Profile is unchanged and unaffected by decideI and can be used again
	decideI(profile ValueProfile) *Decision

	// Prepare the config during loading of a new config
	Prepare()
}
