package loadbalancer

import (
	"log"
	"net/http"
	"patrickbahr/robin/pkg/backend"
)

const (
	//Attempts represents all the attempts of a request
	Attempts int = iota

	//Retry represents all retries of a request
	Retry
)

//Balance will try to balance the load between all backends in a round robin fashion
func Balance(w http.ResponseWriter, r *http.Request) {

	attempts := GetRetryFromContext(r)
	if attempts > 3 {
		log.Printf("%s(%s) Max attempts reached, terminating\n", r.RemoteAddr, r.URL.Path)
		http.Error(w, "Service not available", http.StatusServiceUnavailable)
		return
	}

	sp := backend.ServerPool{}
	peer := sp.GetNextPeer()
	if peer == nil {
		http.Error(w, "Service not available", http.StatusServiceUnavailable)
	}

	peer.ReverseProxy.ServeHTTP(w, r)

}

//GetAttemptsFromContext will use the context of a request to get how many attempts have been made
func GetAttemptsFromContext(r *http.Request) int {
	if attempts, ok := r.Context().Value(Attempts).(int); ok {
		return attempts
	}
	return 1
}

//GetRetryFromContext will use the context of a request to get how many retries have been made
func GetRetryFromContext(r *http.Request) int {
	if retry, ok := r.Context().Value(Retry).(int); ok {
		return retry
	}
	return 0
}
