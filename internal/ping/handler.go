// Package ping implements a simple ping handler to confirm the service is running.
package ping

import (
	"fmt"
	"net/http"

	"github.com/carlmjohnson/versioninfo"
)

var Version = versioninfo.Revision

func Handler(w http.ResponseWriter, r *http.Request) {
	// Set version in response header
	w.Header().Set("Starling-Sweeper-Version", Version)
	fmt.Fprint(w, "PONG\n")
}
