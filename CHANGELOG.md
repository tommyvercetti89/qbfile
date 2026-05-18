# Changelog

All notable changes to the **QBFile** project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

---

## [1.0.0] - 2026-05-18

This is the initial production release of **QBFile** - the ultimate secure P2P file sharing and E2EE instant messaging desktop client.

### Added
* **End-to-End Encryption (E2EE)**: Direct chat messages and chunked file/folder streams are encrypted using Elliptic-Curve Diffie-Hellman (**ECDH - Curve25519**) key exchange and **256-bit AES-GCM**.
* **Cryptographic Friend Management System**: Added zero-knowledge privacy filtering. Users must explicitly add peers using their unique 22-character Peer ID to communicate.
* **Persistent Encrypted Profile Vault**: User parameters, customization colors, and friends list are stored locally using highly secure AES-256-GCM.
* **High-Fidelity Glassmorphism UI**: Beautiful frosted-glass UI styling with CSS micro-animations and custom neon highlight themes.
* **Multi-Language Support**: Added bilingual support (Turkish & English) with instant dynamic runtime translation switching.
* **Linux Matchmaking & Relay Server**: Hosted in a dedicated `SUNUCU` directory containing step-by-step guides and ready-to-run Linux AMD64 and ARM64 pre-compiled binaries.

### Fixed
* **Sprintf Peer ID Mismatch**: Resolved a Go formatting mismatch in `GeneratePeerID` where byte slices caused a format parse error, and replaced it with a flawless tekil-byte UUID formatting scheme.
* **Active Peer ID Auto-Repair**: Implemented on-the-fly automated profile checking and recovery routines to repair previously corrupted Peer ID formats on application startup.
