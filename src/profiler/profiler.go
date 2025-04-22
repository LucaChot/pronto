package profiler

import (
    "log"
    "net/http"
    _ "net/http/pprof"
)

func StartProfilerServer(addr string) {
	go func() {
		log.Printf("Profiling server running on %s", addr)
		log.Println(http.ListenAndServe(addr, nil))
	}()
}

