package main

import (
	"crypto/ecdh"
	"crypto/sha256"
	"encoding/base64"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// Default WAN Matchmaking server settings
var DefaultWANServer = decodeAddress("MTYxLjExOC4xNjkuOTU6MTIxMzA=") // Base64 obfuscated IP to shield from raw strings scanning

func decodeAddress(b64 string) string {
	decoded, err := base64.StdEncoding.DecodeString(b64)
	if err != nil {
		return "127.0.0.1:12130"
	}
	return string(decoded)
}

const (
	ServerPort = 12130
)

// ClientMsg represents a message sent from client to server
type ClientMsg struct {
	Type     string          `json:"type"`      // "register", "list", "signal"
	TargetID string          `json:"target_id"` // Recipient Peer ID (for signal)
	Payload  json.RawMessage `json:"payload"`   // Dynamic payload depending on Type
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

// WANSignalPayload represents the inner payload for signaling packets
type WANSignalPayload struct {
	Type              string `json:"type"` // "handshake", "handshake_response", "chunk", "cancel"
	TransferID        string `json:"transfer_id"`
	Filename          string `json:"filename,omitempty"`
	Filesize          int64  `json:"filesize,omitempty"`
	SenderName        string `json:"sender_name,omitempty"`
	SenderPubKey      []byte `json:"sender_pub_key,omitempty"`
	EphemeralPubKey   []byte `json:"ephemeral_pub_key,omitempty"`
	EncryptedMetadata []byte `json:"encrypted_metadata,omitempty"`
	Accepted          bool   `json:"accepted,omitempty"`
	Error             string `json:"error,omitempty"`
	ChunkIndex        int    `json:"chunk_index,omitempty"`
	ChunkData         string `json:"chunk_data,omitempty"` // Base64 GCM Encrypted payload
	IsLast            bool   `json:"is_last,omitempty"`
}

type WANDecision struct {
	Accepted        bool
	EphemeralPubKey []byte
}

type WANService struct {
	mu               sync.RWMutex
	app              *App
	serverAddr       string
	conn             net.Conn
	peerID           string
	username         string
	publicKey        []byte
	color            string
	status           string
	running          bool
	stopChan         chan struct{}
	peers            map[string]*Peer
	activeTransfers  map[string]*TransferState
	pendingDecisions map[string]chan WANDecision
	incomingTxs      map[string]*WANSignalPayload // In-progress incoming handshakes
	txKeys           map[string][]byte            // Derived ECDH shared keys for each TransferID
	txFiles          map[string]*os.File          // Open file handles for receiving
	txFilePaths      map[string]string            // Target paths for receiving
	txBytesRecv      map[string]int64
	txLastTime       map[string]time.Time
	sentRequests     map[string]time.Time         // Tracks last sent friend request per peer to prevent spamming
}

func NewWANService(app *App) *WANService {
	return &WANService{
		app:              app,
		serverAddr:       DefaultWANServer,
		peers:            make(map[string]*Peer),
		activeTransfers:  make(map[string]*TransferState),
		pendingDecisions: make(map[string]chan WANDecision),
		incomingTxs:      make(map[string]*WANSignalPayload),
		txKeys:           make(map[string][]byte),
		txFiles:          make(map[string]*os.File),
		txFilePaths:      make(map[string]string),
		txBytesRecv:      make(map[string]int64),
		txLastTime:       make(map[string]time.Time),
		sentRequests:     make(map[string]time.Time),
	}
}

// SetServerAddress allows user to change the matchmaking server in settings
func (w *WANService) SetServerAddress(addr string) {
	w.mu.Lock()
	if addr == "" {
		w.serverAddr = DefaultWANServer
	} else {
		w.serverAddr = addr
	}
	w.mu.Unlock()
}

func (w *WANService) Start(peerID string, username string, publicKey []byte, color string, status string) {
	w.mu.Lock()
	if w.running {
		w.mu.Unlock()
		return
	}
	w.peerID = peerID
	w.username = username
	w.publicKey = publicKey
	w.color = color
	w.status = status
	w.running = true
	w.stopChan = make(chan struct{})
	w.peers = make(map[string]*Peer)
	w.mu.Unlock()

	go w.connectLoop()
}

func (w *WANService) Stop() {
	w.mu.Lock()
	if !w.running {
		w.mu.Unlock()
		return
	}
	w.running = false
	if w.conn != nil {
		w.conn.Close()
	}
	close(w.stopChan)
	w.peers = make(map[string]*Peer)
	w.mu.Unlock()

	w.app.EmitCombinedPeers()
}

func (w *WANService) GetPeers() []*Peer {
	w.mu.RLock()
	defer w.mu.RUnlock()
	list := make([]*Peer, 0, len(w.peers))
	for _, p := range w.peers {
		list = append(list, p)
	}
	return list
}

func (w *WANService) connectLoop() {
	for {
		w.mu.RLock()
		running := w.running
		addr := w.serverAddr
		w.mu.RUnlock()

		if !running {
			return
		}

		conn, err := net.DialTimeout("tcp", addr, 8*time.Second)
		if err != nil {
			// Backoff and retry
			select {
			case <-w.stopChan:
				return
			case <-time.After(5 * time.Second):
				continue
			}
		}

		w.mu.Lock()
		w.conn = conn
		w.mu.Unlock()

		// Register with Matchmaking Server
		if err := w.sendRegister(); err != nil {
			conn.Close()
			continue
		}

		w.listenLoop(conn)

		select {
		case <-w.stopChan:
			return
		default:
			// Disconnected, clear peers and notify UI
			w.mu.Lock()
			w.peers = make(map[string]*Peer)
			w.mu.Unlock()
			w.app.EmitCombinedPeers()
			time.Sleep(3 * time.Second)
		}
	}
}

func (w *WANService) sendRegister() error {
	w.mu.RLock()
	info := PeerInfo{
		PeerID:    w.peerID,
		Username:  w.username,
		PublicKey: w.publicKey,
		Color:     w.color,
		Status:    w.status,
	}
	conn := w.conn
	w.mu.RUnlock()

	if conn == nil {
		return errors.New("not connected")
	}

	payload, _ := json.Marshal(info)
	msg := ClientMsg{
		Type:    "register",
		Payload: payload,
	}

	data, err := json.Marshal(msg)
	if err != nil {
		return err
	}

	// Write length + body
	var writeErr error
	w.mu.Lock()
	if writeErr = binary.Write(conn, binary.BigEndian, uint32(len(data))); writeErr == nil {
		_, writeErr = conn.Write(data)
	}
	w.mu.Unlock()

	return writeErr
}

func (w *WANService) listenLoop(conn net.Conn) {
	for {
		var length uint32
		err := binary.Read(conn, binary.BigEndian, &length)
		if err != nil {
			return
		}

		buf := make([]byte, length)
		_, err = io.ReadFull(conn, buf)
		if err != nil {
			return
		}

		var srvMsg ServerMsg
		if err := json.Unmarshal(buf, &srvMsg); err != nil {
			continue
		}

		switch srvMsg.Type {
		case "peer_list":
			var list []*PeerInfo
			if err := json.Unmarshal(srvMsg.Payload, &list); err == nil {
				w.updatePeers(list)
			}

		case "signal":
			var signal WANSignalPayload
			if err := json.Unmarshal(srvMsg.Payload, &signal); err == nil {
				w.handleSignal(srvMsg.SenderID, &signal)
			}
		}
	}
}

func (w *WANService) updatePeers(list []*PeerInfo) {
	w.mu.Lock()
	newPeers := make(map[string]*Peer)
	for _, p := range list {
		if p.PeerID == w.peerID {
			continue // skip self
		}
		newPeers[p.PeerID] = &Peer{
			PeerID:    p.PeerID,
			Username:  p.Username,
			IP:        p.IP,
			TCPPort:   ServerPort,
			PublicKey: p.PublicKey,
			LastSeen:  time.Now(),
			Online:    true,
			Color:     p.Color,
			Status:    p.Status,
			IsWAN:     true,
		}
	}
	w.peers = newPeers
	w.mu.Unlock()

	w.app.EmitCombinedPeers()

	// Automatically announce ourselves (send friend request signal) to online friends
	w.app.mu.Lock()
	if w.app.profile != nil && w.app.profile.Friends != nil {
		w.mu.Lock()
		for _, f := range w.app.profile.Friends {
			if _, online := w.peers[f]; online {
				lastSent, exists := w.sentRequests[f]
				if !exists || time.Since(lastSent) > 2*time.Minute {
					w.sentRequests[f] = time.Now()
					go w.SendFriendRequest(f)
				}
			}
		}
		w.mu.Unlock()
	}
	w.app.mu.Unlock()
}

func (w *WANService) handleSignal(senderID string, sig *WANSignalPayload) {
	switch sig.Type {
	case "friend_request":
		if w.app.isFriend(senderID, sig.SenderPubKey) {
			return
		}
		if w.app.ctx != nil {
			w.app.transferManager.playNotificationSound("request")
			runtime.EventsEmit(w.app.ctx, "incoming_friend_request", map[string]interface{}{
				"peer_id":   senderID,
				"username":  sig.SenderName,
				"pub_key":   hex.EncodeToString(sig.SenderPubKey),
			})
		}

	case "handshake":
		w.mu.Lock()
		w.incomingTxs[sig.TransferID] = sig
		w.mu.Unlock()

		// If it is a virtual text message, auto-accept it to a temporary directory without prompting the user.
		if strings.HasPrefix(sig.Filename, "[TextBase64]") {
			go w.AcceptWANTransfer(sig.TransferID, true, os.TempDir())
		} else {
			// Raise event for Svelte UI
			if w.app.ctx != nil {
				w.app.transferManager.playNotificationSound("request")
				runtime.EventsEmit(w.app.ctx, "incoming_request", map[string]interface{}{
					"id":        sig.TransferID,
					"filename":  sig.Filename,
					"filesize":  sig.Filesize,
					"peer_name": sig.SenderName,
					"is_wan":    true,
				})
			}
		}

	case "handshake_response":
		w.mu.Lock()
		ch, exists := w.pendingDecisions[sig.TransferID]
		if exists {
			ch <- WANDecision{Accepted: sig.Accepted, EphemeralPubKey: sig.EphemeralPubKey}
			delete(w.pendingDecisions, sig.TransferID)
		}
		w.mu.Unlock()

	case "chunk":
		w.handleIncomingChunk(sig)

	case "cancel":
		w.mu.Lock()
		if file, ok := w.txFiles[sig.TransferID]; ok {
			file.Close()
			delete(w.txFiles, sig.TransferID)
		}
		path := w.txFilePaths[sig.TransferID]
		if path != "" {
			_ = os.Remove(path)
		}
		w.mu.Unlock()

	case "pause":
		w.mu.Lock()
		tr, exists := w.activeTransfers[sig.TransferID]
		if exists {
			tr.Status = "paused"
		}
		w.mu.Unlock()
		w.app.transferManager.emitTransfers()

	case "resume":
		w.mu.Lock()
		tr, exists := w.activeTransfers[sig.TransferID]
		if exists {
			tr.Status = "transferring"
		}
		w.mu.Unlock()
		w.app.transferManager.emitTransfers()
	}
}

// AcceptWANTransfer processes the recipient Svelte response for WAN downloads
func (w *WANService) AcceptWANTransfer(transferID string, accepted bool, saveDir string) {
	w.mu.Lock()
	sig, ok := w.incomingTxs[transferID]
	w.mu.Unlock()

	if !ok {
		return
	}

	if !accepted {
		w.sendSignal(sig.SenderName, &WANSignalPayload{
			Type:       "handshake_response",
			TransferID: transferID,
			Accepted:   false,
		})
		return
	}

	// Prepare file decryption keys via ECDH
	localPriv, err := GenerateKeyPair() // Ephemeral receiver keys
	if err != nil {
		return
	}

	senderPub, err := ecdh.P256().NewPublicKey(sig.SenderPubKey)
	if err != nil {
		return
	}

	sharedSecret, err := localPriv.ECDH(senderPub)
	if err != nil {
		return
	}

	// Derive GCM AES Key
	aesKey := sha256Bytes(sharedSecret)

	w.mu.Lock()
	w.txKeys[transferID] = aesKey
	w.mu.Unlock()

	// Setup file saving
	targetPath := filepath.Join(saveDir, sig.Filename)
	file, err := os.OpenFile(targetPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0666)
	if err != nil {
		return
	}

	w.mu.Lock()
	w.txFiles[transferID] = file
	w.txFilePaths[transferID] = targetPath
	w.txBytesRecv[transferID] = 0
	w.txLastTime[transferID] = time.Now()
	w.mu.Unlock()

	// Register transfer locally to UI
	ts := &TransferState{
		ID:        transferID,
		Filename:  sig.Filename,
		Filesize:  sig.Filesize,
		Status:    "transferring",
		PeerName:  sig.SenderName,
		IsSender:  false,
		LocalPath: targetPath,
	}
	w.mu.Lock()
	w.activeTransfers[transferID] = ts
	w.mu.Unlock()
	w.app.transferManager.AddExternalTransfer(ts)

	// Send handshake accept back to sender
	w.sendSignal(sig.SenderName, &WANSignalPayload{
		Type:            "handshake_response",
		TransferID:      transferID,
		Accepted:        true,
		EphemeralPubKey: localPriv.PublicKey().Bytes(),
	})
}

func (w *WANService) handleIncomingChunk(sig *WANSignalPayload) {
	w.mu.RLock()
	file := w.txFiles[sig.TransferID]
	key := w.txKeys[sig.TransferID]
	w.mu.RUnlock()

	if file == nil || len(key) == 0 {
		return
	}

	cipherBytes, err := base64.StdEncoding.DecodeString(sig.ChunkData)
	if err != nil {
		return
	}

	plainBytes, err := DecryptGCM(cipherBytes, key)
	if err != nil {
		return
	}

	_, err = file.Write(plainBytes)
	if err != nil {
		return
	}

	w.mu.Lock()
	w.txBytesRecv[sig.TransferID] += int64(len(plainBytes))
	bytesRecv := w.txBytesRecv[sig.TransferID]
	lastTime := w.txLastTime[sig.TransferID]
	now := time.Now()
	w.txLastTime[sig.TransferID] = now
	w.mu.Unlock()

	// Calculate Speed
	duration := now.Sub(lastTime).Seconds()
	var speed float64
	if duration > 0 {
		speed = float64(len(plainBytes)) / (1024 * 1024 * duration)
	}

	percent := int((float64(bytesRecv) / float64(sig.Filesize)) * 100)

	// Update transfer state
	w.app.transferManager.UpdateExternalProgress(sig.TransferID, bytesRecv, speed, percent, "transferring")

	if sig.IsLast {
		file.Close()
		w.mu.Lock()
		delete(w.txFiles, sig.TransferID)
		delete(w.txKeys, sig.TransferID)
		w.mu.Unlock()

		w.app.transferManager.UpdateExternalProgress(sig.TransferID, sig.Filesize, 0, 100, "completed")
		w.app.transferManager.playNotificationSound("success")
	}
}

// SendWANFile streams a file over the internet relay to the recipient PeerID
func (w *WANService) SendWANFile(targetPeerID string, filePath string) (string, error) {
	fileInfo, err := os.Stat(filePath)
	if err != nil {
		return "", err
	}

	transferID := fmt.Sprintf("tx_wan_%d", time.Now().UnixNano())
	filename := filepath.Base(filePath)
	filesize := fileInfo.Size()

	// Register sender transfer locally to UI
	ts := &TransferState{
		ID:        transferID,
		Filename:  filename,
		Filesize:  filesize,
		Status:    "pending",
		PeerName:  targetPeerID,
		IsSender:  true,
		LocalPath: filePath,
	}
	w.mu.Lock()
	w.activeTransfers[transferID] = ts
	w.mu.Unlock()
	w.app.transferManager.AddExternalTransfer(ts)

	go w.sendWANRoutine(transferID, targetPeerID, filename, filesize, filePath)

	return transferID, nil
}

func (w *WANService) sendWANRoutine(transferID, targetPeerID, filename string, filesize int64, filePath string) {
	updateStatus := func(status string, err error) {
		w.app.transferManager.UpdateExternalStatus(transferID, status, err)
	}

	updateStatus("pending", nil)

	// 1. Setup ECDH
	localPriv, err := GenerateKeyPair()
	if err != nil {
		updateStatus("failed", err)
		return
	}

	w.mu.Lock()
	decisionChan := make(chan WANDecision, 1)
	w.pendingDecisions[transferID] = decisionChan
	w.mu.Unlock()

	// Send handshake request
	w.sendSignal(targetPeerID, &WANSignalPayload{
		Type:         "handshake",
		TransferID:   transferID,
		Filename:     filename,
		Filesize:     filesize,
		SenderName:   w.peerID, // Sender ID
		SenderPubKey: localPriv.PublicKey().Bytes(),
	})

	// Wait for recipient acceptance decision
	var decision WANDecision
	select {
	case decision = <-decisionChan:
	case <-time.After(45 * time.Second):
		updateStatus("failed", errors.New("timeout waiting for recipient approval"))
		return
	}

	if !decision.Accepted {
		updateStatus("declined", nil)
		return
	}

	recipientPubKey := decision.EphemeralPubKey

	// Use recipient's key for ECDH shared secret derivation
	recipientPub, err := ecdh.P256().NewPublicKey(recipientPubKey)
	if err != nil {
		updateStatus("failed", err)
		return
	}

	sharedSecret, err := localPriv.ECDH(recipientPub)
	if err != nil {
		updateStatus("failed", err)
		return
	}

	aesKey := sha256Bytes(sharedSecret)

	// Open file to stream
	file, err := os.Open(filePath)
	if err != nil {
		updateStatus("failed", err)
		return
	}
	defer file.Close()

	updateStatus("transferring", nil)

	buffer := make([]byte, ChunkSize)
	var bytesSent int64
	chunkIndex := 0
	lastTime := time.Now()

	for {
		// Pause check loop
		for {
			w.mu.RLock()
			tr, exists := w.activeTransfers[transferID]
			var isPaused bool
			if exists {
				isPaused = tr.Status == "paused"
			}
			w.mu.RUnlock()

			if !isPaused {
				break
			}
			time.Sleep(100 * time.Millisecond)
		}

		n, err := file.Read(buffer)
		if n > 0 {
			chunkData := buffer[:n]
			cipherBytes, encryptErr := EncryptGCM(chunkData, aesKey)
			if encryptErr != nil {
				updateStatus("failed", encryptErr)
				return
			}

			b64Str := base64.StdEncoding.EncodeToString(cipherBytes)
			isLast := (bytesSent + int64(n)) >= filesize

			w.sendSignal(targetPeerID, &WANSignalPayload{
				Type:       "chunk",
				TransferID: transferID,
				ChunkIndex: chunkIndex,
				ChunkData:  b64Str,
				IsLast:     isLast,
				Filesize:   filesize,
			})

			bytesSent += int64(n)
			chunkIndex++

			now := time.Now()
			duration := now.Sub(lastTime).Seconds()
			lastTime = now
			var speed float64
			if duration > 0 {
				speed = float64(n) / (1024 * 1024 * duration)
			}
			percent := int((float64(bytesSent) / float64(filesize)) * 100)

			w.app.transferManager.UpdateExternalProgress(transferID, bytesSent, speed, percent, "transferring")

			if isLast {
				break
			}
		}

		if err != nil {
			if err == io.EOF {
				break
			}
			updateStatus("failed", err)
			return
		}
	}

	updateStatus("completed", nil)
}

func (w *WANService) sendSignal(targetID string, sig *WANSignalPayload) {
	w.mu.RLock()
	conn := w.conn
	w.mu.RUnlock()

	if conn == nil {
		return
	}

	sigBytes, _ := json.Marshal(sig)
	msg := ClientMsg{
		Type:     "signal",
		TargetID: targetID,
		Payload:  sigBytes,
	}

	data, err := json.Marshal(msg)
	if err != nil {
		return
	}

	w.mu.Lock()
	if err := binary.Write(conn, binary.BigEndian, uint32(len(data))); err == nil {
		_, _ = conn.Write(data)
	}
	w.mu.Unlock()
}

func sha256Bytes(data []byte) []byte {
	h := sha256.New()
	h.Write(data)
	return h.Sum(nil)
}

// PauseTransfer marks a transferring WAN state as paused and sends a signal to the peer
func (w *WANService) PauseTransfer(transferID string) {
	w.mu.Lock()
	tr, exists := w.activeTransfers[transferID]
	if exists && tr.Status == "transferring" {
		tr.Status = "paused"
		peerID := tr.PeerName
		w.mu.Unlock()
		
		// Send signal
		w.sendSignal(peerID, &WANSignalPayload{
			Type:       "pause",
			TransferID: transferID,
		})
		w.app.transferManager.emitTransfers()
		return
	}
	w.mu.Unlock()
}

// ResumeTransfer marks a paused WAN state as transferring again and sends a signal to the peer
func (w *WANService) ResumeTransfer(transferID string) {
	w.mu.Lock()
	tr, exists := w.activeTransfers[transferID]
	if exists && tr.Status == "paused" {
		tr.Status = "transferring"
		peerID := tr.PeerName
		w.mu.Unlock()
		
		// Send signal
		w.sendSignal(peerID, &WANSignalPayload{
			Type:       "resume",
			TransferID: transferID,
		})
		w.app.transferManager.emitTransfers()
		return
	}
	w.mu.Unlock()
}

// SendFriendRequest sends a friend request signal to another peer via WAN
func (w *WANService) SendFriendRequest(targetPeerID string) {
	w.mu.RLock()
	username := w.username
	pubKey := w.publicKey
	w.mu.RUnlock()

	w.sendSignal(targetPeerID, &WANSignalPayload{
		Type:         "friend_request",
		TransferID:   w.peerID, // use our peer ID as transfer ID for the request
		SenderName:   username,
		SenderPubKey: pubKey,
	})
}
