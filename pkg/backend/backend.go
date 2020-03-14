package backend

import (
	"log"
	"net"
	"net/http/httputil"
	"net/url"
	"sync"
	"sync/atomic"
	"time"
)

//Backend represents some backend that the load will be balanced to
type Backend struct {
	URL          *url.URL
	Alive        bool
	mux          sync.RWMutex
	ReverseProxy *httputil.ReverseProxy
}

//SetAlive will set the state of a backend with a mutex to avoid race condition
func (b *Backend) SetAlive(alive bool) {
	b.mux.Lock()
	defer b.mux.Unlock()
	b.Alive = alive
}

//IsAlive will set the state of a backend with a mutex to avoid race condition
func (b *Backend) IsAlive() bool {
	b.mux.RLock()
	defer b.mux.RUnlock()
	return b.Alive
}

//ServerPool represents a cluster os backends
type ServerPool struct {
	backends []*Backend
	current  uint64
}

//NextIndex will return the next index for a backend
func (s *ServerPool) NextIndex() int {
	return int(atomic.AddUint64(&s.current, uint64(1)) % uint64(len(s.backends)))
}

//GetNextPeer will return the next available backend from the pool
func (s *ServerPool) GetNextPeer() *Backend {
	next := s.NextIndex()
	l := len(s.backends) + next
	for i := next; i < l; i++ {
		idx := i % len(s.backends)
		if s.backends[idx].IsAlive() {
			if i != next {
				atomic.StoreUint64(&s.current, uint64(idx))
			}

			return s.backends[idx]
		}
	}
	return nil
}

//AddBackend will add a backend to the pool
func (s *ServerPool) AddBackend(backend *Backend) {
	s.backends = append(s.backends, backend)
}

// MarkBackendStatus changes a status of a backend
func (s *ServerPool) MarkBackendStatus(backendURL *url.URL, alive bool) {
	for _, b := range s.backends {
		if b.URL.String() == backendURL.String() {
			b.SetAlive(alive)
			break
		}
	}
}

//HealthCheck will trigger every 2 minutes to see if the backends are ok
func HealthCheck(serverPool ServerPool) {
	t := time.NewTicker(time.Minute * 2)
	for {
		select {
		case <-t.C:
			log.Println("Starting health check...")
			serverPool.healthCheck()
			log.Println("Health check completed")
		}
	}
}

func (s *ServerPool) healthCheck() {
	for _, b := range s.backends {
		status := "up"
		alive := isBackendAlive(b.URL)
		b.SetAlive(alive)
		if !alive {
			status = "down"
		}
		log.Printf("%s [%s]\n", b.URL, status)
	}
}

func isBackendAlive(u *url.URL) bool {
	timeout := 2 * time.Second
	conn, err := net.DialTimeout("tcp", u.Host, timeout)
	defer conn.Close()

	if err != nil {
		log.Println("Site unreachable, error: ", err)
		return false
	}
	return true
}
