# 🛡️ QBFile - End-to-End Encrypted (E2EE) P2P File Transfer & Chat System

🇹🇷 [Türkçe Dökümantasyon](README.tr.md)

[![License](https://img.shields.io/badge/License-Apache_2.0-blue.svg)](https://opensource.org/licenses/Apache-2.0)
[![Go Version](https://img.shields.io/badge/Go-1.21+-00ADD8.svg?style=flat&logo=go)](https://golang.org)
[![Svelte](https://img.shields.io/badge/Svelte-4.0+-FF3E00.svg?style=flat&logo=svelte)](https://svelte.dev)
[![Wails](https://img.shields.io/badge/Wails-2.0+-red.svg)](https://wails.io)

QBFile is a high-performance, **End-to-End Encrypted (E2EE) peer-to-peer (P2P) file transfer and instant messaging** utility built with a zero-knowledge privacy-first architecture, functioning seamlessly across local area networks (LAN) and the global internet (WAN).

---

## 💎 Core Features

### 🔒 1. High-Grade Security & Cryptography
* **End-to-End Encryption (E2EE)**: All files, directories, and chat messages are encrypted directly between devices using Elliptic-Curve Diffie-Hellman (**ECDH - Curve25519**) key exchange and **256-bit AES-GCM** symmetric encryption.
* **Encrypted Local Storage**: User profile parameters, status messages, and friend databases are stored securely on disk using **AES-256-GCM**. No unauthorized local users can access your profile database.
* **Zero-Knowledge Signaling**: The lightweight matchmaking/signaling server **never sees, logs, or intercepts** any transmitted data or E2EE payloads; it only acts as a secure network handshake coordinator.

### 👥 2. Cryptographic Friend System (Privacy Shield)
* **Strangers Blocked by Default**: Even when connected to the public matchmaking network, **peers will never discover or view each other unless they explicitly add one another** via their 22-character unique Peer ID.
* **Keşif Filtresi (Discovery Filtering)**: Network packets are securely filtered in the backend. Only authenticated peers mapped to your active friend keys will trigger a UI presence.
* **Offline Vault Memory**: Added friends remain visible as "Offline" with elegant frosted-glass styling even when disconnected, allowing you to access historic chats at any time.

### 📂 3. High-Speed P2P File & Folder Transfer
* **Large File Optimizations**: Stream massive files smoothly via structured, memory-friendly chunked transfers over secure raw TCP sockets.
* **Automatic Directory Zipping**: Folders are zipped on-the-fly, transferred securely, and automatically extracted at the recipient's destination maintaining structure.
* **Granular Flow Control**: Pause, resume, or cancel active file transfers instantaneously with micro-second responsiveness.

### 🎨 4. Premium Aesthetic & UI/UX Experience
* **Glassmorphism Interface**: Sleek dark mode featuring gorgeous frosted-glass panels (`backdrop-filter: blur`), glowing neon indicators, and fluid CSS micro-animations.
* **Accent Colors**: Custom color themes including Cyber Green, Cobalt Blue, Neon Purple, Rich Gold, and Sakura Pink.
* **Localization**: Fully localized dynamic language switcher offering English and Turkish (TR) experiences out of the box.

---

## 🏗️ Architecture

```mermaid
graph TD
    subgraph Device A (Client)
        A1[User Interface - Svelte] <--> A2[Go Backend Core]
        A2 <--> A3[(AES-256 Encrypted Profile)]
    end
    
    subgraph Global WAN / LAN
        S[Matchmaking & Relay Server]
    end
    
    subgraph Device B (Client)
        B1[User Interface - Svelte] <--> B2[Go Backend Core]
        B2 <--> B3[(AES-256 Encrypted Profile)]
    end
    
    A2 <-->|1. Handshake & ECDH negotiation| S
    B2 <-->|1. Handshake & ECDH negotiation| S
    A2 ====|2. Direct E2EE AES-GCM P2P Stream| B2
```

---

## 🛠️ Development & Compilation Guide

### Prerequisites
* **Go** (v1.21 or higher)
* **Node.js** (v18 or higher)
* **Wails CLI** (Install via: `go install github.com/wailsapp/wails/v2/cmd/wails@latest`)

### 1. Live Development Mode
Run the application in hot-reload live development mode:
```bash
cd qbfile
wails dev
```

### 2. Building Production Binary
To build a highly optimized, single portable Windows executable (`.exe`):
```bash
wails build
```

---

## 🛡️ Secure Deployment & Remote Server Injections

When publicizing your project on GitHub, you can **protect your signaling server's IP address** while keeping the client binary fully pre-configured for friends by compiling with Linker flags:

Simply execute this build command inside your local environment before distribution:
```bash
wails build -ldflags "-X main.DefaultWANServer=YOUR_SERVER_IP:12130"
```
This compilation dynamically binds your private signaling endpoint directly to the built `.exe` without altering the generic localhost configurations in the Git history!

---

## 📜 License

This project is licensed under the **Apache License 2.0**. For more details, see the [LICENSE](LICENSE) file.

---

## 🤝 Credits & Acknowledgements

* **Developer / Owner**: [tommyvercetti89](https://github.com/tommyvercetti89)
* **AI Architecture & Collaborator**: This project was developed in close pair-programming collaboration with **Antigravity** (Google Deepmind's Advanced Agentic Coding AI).
