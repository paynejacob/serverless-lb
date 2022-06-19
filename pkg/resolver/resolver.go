package resolver

import (
	"strings"
	"sync"

	"github.com/miekg/dns"
)

type Resolver struct {
	poolNameIndex map[string]*AddressPool
	hosts         map[string]*AddressPool
	wildcards     map[string]*AddressPool

	mu sync.RWMutex
}

func NewResolver() *Resolver {
	return &Resolver{
		poolNameIndex: map[string]*AddressPool{},
		hosts:         map[string]*AddressPool{},
		wildcards:     map[string]*AddressPool{},
		mu:            sync.RWMutex{},
	}
}

func (r *Resolver) Answer(questions ...dns.Question) (answer []dns.RR) {
	r.mu.RLock()
	for _, q := range questions {
		switch q.Qtype {
		case dns.TypeA:
			ip := r.resolve(q.Name)
			if ip != "" {
				rr, err := dns.NewRR(q.Name + " A " + ip)
				if err == nil {
					answer = append(answer, rr)
				}
			}
		}
	}
	r.mu.RUnlock()

	return answer
}

func (r *Resolver) AddAddress(pool, addr string) {
	r.mu.Lock()
	if _, ok := r.poolNameIndex[pool]; !ok {
		r.poolNameIndex[pool] = &AddressPool{}
	}

	r.poolNameIndex[pool].AddAddress(addr)
	r.mu.Unlock()
}

func (r *Resolver) RemoveAddress(pool, addr string) {
	r.mu.Lock()
	if _, ok := r.poolNameIndex[pool]; ok {
		r.poolNameIndex[pool].RemoveAddress(addr)
	}
	r.mu.Unlock()
}

func (r *Resolver) AddHost(rec, pool string) {
	if strings.HasPrefix(rec, "*") {
		r.addWildCard(rec, pool)
		return
	}

	r.mu.Lock()
	if _, ok := r.poolNameIndex[pool]; !ok {
		r.poolNameIndex[pool] = &AddressPool{}
	}

	r.hosts[rec] = r.poolNameIndex[pool]
	r.mu.Unlock()
}

func (r *Resolver) RemoveHost(rec string) {
	if strings.HasPrefix(rec, "*") {
		r.removeWildCard(rec)
		return
	}

	r.mu.Lock()
	delete(r.hosts, rec)
	r.mu.Unlock()
}

func (r *Resolver) addWildCard(rec, pool string) {
	rec = normalizeWildcard(rec)

	r.mu.Lock()
	if _, ok := r.poolNameIndex[pool]; !ok {
		r.poolNameIndex[pool] = &AddressPool{}
	}

	r.wildcards[rec] = r.poolNameIndex[pool]
	r.mu.Unlock()
}

func (r *Resolver) removeWildCard(rec string) {
	rec = normalizeWildcard(rec)

	r.mu.Lock()
	delete(r.wildcards, rec)
	r.mu.Unlock()
}

func (r *Resolver) resolve(host string) string {
	pool := r.hosts[normalizeHost(host)]
	if pool != nil {
		return pool.Next()
	}

	parts := strings.Split(host, ".")
	for i, j := 0, len(parts)-1; i < j; i, j = i+1, j-1 {
		parts[i], parts[j] = parts[j], parts[i]
	}

	for {
		if len(parts) == 0 {
			return ""
		}

		if answer, ok := r.wildcards[strings.Join(parts, ".")]; ok {
			return answer.Next()
		}

		parts = parts[:len(parts)-1]
	}

}

func normalizeWildcard(pattern string) string {
	pattern = pattern[1:]

	parts := strings.Split(pattern, ".")
	for i, j := 0, len(parts)-1; i < j; i, j = i+1, j-1 {
		parts[i], parts[j] = parts[j], parts[i]
	}

	return strings.Join(parts, ".")
}

func normalizeHost(pattern string) string {
	return pattern[:len(pattern)-1]
}
