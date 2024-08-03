// to solve cache Penetration, e.g. when sending huge concurrent requests ?key=Tom
// we will send them all to local cache/source, or to remote node. It will pressure
// the server in short time, leading to penetration.

// Therefore, we use singleflight to make sure concurrent requests to the same key
// waits for the 1st to finish with c.wg.Wait(), and then reuse the response from 1st request.

package singleflight

import (
	"sync"
)

type Call struct {
	wg  sync.WaitGroup
	val interface{}
	err error
}

type Group struct {
	mu   sync.Mutex // protects call from concurrent r/w
	call map[string]*Call
}

//	ensure that only one execution of a function with the same key is in progress at any given time.
//
// If a request with the same key is already in progress, other requests will wait for the result of the ongoing request instead of starting a new one.
func (g *Group) Do(key string, fn func() (interface{}, error)) (interface{}, error) {

	// delayed init
	if g == nil || g.call == nil {
		g = &Group{
			call: make(map[string]*Call),
		}
	}
	g.mu.Lock()
	// wait existing request finish and reuses
	if c, ok := g.call[key]; ok {
		// read call complete, unlock
		g.mu.Unlock()
		c.wg.Wait()
		return c.val, c.err
	}
	// Initiate a new request
	c := new(Call)
	c.wg.Add(1)
	g.call[key] = c
	g.mu.Unlock() // write call complete, unlock

	// send the actual request
	c.val, c.err = fn()
	c.wg.Done() // request complete

	// lock and unlock to delete call with key
	g.mu.Lock()
	delete(g.call, key)
	g.mu.Unlock()

	return c.val, c.err
}
