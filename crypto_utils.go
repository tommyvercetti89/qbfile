package main

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/ecdh"
	"crypto/rand"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/crypto/pbkdf2"
)

const (
	ProfileFileName = "qbfile_profile.enc"
	SaltSize        = 16
	PBKDF2Iterations = 100000
	KeySize         = 32 // AES-256
	AppSecret       = "qbfile_offline_local_key_exchange_salt_2026"
)

// GetStoragePath determines the best writable storage directory.
// 1. First, it tries the directory where the executable (qbfile.exe) is located.
//    This allows the application to be fully portable and keep all data inside its installed folder.
// 2. If the executable directory is not writable (e.g. C:\Program Files which has OS protections),
//    it automatically falls back to '%APPDATA%\qbfile\', ensuring standard Windows non-admin write compatibility.
func GetStoragePath(filename string) string {
	exePath, err := os.Executable()
	if err == nil {
		exeDir := filepath.Dir(exePath)
		testFile := filepath.Join(exeDir, ".write_test")
		
		// Test write access
		err := os.WriteFile(testFile, []byte{0}, 0600)
		if err == nil {
			_ = os.Remove(testFile)
			return filepath.Join(exeDir, filename)
		}
	}
	
	// Fallback to offline local APPDATA folder
	appData := os.Getenv("APPDATA")
	if appData == "" {
		return filename
	}
	
	targetFolder := filepath.Join(appData, "qbfile")
	_ = os.MkdirAll(targetFolder, 0755)
	return filepath.Join(targetFolder, filename)
}

// Profile represents the user's secure account metadata
type Profile struct {
	PeerID            string   `json:"peer_id"`            // Unique Cryptographic ID: QB-XXXX-XXXX-XXXX-XXXX
	Username          string   `json:"username"`           // Optional display name chosen by the user
	PrivateKey        []byte   `json:"private_key"`        // ECDH raw private key bytes
	PublicKey         []byte   `json:"public_key"`         // ECDH raw public key bytes
	Color             string   `json:"color"`              // Custom profile avatar color
	Status            string   `json:"status"`             // Custom profile status message
	MatchmakingServer string   `json:"matchmaking_server"` // Custom matchmaking server address configured by user
	Friends           []string          `json:"friends"`            // Persistent list of added friends (Peer IDs or public key strings)
	FriendNames       map[string]string `json:"friend_names"`       // Map of PeerID -> Last known username
	FriendColors      map[string]string `json:"friend_colors"`      // Map of PeerID -> Last known color
}

// GeneratePeerID generates a mathematically guaranteed globally unique 128-bit Peer ID
func GeneratePeerID() (string, error) {
	uuid := make([]byte, 16)
	_, err := rand.Read(uuid)
	if err != nil {
		return "", err
	}
	// UUID v4 format specifications
	uuid[6] = (uuid[6] & 0x0f) | 0x40
	uuid[8] = (uuid[8] & 0x3f) | 0x80
	
	return fmt.Sprintf("QB-%02X%02X-%02X%02X-%02X%02X-%02X%02X",
		uuid[0], uuid[1], uuid[2], uuid[3], uuid[4], uuid[5], uuid[6], uuid[7]), nil
}

// DeriveKeyAuto derives a secure AES key using the hardcoded AppSecret
func DeriveKeyAuto(salt []byte) []byte {
	return pbkdf2.Key([]byte(AppSecret), salt, PBKDF2Iterations, KeySize, sha256.New)
}

// DeriveKey derives a secure 256-bit AES key from a password and salt using PBKDF2
func DeriveKey(password string, salt []byte) []byte {
	return pbkdf2.Key([]byte(password), salt, PBKDF2Iterations, KeySize, sha256.New)
}

// EncryptGCM encrypts plaintext using AES-256-GCM with a derived key
func EncryptGCM(plaintext []byte, key []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}

	ciphertext := gcm.Seal(nil, nonce, plaintext, nil)
	
	// Prepend nonce to ciphertext
	result := append(nonce, ciphertext...)
	return result, nil
}

// DecryptGCM decrypts ciphertext using AES-256-GCM with a derived key
func DecryptGCM(encryptedData []byte, key []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	nonceSize := gcm.NonceSize()
	if len(encryptedData) < nonceSize {
		return nil, errors.New("ciphertext too short")
	}

	nonce := encryptedData[:nonceSize]
	ciphertext := encryptedData[nonceSize:]

	return gcm.Open(nil, nonce, ciphertext, nil)
}

// GenerateKeyPair generates a new Elliptic Curve Diffie-Hellman P-256 key pair
func GenerateKeyPair() (*ecdh.PrivateKey, error) {
	return ecdh.P256().GenerateKey(rand.Reader)
}

// CheckProfileExists checks if the local secure profile exists
func CheckProfileExists() bool {
	_, err := os.Stat(GetStoragePath(ProfileFileName))
	return !os.IsNotExist(err)
}

// CreateProfile creates and encrypts a new user profile with their password
func CreateProfile(username string, password string) (*Profile, []byte, error) {
	privKey, err := GenerateKeyPair()
	if err != nil {
		return nil, nil, err
	}

	pubKey := privKey.PublicKey()

	profile := &Profile{
		Username:   username,
		PrivateKey: privKey.Bytes(),
		PublicKey:  pubKey.Bytes(),
		Color:      "#00a884",
		Status:     "Dosya almaya hazır",
		Friends:           []string{},
		FriendNames:       make(map[string]string),
		FriendColors:      make(map[string]string),
	}

	profileJSON, err := json.Marshal(profile)
	if err != nil {
		return nil, nil, err
	}

	// Generate salt
	salt := make([]byte, SaltSize)
	if _, err := io.ReadFull(rand.Reader, salt); err != nil {
		return nil, nil, err
	}

	// Derive AES key
	key := DeriveKey(password, salt)

	// Encrypt Profile
	encryptedProfile, err := EncryptGCM(profileJSON, key)
	if err != nil {
		return nil, nil, err
	}

	// Write salt + encrypted data to profile file
	fileData := append(salt, encryptedProfile...)
	err = os.WriteFile(GetStoragePath(ProfileFileName), fileData, 0600)
	if err != nil {
		return nil, nil, err
	}

	return profile, key, nil
}

// UnlockProfile decrypts and loads the user profile using their password
func UnlockProfile(password string) (*Profile, []byte, error) {
	fileData, err := os.ReadFile(GetStoragePath(ProfileFileName))
	if err != nil {
		return nil, nil, err
	}

	if len(fileData) < SaltSize {
		return nil, nil, errors.New("profile file is corrupted")
	}

	salt := fileData[:SaltSize]
	encryptedProfile := fileData[SaltSize:]

	// Derive key
	key := DeriveKey(password, salt)

	// Decrypt
	profileJSON, err := DecryptGCM(encryptedProfile, key)
	if err != nil {
		return nil, nil, errors.New("incorrect password or corrupted profile")
	}

	var profile Profile
	err = json.Unmarshal(profileJSON, &profile)
	if err != nil {
		return nil, nil, err
	}

	// Fallback for older profiles
	if profile.Color == "" {
		profile.Color = "#00a884"
	}
	if profile.Status == "" {
		profile.Status = "Dosya almaya hazır"
	}
	if profile.Friends == nil {
		profile.Friends = []string{}
	}
	if profile.FriendNames == nil {
		profile.FriendNames = make(map[string]string)
	}
	if profile.FriendColors == nil {
		profile.FriendColors = make(map[string]string)
	}
	// Auto repair Peer IDs containing formatting leftovers
	if profile.PeerID == "" || strings.Contains(profile.PeerID, "%!X") || strings.Contains(profile.PeerID, "MISSING") {
		pID, errGen := GeneratePeerID()
		if errGen == nil {
			profile.PeerID = pID
			_ = SaveProfile(&profile, key)
		}
	}

	return &profile, key, nil
}

// SaveProfile re-encrypts and saves the profile using the session key
func SaveProfile(profile *Profile, key []byte) error {
	profileJSON, err := json.Marshal(profile)
	if err != nil {
		return err
	}

	// Read the original salt from the existing profile file to keep PBKDF2 key stable
	salt := make([]byte, SaltSize)
	originalData, err := os.ReadFile(GetStoragePath(ProfileFileName))
	if err == nil && len(originalData) >= SaltSize {
		copy(salt, originalData[:SaltSize])
	} else {
		// Fallback to generating a fresh salt if the file doesn't exist
		if _, err := io.ReadFull(rand.Reader, salt); err != nil {
			return err
		}
	}

	encryptedProfile, err := EncryptGCM(profileJSON, key)
	if err != nil {
		return err
	}

	fileData := append(salt, encryptedProfile...)
	return os.WriteFile(GetStoragePath(ProfileFileName), fileData, 0600)
}

// CreateProfileAuto automatically creates a profile with a unique PeerID and no user-entered password
func CreateProfileAuto(username string) (*Profile, []byte, error) {
	privKey, err := GenerateKeyPair()
	if err != nil {
		return nil, nil, err
	}

	pubKey := privKey.PublicKey()

	peerID, err := GeneratePeerID()
	if err != nil {
		return nil, nil, err
	}

	profile := &Profile{
		PeerID:     peerID,
		Username:   username,
		PrivateKey: privKey.Bytes(),
		PublicKey:  pubKey.Bytes(),
		Color:      "#00a884",
		Status:     "Dosya almaya hazır",
		Friends:           []string{},
		FriendNames:       make(map[string]string),
		FriendColors:      make(map[string]string),
	}

	profileJSON, err := json.Marshal(profile)
	if err != nil {
		return nil, nil, err
	}

	salt := make([]byte, SaltSize)
	if _, err := io.ReadFull(rand.Reader, salt); err != nil {
		return nil, nil, err
	}

	// Derive key automatically using AppSecret
	key := DeriveKeyAuto(salt)

	encryptedProfile, err := EncryptGCM(profileJSON, key)
	if err != nil {
		return nil, nil, err
	}

	fileData := append(salt, encryptedProfile...)
	err = os.WriteFile(GetStoragePath(ProfileFileName), fileData, 0600)
	if err != nil {
		return nil, nil, err
	}

	return profile, key, nil
}

// UnlockProfileAuto automatically loads the encrypted profile using the AppSecret
func UnlockProfileAuto() (*Profile, []byte, error) {
	fileData, err := os.ReadFile(GetStoragePath(ProfileFileName))
	if err != nil {
		return nil, nil, err
	}

	if len(fileData) < SaltSize {
		return nil, nil, errors.New("profile file is corrupted")
	}

	salt := fileData[:SaltSize]
	encryptedProfile := fileData[SaltSize:]

	// Derive key automatically using AppSecret
	key := DeriveKeyAuto(salt)

	profileJSON, err := DecryptGCM(encryptedProfile, key)
	if err != nil {
		return nil, nil, err
	}

	var profile Profile
	err = json.Unmarshal(profileJSON, &profile)
	if err != nil {
		return nil, nil, err
	}

	// Ensure PeerID exists (fallback for older test profiles) and auto-repair broken IDs
	if profile.PeerID == "" || strings.Contains(profile.PeerID, "%!X") || strings.Contains(profile.PeerID, "MISSING") {
		pID, errGen := GeneratePeerID()
		if errGen == nil {
			profile.PeerID = pID
			_ = SaveProfile(&profile, key)
		}
	}
	if profile.Color == "" {
		profile.Color = "#00a884"
	}
	if profile.Status == "" {
		profile.Status = "Dosya almaya hazır"
	}
	if profile.Friends == nil {
		profile.Friends = []string{}
	}
	if profile.FriendNames == nil {
		profile.FriendNames = make(map[string]string)
	}
	if profile.FriendColors == nil {
		profile.FriendColors = make(map[string]string)
	}

	return &profile, key, nil
}
