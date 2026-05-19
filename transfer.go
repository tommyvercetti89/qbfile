package main

import (
	"archive/zip"
	"crypto/ecdh"
	"crypto/sha256"
	"encoding/binary"
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

const (
	ChunkSize       = 256 * 1024 // 256 KB chunks for high-speed transfers
	TransferTimeout = 60 * time.Second
)

// TransferState tracks active and past transfers
type TransferState struct {
	ID        string  `json:"id"`
	Filename  string  `json:"filename"`
	Filesize  int64   `json:"filesize"`
	BytesSent int64   `json:"bytes_sent"`
	BytesRecv int64   `json:"bytes_recv"`
	SpeedMB   float64 `json:"speed_mb"` // MB/s
	Percent   int     `json:"percent"`
	Status    string  `json:"status"` // "pending", "transferring", "completed", "failed", "declined"
	PeerName  string  `json:"peer_name"`
	IsSender  bool    `json:"is_sender"`
	LocalPath string  `json:"local_path"`
}

// HandshakeMessage contains sender identification and ephemeral keys
type HandshakeMessage struct {
	SenderName       string `json:"sender_name"`
	SenderPubKey     []byte `json:"sender_pub_key"`     // Sender Profile Public Key (P-256 bytes)
	EphemeralPubKey  []byte `json:"ephemeral_pub_key"`  // Sender Ephemeral Public Key
	EncryptedMetadata []byte `json:"encrypted_metadata"` // Encrypted TransferMetadata
}

// TransferMetadata contains file details encrypted during handshake
type TransferMetadata struct {
	ID       string `json:"id"`
	Filename string `json:"filename"`
	Filesize int64  `json:"filesize"`
}

// HandshakeResponse tells the sender if the transfer is approved
type HandshakeResponse struct {
	Accepted bool   `json:"accepted"`
	Error    string `json:"error"`
}

// Decision represents the receiver's choice
type Decision struct {
	Accepted bool
	SaveDir  string
}

// TransferManager manages active TCP transfers
type TransferManager struct {
	mu               sync.RWMutex
	app              *App
	tcpListener      net.Listener
	port             int
	activeTransfers  map[string]*TransferState
	pendingDecisions map[string]chan Decision
	activeConns      map[string]net.Conn
	profile          *Profile
	stopChan         chan struct{}
	tempFiles        sync.Map // Tracks temporary zip files to clean up
}

// NewTransferManager initializes the manager
func NewTransferManager(app *App) *TransferManager {
	tm := &TransferManager{
		app:              app,
		activeTransfers:  make(map[string]*TransferState),
		pendingDecisions: make(map[string]chan Decision),
		activeConns:      make(map[string]net.Conn),
		stopChan:         make(chan struct{}),
	}
	tm.LoadTransfers()
	return tm
}

// Start listens on a random or standard TCP port
func (t *TransferManager) Start(profile *Profile) (int, error) {
	t.mu.Lock()
	t.profile = profile
	t.mu.Unlock()

	// Try to listen on 0 (random open port) to ensure no port conflicts
	listener, err := net.Listen("tcp", "0.0.0.0:0")
	if err != nil {
		// fallback to random port
		listener, err = net.Listen("tcp", ":0")
		if err != nil {
			return 0, err
		}
	}
	t.tcpListener = listener
	t.port = listener.Addr().(*net.TCPAddr).Port

	go t.listenLoop()

	return t.port, nil
}

// Stop shuts down the transfer manager
func (t *TransferManager) Stop() {
	if t.tcpListener != nil {
		t.tcpListener.Close()
	}
	close(t.stopChan)
}

// AcceptTransfer receives frontend decision for an incoming file
func (t *TransferManager) AcceptTransfer(transferID string, accepted bool, saveDir string) {
	t.mu.Lock()
	ch, exists := t.pendingDecisions[transferID]
	if exists {
		ch <- Decision{Accepted: accepted, SaveDir: saveDir}
		delete(t.pendingDecisions, transferID)
	}
	t.mu.Unlock()
}

// SendFile initiates secure file transfer to another peer
func (t *TransferManager) SendFile(peerIP string, peerPort int, peerProfilePubKey []byte, filePath string) (string, error) {
	t.mu.RLock()
	localProfile := t.profile
	t.mu.RUnlock()

	if localProfile == nil {
		return "", errors.New("profile not unlocked")
	}

	// 1. Check file exists
	fileInfo, err := os.Stat(filePath)
	if err != nil {
		return "", fmt.Errorf("file error: %w", err)
	}

	transferID := fmt.Sprintf("tx_%d", time.Now().UnixNano())
	filename := filepath.Base(filePath)
	filesize := fileInfo.Size()

	transfer := &TransferState{
		ID:        transferID,
		Filename:  filename,
		Filesize:  filesize,
		Status:    "pending",
		PeerName:  "Connecting...",
		IsSender:  true,
		LocalPath: filePath,
	}

	t.mu.Lock()
	t.activeTransfers[transferID] = transfer
	t.mu.Unlock()
	t.emitTransfers()
	t.SaveTransfers()

	go t.sendFileRoutine(transferID, peerIP, peerPort, peerProfilePubKey, filePath)

	return transferID, nil
}

// sendFileRoutine runs in background to connect, handshake, encrypt, and stream
func (t *TransferManager) sendFileRoutine(transferID, peerIP string, peerPort int, peerProfilePubKey []byte, filePath string) {
	defer func() {
		if _, isTemp := t.tempFiles.LoadAndDelete(filePath); isTemp {
			_ = os.Remove(filePath)
		}
	}()

	updateStatus := func(status string, err error) {
		t.mu.Lock()
		p, exists := t.activeTransfers[transferID]
		if exists {
			p.Status = status
			if err != nil {
				p.PeerName = fmt.Sprintf("Error: %s", err.Error())
			}
		}
		t.mu.Unlock()
		t.emitTransfers()
		t.SaveTransfers()
	}

	// Connect
	conn, err := net.DialTimeout("tcp", fmt.Sprintf("%s:%d", peerIP, peerPort), 10*time.Second)
	if err != nil {
		updateStatus("failed", err)
		return
	}
	defer conn.Close()

	if tcpConn, ok := conn.(*net.TCPConn); ok {
		_ = tcpConn.SetNoDelay(true)
	}

	t.mu.Lock()
	t.activeConns[transferID] = conn
	t.mu.Unlock()
	defer func() {
		t.mu.Lock()
		delete(t.activeConns, transferID)
		t.mu.Unlock()
	}()

	// Start a background loop to listen for pause/resume/cancel signals from the receiver
	go func() {
		buf := make([]byte, 16)
		for {
			n, err := conn.Read(buf)
			if err != nil {
				return
			}
			cmd := string(buf[:n])
			if strings.Contains(cmd, "pause") {
				t.mu.Lock()
				if tr, exists := t.activeTransfers[transferID]; exists {
					tr.Status = "paused"
				}
				t.mu.Unlock()
				t.emitTransfers()
			} else if strings.Contains(cmd, "resume") {
				t.mu.Lock()
				if tr, exists := t.activeTransfers[transferID]; exists {
					tr.Status = "transferring"
				}
				t.mu.Unlock()
				t.emitTransfers()
			} else if strings.Contains(cmd, "cancel") {
				t.mu.Lock()
				if tr, exists := t.activeTransfers[transferID]; exists {
					tr.Status = "failed"
				}
				t.mu.Unlock()
				t.emitTransfers()
				_ = conn.Close()
				return
			}
		}
	}()

	// 1. Perform ECDH Shared Secret Agreement
	localPriv, err := ecdh.P256().NewPrivateKey(t.profile.PrivateKey)
	if err != nil {
		updateStatus("failed", err)
		return
	}

	peerPub, err := ecdh.P256().NewPublicKey(peerProfilePubKey)
	if err != nil {
		updateStatus("failed", err)
		return
	}

	// Derive master shared secret
	sharedSecret, err := localPriv.ECDH(peerPub)
	if err != nil {
		updateStatus("failed", err)
		return
	}

	// Derive AES-256 session key
	hash := sha256.Sum256(sharedSecret)
	sessionKey := hash[:]

	// 2. Prepare Metadata Payload
	meta := TransferMetadata{
		ID:       transferID,
		Filename: filepath.Base(filePath),
		Filesize: mustGetSize(filePath),
	}

	metaJSON, _ := json.Marshal(meta)
	encryptedMeta, err := EncryptGCM(metaJSON, sessionKey)
	if err != nil {
		updateStatus("failed", err)
		return
	}

	// Ephemeral key for additional session PFS (Perfect Forward Secrecy)
	ephemeralPriv, err := GenerateKeyPair()
	if err != nil {
		updateStatus("failed", err)
		return
	}
	ephemeralPubBytes := ephemeralPriv.PublicKey().Bytes()

	// Update transfer metadata
	t.mu.Lock()
	t.activeTransfers[transferID].PeerName = "Waiting for approval..."
	t.mu.Unlock()
	t.emitTransfers()
	t.SaveTransfers()

	// Send Handshake Message
	handshake := HandshakeMessage{
		SenderName:        t.profile.Username,
		SenderPubKey:      t.profile.PublicKey,
		EphemeralPubKey:   ephemeralPubBytes,
		EncryptedMetadata: encryptedMeta,
	}

	handshakeBytes, err := json.Marshal(handshake)
	if err != nil {
		updateStatus("failed", err)
		return
	}

	// Send handshake size, then handshake
	sizeBuf := make([]byte, 4)
	binary.BigEndian.PutUint32(sizeBuf, uint32(len(handshakeBytes)))
	if _, err := conn.Write(sizeBuf); err != nil {
		updateStatus("failed", err)
		return
	}
	if _, err := conn.Write(handshakeBytes); err != nil {
		updateStatus("failed", err)
		return
	}

	// Read response
	var resp HandshakeResponse
	dec := json.NewDecoder(conn)
	if err := dec.Decode(&resp); err != nil {
		updateStatus("failed", err)
		return
	}

	if !resp.Accepted {
		if resp.Error == "declined" {
			updateStatus("declined", nil)
		} else {
			updateStatus("failed", errors.New(resp.Error))
		}
		return
	}

	// Peer accepted! Update status
	t.mu.Lock()
	p := t.activeTransfers[transferID]
	p.Status = "transferring"
	p.PeerName = handshake.SenderName // Will update with recipient name from response if needed, let's keep it simple
	t.mu.Unlock()
	t.emitTransfers()
	t.SaveTransfers()

	// Open file to stream
	file, err := os.Open(filePath)
	if err != nil {
		updateStatus("failed", err)
		return
	}
	defer file.Close()

	// Stream chunks encrypted with AES-GCM
	fileBuf := make([]byte, ChunkSize)
	var bytesSent int64
	startTime := time.Now()
	lastReportTime := time.Now()
	var lastReportBytes int64

	for {
		// Pause / cancel check loop
		for {
			t.mu.RLock()
			tr, exists := t.activeTransfers[transferID]
			var status string
			if exists {
				status = tr.Status
			}
			t.mu.RUnlock()

			if status == "failed" {
				// Cancelled — bail out
				return
			}
			if status != "paused" {
				break
			}
			// Reset last report times to avoid speed calculation spikes and timeouts
			lastReportTime = time.Now()
			time.Sleep(100 * time.Millisecond)
		}

		n, err := file.Read(fileBuf)
		if n > 0 {
			// Encrypt chunk
			encryptedChunk, err := EncryptGCM(fileBuf[:n], sessionKey)
			if err != nil {
				updateStatus("failed", err)
				return
			}

			// Send encrypted chunk size (4 bytes)
			binary.BigEndian.PutUint32(sizeBuf, uint32(len(encryptedChunk)))
			if _, err := conn.Write(sizeBuf); err != nil {
				updateStatus("failed", err)
				return
			}

			// Send encrypted chunk
			if _, err := conn.Write(encryptedChunk); err != nil {
				updateStatus("failed", err)
				return
			}

			bytesSent += int64(n)

			// Periodic updates
			if time.Since(lastReportTime) >= 500*time.Millisecond {
				now := time.Now()
				duration := now.Sub(lastReportTime).Seconds()
				deltaBytes := bytesSent - lastReportBytes
				speed := (float64(deltaBytes) / (1024 * 1024)) / duration

				t.mu.Lock()
				if p, exists := t.activeTransfers[transferID]; exists {
					p.BytesSent = bytesSent
					p.Percent = int((float64(bytesSent) / float64(p.Filesize)) * 100)
					p.SpeedMB = speed
				}
				t.mu.Unlock()
				t.emitTransfers()

				lastReportTime = now
				lastReportBytes = bytesSent
			}
		}

		if err == io.EOF {
			break
		}
		if err != nil {
			updateStatus("failed", err)
			return
		}
	}

	// Final completed report
	totalDuration := time.Since(startTime).Seconds()
	if totalDuration == 0 {
		totalDuration = 0.1
	}
	t.mu.Lock()
	if p, exists := t.activeTransfers[transferID]; exists {
		p.Status = "completed"
		p.BytesSent = p.Filesize
		p.Percent = 100
		p.SpeedMB = (float64(p.Filesize) / (1024 * 1024)) / totalDuration
	}
	t.mu.Unlock()
	t.emitTransfers()
	t.SaveTransfers()
}

// listenLoop accepts incoming TCP file transfer requests
func (t *TransferManager) listenLoop() {
	for {
		conn, err := t.tcpListener.Accept()
		if err != nil {
			// Listener closed, stop
			return
		}

		go t.receiveTransferRoutine(conn)
	}
}

// receiveTransferRoutine handles handshakes, requests authorization, decrypts and writes files
func (t *TransferManager) receiveTransferRoutine(conn net.Conn) {
	defer conn.Close()

	if tcpConn, ok := conn.(*net.TCPConn); ok {
		_ = tcpConn.SetNoDelay(true)
	}

	// 1. Read handshake size (4 bytes)
	sizeBuf := make([]byte, 4)
	if _, err := io.ReadFull(conn, sizeBuf); err != nil {
		return
	}
	handshakeSize := binary.BigEndian.Uint32(sizeBuf)

	// Read handshake payload
	handshakeBytes := make([]byte, handshakeSize)
	if _, err := io.ReadFull(conn, handshakeBytes); err != nil {
		return
	}

	var handshake HandshakeMessage
	if err := json.Unmarshal(handshakeBytes, &handshake); err != nil {
		return
	}

	// 2. Perform ECDH Key Agreement to Decrypt Metadata
	t.mu.RLock()
	localProfile := t.profile
	t.mu.RUnlock()

	if localProfile == nil {
		// Reject
		json.NewEncoder(conn).Encode(HandshakeResponse{Accepted: false, Error: "Profile locked"})
		return
	}

	localPriv, err := ecdh.P256().NewPrivateKey(localProfile.PrivateKey)
	if err != nil {
		return
	}

	senderPub, err := ecdh.P256().NewPublicKey(handshake.SenderPubKey)
	if err != nil {
		json.NewEncoder(conn).Encode(HandshakeResponse{Accepted: false, Error: "Invalid sender public key"})
		return
	}

	// Derive session key
	sharedSecret, err := localPriv.ECDH(senderPub)
	if err != nil {
		return
	}

	hash := sha256.Sum256(sharedSecret)
	sessionKey := hash[:]

	// Decrypt metadata
	metaBytes, err := DecryptGCM(handshake.EncryptedMetadata, sessionKey)
	if err != nil {
		json.NewEncoder(conn).Encode(HandshakeResponse{Accepted: false, Error: "Decryption failed"})
		return
	}

	var meta TransferMetadata
	if err := json.Unmarshal(metaBytes, &meta); err != nil {
		return
	}

	// 3. Create Transfer State
	transferID := meta.ID
	transfer := &TransferState{
		ID:        transferID,
		Filename:  meta.Filename,
		Filesize:  meta.Filesize,
		Status:    "pending",
		PeerName:  handshake.SenderName,
		IsSender:  false,
		LocalPath: "",
	}

	t.mu.Lock()
	t.activeTransfers[transferID] = transfer
	decisionChan := make(chan Decision, 1)
	t.pendingDecisions[transferID] = decisionChan
	t.mu.Unlock()
	t.emitTransfers()
	t.SaveTransfers()

	// If it is a virtual text message, auto-accept it to a temporary directory without prompting the user.
	if strings.HasPrefix(meta.Filename, "[TextBase64]") {
		decisionChan <- Decision{Accepted: true, SaveDir: os.TempDir()}
	} else {
		// Notify Frontend of incoming transfer request
		if t.app != nil && t.app.ctx != nil {
			runtime.EventsEmit(t.app.ctx, "incoming_request", transfer)
		}
	}

	// Wait for user decision (Accept / Decline) with timeout
	var decision Decision
	select {
	case d := <-decisionChan:
		decision = d
	case <-time.After(TransferTimeout):
		decision = Decision{Accepted: false, SaveDir: ""}
		t.mu.Lock()
		delete(t.pendingDecisions, transferID)
		t.mu.Unlock()
	}

	// If declined
	if !decision.Accepted {
		json.NewEncoder(conn).Encode(HandshakeResponse{Accepted: false, Error: "declined"})
		t.mu.Lock()
		transfer.Status = "declined"
		t.mu.Unlock()
		t.emitTransfers()
		t.SaveTransfers()
		return
	}

	// Create local file to save
	savePath := filepath.Join(decision.SaveDir, meta.Filename)
	
	// Deduplicate filename if already exists
	savePath = deduplicatePath(savePath)

	file, err := os.Create(savePath)
	if err != nil {
		json.NewEncoder(conn).Encode(HandshakeResponse{Accepted: false, Error: fmt.Sprintf("Failed to create file: %s", err.Error())})
		t.mu.Lock()
		transfer.Status = "failed"
		transfer.PeerName = fmt.Sprintf("Error creating file: %s", err.Error())
		t.mu.Unlock()
		t.emitTransfers()
		t.SaveTransfers()
		return
	}
	defer file.Close()

	t.mu.Lock()
	transfer.LocalPath = savePath
	transfer.Status = "transferring"
	t.activeConns[transferID] = conn
	t.mu.Unlock()
	defer func() {
		t.mu.Lock()
		delete(t.activeConns, transferID)
		t.mu.Unlock()
	}()
	t.emitTransfers()
	t.SaveTransfers()

	// Send acceptance response
	err = json.NewEncoder(conn).Encode(HandshakeResponse{Accepted: true})
	if err != nil {
		t.mu.Lock()
		transfer.Status = "failed"
		t.mu.Unlock()
		t.emitTransfers()
		t.SaveTransfers()
		return
	}

	// Read and decrypt chunks
	var bytesRecv int64
	startTime := time.Now()
	lastReportTime := time.Now()
	var lastReportBytes int64

	for {
		// Pause / cancel check loop
		for {
			t.mu.RLock()
			tr, exists := t.activeTransfers[transferID]
			var status string
			if exists {
				status = tr.Status
			}
			t.mu.RUnlock()

			if status == "failed" {
				// Cancelled by receiver — clean up partial file
				file.Close()
				_ = os.Remove(transfer.LocalPath)
				t.emitTransfers()
				t.SaveTransfers()
				return
			}
			if status != "paused" {
				break
			}
			lastReportTime = time.Now()
			time.Sleep(100 * time.Millisecond)
		}

		// Read encrypted chunk size (4 bytes)
		if _, err := io.ReadFull(conn, sizeBuf); err != nil {
			if err == io.EOF || err == io.ErrUnexpectedEOF {
				// Completed or sender disconnected
				break
			}
			t.mu.Lock()
			transfer.Status = "failed"
			t.mu.Unlock()
			t.emitTransfers()
			t.SaveTransfers()
			return
		}
		chunkSize := binary.BigEndian.Uint32(sizeBuf)

		// Read encrypted chunk bytes
		encryptedChunk := make([]byte, chunkSize)
		if _, err := io.ReadFull(conn, encryptedChunk); err != nil {
			t.mu.Lock()
			transfer.Status = "failed"
			t.mu.Unlock()
			t.emitTransfers()
			t.SaveTransfers()
			return
		}

		// Decrypt chunk
		decryptedChunk, err := DecryptGCM(encryptedChunk, sessionKey)
		if err != nil {
			t.mu.Lock()
			transfer.Status = "failed"
			transfer.PeerName = "Decryption error on stream"
			t.mu.Unlock()
			t.emitTransfers()
			t.SaveTransfers()
			return
		}

		// Write to local disk
		if _, err := file.Write(decryptedChunk); err != nil {
			t.mu.Lock()
			transfer.Status = "failed"
			transfer.PeerName = "Disk write error"
			t.mu.Unlock()
			t.emitTransfers()
			t.SaveTransfers()
			return
		}

		bytesRecv += int64(len(decryptedChunk))

		// Periodic progress updates
		if time.Since(lastReportTime) >= 500*time.Millisecond {
			now := time.Now()
			duration := now.Sub(lastReportTime).Seconds()
			deltaBytes := bytesRecv - lastReportBytes
			speed := (float64(deltaBytes) / (1024 * 1024)) / duration

			t.mu.Lock()
			transfer.BytesRecv = bytesRecv
			transfer.Percent = int((float64(bytesRecv) / float64(transfer.Filesize)) * 100)
			transfer.SpeedMB = speed
			t.mu.Unlock()
			t.emitTransfers()

			lastReportTime = now
			lastReportBytes = bytesRecv
		}
	}

	// Verify size matched
	t.mu.Lock()
	totalDuration := time.Since(startTime).Seconds()
	if totalDuration == 0 {
		totalDuration = 0.1
	}

	if bytesRecv == transfer.Filesize {
		transfer.Status = "completed"
		transfer.Percent = 100
		transfer.BytesRecv = bytesRecv
		transfer.SpeedMB = (float64(transfer.Filesize) / (1024 * 1024)) / totalDuration
	} else {
		transfer.Status = "failed"
		transfer.PeerName = "Transfer cut off before completion"
	}
	t.mu.Unlock()
	t.emitTransfers()
	t.SaveTransfers()
}

// GetTransfers returns all active transfers
func (t *TransferManager) GetTransfers() []*TransferState {
	t.mu.RLock()
	defer t.mu.RUnlock()

	list := make([]*TransferState, 0, len(t.activeTransfers))
	for _, tr := range t.activeTransfers {
		list = append(list, tr)
	}
	return list
}

// emitTransfers sends active transfers to frontend
func (t *TransferManager) emitTransfers() {
	t.mu.RLock()
	list := make([]*TransferState, 0, len(t.activeTransfers))
	for _, tr := range t.activeTransfers {
		list = append(list, tr)
	}
	t.mu.RUnlock()

	if t.app != nil && t.app.ctx != nil {
		runtime.EventsEmit(t.app.ctx, "transfers_updated", list)
	}
}

// Helper: gets size of file
func mustGetSize(path string) int64 {
	stat, err := os.Stat(path)
	if err != nil {
		return 0
	}
	return stat.Size()
}

// Deduplicate local path to prevent overwriting
func deduplicatePath(path string) string {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return path
	}

	ext := filepath.Ext(path)
	base := path[:len(path)-len(ext)]
	counter := 1

	for {
		newPath := fmt.Sprintf("%s (%d)%s", base, counter, ext)
		if _, err := os.Stat(newPath); os.IsNotExist(err) {
			return newPath
		}
		counter++
	}
}

// SaveTransfers saves active and past transfers to a local JSON file
func (t *TransferManager) SaveTransfers() {
	t.mu.RLock()
	defer t.mu.RUnlock()

	list := make([]*TransferState, 0, len(t.activeTransfers))
	for _, tr := range t.activeTransfers {
		savedTr := *tr
		// Reset ongoing/paused states to failed when saving to prevent stuck states on restart
		if savedTr.Status == "transferring" || savedTr.Status == "pending" || savedTr.Status == "paused" {
			savedTr.Status = "failed"
			savedTr.PeerName = "Kesildi"
		}
		savedTr.SpeedMB = 0
		list = append(list, &savedTr)
	}

	data, err := json.Marshal(list)
	if err == nil {
		_ = os.WriteFile(GetStoragePath("qbfile_transfers.json"), data, 0600)
	}
}

// LoadTransfers loads transfer history from the local JSON file
func (t *TransferManager) LoadTransfers() {
	data, err := os.ReadFile(GetStoragePath("qbfile_transfers.json"))
	if err != nil {
		return
	}

	var list []*TransferState
	err = json.Unmarshal(data, &list)
	if err != nil {
		return
	}

	t.mu.Lock()
	for _, tr := range list {
		t.activeTransfers[tr.ID] = tr
	}
	t.mu.Unlock()
}

// UpdateExternalStatus updates a transfer's status from an external source (e.g. WAN service)
func (t *TransferManager) UpdateExternalStatus(transferID string, status string, err error) {
	t.mu.Lock()
	if tr, exists := t.activeTransfers[transferID]; exists {
		tr.Status = status
		if err != nil {
			tr.PeerName = err.Error()
		}
	}
	t.mu.Unlock()
	t.emitTransfers()
	t.SaveTransfers()
}

// ClearTransferHistory clears the local history file and memory state
func (t *TransferManager) ClearTransferHistory() {
	t.mu.Lock()
	t.activeTransfers = make(map[string]*TransferState)
	t.mu.Unlock()

	t.emitTransfers()
	_ = os.Remove(GetStoragePath("qbfile_transfers.json"))
}

// PauseTransfer marks a transferring state as paused
func (t *TransferManager) PauseTransfer(transferID string) {
	t.mu.Lock()
	tr, exists := t.activeTransfers[transferID]
	if exists && tr.Status == "transferring" {
		tr.Status = "paused"
		if conn, hasConn := t.activeConns[transferID]; hasConn {
			if !tr.IsSender {
				go conn.Write([]byte("pause"))
			}
		}
	}
	t.mu.Unlock()
	t.emitTransfers()
	t.SaveTransfers()
}

// ResumeTransfer marks a paused state as transferring again
func (t *TransferManager) ResumeTransfer(transferID string) {
	t.mu.Lock()
	tr, exists := t.activeTransfers[transferID]
	if exists && tr.Status == "paused" {
		tr.Status = "transferring"
		if conn, hasConn := t.activeConns[transferID]; hasConn {
			if !tr.IsSender {
				go conn.Write([]byte("resume"))
			}
		}
	}
	t.mu.Unlock()
	t.emitTransfers()
	t.SaveTransfers()
}

// CancelTransfer cancels an active LAN transfer, sends cancel command to peer if receiver, and cleans up local files
func (t *TransferManager) CancelTransfer(transferID string) {
	t.mu.Lock()
	tr, exists := t.activeTransfers[transferID]
	if exists && (tr.Status == "transferring" || tr.Status == "paused" || tr.Status == "pending") {
		tr.Status = "failed"
		if conn, hasConn := t.activeConns[transferID]; hasConn {
			if !tr.IsSender {
				go func() {
					_, _ = conn.Write([]byte("cancel"))
					_ = conn.Close()
				}()
			} else {
				_ = conn.Close()
			}
		}
		localPath := tr.LocalPath
		t.mu.Unlock()

		if localPath != "" {
			_ = os.Remove(localPath)
		}
	} else {
		t.mu.Unlock()
	}
	t.emitTransfers()
	t.SaveTransfers()
}

// zipDirectory zips a folder to a temporary file and returns its path
func zipDirectory(sourceDir string) (string, error) {
	tempFile, err := os.CreateTemp("", "qbfile_dir_*.zip")
	if err != nil {
		return "", err
	}
	defer tempFile.Close()

	archive := zip.NewWriter(tempFile)
	defer archive.Close()

	err = filepath.Walk(sourceDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		relPath, err := filepath.Rel(sourceDir, path)
		if err != nil {
			return err
		}

		if relPath == "." {
			return nil
		}

		header, err := zip.FileInfoHeader(info)
		if err != nil {
			return err
		}

		header.Name = filepath.ToSlash(relPath)
		if info.IsDir() {
			header.Name += "/"
		} else {
			header.Method = zip.Deflate
		}

		writer, err := archive.CreateHeader(header)
		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		file, err := os.Open(path)
		if err != nil {
			return err
		}
		defer file.Close()

		_, err = io.Copy(writer, file)
		return err
	})

	if err != nil {
		os.Remove(tempFile.Name())
		return "", err
	}

	return tempFile.Name(), nil
}

// AddExternalTransfer inserts a transfer state managed by an external service (like WAN)
func (t *TransferManager) AddExternalTransfer(ts *TransferState) {
	t.mu.Lock()
	t.activeTransfers[ts.ID] = ts
	t.mu.Unlock()
	t.emitTransfers()
	t.SaveTransfers()
}

// UpdateExternalProgress updates bytes, speed, percentage, and status for a transfer
func (t *TransferManager) UpdateExternalProgress(id string, bytes int64, speed float64, percent int, status string) {
	t.mu.Lock()
	tr, exists := t.activeTransfers[id]
	if exists {
		if tr.IsSender {
			tr.BytesSent = bytes
		} else {
			tr.BytesRecv = bytes
		}
		tr.SpeedMB = speed
		tr.Percent = percent
		if tr.Status != "paused" {
			tr.Status = status
		}
	}
	t.mu.Unlock()
	t.emitTransfers()
}



// playNotificationSound triggers a sound event in the frontend context
func (t *TransferManager) playNotificationSound(soundType string) {
	if t.app != nil && t.app.ctx != nil {
		runtime.EventsEmit(t.app.ctx, "play_sound", soundType)
	}
}
