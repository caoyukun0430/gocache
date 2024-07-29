package gocache

import (
	"fmt"
	"gocache/consistenthash"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"strings"
	"sync"
)

const (
	defaultPrefix   = "/_gocache/"
	defaultReplicas = 3
)

// HTTPPool works as 1. client implements PeerPicker for a pool of HTTP peers.
// 2. server implements ServeHTTP
type HTTPPool struct {
	// this peer's base URL, e.g. "https://example.net:8000"
	base string
	// prefix for peer communication
	prefix      string
	mu          sync.Mutex               // guards peers and httpGetters
	peerRing    *consistenthash.HashRing // inside the ring is peer names
	httpClients map[string]*httpClient   // each remote node is a httpClient with addr baseURL
}

// NewHTTPPool initializes an HTTP pool of peers.
func NewHTTPPool(base string) *HTTPPool {
	return &HTTPPool{
		base:   base,
		prefix: defaultPrefix,
	}
}

// Add peer names into the consistenthash ring, wrapped consisenthash Add
func (p *HTTPPool) Add(peers ...string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.peerRing = consistenthash.New(defaultReplicas, nil)
	p.peerRing.Add(peers...)
	p.httpClients = make(map[string]*httpClient, len(peers))
	// each peer act as a client ready to send requests
	for _, peer := range peers {
		p.httpClients[peer] = &httpClient{baseURL: peer + p.prefix}
	}
}

// implements the peerPicker interface methods
// all peers are stored in ring p.httpClients
func (p *HTTPPool) PickPeer(key string) (PeerClient, bool) {
	p.mu.Lock()
	defer p.mu.Unlock()
	// based on the key, we select the node/peer on the ring, consistent hash makes sure certains key belongs to one node
	if vnode := p.peerRing.Get(key); vnode != "" && vnode != p.base {
		// vnode is not myself
		p.Log("Pick peer %s", vnode)
		return p.httpClients[vnode], true
	}
	// no peer picked, get locally myself
	return nil, false
}

var _ PeerPicker = (*HTTPPool)(nil)

// Log info with server name
func (p *HTTPPool) Log(format string, v ...interface{}) {
	log.Printf("[Server %s] %s", p.base, fmt.Sprintf(format, v...))
}

// ServeHTTP handle all http requests
func (p *HTTPPool) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if !strings.HasPrefix(r.URL.Path, p.prefix) {
		panic("HTTPPool serving unexpected path: " + r.URL.Path)
	}
	p.Log("%s %s", r.Method, r.URL.Path)
	// /<basepath>/<groupname>/<key> required
	parts := strings.SplitN(r.URL.Path[len(p.prefix):], "/", 2)
	if len(parts) != 2 {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	groupName := parts[0]
	key := parts[1]

	group := GetGroup(groupName)
	if group == nil {
		http.Error(w, "no such group: "+groupName, http.StatusNotFound)
		return
	}

	view, err := group.Get(key)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/octet-stream")
	w.Write(view.ByteSlice())
}

// httpClient implements the peerClient interface, it's peer as a client role
type httpClient struct {
	// baseURL is the addr of the remote server
	baseURL string
}

// the httpClient peer send GET request to remote with addr link
func (h *httpClient) Request(group string, key string) ([]byte, error) {
	link := fmt.Sprintf(
		"%v%v/%v",
		h.baseURL,
		url.QueryEscape(group),
		url.QueryEscape(key),
	)
	// send http GET
	res, err := http.Get(link)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("server returned: %v", res.Status)
	}

	bytes, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response body: %v", err)
	}

	return bytes, nil
}

// Interface Compliance Check, Go compiler checks at compile time that httpClient implements all the methods required by the PeerClient interface.
var _ PeerClient = (*httpClient)(nil)
