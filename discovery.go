package main

import (
	"encoding/json"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

const (
	DiscoveryPort = 12120
	BroadcastInterval = 3 * time.Second
	PeerTimeout      = 8 * time.Second
)

// DiscoveryMessage is the payload broadcasted over UDP
type DiscoveryMessage struct {
	PeerID    string `json:"peer_id"`
	Username  string `json:"username"`
	PublicKey []byte `json:"public_key"`
	TCPPort   int    `json:"tcp_port"`
	Color     string `json:"color"`  // Hex color
	Status    string `json:"status"` // Status message
}

// Peer represents an active discovered peer
type Peer struct {
	PeerID    string    `json:"peer_id"`
	Username  string    `json:"username"`
	IP        string    `json:"ip"`
	TCPPort   int       `json:"tcp_port"`
	PublicKey []byte    `json:"public_key"`
	LastSeen  time.Time `json:"last_seen"`
	Online    bool      `json:"online"`
	Color     string    `json:"color"`  // Discovered peer color
	Status    string    `json:"status"` // Discovered peer status
	IsWAN     bool      `json:"is_wan"`  // True if connected via Internet Matchmaking Relay
}

// DiscoveryService manages LAN peer discovery
type DiscoveryService struct {
	mu           sync.RWMutex
	peers        map[string]*Peer // key: IP
	app          *App
	tcpPort      int
	peerID       string
	username     string
	publicKey    []byte
	color        string
	status       string
	udpConn      *net.UDPConn
	stopChan     chan struct{}
	running      bool
}

// NewDiscoveryService creates a new discovery service instance
func NewDiscoveryService(app *App) *DiscoveryService {
	return &DiscoveryService{
		peers:    make(map[string]*Peer),
		app:      app,
		stopChan: make(chan struct{}),
	}
}

// Start initiates the UDP broadcast and receiver routines
func (d *DiscoveryService) Start(peerID string, username string, publicKey []byte, tcpPort int, color string, status string) error {
	d.mu.Lock()
	if d.running {
		d.mu.Unlock()
		return nil
	}
	d.peerID = peerID
	d.username = username
	d.publicKey = publicKey
	d.tcpPort = tcpPort
	d.color = color
	d.status = status
	d.running = true
	d.stopChan = make(chan struct{})
	d.peers = make(map[string]*Peer) // clear old peers
	d.mu.Unlock()

	// 1. Setup UDP listener
	addr, err := net.ResolveUDPAddr("udp", fmt.Sprintf(":%d", DiscoveryPort))
	if err != nil {
		d.Stop()
		return err
	}

	conn, err := net.ListenUDP("udp", addr)
	if err != nil {
		d.Stop()
		return err
	}
	d.udpConn = conn

	// 2. Start Receiver Goroutine
	go d.listenLoop()

	// 3. Start Broadcaster Goroutine
	go d.broadcastLoop()

	// 4. Start Sweeper Goroutine
	go d.sweeperLoop()

	return nil
}

// Stop halts all discovery activities
func (d *DiscoveryService) Stop() {
	d.mu.Lock()
	if !d.running {
		d.mu.Unlock()
		return
	}
	d.running = false
	close(d.stopChan)
	if d.udpConn != nil {
		d.udpConn.Close()
	}
	d.mu.Unlock()
}

// GetPeers returns the list of currently active peers
func (d *DiscoveryService) GetPeers() []*Peer {
	d.mu.RLock()
	defer d.mu.RUnlock()

	peerList := make([]*Peer, 0, len(d.peers))
	for _, p := range d.peers {
		if p.Online {
			peerList = append(peerList, p)
		}
	}
	return peerList
}

// broadcastLoop broadcasts our identity to the LAN periodically
func (d *DiscoveryService) broadcastLoop() {
	ticker := time.NewTicker(BroadcastInterval)
	defer ticker.Stop()

	// Setup broadcast address
	broadcastAddr, err := net.ResolveUDPAddr("udp", fmt.Sprintf("255.255.255.255:%d", DiscoveryPort))
	if err != nil {
		return
	}

	// Bind socket for sending
	sendConn, err := net.DialUDP("udp", nil, broadcastAddr)
	if err != nil {
		return
	}
	defer sendConn.Close()

	for {
		d.mu.RLock()
		msg := DiscoveryMessage{
			PeerID:    d.peerID,
			Username:  d.username,
			PublicKey: d.publicKey,
			TCPPort:   d.tcpPort,
			Color:     d.color,
			Status:    d.status,
		}
		d.mu.RUnlock()

		payload, err := json.Marshal(msg)
		if err == nil {
			_, _ = sendConn.Write(payload)
		}

		select {
		case <-d.stopChan:
			return
		case <-ticker.C:
			// Loop continues and broadcasts again
		}
	}
}

// broadcastOnce triggers an immediate single UDP broadcast (e.g. on profile change)
func (d *DiscoveryService) broadcastOnce() {
	broadcastAddr, err := net.ResolveUDPAddr("udp", fmt.Sprintf("255.255.255.255:%d", DiscoveryPort))
	if err != nil {
		return
	}

	sendConn, err := net.DialUDP("udp", nil, broadcastAddr)
	if err != nil {
		return
	}
	defer sendConn.Close()

	d.mu.RLock()
	msg := DiscoveryMessage{
		PeerID:    d.peerID,
		Username:  d.username,
		PublicKey: d.publicKey,
		TCPPort:   d.tcpPort,
		Color:     d.color,
		Status:    d.status,
	}
	d.mu.RUnlock()

	payload, err := json.Marshal(msg)
	if err == nil {
		_, _ = sendConn.Write(payload)
	}
}

// UpdateProfile updates our broadcast parameters and announces them immediately
func (d *DiscoveryService) UpdateProfile(username string, color string, status string) {
	d.mu.Lock()
	if username != "" {
		d.username = username
	}
	d.color = color
	d.status = status
	running := d.running
	d.mu.Unlock()

	if running {
		d.broadcastOnce()
	}
}

// listenLoop listens for broadcasts from other peers
func (d *DiscoveryService) listenLoop() {
	buf := make([]byte, 2048)

	for {
		d.mu.RLock()
		running := d.running
		d.mu.RUnlock()
		if !running {
			return
		}

		n, remoteAddr, err := d.udpConn.ReadFromUDP(buf)
		if err != nil {
			// If socket closed, exit loop
			return
		}

		var msg DiscoveryMessage
		err = json.Unmarshal(buf[:n], &msg)
		if err != nil {
			continue // invalid packet
		}

		// Don't register ourselves
		d.mu.RLock()
		isSelf := msg.PeerID == d.peerID
		d.mu.RUnlock()
		if isSelf {
			continue
		}

		peerIP := remoteAddr.IP.String()

		d.mu.Lock()
		p, exists := d.peers[peerIP]
		updated := false

		if !exists {
			p = &Peer{
				PeerID:    msg.PeerID,
				Username:  msg.Username,
				IP:        peerIP,
				TCPPort:   msg.TCPPort,
				PublicKey: msg.PublicKey,
				LastSeen:  time.Now(),
				Online:    true,
				Color:     msg.Color,
				Status:    msg.Status,
			}
			d.peers[peerIP] = p
			updated = true
		} else {
			// Update peer status
			p.LastSeen = time.Now()
			p.TCPPort = msg.TCPPort
			if !p.Online {
				p.Online = true
				updated = true
			}
			// Update PeerID, username or color/status if they changed
			if p.PeerID != msg.PeerID {
				p.PeerID = msg.PeerID
				updated = true
			}
			if p.Username != msg.Username {
				p.Username = msg.Username
				updated = true
			}
			// Update color or status if they changed
			if p.Color != msg.Color {
				p.Color = msg.Color
				updated = true
			}
			if p.Status != msg.Status {
				p.Status = msg.Status
				updated = true
			}
		}
		d.mu.Unlock()

		if updated {
			d.emitPeerList()
		}
	}
}

// sweeperLoop periodically marks inactive peers as offline
func (d *DiscoveryService) sweeperLoop() {
	ticker := time.NewTicker(3 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-d.stopChan:
			return
		case <-ticker.C:
			d.mu.Lock()
			updated := false
			now := time.Now()

			for ip, p := range d.peers {
				if p.Online && now.Sub(p.LastSeen) > PeerTimeout {
					p.Online = false
					updated = true
					// Optional: delete from map or keep in history as offline
					delete(d.peers, ip)
				}
			}
			d.mu.Unlock()

			if updated {
				d.emitPeerList()
			}
		}
	}
}

// emitPeerList sends the updated active peer list to the frontend
func (d *DiscoveryService) emitPeerList() {
	d.mu.RLock()
	peerList := make([]*Peer, 0, len(d.peers))
	for _, p := range d.peers {
		if p.Online {
			peerList = append(peerList, p)
		}
	}
	d.mu.RUnlock()

	// Emit Wails event
	if d.app != nil && d.app.ctx != nil {
		runtime.EventsEmit(d.app.ctx, "peers_updated", peerList)
	}
}
