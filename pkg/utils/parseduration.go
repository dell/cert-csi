/*
 *
 * Copyright © 2022-2023 Dell Inc. or its subsidiaries. All Rights Reserved.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *      http://www.apache.org/licenses/LICENSE-2.0
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 *
 */

package utils

import (
	"errors"
	"regexp"
	"strconv"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
)

const (
	hoursinweek = 168
	hoursinday  = 24
)

// ExtendedDuration in weeks, days, hours, minutes and seconds
type ExtendedDuration struct {
	Weeks   int
	Days    int
	Hours   int
	Minutes int
	Seconds int
}

// ParseDuration parse duration string and returns ExtendedDuration struct
func ParseDuration(str string) (*ExtendedDuration, error) {
	r := regexp.MustCompile(`(?P<weeks>\d+[wW])?(?P<days>\d+[dD])?(?P<hours>\d+[hH])?(?P<minutes>\d+[mM])?(?P<seconds>\d+[sS])?`)
	result := &ExtendedDuration{}
	var err error
	res := r.FindAllStringSubmatch(str, -1)
	for idx := range res {
		if result.Weeks == 0 && strings.ContainsRune(strings.ToLower(res[idx][1]), 'w') {
			result.Weeks, err = strconv.Atoi(res[idx][1][:len(res[idx][1])-1])
			if err != nil {
				log.Errorf("")
			}
		}
		if result.Days == 0 && strings.ContainsRune(strings.ToLower(res[idx][2]), 'd') {
			result.Days, err = strconv.Atoi(res[idx][2][:len(res[idx][2])-1])
			if err != nil {
				log.Errorf("")
			}
		}
		if result.Hours == 0 && strings.ContainsRune(strings.ToLower(res[idx][3]), 'h') {
			result.Hours, err = strconv.Atoi(res[idx][3][:len(res[idx][3])-1])
			if err != nil {
				log.Errorf("")
			}
		}
		if result.Minutes == 0 && strings.ContainsRune(strings.ToLower(res[idx][4]), 'm') {
			result.Minutes, err = strconv.Atoi(res[idx][4][:len(res[idx][4])-1])
			if err != nil {
				log.Errorf("")
			}
		}
		if result.Seconds == 0 && strings.ContainsRune(strings.ToLower(res[idx][5]), 's') {
			result.Seconds, err = strconv.Atoi(res[idx][5][:len(res[idx][5])-1])
			if err != nil {
				log.Errorf("")
			}
		}
	}
	if result.Duration() == 0 {
		return nil, errors.New("wrongly formatted")
	}
	return result, err
}

// Duration returns time duration
func (s ExtendedDuration) Duration() time.Duration {
	var dur time.Duration
	dur += time.Duration(s.Hours) * time.Hour
	dur += time.Duration(s.Days*hoursinday) * time.Hour
	dur += time.Duration(s.Weeks*hoursinweek) * time.Hour
	dur += time.Duration(s.Minutes) * time.Minute
	dur += time.Duration(s.Seconds) * time.Second
	return dur
}
