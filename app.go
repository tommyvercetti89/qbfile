package main

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// App struct
type App struct {
	ctx              context.Context
	discoveryService *DiscoveryService
	transferManager  *TransferManager
	wanService       *WANService
	profile          *Profile
	sessionKey       []byte
	mu               sync.RWMutex
	startupFilePath  string
}

// NewApp creates a new App application struct
func NewApp() *App {
	return &App{}
}

// startup is called when the app starts. The context is saved
// so we can call the runtime methods
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	a.discoveryService = NewDiscoveryService(a)
	a.transferManager = NewTransferManager(a)
	a.wanService = NewWANService(a)
}

// CheckProfileExists returns true if a secure profile has been created
func (a *App) CheckProfileExists() bool {
	return CheckProfileExists()
}

// RegisterProfile creates a brand new profile and initiates services
func (a *App) RegisterProfile(username string, password string) error {
	if username == "" || password == "" {
		return errors.New("username and password cannot be empty")
	}

	profile, key, err := CreateProfile(username, password)
	if err != nil {
		return err
	}

	a.mu.Lock()
	a.profile = profile
	a.sessionKey = key
	a.mu.Unlock()

	return a.startServices()
}

// LoginProfile decrypts the profile using password and initiates services
func (a *App) LoginProfile(password string) error {
	profile, key, err := UnlockProfile(password)
	if err != nil {
		return err
	}

	a.mu.Lock()
	a.profile = profile
	a.sessionKey = key
	a.mu.Unlock()

	return a.startServices()
}

// AutoLoadProfile automatically loads the profile without any password.
// If the profile is corrupted or in an old password-encrypted format, it automatically recreates it.
func (a *App) AutoLoadProfile() error {
	profile, key, err := UnlockProfileAuto()
	if err != nil {
		// Auto delete corrupted or old password-encrypted profile file
		_ = os.Remove(GetStoragePath(ProfileFileName))

		// Create a fresh automatic profile
		newProfile, newKey, createErr := CreateProfileAuto("Cihaz")
		if createErr != nil {
			return fmt.Errorf("failed to auto-recreate corrupted profile: %w", createErr)
		}

		a.mu.Lock()
		a.profile = newProfile
		a.sessionKey = newKey
		a.mu.Unlock()

		return a.startServices()
	}

	a.mu.Lock()
	a.profile = profile
	a.sessionKey = key
	a.mu.Unlock()

	return a.startServices()
}

// AutoCreateProfile creates a new profile with a generated unique ID and starts services
func (a *App) AutoCreateProfile(username string) error {
	if username == "" {
		username = "Cihaz"
	}
	profile, key, err := CreateProfileAuto(username)
	if err != nil {
		return err
	}

	a.mu.Lock()
	a.profile = profile
	a.sessionKey = key
	a.mu.Unlock()

	return a.startServices()
}

// GetPeerID returns the current active profile's unique peer ID
func (a *App) GetPeerID() string {
	a.mu.RLock()
	defer a.mu.RUnlock()
	if a.profile == nil {
		return ""
	}
	return a.profile.PeerID
}

// GetUsername returns the current active profile's username
func (a *App) GetUsername() string {
	a.mu.RLock()
	defer a.mu.RUnlock()
	if a.profile == nil {
		return ""
	}
	return a.profile.Username
}

// GetPublicKeyHex returns the public key as Hex string
func (a *App) GetPublicKeyHex() string {
	a.mu.RLock()
	defer a.mu.RUnlock()
	if a.profile == nil {
		return ""
	}
	return hex.EncodeToString(a.profile.PublicKey)
}

// startServices initiates the file receiver TCP server and LAN UDP broadcasting
func (a *App) startServices() error {
	a.mu.RLock()
	p := a.profile
	a.mu.RUnlock()

	if p == nil {
		return errors.New("profile not loaded")
	}

	// 1. Start TCP receiver
	tcpPort, err := a.transferManager.Start(p)
	if err != nil {
		return fmt.Errorf("failed to start transfer manager: %w", err)
	}

	// 2. Start LAN Discovery
	err = a.discoveryService.Start(p.PeerID, p.Username, p.PublicKey, tcpPort, p.Color, p.Status)
	if err != nil {
		println("Warning: failed to start LAN discovery:", err.Error())
	}

	// 3. Start WAN Client
	if a.wanService != nil {
		if p.MatchmakingServer != "" {
			a.wanService.SetServerAddress(p.MatchmakingServer)
		}
		a.wanService.Start(p.PeerID, p.Username, p.PublicKey, p.Color, p.Status)
	}

	return nil
}

// isFriend checks if a peer ID or public key is in our local friends list
func (a *App) isFriend(peerID string, pubKey []byte) bool {
	a.mu.RLock()
	defer a.mu.RUnlock()
	if a.profile == nil {
		return false
	}
	
	if a.profile.Friends == nil {
		return false
	}

	pubKeyHex := hex.EncodeToString(pubKey)
	pubKeyB64 := base64.StdEncoding.EncodeToString(pubKey)

	for _, f := range a.profile.Friends {
		fClean := strings.TrimSpace(f)
		if fClean == "" {
			continue
		}
		if strings.EqualFold(fClean, peerID) || strings.EqualFold(fClean, pubKeyHex) || fClean == pubKeyB64 {
			return true
		}
	}
	return false
}

// AddFriend adds a friend's Peer ID or Public Key Hex to the persistent profile
func (a *App) AddFriend(friendIDOrKey string) error {
	fClean := strings.TrimSpace(friendIDOrKey)
	if fClean == "" {
		return errors.New("arkadaş kimliği boş olamaz")
	}

	a.mu.Lock()
	if a.profile == nil {
		a.mu.Unlock()
		return errors.New("profil yüklenmedi")
	}

	if a.profile.Friends == nil {
		a.profile.Friends = []string{}
	}

	// Check if already a friend
	exists := false
	for _, f := range a.profile.Friends {
		if strings.EqualFold(strings.TrimSpace(f), fClean) {
			exists = true
			break
		}
	}

	if !exists {
		a.profile.Friends = append(a.profile.Friends, fClean)
	}
	p := a.profile
	key := a.sessionKey
	a.mu.Unlock()

	err := SaveProfile(p, key)
	if err != nil {
		return fmt.Errorf("arkadaş kaydedilemedi: %w", err)
	}

	a.EmitCombinedPeers()
	return nil
}

// RemoveFriend removes a friend's Peer ID or Public Key Hex from the persistent profile
func (a *App) RemoveFriend(friendIDOrKey string) error {
	fClean := strings.TrimSpace(friendIDOrKey)
	if fClean == "" {
		return errors.New("arkadaş kimliği boş olamaz")
	}

	a.mu.Lock()
	if a.profile == nil {
		a.mu.Unlock()
		return errors.New("profil yüklenmedi")
	}

	if a.profile.Friends == nil {
		a.profile.Friends = []string{}
	}

	var newList []string
	for _, f := range a.profile.Friends {
		if !strings.EqualFold(strings.TrimSpace(f), fClean) {
			newList = append(newList, f)
		}
	}

	a.profile.Friends = newList
	p := a.profile
	key := a.sessionKey
	a.mu.Unlock()

	err := SaveProfile(p, key)
	if err != nil {
		return fmt.Errorf("arkadaş silinemedi: %w", err)
	}

	a.EmitCombinedPeers()
	return nil
}

// EmitCombinedPeers broadcasts LAN + WAN discovered contacts to the frontend
func (a *App) EmitCombinedPeers() {
	if a.ctx == nil {
		return
	}
	lanPeers := a.discoveryService.GetPeers()
	var wanPeers []*Peer
	if a.wanService != nil {
		wanPeers = a.wanService.GetPeers()
	}
	combined := append(lanPeers, wanPeers...)

	a.mu.RLock()
	profile := a.profile
	a.mu.RUnlock()

	if profile == nil {
		runtime.EventsEmit(a.ctx, "peers_updated", []*Peer{})
		return
	}

	// Track online friends
	onlineFriends := make(map[string]bool)
	var friendsOnly []*Peer

	for _, p := range combined {
		if a.isFriend(p.PeerID, p.PublicKey) {
			friendsOnly = append(friendsOnly, p)
			onlineFriends[p.PeerID] = true
			if len(p.PublicKey) > 0 {
				onlineFriends[hex.EncodeToString(p.PublicKey)] = true
				onlineFriends[base64.StdEncoding.EncodeToString(p.PublicKey)] = true
			}
			// Update friend metadata persistently when they come online
			a.updateFriendMetadata(p.PeerID, p.Username, p.Color)
		}
	}

	// For any offline friends, append a virtual offline Peer card
	for _, f := range profile.Friends {
		fClean := strings.TrimSpace(f)
		if fClean == "" {
			continue
		}
		if !onlineFriends[fClean] {
			peerID := fClean
			username := "Arkadaş (" + fClean + ")"
			
			a.mu.RLock()
			if a.profile != nil && a.profile.FriendNames != nil {
				if savedName, ok := a.profile.FriendNames[fClean]; ok && savedName != "" {
					username = savedName
				}
			}
			color := "#6c757d"
			if a.profile != nil && a.profile.FriendColors != nil {
				if savedColor, ok := a.profile.FriendColors[fClean]; ok && savedColor != "" {
					color = savedColor
				}
			}
			a.mu.RUnlock()

			friendsOnly = append(friendsOnly, &Peer{
				PeerID:    peerID,
				Username:  username,
				IP:        "offline",
				TCPPort:   0,
				PublicKey: []byte{},
				Online:    false,
				Color:     color, // Use last known color
				Status:    "Çevrimdışı",
			})
		}
	}

	runtime.EventsEmit(a.ctx, "peers_updated", friendsOnly)
}

func (a *App) updateFriendMetadata(peerID string, username string, color string) {
	a.mu.Lock()
	if a.profile == nil {
		a.mu.Unlock()
		return
	}
	if a.profile.FriendNames == nil {
		a.profile.FriendNames = make(map[string]string)
	}
	if a.profile.FriendColors == nil {
		a.profile.FriendColors = make(map[string]string)
	}

	changed := false
	if a.profile.FriendNames[peerID] != username && username != "" && !strings.Contains(username, "Arkadaş (") {
		a.profile.FriendNames[peerID] = username
		changed = true
	}
	if a.profile.FriendColors[peerID] != color && color != "" && color != "#6c757d" {
		a.profile.FriendColors[peerID] = color
		changed = true
	}
	p := a.profile
	key := a.sessionKey
	a.mu.Unlock()

	if changed {
		_ = SaveProfile(p, key)
	}
}

// AcceptTransfer routes frontend's decision for an incoming file
func (a *App) AcceptTransfer(transferID string, accepted bool, saveDir string) {
	if a.wanService != nil {
		a.wanService.mu.RLock()
		_, isWAN := a.wanService.incomingTxs[transferID]
		a.wanService.mu.RUnlock()

		if isWAN {
			a.wanService.AcceptWANTransfer(transferID, accepted, saveDir)
			return
		}
	}
	a.transferManager.AcceptTransfer(transferID, accepted, saveDir)
}

// SendFileToPeer triggers peer-to-peer file transmission
func (a *App) SendFileToPeer(peerIP string, peerPort int, peerPubKeyHex string, filePath string) (string, error) {
	pubKeyBytes, err := decodePublicKey(peerPubKeyHex)
	if err != nil {
		return "", fmt.Errorf("invalid peer public key: %w", err)
	}

	return a.transferManager.SendFile(peerIP, peerPort, pubKeyBytes, filePath)
}

// GetActivePeers returns currently discovered online LAN and WAN peers
func (a *App) GetActivePeers() []*Peer {
	lanPeers := a.discoveryService.GetPeers()
	var wanPeers []*Peer
	if a.wanService != nil {
		wanPeers = a.wanService.GetPeers()
	}
	combined := append(lanPeers, wanPeers...)

	var friendsOnly []*Peer
	for _, p := range combined {
		if a.isFriend(p.PeerID, p.PublicKey) {
			friendsOnly = append(friendsOnly, p)
		}
	}
	return friendsOnly
}

// GetTransfers returns the list of active/past transfers
func (a *App) GetTransfers() []*TransferState {
	return a.transferManager.GetTransfers()
}

// GetDownloadsFolder returns user's default Downloads directory
func (a *App) GetDownloadsFolder() string {
	home, err := os.UserHomeDir()
	if err != nil {
		// Fallback to current dir
		dir, _ := os.Getwd()
		return dir
	}
	
	downloads := filepath.Join(home, "Downloads")
	// Make sure it exists, if not use home
	if _, err := os.Stat(downloads); err != nil {
		return home
	}
	return downloads
}

// SelectFile launches OS native open file selector dialog
func (a *App) SelectFile() string {
	filePath, err := runtime.OpenFileDialog(a.ctx, runtime.OpenDialogOptions{
		Title: "Select File to Send",
	})
	if err != nil {
		return ""
	}
	return filePath
}

// SelectDirectory launches OS native choose folder dialog
func (a *App) SelectDirectory() string {
	dirPath, err := runtime.OpenDirectoryDialog(a.ctx, runtime.OpenDialogOptions{
		Title: "Choose Download Folder",
	})
	if err != nil {
		return ""
	}
	return dirPath
}

// OpenReceivedFile launches file with standard OS viewer
func (a *App) OpenReceivedFile(filePath string) error {
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return fmt.Errorf("file does not exist: %s", filePath)
	}

	// Windows implementation
	cmd := exec.Command("cmd", "/c", "start", "", filePath)
	return cmd.Run()
}

// OpenFolderAndSelect opens Windows Explorer with file highlighted
func (a *App) OpenFolderAndSelect(filePath string) error {
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return fmt.Errorf("file does not exist: %s", filePath)
	}

	// Windows implementation: opens folder and highlights file
	cmd := exec.Command("explorer", "/select,", filePath)
	return cmd.Run()
}

// GetProfileColor returns the custom color of local profile
func (a *App) GetProfileColor() string {
	a.mu.RLock()
	defer a.mu.RUnlock()
	if a.profile == nil {
		return ""
	}
	return a.profile.Color
}

// GetProfileStatus returns the custom status of local profile
func (a *App) GetProfileStatus() string {
	a.mu.RLock()
	defer a.mu.RUnlock()
	if a.profile == nil {
		return ""
	}
	return a.profile.Status
}

// UpdateProfile updates username, color and status, re-encrypts local file, and broadcasts changes
func (a *App) UpdateProfile(username string, color string, status string) error {
	a.mu.Lock()
	if a.profile == nil {
		a.mu.Unlock()
		return errors.New("profile not logged in")
	}
	if username != "" {
		a.profile.Username = username
	}
	a.profile.Color = color
	a.profile.Status = status
	p := a.profile
	key := a.sessionKey
	a.mu.Unlock()

	// Persist the updated profile to the encrypted file
	err := SaveProfile(p, key)
	if err != nil {
		return fmt.Errorf("failed to save updated profile: %w", err)
	}

	// Notify discovery service to broadcast immediately
	a.discoveryService.UpdateProfile(username, color, status)

	// Restart WAN Service to re-register new identity immediately on matchmaking server
	if a.wanService != nil {
		a.wanService.Stop()
		a.wanService.Start(p.PeerID, p.Username, p.PublicKey, p.Color, p.Status)
	}
	return nil
}

// Logout stops active routines and clears profile memory
func (a *App) Logout() {
	a.discoveryService.Stop()
	a.transferManager.Stop()
	if a.wanService != nil {
		a.wanService.Stop()
	}
	
	a.mu.Lock()
	a.profile = nil
	a.sessionKey = nil
	a.mu.Unlock()
}

// GetServerAddress returns the currently configured matchmaking server address
func (a *App) GetServerAddress() string {
	a.mu.RLock()
	defer a.mu.RUnlock()
	if a.profile == nil {
		return ""
	}
	// Hide default server IP from settings UI textbox
	if a.profile.MatchmakingServer == DefaultWANServer {
		return ""
	}
	return a.profile.MatchmakingServer
}

// UpdateServerAddress saves a custom matchmaking server IP/domain and restarts the P2P internet client
func (a *App) UpdateServerAddress(addr string) error {
	a.mu.Lock()
	if a.profile == nil {
		a.mu.Unlock()
		return errors.New("profile not logged in")
	}
	a.profile.MatchmakingServer = addr
	p := a.profile
	key := a.sessionKey
	a.mu.Unlock()

	// Persist the updated profile with the new server address
	err := SaveProfile(p, key)
	if err != nil {
		return fmt.Errorf("failed to save custom server address: %w", err)
	}

	// Restart WAN Service with new server address immediately
	if a.wanService != nil {
		a.wanService.Stop()
		a.wanService.SetServerAddress(addr)
		a.wanService.Start(p.PeerID, p.Username, p.PublicKey, p.Color, p.Status)
	}
	return nil
}

// SelectMultipleFiles launches OS native open file selector allowing multiple selections
func (a *App) SelectMultipleFiles() []string {
	filePaths, err := runtime.OpenMultipleFilesDialog(a.ctx, runtime.OpenDialogOptions{
		Title: "Select Files to Send",
	})
	if err != nil {
		return nil
	}
	return filePaths
}

// SelectFolderToSend launches OS native open folder dialog to send a directory
func (a *App) SelectFolderToSend() string {
	dirPath, err := runtime.OpenDirectoryDialog(a.ctx, runtime.OpenDialogOptions{
		Title: "Select Folder to Send",
	})
	if err != nil {
		return ""
	}
	return dirPath
}

// SendFolderToPeer zips the folder and sends it as a single zip file to peer
func (a *App) SendFolderToPeer(peerIP string, peerPort int, peerPubKeyHex string, folderPath string) (string, error) {
	// Zip folder first
	zipPath, err := zipDirectory(folderPath)
	if err != nil {
		return "", fmt.Errorf("klasör sıkıştırılamadı: %w", err)
	}

	pubKeyBytes, err := decodePublicKey(peerPubKeyHex)
	if err != nil {
		os.Remove(zipPath)
		return "", fmt.Errorf("geçersiz peer public key: %w", err)
	}

	// Rename the zipped temp file to FolderName.zip so the recipient gets a clean name
	folderName := filepath.Base(folderPath)
	actualZipPath := filepath.Join(os.TempDir(), folderName+".zip")
	actualZipPath = deduplicatePath(actualZipPath)

	err = os.Rename(zipPath, actualZipPath)
	if err != nil {
		actualZipPath = zipPath // fallback
	}

	// Register in TransferManager temp files tracker for automated cleanup
	a.transferManager.tempFiles.Store(actualZipPath, true)

	return a.transferManager.SendFile(peerIP, peerPort, pubKeyBytes, actualZipPath)
}

// PauseTransfer triggers active chunk pause
func (a *App) PauseTransfer(transferID string) {
	a.transferManager.PauseTransfer(transferID)
	if a.wanService != nil {
		a.wanService.PauseTransfer(transferID)
	}
}

// ResumeTransfer resumes active chunk transfer
func (a *App) ResumeTransfer(transferID string) {
	a.transferManager.ResumeTransfer(transferID)
	if a.wanService != nil {
		a.wanService.ResumeTransfer(transferID)
	}
}

// HandleStartupFilePath sets startupFilePath and notifies UI
func (a *App) HandleStartupFilePath(filePath string) {
	a.mu.Lock()
	a.startupFilePath = filePath
	a.mu.Unlock()

	if a.ctx != nil {
		runtime.EventsEmit(a.ctx, "startup_file_received", filePath)
		runtime.WindowShow(a.ctx)
		runtime.WindowUnminimise(a.ctx)
	}
}

// ClearTransferHistory deletes memory and persistent JSON logs
func (a *App) ClearTransferHistory() {
	a.transferManager.ClearTransferHistory()
}

// SendPathToPeer dynamically handles either a single file or a zipped folder transfer
func (a *App) SendPathToPeer(peerIP string, peerPort int, peerPubKeyHex string, path string) (string, error) {
	// Standardize and decode target public key bytes
	pubKeyBytes, err := decodePublicKey(peerPubKeyHex)
	if err != nil {
		return "", fmt.Errorf("invalid peer public key format: %w", err)
	}

	// Check if this peer is a WAN peer
	if a.wanService != nil {
		a.wanService.mu.RLock()
		var targetPeer *Peer
		for _, p := range a.wanService.peers {
			if bytes.Equal(p.PublicKey, pubKeyBytes) {
				targetPeer = p
				break
			}
		}
		a.wanService.mu.RUnlock()

		if targetPeer != nil {
			// This is an internet WAN Peer! Route through WAN relay tunnel
			fi, err := os.Stat(path)
			if err != nil {
				return "", fmt.Errorf("stat error: %w", err)
			}
			actualPath := path
			if fi.IsDir() {
				zipPath, err := zipDirectory(path)
				if err != nil {
					return "", fmt.Errorf("folder compression error: %w", err)
				}
				folderName := filepath.Base(path)
				actualZipPath := filepath.Join(os.TempDir(), folderName+".zip")
				actualZipPath = deduplicatePath(actualZipPath)
				_ = os.Rename(zipPath, actualZipPath)
				actualPath = actualZipPath
				a.transferManager.tempFiles.Store(actualZipPath, true)
			}
			return a.wanService.SendWANFile(targetPeer.PeerID, actualPath)
		}
	}

	fi, err := os.Stat(path)
	if err != nil {
		return "", fmt.Errorf("stat error: %w", err)
	}

	if fi.IsDir() {
		return a.SendFolderToPeer(peerIP, peerPort, peerPubKeyHex, path)
	}

	return a.transferManager.SendFile(peerIP, peerPort, pubKeyBytes, path)
}

// SendTextMessage sends an encrypted text message to a peer (either LAN or WAN).
func (a *App) SendTextMessage(peerIP string, peerPort int, peerPubKeyHex string, message string) (string, error) {
	if message == "" {
		return "", errors.New("mesaj boş olamaz")
	}

	// 1. Encode message inside base64 so it can safely be part of the filename
	encodedMsg := base64.StdEncoding.EncodeToString([]byte(message))
	virtualFilename := fmt.Sprintf("[TextBase64]%s.txt", encodedMsg)

	// 2. Create the temporary file under this virtual filename inside the system temp directory
	tempPath := filepath.Join(os.TempDir(), virtualFilename)
	
	// Make sure to clean up any existing file
	_ = os.Remove(tempPath)

	err := os.WriteFile(tempPath, []byte(message), 0666)
	if err != nil {
		return "", fmt.Errorf("geçici mesaj dosyası oluşturulamadı: %w", err)
	}

	// Register in TransferManager temp files tracker for automated cleanup after completion
	a.transferManager.tempFiles.Store(tempPath, true)

	// 3. Trigger the standard encrypted transfer pipeline!
	txID, err := a.SendPathToPeer(peerIP, peerPort, peerPubKeyHex, tempPath)
	if err != nil {
		_ = os.Remove(tempPath)
		return "", err
	}

	return txID, nil
}

// GetStartupFilePath returns any file path passed via command line argument on launch
func (a *App) GetStartupFilePath() string {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.startupFilePath
}

// ClearStartupFilePath clears the stored command-line argument file path
func (a *App) ClearStartupFilePath() {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.startupFilePath = ""
}

// decodePublicKey standardizes public key parsing by handling both Hex and Base64 formats from Wails JSON.
func decodePublicKey(encoded string) ([]byte, error) {
	// Try hex decoding first
	bytesDec, err := hex.DecodeString(encoded)
	if err == nil && len(bytesDec) > 0 {
		return bytesDec, nil
	}
	// Try base64 decoding
	bytesDec, err = base64.StdEncoding.DecodeString(encoded)
	if err == nil {
		return bytesDec, nil
	}
	return nil, fmt.Errorf("unable to decode public key: %s", encoded)
}
