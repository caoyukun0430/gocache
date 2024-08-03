package gocache

import (
	"context"
	"fmt"
	"gocache/consistenthash"
	pb "gocache/gocachepb"
	"log"
	"net"
	"sync"
	"strings"

	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
	"time"
)

const (
	defaultPrefix   = "/_gocache/"
	defaultReplicas = 3
)

// HTTPPool works as 1. client implements PeerPicker for a pool of HTTP peers.
// 2. server implements ServeHTTP
type GrpcPool struct {
	pb.UnimplementedGroupCacheServer
	// this peer's base URL, e.g. "https://example.net:8000"
	base string
	// prefix for peer communication
	prefix      string
	mu          sync.Mutex               // guards peers and httpGetters
	peerRing    *consistenthash.HashRing // inside the ring is peer names
	grpcClients map[string]*grpcClient   // each remote node is a httpClient with addr baseURL
	
}

var _ PeerPicker = (*GrpcPool)(nil)

type grpcClient struct {
	// baseURL is the addr of the remote server
	baseURL string
}

// Interface Compliance Check, Go compiler checks at compile time that grpcClient implements all the methods required by the PeerClient interface.
var _ PeerClient = (*grpcClient)(nil)

// NewGrpcPool initializes an GRPC pool of peers.
func NewGrpcPool(base string) *GrpcPool {
	return &GrpcPool{
		base:   base,
		prefix: defaultPrefix,
	}
}

// Add peer names into the consistenthash ring, wrapped consisenthash Add
func (p *GrpcPool) Add(peers ...string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.peerRing = consistenthash.New(defaultReplicas, nil)
	p.peerRing.Add(peers...)
	p.grpcClients = make(map[string]*grpcClient, len(peers))
	// each peer act as a client ready to send requests
	for _, peer := range peers {
		p.grpcClients[peer] = &grpcClient{baseURL: peer + p.prefix}
	}
}

// implements the peerPicker interface methods
// all peers are stored in ring p.httpClients
func (p *GrpcPool) PickPeer(key string) (PeerClient, bool) {
	p.mu.Lock()
	defer p.mu.Unlock()
	// based on the key, we select the node/peer on the ring, consistent hash makes sure certains key belongs to one node
	if vnode := p.peerRing.Get(key); vnode != "" && vnode != p.base {
		// vnode is not myself
		p.Log("Pick peer %s", vnode)
		return p.grpcClients[vnode], true
	}
	// no peer picked, get locally myself
	return nil, false
}

// Log info with server name
func (p *GrpcPool) Log(format string, v ...interface{}) {
	log.Printf("[Server %s] %s", p.base, fmt.Sprintf(format, v...))
}

// func name matches .proto service, similarly to ServeHTTP
func (p *GrpcPool) Get(ctx context.Context, in *pb.Request) (*pb.Response, error) {
	p.Log("Get %s %s", in.Group, in.Key)
	response := &pb.Response{}

	group := GetGroup(in.Group)
	if group == nil {
		p.Log("no such group %v", in.Group)
		return response, fmt.Errorf("no such group %v", in.Group)
	}
	value, err := group.Get(in.Key)
	if err != nil {
		p.Log("get key %v error %v", in.Key, err)
		return response, err
	}

	response.Value = value.ByteSlice()
	return response, nil
}

func (p *GrpcPool) Run() {
	listen, err := net.Listen("tcp", p.base)
	if err != nil {
		panic(err)
	}

	server := grpc.NewServer()
	pb.RegisterGroupCacheServer(server, p)
	// p.Log("Run listen %+v p.base %s server %+v", listen, p.base, server)

	reflection.Register(server)
	err = server.Serve(listen)
	if err != nil {
		panic(err)
	}
}

// GRPC CLIENT
// func name matches .proto service also for CLIENT!
func (g *grpcClient) Get(in *pb.Request, out *pb.Response) error {
	// Dial addr should not contain BaseURL /_gocache/, only ip:port
	portIndex := strings.Index(g.baseURL, "/")
	c, err := grpc.Dial(g.baseURL[:portIndex], grpc.WithInsecure())
	if err != nil {
		return err
	}
	client := pb.NewGroupCacheClient(c)
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()
	response, err := client.Get(ctx, in)
	out.Value = response.Value
	return err
}
