package cookies

import (
	"sort"
	"sync"
	"time"
)

// NodeKind classifies a node in the cross-site request graph.
type NodeKind uint8

const (
	NodeFirstParty NodeKind = iota
	NodeThirdParty
	NodeCloakedThirdParty
	NodeTrackerGroup
)

// Node is an origin in the request graph.
type Node struct {
	Host        string
	Canonical   string
	Registrable string
	Kind        NodeKind
}

// Edge records that a first-party site requested a host.
type Edge struct {
	TopLevel string
	From     string
	To       string
}

// DependencyGraph records cross-site requests and detects CNAME-cloaked
// third-party endpoints.
type DependencyGraph struct {
	mutex  sync.Mutex
	nodes  map[string]*Node
	edges  []Edge
	groups map[string]map[string]struct{}
}

// NewDependencyGraph returns an empty graph.
func NewDependencyGraph() *DependencyGraph {
	return &DependencyGraph{
		nodes:  make(map[string]*Node),
		groups: make(map[string]map[string]struct{}),
	}
}

// Record notes that topLevel requested requestedHost, which resolved via DNS
// to canonicalHost (empty when there was no CNAME).
func (graph *DependencyGraph) Record(topLevel, requestedHost, canonicalHost string) {
	if requestedHost == "" {
		return
	}
	graph.mutex.Lock()
	defer graph.mutex.Unlock()

	requestedRegistrable := registrableDomain(requestedHost)
	node := &Node{Host: requestedHost, Registrable: requestedRegistrable, Kind: NodeFirstParty}
	if !equalFold(requestedRegistrable, registrableDomain(topLevel)) {
		node.Kind = NodeThirdParty
	}
	if canonicalHost != "" && !equalFold(canonicalHost, requestedHost) {
		canonicalRegistrable := registrableDomain(canonicalHost)
		if !equalFold(canonicalRegistrable, requestedRegistrable) {
			node.Canonical = canonicalHost
			node.Kind = NodeCloakedThirdParty
			if graph.groups[canonicalRegistrable] == nil {
				graph.groups[canonicalRegistrable] = make(map[string]struct{})
			}
			graph.groups[canonicalRegistrable][requestedHost] = struct{}{}
		}
	}
	graph.nodes[requestedHost] = node
	graph.edges = append(graph.edges, Edge{TopLevel: topLevel, From: topLevel, To: requestedHost})
}

// ClearableHosts returns all third-party and cloaked hosts contacted by a
// top-level site, including other hosts that share a tracker group.
func (graph *DependencyGraph) ClearableHosts(topLevel string) []string {
	graph.mutex.Lock()
	defer graph.mutex.Unlock()
	clearable := make(map[string]struct{})
	trackerGroups := make(map[string]struct{})
	for _, edge := range graph.edges {
		if !equalFold(edge.TopLevel, topLevel) {
			continue
		}
		node := graph.nodes[edge.To]
		if node == nil {
			continue
		}
		switch node.Kind {
		case NodeThirdParty:
			clearable[node.Host] = struct{}{}
		case NodeCloakedThirdParty:
			clearable[node.Host] = struct{}{}
			if node.Canonical != "" {
				trackerGroups[registrableDomain(node.Canonical)] = struct{}{}
			}
		}
	}
	for canonical := range trackerGroups {
		for host := range graph.groups[canonical] {
			clearable[host] = struct{}{}
		}
	}
	hosts := make([]string, 0, len(clearable))
	for host := range clearable {
		hosts = append(hosts, host)
	}
	sort.Strings(hosts)
	return hosts
}

// SessionManager ties the cookie jar to the dependency graph and performs
// tracking-session resets.
type SessionManager struct {
	jar   *Jar
	graph *DependencyGraph
}

// NewSessionManager returns a session manager for a jar.
func NewSessionManager(jar *Jar) *SessionManager {
	return &SessionManager{jar: jar, graph: NewDependencyGraph()}
}

// RecordRequest records a request in the graph.
func (manager *SessionManager) RecordRequest(topLevel, requestedHost, canonicalHost string) {
	manager.graph.Record(topLevel, requestedHost, canonicalHost)
}

// ResetTrackingSession clears third-party and cloaked-third-party cookies for a
// top-level site, and first-party session cookies, so the next requests look
// like a new user. It returns the number of cookies removed.
func (manager *SessionManager) ResetTrackingSession(partition, topLevel string, now time.Time) int {
	deleted := 0
	for _, host := range manager.graph.ClearableHosts(topLevel) {
		deleted += manager.jar.DeleteDomain(partition, host)
	}
	for _, cookie := range manager.jar.All(partition) {
		isSession := cookie.Expires.IsZero() && cookie.MaxAge == 0
		if !isSession {
			continue
		}
		if equalFold(registrableDomain(cookie.Domain), registrableDomain(topLevel)) {
			manager.jar.Delete(partition, cookie.Key())
			deleted++
		}
	}
	return deleted
}

// Graph exposes the dependency graph for the privacy dashboard.
func (manager *SessionManager) Graph() *DependencyGraph {
	return manager.graph
}
