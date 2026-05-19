package main

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"sync"
)

// ServerPort is the default port the matchmaking and relay server listens on
const ServerPort = 12130

// ClientMsg represents a message sent from client to server
type ClientMsg struct {
	Type      string          `json:"type"`       // "register", "list", "signal"
	TargetID  string          `json:"target_id"`  // Recipient Peer ID (for signal)
	Payload   json.RawMessage `json:"payload"`    // Dynamic payload depending on Type
}

// PeerInfo represents the active state of an internet-connected peer
type PeerInfo struct {
	PeerID    string `json:"peer_id"`
	Username  string `json:"username"`
	PublicKey []byte `json:"public_key"`
	Color     string `json:"color"`
	Status    string `json:"status"`
	IP        string `json:"ip"`
}

// ServerMsg represents a message sent from server to client
type ServerMsg struct {
	Type     string          `json:"type"` // "peer_list", "signal", "error"
	SenderID string          `json:"sender_id"`
	Payload  json.RawMessage `json:"payload"`
}

type ClientConnection struct {
	PeerID string
	Conn   net.Conn
	Mu     sync.Mutex
}

var (
	clients   = make(map[string]*ClientConnection) // key: PeerID
	peersInfo = make(map[string]*PeerInfo)         // key: PeerID
	mu        sync.RWMutex
)

func main() {
	addr := fmt.Sprintf("0.0.0.0:%d", ServerPort)
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		log.Fatalf("Error starting matchmaking server: %v", err)
	}
	defer listener.Close()

	log.Printf("QBFile Matchmaking & Secure Relay Server running on %s", addr)

	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Printf("Connection accept error: %v", err)
			continue
		}

		go handleClient(conn)
	}
}

func handleClient(conn net.Conn) {
	defer conn.Close()
	var clientID string
	var registered bool

	defer func() {
		if registered {
			mu.Lock()
			delete(clients, clientID)
			delete(peersInfo, clientID)
			mu.Unlock()
			log.Printf("Peer disconnected and unregistered: %s", clientID)
			broadcastPeerList()
		}
	}()

	log.Printf("New connection from %s", conn.RemoteAddr().String())

	for {
		// Read 4-byte length prefix
		var length uint32
		err := binary.Read(conn, binary.BigEndian, &length)
		if err != nil {
			if err != io.EOF {
				log.Printf("Read error length prefix: %v", err)
			}
			return
		}

		if length > 50*1024*1024 { // 50 MB safety limit for single frame
			log.Printf("Frame size too large: %d bytes", length)
			return
		}

		// Read packet body
		buf := make([]byte, length)
		_, err = io.ReadFull(conn, buf)
		if err != nil {
			log.Printf("Read error body: %v", err)
			return
		}

		var msg ClientMsg
		if err := json.Unmarshal(buf, &msg); err != nil {
			log.Printf("JSON unmarshal error: %v", err)
			continue
		}

		switch msg.Type {
		case "register":
			var info PeerInfo
			if err := json.Unmarshal(msg.Payload, &info); err != nil {
				log.Printf("Register payload error: %v", err)
				continue
			}

			info.IP = conn.RemoteAddr().(*net.TCPAddr).IP.String()
			clientID = info.PeerID

			mu.Lock()
			// Evict old session if it existed
			if old, exists := clients[clientID]; exists {
				old.Conn.Close()
			}
			clients[clientID] = &ClientConnection{PeerID: clientID, Conn: conn}
			peersInfo[clientID] = &info
			mu.Unlock()

			registered = true
			log.Printf("Peer registered successfully: %s (%s) from %s", info.Username, clientID, info.IP)

			// Broadcast updated peer list to everyone
			broadcastPeerList()

		case "list":
			sendPeerList(conn)

		case "signal":
			if !registered {
				sendError(conn, "Must register before signaling")
				continue
			}

			// Forward the encrypted signal (handshake, decision, file chunks) to the target peer
			mu.RLock()
			target, exists := clients[msg.TargetID]
			mu.RUnlock()

			if !exists {
				sendError(conn, fmt.Sprintf("Target peer %s is offline", msg.TargetID))
				continue
			}

			// Relay the raw signal frame directly to the target
			forwardMsg := ServerMsg{
				Type:     "signal",
				SenderID: clientID,
				Payload:  msg.Payload,
			}

			forwardData, err := json.Marshal(forwardMsg)
			if err != nil {
				log.Printf("Relay marshal error: %v", err)
				continue
			}

			target.Mu.Lock()
			// Write length header + data
			var writeErr error
			if writeErr = binary.Write(target.Conn, binary.BigEndian, uint32(len(forwardData))); writeErr == nil {
				_, writeErr = target.Conn.Write(forwardData)
			}
			target.Mu.Unlock()

			if writeErr != nil {
				log.Printf("Failed to relay signal to %s: %v", msg.TargetID, writeErr)
				target.Conn.Close()
			}
		}
	}
}

func sendPeerList(conn net.Conn) {
	mu.RLock()
	list := make([]*PeerInfo, 0, len(peersInfo))
	for _, p := range peersInfo {
		list = append(list, p)
	}
	mu.RUnlock()

	payload, _ := json.Marshal(list)
	srvMsg := ServerMsg{
		Type:    "peer_list",
		Payload: payload,
	}

	data, _ := json.Marshal(srvMsg)

	var err error
	if err = binary.Write(conn, binary.BigEndian, uint32(len(data))); err == nil {
		_, err = conn.Write(data)
	}
	if err != nil {
		log.Printf("Error sending peer list: %v", err)
	}
}

func broadcastPeerList() {
	mu.RLock()
	list := make([]*PeerInfo, 0, len(peersInfo))
	for _, p := range peersInfo {
		list = append(list, p)
	}

	payload, _ := json.Marshal(list)
	srvMsg := ServerMsg{
		Type:    "peer_list",
		Payload: payload,
	}
	data, _ := json.Marshal(srvMsg)

	// Send to all connected clients
	for _, client := range clients {
		client.Mu.Lock()
		if err := binary.Write(client.Conn, binary.BigEndian, uint32(len(data))); err == nil {
			_, _ = client.Conn.Write(data)
		}
		client.Mu.Unlock()
	}
	mu.RUnlock()
}

func sendError(conn net.Conn, errMsg string) {
	payload, _ := json.Marshal(map[string]string{"message": errMsg})
	srvMsg := ServerMsg{
		Type:    "error",
		Payload: payload,
	}
	data, _ := json.Marshal(srvMsg)

	if err := binary.Write(conn, binary.BigEndian, uint32(len(data))); err == nil {
		_, _ = conn.Write(data)
	}
}
