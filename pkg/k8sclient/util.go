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
