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

package k8sclient

import (
	"crypto/rand"
	"fmt"

	log "github.com/sirupsen/logrus"
)

// RandomSuffix returns a random suffix to use when naming resources
func RandomSuffix() string {
	b := make([]byte, 4)
	_, err := rand.Read(b)
	if err != nil {
		log.Errorf("Can't generate UID; error = %v", err)
	}
	suff := fmt.Sprintf("%x", b[0:])
	return suff
}
