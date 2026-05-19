<script>
  import { onMount } from 'svelte';
  import { 
    CheckProfileExists, 
    RegisterProfile, 
    LoginProfile, 
    AutoLoadProfile,
    AutoCreateProfile,
    GetPeerID,
    GetUsername,
    GetPublicKeyHex,
    AcceptTransfer, 
    SendFileToPeer, 
    GetDownloadsFolder,
    SelectFile, 
    SelectDirectory, 
    OpenReceivedFile, 
    OpenFolderAndSelect,
    Logout,
    GetProfileColor,
    GetProfileStatus,
    UpdateProfile,
    SelectMultipleFiles,
    SelectFolderToSend,
    SendPathToPeer,
    GetStartupFilePath,
    ClearStartupFilePath,
    PauseTransfer,
    ResumeTransfer,
    CancelTransfer,
    ClearTransferHistory,
    GetServerAddress,
    UpdateServerAddress,
    SendTextMessage,
    AddFriend,
    RemoveFriend
  } from '../wailsjs/go/main/App.js';
  
  import { EventsOn } from '../wailsjs/runtime/runtime.js';

  // Authentication State
  let isProfileExists = false;
  let isLoggedIn = false;
  let username = "";
  let peerID = "";
  let copiedMessage = "";
  let publicKeyHex = "";
  let password = "";
  
  // Registration Inputs
  let regUsername = "";
  let regPassword = "";
  let regConfirmPassword = "";
  
  // UI States
  let authLoading = false;
  let errorMsg = "";
  let successMsg = "";
  let showPassword = false;
  let showRegPassword = false;
  
  // Dashboard & Transfer States
  let peers = [];
  let transfers = [];
  let searchQuery = "";
  let selectedPeer = null;
  let viewMode = 'chat'; // 'chat' or 'transfers'
  let friendRequests = [];
  
  // Incoming Request Modal State
  let incomingRequest = null;
  let targetSaveDir = "";

  // Profile Customization & Language Settings
  let profileColor = "#00a884";
  let profileStatus = "Çevrimiçi";
  let showColorPicker = false;
  let availableColors = ["#00a884", "#4285f4", "#9060eb", "#ea4335", "#fbbc05", "#ff6b00", "#d33682", "#2aa198", "#202c33"];
  
  // Settings & Localization State
  let showSettings = false;
  let currentLang = "tr";
  let tempUsername = "";
  let tempStatus = "";
  let tempColor = "";
  let matchmakingServer = "";
  let tempMatchmakingServer = "";

  const translations = {
    tr: {
      settingsTitle: "Uygulama Ayarları",
      settingsInfo: "QBFile tamamen yerel çalışan uçtan uca şifreli bir paylaşım aracıdır. Herhangi bir bulut yedeklemesi veya merkezi veri depolaması bulunmamaktadır. Profiliniz ve transfer geçmişiniz sadece bu cihazda şifreli olarak saklanır. Cihazınızı değiştirdiğinizde veya uygulamayı kaldırdığınızda verileriniz ve benzersiz kimliğiniz geri alınamaz şekilde kaybolur.",
      usernameLabel: "Kullanıcı Adı / Cihaz İsmi",
      colorLabel: "Profil Teması / Rengi",
      statusLabel: "Durum Mesajı",
      saveBtn: "Ayarları Kaydet",
      langLabel: "Uygulama Dili",
      infoTitle: "Önemli Kriptografik Bilgilendirme",
      ipAddress: "IP Adresi",
      e2eeActive: "AES-256 E2EE Korumalı",
      noBackupWarn: "⚠️ Bulut Yedeklemesi Yoktur!",
      copiedMsg: "Kimlik panoya kopyalandı! 📋",
      searchPlaceholder: "Kullanıcı veya IP ara...",
      searchingPeers: "LAN üzerinde başka kullanıcı aranıyor...",
      searchingSub: "Diğer bilgisayarlarda da QBFile uygulamasının açık olduğundan emin olun.",
      emptyHistory: "Henüz dosya paylaşımı yapılmadı.",
      emptyHistorySub: "Aşağıdaki butonları veya sürükle-bırak alanını kullanarak ilk dosyanızı gönderin.",
      clearHistory: "Geçmişi Temizle",
      historyCleared: "Tüm transfer geçmişini kalıcı olarak temizlemek istiyor musunuz?",
      dragOverlayTitle: "Dosyaları Paylaşmak İçin Bırakın",
      dragOverlaySub: "Klasörler otomatik olarak zipleşir ve tüm aktarım uçtan uca şifrelenir.",
      receivedTitle: "Gelen Dosya İsteği",
      receivedSub: "adlı kullanıcı size bir dosya göndermek istiyor.",
      acceptBtn: "Kabul Et ve İndir",
      declineBtn: "Reddet",
      changeSaveDir: "Kayıt Klasörü:",
      selectFolderTitle: "Klasör Seç",
      logoutTitle: "Profili Kilitle ve Çık",
      settingsBtn: "Uygulama Ayarları",
      statusReady: "Dosya almaya hazır",
      cancelBtn: "İptal Et",
      usernameEmpty: "Kullanıcı adı boş olamaz!",
      offlineText: "Çevrimdışı",
      onlineText: "Çevrimiçi",
      serverLabel: "Çöpçatan Sunucu Adresi (WAN)",
      splashTitle: "Uçtan Uca Şifreli P2P Transfer",
      splashDesc: "Ağınızdaki diğer cihazları otomatik olarak keşfedin ve dosya boyutu sınırı olmaksızın, tamamen yerel bellek üzerinden doğrudan cihazlar arası aktarım gerçekleştirin.",
      splashFeature1Title: "🔑 Yerel Kimlik",
      splashFeature1Desc: "Anahtar çiftiniz ve Peer ID'niz sadece bu cihazda saklanır.",
      splashFeature2Title: "⚡ Sıfır Bulut",
      splashFeature2Desc: "Dosyalarınız hiçbir bulut sunucusuna yüklenmez, doğrudan aktarılır.",
      splashSecurityBadge: "AES-256 GCM Şifreleme Koruması Aktif",
      clearHistoryBtn: "🗑️ Geçmişi Temizle",
      e2eWarning: "Bu sohbetteki tüm dosya aktarımları uçtan uca şifrelenmiştir. Aracı sunucular veya bulut depoları yoktur.",
      statusTransferring: "Aktarılıyor...",
      statusPaused: "Duraklatıldı",
      statusPending: "Karşı tarafın onayı bekleniyor...",
      statusCompleted: "Tamamlandı",
      statusDeclined: "Aktarım Reddedildi",
      statusFailed: "Aktarım Başarısız",
      actionOpen: "Aç",
      actionShow: "📁 Göster",
      btnSendFiles: "Dosya(lar) Gönder",
      btnSendFolder: "Klasör Gönder",
      offlineNotice: "Kullanıcı şu anda çevrimdışı. Dosya gönderilemez.",
      incomingTitle: "Güvenli Dosya Talebi",
      incomingDesc: "size bir dosya göndermek istiyor:",
      saveFolder: "Kaydedilecek Klasör",
      btnChange: "Değiştir",
      activeE2EE: "AES-256 E2EE",
      actionPause: "⏸️ Duraklat",
      actionResume: "▶️ Devam Et",
      rightClickReady: "Sağ tık ile seçilen öge gönderilmeye hazır:",
      btnSendNow: "Şimdi Gönder",
      myConnections: "Bağlantılarım",
      secureChannel: "Uçtan Uca Güvenli",
      deleteFriend: "Arkadaşı Sil",
      deleteFriendTitle: "Arkadaşı Listemden Sil",
      addFriend: "Arkadaş Ekle",
      sentItems: "Gönderilenler",
      friendRequests: "Gelen Arkadaşlık İstekleri",
      acceptBtnShort: "Kabul Et",
      rejectBtnShort: "Reddet",
      noTransfersYet: "Henüz bu kişiyle transfer bulunmuyor.",
      allTransfersDashboard: "Transfer Geçmişi",
      transferHistorySub: "Bu arkadaşla gönderilen ve alınan dosyalar",
      actionCancel: "❌ İptal Et"
    },
    en: {
      settingsTitle: "Application Settings",
      settingsInfo: "QBFile is a fully local, end-to-end encrypted sharing tool. There is absolutely no cloud backup or centralized data storage. Your profile and transfer history are encrypted and stored solely on this device. If you change your device or uninstall the application, your data and unique identity will be lost permanently.",
      usernameLabel: "Username / Device Nickname",
      colorLabel: "Profile Accent Color",
      statusLabel: "Status Message",
      saveBtn: "Save Settings",
      langLabel: "App Language",
      infoTitle: "Important Cryptographic Notice",
      ipAddress: "IP Address",
      e2eeActive: "AES-256 E2EE Protected",
      noBackupWarn: "⚠️ No Cloud Backups!",
      copiedMsg: "ID copied to clipboard! 📋",
      searchPlaceholder: "Search user or IP...",
      searchingPeers: "Scanning LAN for other peers...",
      searchingSub: "Make sure QBFile is open on the other computers.",
      emptyHistory: "No files shared yet.",
      emptyHistorySub: "Use the buttons below or the drag-and-drop area to send your first file.",
      clearHistory: "Clear History",
      historyCleared: "Do you want to permanently clear the entire transfer history?",
      dragOverlayTitle: "Drop Files to Share",
      dragOverlaySub: "Folders zip automatically and all transfers are fully encrypted.",
      receivedTitle: "Incoming File Request",
      receivedSub: "wants to send you a file.",
      acceptBtn: "Accept & Download",
      declineBtn: "Decline",
      changeSaveDir: "Save Folder:",
      selectFolderTitle: "Select Folder",
      logoutTitle: "Lock Profile & Exit",
      settingsBtn: "Application Settings",
      statusReady: "Ready to receive files",
      cancelBtn: "Cancel",
      usernameEmpty: "Username cannot be empty!",
      offlineText: "Offline",
      onlineText: "Online",
      serverLabel: "Matchmaking Server Address (WAN)",
      splashTitle: "End-to-End Encrypted P2P Transfer",
      splashDesc: "Automatically discover other devices on your network and perform direct device-to-device transfers without file size limits, completely via local memory.",
      splashFeature1Title: "🔑 Local Identity",
      splashFeature1Desc: "Your key pair and Peer ID are stored solely on this device.",
      splashFeature2Title: "⚡ Zero Cloud",
      splashFeature2Desc: "Your files are never uploaded to any cloud server, they are transferred directly.",
      splashSecurityBadge: "AES-256 GCM Encryption Protection Active",
      clearHistoryBtn: "🗑️ Clear History",
      e2eWarning: "All file transfers in this chat are end-to-end encrypted. There are no intermediate servers or cloud storage.",
      statusTransferring: "Transferring...",
      statusPaused: "Paused",
      statusPending: "Waiting for peer approval...",
      statusCompleted: "Completed",
      statusDeclined: "Transfer Declined",
      statusFailed: "Transfer Failed",
      actionOpen: "Open",
      actionShow: "📁 Show",
      btnSendFiles: "Send File(s)",
      btnSendFolder: "Send Folder",
      offlineNotice: "User is currently offline. Files cannot be sent.",
      incomingTitle: "Secure File Request",
      incomingDesc: "wants to send you a file:",
      saveFolder: "Folder to Save",
      btnChange: "Change",
      activeE2EE: "AES-256 E2EE",
      actionPause: "⏸️ Pause",
      actionResume: "▶️ Resume",
      rightClickReady: "Right-click selected item is ready to send:",
      btnSendNow: "Send Now",
      myConnections: "My Connections",
      secureChannel: "End-to-End Secure",
      deleteFriend: "Delete Friend",
      deleteFriendTitle: "Remove Friend from My List",
      addFriend: "Add Friend",
      sentItems: "Sent Items",
      friendRequests: "Incoming Friend Requests",
      acceptBtnShort: "Accept",
      rejectBtnShort: "Decline",
      noTransfersYet: "No transfers with this contact yet.",
      allTransfersDashboard: "Transfer History",
      transferHistorySub: "Files exchanged with this friend",
      actionCancel: "❌ Cancel"
    }
  };

  $: t = translations[currentLang];

  // Drag & Drop State
  let isDragging = false;
  let startupFilePath = "";

  function handleSendStartupPath() {
    if (!selectedPeer || !startupFilePath) return;
    const path = startupFilePath;
    startupFilePath = "";
    ClearStartupFilePath();
    
    SendPathToPeer(selectedPeer.ip, selectedPeer.tcp_port, selectedPeer.public_key, path)
      .then(() => {
        // Success!
      })
      .catch(err => {
        alert("Gönderim başarısız: " + err);
      });
  }

  // Audio elements or simple sound synth for alerts
  function playNotificationSound(type) {
    try {
      const ctx = new (window.AudioContext || window.webkitAudioContext)();
      const osc = ctx.createOscillator();
      const gain = ctx.createGain();
      osc.connect(gain);
      gain.connect(ctx.destination);
      
      if (type === 'success') {
        // High double beep
        osc.frequency.setValueAtTime(523.25, ctx.currentTime); // C5
        gain.gain.setValueAtTime(0.1, ctx.currentTime);
        osc.start();
        osc.stop(ctx.currentTime + 0.1);
        
        setTimeout(() => {
          const osc2 = ctx.createOscillator();
          const gain2 = ctx.createGain();
          osc2.connect(gain2);
          gain2.connect(ctx.destination);
          osc2.frequency.setValueAtTime(659.25, ctx.currentTime); // E5
          gain2.gain.setValueAtTime(0.1, ctx.currentTime);
          osc2.start();
          osc2.stop(ctx.currentTime + 0.15);
        }, 120);
      } else if (type === 'request') {
        // WhatsApp ping style
        osc.type = 'sine';
        osc.frequency.setValueAtTime(587.33, ctx.currentTime); // D5
        gain.gain.setValueAtTime(0.15, ctx.currentTime);
        osc.start();
        osc.stop(ctx.currentTime + 0.2);
      }
    } catch (e) {
      console.log("Audio not supported or blocked", e);
    }
  }

  // Reactively filter peers based on search input
  $: filteredPeers = peers.filter(p => 
    p.username.toLowerCase().includes(searchQuery.toLowerCase()) ||
    p.ip.includes(searchQuery)
  );

  // Decodes base64 safely supporting UTF-8 Turkish characters
  function decodeBase64Utf8(str) {
    try {
      return decodeURIComponent(escape(atob(str)));
    } catch (e) {
      try {
        return atob(str);
      } catch (err) {
        return str;
      }
    }
  }

  // Reactively get transfers matching selected peer
  $: filteredTransfers = transfers
    .filter(t => {
      if (!selectedPeer) return false;
      return t.peer_name === selectedPeer.username || t.peer_name === selectedPeer.peer_id || t.peer_name === "Connecting...";
    })
    .sort((a, b) => a.id.localeCompare(b.id));

  onMount(() => {
    // Initial check
    checkExistingProfile();

    // Check if launched with a context menu path
    GetStartupFilePath().then(path => {
      if (path) {
        startupFilePath = path;
      }
    });

    EventsOn("startup_file_received", (path) => {
      if (path) {
        startupFilePath = path;
      }
    });

    // Live active peers from discovery
    EventsOn("peers_updated", (peerList) => {
      peers = peerList || [];
      if (selectedPeer) {
        const found = peers.find(p => p.username === selectedPeer.username);
        if (found) {
          selectedPeer = found;
        } else {
          selectedPeer = { ...selectedPeer, online: false };
        }
      }
    });

    // Live active transfers
    EventsOn("transfers_updated", (transferList) => {
      const oldTransfers = [...transfers];
      transfers = transferList || [];

      // Check if a transfer just completed to trigger a notification sound
      transfers.forEach(t => {
        const old = oldTransfers.find(o => o.id === t.id);
        if (t.status === 'completed' && (!old || old.status !== 'completed')) {
          playNotificationSound('success');
        }
      });
    });

    // Incoming file request trigger
    EventsOn("incoming_request", (req) => {
      incomingRequest = req;
      playNotificationSound('request');
      GetDownloadsFolder().then(folder => {
        targetSaveDir = folder;
      });
    });

    EventsOn("incoming_friend_request", (req) => {
      if (req && req.peer_id) {
        if (!friendRequests.some(r => r.peer_id === req.peer_id)) {
          friendRequests = [...friendRequests, req];
          playNotificationSound('request');
        }
      }
    });
  });

  function copyToClipboard(text) {
    navigator.clipboard.writeText(text).then(() => {
      copiedMessage = "Kimlik panoya kopyalandı! 📋";
      setTimeout(() => {
        copiedMessage = "";
      }, 2500);
    });
  }

  function checkExistingProfile() {
    CheckProfileExists().then(exists => {
      isProfileExists = exists;
      if (exists) {
        authLoading = true;
        AutoLoadProfile()
          .then(() => {
            GetUsername().then(u => username = u);
            GetPeerID().then(id => peerID = id);
            GetPublicKeyHex().then(key => publicKeyHex = key);
            GetProfileColor().then(c => profileColor = c || "#00a884");
            GetProfileStatus().then(s => profileStatus = s || "Dosya almaya hazır");
            isLoggedIn = true;
            authLoading = false;
          })
          .catch(err => {
            errorMsg = "Profil yüklenemedi: " + err;
            authLoading = false;
          });
      }
    });
  }

  function handleAutoCreateProfile() {
    errorMsg = "";
    successMsg = "";
    
    let dispName = regUsername.trim();
    if (!dispName) {
      dispName = "Cihaz";
    }

    authLoading = true;
    AutoCreateProfile(dispName)
      .then(() => {
        GetUsername().then(u => username = u);
        GetPeerID().then(id => peerID = id);
        GetPublicKeyHex().then(key => publicKeyHex = key);
        GetProfileColor().then(c => profileColor = c || "#00a884");
        GetProfileStatus().then(s => profileStatus = s || "Dosya almaya hazır");
        isLoggedIn = true;
        authLoading = false;
      })
      .catch(err => {
        errorMsg = "Profil oluşturulamadı: " + err;
        authLoading = false;
      });
  }

  function triggerMultipleFileSelect() {
    if (!selectedPeer || !selectedPeer.online) return;

    SelectMultipleFiles().then(paths => {
      if (paths && paths.length > 0) {
        paths.forEach(filePath => {
          SendPathToPeer(selectedPeer.ip, selectedPeer.tcp_port, selectedPeer.public_key, filePath)
            .catch(err => {
              alert("Dosya gönderilemedi: " + err);
            });
        });
      }
    });
  }

  function triggerFolderSelect() {
    if (!selectedPeer || !selectedPeer.online) return;

    SelectFolderToSend().then(dirPath => {
      if (dirPath) {
        SendPathToPeer(selectedPeer.ip, selectedPeer.tcp_port, selectedPeer.public_key, dirPath)
          .catch(err => {
            alert("Klasör gönderilemedi: " + err);
          });
      }
    });
  }

  function handleDragOver(e) {
    e.preventDefault();
    if (selectedPeer && selectedPeer.online) {
      isDragging = true;
    }
  }

  function handleDragLeave(e) {
    e.preventDefault();
    isDragging = false;
  }

  function handleDrop(e) {
    e.preventDefault();
    isDragging = false;
    if (!selectedPeer || !selectedPeer.online) return;

    const files = e.dataTransfer.files;
    if (files && files.length > 0) {
      for (let i = 0; i < files.length; i++) {
        const path = files[i].path; // absolute path injected by Wails Webview2!
        if (path) {
          SendPathToPeer(selectedPeer.ip, selectedPeer.tcp_port, selectedPeer.public_key, path)
            .catch(err => {
              alert("Gönderim başarısız: " + err);
            });
        }
      }
    }
  }

  function togglePauseResume(transfer) {
    if (transfer.status === 'transferring') {
      PauseTransfer(transfer.id);
    } else if (transfer.status === 'paused') {
      ResumeTransfer(transfer.id);
    }
  }

  function cancelTransfer(transfer) {
    CancelTransfer(transfer.id);
  }

  function handleClearHistory() {
    if (confirm("Tüm transfer geçmişini kalıcı olarak temizlemek istiyor musunuz?")) {
      ClearTransferHistory();
    }
  }

  function openSettings() {
    tempUsername = username;
    tempStatus = profileStatus;
    tempColor = profileColor;
    
    GetServerAddress().then(addr => {
      matchmakingServer = addr;
      tempMatchmakingServer = addr;
    }).catch(() => {});
    
    showSettings = true;
  }

  function handleProfileUpdate() {
    if (!tempUsername.trim()) {
      alert(t.usernameEmpty);
      return;
    }
    
    UpdateProfile(tempUsername.trim(), tempColor, tempStatus.trim())
      .then(() => {
        username = tempUsername.trim();
        profileColor = tempColor;
        profileStatus = tempStatus.trim();
        return UpdateServerAddress(tempMatchmakingServer.trim());
      })
      .then(() => {
        matchmakingServer = tempMatchmakingServer.trim();
        showSettings = false;
      })
      .catch(err => {
        alert("Profil veya sunucu ayarları güncellenemedi: " + err);
      });
  }

  let showAddFriendModal = false;
  let addFriendInput = "";

  async function handleAddFriend() {
    if (!addFriendInput.trim()) return;
    try {
      await AddFriend(addFriendInput.trim());
      addFriendInput = "";
      showAddFriendModal = false;
    } catch (err) {
      alert("Arkadaş eklenemedi: " + err);
    }
  }

  async function acceptFriendRequest(req) {
    if (!req || !req.peer_id) return;
    try {
      await AddFriend(req.peer_id);
      friendRequests = friendRequests.filter(r => r.peer_id !== req.peer_id);
    } catch (err) {
      alert("İstek kabul edilemedi: " + err);
    }
  }

  function rejectFriendRequest(req) {
    if (!req || !req.peer_id) return;
    friendRequests = friendRequests.filter(r => r.peer_id !== req.peer_id);
  }

  async function handleRemoveFriend(peer) {
    if (!peer) return;
    const confirmDelete = confirm(`${peer.username} adlı arkadaşınızı silmek istediğinize emin misiniz?`);
    if (!confirmDelete) return;

    try {
      // Use peer_id if it's a valid QB- id, else fall back to public_key or pub_key_hex
      let targetID = peer.peer_id;
      if (!targetID || targetID === "QB-OFFLINE") {
        targetID = peer.pub_key_hex || peer.public_key;
      }
      await RemoveFriend(targetID);
      selectedPeer = null;
    } catch (err) {
      alert("Arkadaş silinemedi: " + err);
    }
  }

  let chatMessageInput = "";

  async function handleSendChatMessage() {
    if (!chatMessageInput.trim() || !selectedPeer) return;
    const msg = chatMessageInput.trim();
    chatMessageInput = "";
    
    try {
      const pubKey = selectedPeer.pub_key_hex || selectedPeer.public_key;
      await SendTextMessage(
        selectedPeer.ip,
        selectedPeer.tcp_port,
        pubKey,
        msg
      );
    } catch (err) {
      console.error("Mesaj gönderilemedi:", err);
      alert("Mesaj gönderilemedi: " + err);
    }
  }

  function handleChatKeyDown(event) {
    if (event.key === 'Enter') {
      handleSendChatMessage();
    }
  }

  function selectNewSaveDir() {
    SelectDirectory().then(dir => {
      if (dir) {
        targetSaveDir = dir;
      }
    });
  }

  function respondToTransfer(accept) {
    if (!incomingRequest) return;
    AcceptTransfer(incomingRequest.id, accept, targetSaveDir);
    incomingRequest = null;
  }

  function openFile(path) {
    OpenReceivedFile(path).catch(err => {
      alert("Dosya açılamadı: " + err);
    });
  }

  function openFolder(path) {
    OpenFolderAndSelect(path).catch(err => {
      alert("Klasör açılamadı: " + err);
    });
  }

  function formatBytes(bytes, decimals = 2) {
    if (bytes === 0) return '0 Bytes';
    const k = 1024;
    const dm = decimals < 0 ? 0 : decimals;
    const sizes = ['Bytes', 'KB', 'MB', 'GB', 'TB'];
    const i = Math.floor(Math.log(bytes) / Math.log(k));
    return parseFloat((bytes / Math.pow(k, i)).toFixed(dm)) + ' ' + sizes[i];
  }

  function getFileIcon(filename) {
    const ext = filename.split('.').pop().toLowerCase();
    if (['png', 'jpg', 'jpeg', 'gif', 'svg', 'webp'].includes(ext)) return 'image';
    if (['mp4', 'mkv', 'avi', 'mov', 'wmv'].includes(ext)) return 'video';
    if (['mp3', 'wav', 'ogg', 'flac', 'm4a'].includes(ext)) return 'audio';
    if (['zip', 'rar', '7z', 'tar', 'gz'].includes(ext)) return 'archive';
    if (['pdf', 'doc', 'docx', 'xls', 'xlsx', 'ppt', 'pptx', 'txt', 'md'].includes(ext)) return 'document';
    return 'file';
  }
</script>

<!-- AUTHENTICATION SCREEN -->
{#if !isLoggedIn}
<div class="auth-container">
  <div class="auth-card glass-panel animate-fade">
    <div class="logo-area">
      <svg class="logo-shield animate-pulse" viewBox="0 0 24 24" fill="none" xmlns="http://www.w3.org/2000/svg">
        <path d="M12 22C12 22 20 18 20 12V5L12 2L4 5V12C4 18 12 22 12 22Z" stroke="#00a884" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"/>
        <path d="M12 11V15" stroke="#00a884" stroke-width="2" stroke-linecap="round"/>
        <circle cx="12" cy="8" r="1" fill="#00a884"/>
      </svg>
      <h1>QBFile</h1>
      <p class="subtitle">Uçtan Uca Şifreli Yerel Dosya Paylaşımı</p>
    </div>

    {#if !isProfileExists}
      <!-- Registration Form -->
      <form on:submit|preventDefault={handleAutoCreateProfile} class="auth-form animate-slide">
        <h2>Cihazınızı Tanımlayın</h2>
        <p class="form-info">Tamamen yerel ve uçtan uca şifreli dosya aktarımı için benzersiz bir cihaz kimliği oluşturulacak.</p>
        
        <div class="input-group">
          <label for="reg-user">Cihaz İsmi / Takma Ad (İsteğe Bağlı)</label>
          <input type="text" id="reg-user" placeholder="Örn: Ata, Laptop, Ofis-PC..." bind:value={regUsername} autocomplete="off" disabled={authLoading}/>
        </div>

        {#if errorMsg}
          <div class="alert alert-error">{errorMsg}</div>
        {/if}

        <button type="submit" class="submit-btn glowing" disabled={authLoading}>
          {#if authLoading}
            <div class="btn-loader"></div> Kimlik Oluşturuluyor...
          {:else}
            Benzersiz Kimlik Oluştur ve Başlat
          {/if}
        </button>
      </form>
    {:else}
      <!-- Auto Login / Decryption Loader -->
      <div class="auth-form animate-slide" style="text-align: center; padding: 2rem 0;">
        <h2>Güvenli Kimlik Çözülüyor</h2>
        <p class="form-info" style="margin-bottom: 2rem;">Yerel kriptografik profil dosyası yükleniyor...</p>
        
        <div class="btn-loader" style="width: 40px; height: 40px; border-width: 4px; margin: 0 auto 1.5rem auto;"></div>
        
        {#if errorMsg}
          <div class="alert alert-error">{errorMsg}</div>
        {/if}
      </div>
    {/if}
  </div>
</div>

<!-- MAIN DASHBOARD -->
{:else}
<div class="app-layout animate-fade">
  {#if copiedMessage}
    <div style="position: fixed; top: 20px; right: 20px; z-index: 9999; background: rgba(0, 168, 132, 0.95); color: white; padding: 12px 24px; border-radius: 8px; font-weight: 600; box-shadow: 0 4px 15px rgba(0,0,0,0.4); border: 1px solid rgba(255,255,255,0.15); font-size: 0.85rem;" class="animate-slide">
      {copiedMessage}
    </div>
  {/if}
  
  <!-- LEFT SIDEBAR -->
  <aside class="sidebar">
    <!-- Sidebar Header -->
    <header class="sidebar-header">
      <div class="user-profile">
        <div class="avatar initials" style="background: {profileColor}; cursor: pointer; transition: background 0.3s; font-weight: 700;" on:click={openSettings} title={t.settingsBtn}>
          {username ? username[0].toUpperCase() : 'U'}
        </div>
        <div class="profile-info" style="display: flex; flex-direction: column; justify-content: center; gap: 2px;">
          <h3 style="cursor: pointer; font-size: 0.92rem; font-weight: 600; margin: 0; line-height: 1.2;" on:click={openSettings} title={t.settingsBtn}>{username}</h3>
          <span style="font-size: 0.68rem; color: var(--text-secondary); cursor: pointer; letter-spacing: 0.3px; display: flex; align-items: center; gap: 4px;" on:click={() => copyToClipboard(peerID)} title="Kimliğimi Kopyalamak İçin Tıkla">
            <span class="pulse-dot" style="background-color: {profileColor}; box-shadow: 0 0 6px {profileColor}; width: 6px; height: 6px;"></span>
            {peerID} 📋
          </span>
        </div>
      </div>
      <div style="display: flex; gap: 6px; align-items: center;">
        <button class="logout-btn" on:click={openSettings} title={t.settingsBtn} style="padding: 8px; display: flex; justify-content: center; align-items: center; border-radius: 50%; color: var(--text-secondary); transition: all 0.2s;">
          <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" style="width: 18px; height: 18px;">
            <circle cx="12" cy="12" r="3"/>
            <path d="M19.4 15a1.65 1.65 0 0 0 .33 1.82l.06.06a2 2 0 1 1-2.83 2.83l-.06-.06a1.65 1.65 0 0 0-1.82-.33 1.65 1.65 0 0 0-1 1.51V21a2 2 0 0 1-4 0v-.09A1.65 1.65 0 0 0 9 19.4a1.65 1.65 0 0 0-1.82.33l-.06.06a2 2 0 1 1-2.83-2.83l.06-.06a1.65 1.65 0 0 0 .33-1.82 1.65 1.65 0 0 0-1.51-1H3a2 2 0 0 1 0-4h.09A1.65 1.65 0 0 0 4.6 9a1.65 1.65 0 0 0-.33-1.82l-.06-.06a2 2 0 1 1 2.83-2.83l.06.06a1.65 1.65 0 0 0 1.82.33H9a1.65 1.65 0 0 0 1-1.51V3a2 2 0 0 1 4 0v.09a1.65 1.65 0 0 0 1 1.51 1.65 1.65 0 0 0 1.82-.33l.06-.06a2 2 0 1 1 2.83 2.83l-.06.06a1.65 1.65 0 0 0-.33 1.82V9a1.65 1.65 0 0 0 1.51 1H21a2 2 0 0 1 0 4h-.09a1.65 1.65 0 0 0-1.51 1z"/>
          </svg>
        </button>
      </div>
    </header>

    <!-- Friend Management Action Row -->
    <div style="display: flex; gap: 8px; padding: 4px 16px 12px; align-items: center; justify-content: space-between; border-bottom: 1px solid var(--border-light); flex-wrap: wrap;">
      <div style="display: flex; gap: 4px; background: rgba(255,255,255,0.03); padding: 2px; border-radius: 8px;">
        <button on:click={() => { viewMode = 'chat'; }} class="tab-btn {viewMode === 'chat' ? 'active' : ''}" style="border: none; background: {viewMode === 'chat' ? 'var(--accent)' : 'transparent'}; color: {viewMode === 'chat' ? 'white' : 'var(--text-secondary)'}; font-size: 0.72rem; font-weight: 700; padding: 6px 10px; border-radius: 6px; cursor: pointer; transition: all 0.2s; text-transform: uppercase; letter-spacing: 0.5px;">
          {t.myConnections}
        </button>
        <button on:click={() => { viewMode = 'transfers'; }} class="tab-btn {viewMode === 'transfers' ? 'active' : ''}" style="border: none; background: {viewMode === 'transfers' ? 'var(--accent)' : 'transparent'}; color: {viewMode === 'transfers' ? 'white' : 'var(--text-secondary)'}; font-size: 0.72rem; font-weight: 700; padding: 6px 10px; border-radius: 6px; cursor: pointer; transition: all 0.2s; text-transform: uppercase; letter-spacing: 0.5px;">
          {t.sentItems}
        </button>
      </div>
      
      <button on:click={() => showAddFriendModal = true} class="glowing" style="background: var(--accent); color: white; border: none; padding: 6px 10px; border-radius: 8px; font-size: 0.72rem; font-weight: 600; cursor: pointer; display: flex; align-items: center; gap: 4px; box-shadow: 0 4px 15px var(--accent-glow); transition: all 0.25s;">
        <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" style="width: 12px; height: 12px;"><path d="M16 21v-2a4 4 0 0 0-4-4H5a4 4 0 0 0-4 4v2"/><circle cx="8.5" cy="7" r="4"/><line x1="20" y1="8" x2="20" y2="14"/><line x1="23" y1="11" x2="17" y2="11"/></svg>
        {t.addFriend}
      </button>
    </div>

    <!-- Friend Requests -->
    {#if friendRequests.length > 0}
      <div class="friend-requests-section animate-slide" style="padding: 10px 16px; border-bottom: 1px solid var(--border-light); background: rgba(0, 168, 132, 0.04);">
        <span style="font-size: 0.72rem; font-weight: 700; color: var(--accent); text-transform: uppercase; letter-spacing: 0.5px; display: block; margin-bottom: 8px;">
          {t.friendRequests} ({friendRequests.length})
        </span>
        <div style="display: flex; flex-direction: column; gap: 8px;">
          {#each friendRequests as req}
            <div class="glass-panel" style="padding: 10px; border-radius: 8px; border: 1px solid rgba(0,168,132,0.15); display: flex; flex-direction: column; gap: 6px; background: rgba(20,20,20,0.4);">
              <div style="display: flex; align-items: center; gap: 8px;">
                <div class="avatar initials" style="width: 28px; height: 28px; font-size: 0.8rem; background: var(--accent); display: flex; align-items: center; justify-content: center; border-radius: 50%; color: white; font-weight: 700;">
                  {req.username ? req.username[0].toUpperCase() : '?'}
                </div>
                <div style="display: flex; flex-direction: column; min-width: 0; flex: 1;">
                  <span style="font-size: 0.82rem; font-weight: 600; color: white; overflow: hidden; text-overflow: ellipsis; white-space: nowrap;">{req.username}</span>
                  <span style="font-size: 0.6rem; color: var(--text-secondary); overflow: hidden; text-overflow: ellipsis; white-space: nowrap;">{req.peer_id}</span>
                </div>
              </div>
              <div style="display: flex; gap: 6px;">
                <button on:click={() => acceptFriendRequest(req)} class="bubble-btn accept-btn glowing" style="flex: 1; padding: 4px 8px; font-size: 0.7rem; font-weight: 700; background: var(--accent); color: white; border: none; border-radius: 4px; cursor: pointer;">
                  {t.acceptBtnShort}
                </button>
                <button on:click={() => rejectFriendRequest(req)} class="bubble-btn decline-btn" style="flex: 1; padding: 4px 8px; font-size: 0.7rem; font-weight: 700; background: rgba(255,255,255,0.05); border: 1px solid var(--border-light); color: var(--text-secondary); border-radius: 4px; cursor: pointer;">
                  {t.rejectBtnShort}
                </button>
              </div>
            </div>
          {/each}
        </div>
      </div>
    {/if}

    <!-- Search / Filter -->
    <div class="search-container">
      <div class="search-bar">
        <svg class="search-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><circle cx="11" cy="11" r="8"/><line x1="21" y1="21" x2="16.65" y2="16.65"/></svg>
        <input type="text" placeholder={t.searchPlaceholder} bind:value={searchQuery}/>
      </div>
    </div>

    <!-- Active Peers / Chat List -->
    <div class="contacts-list">
      {#if filteredPeers.length === 0}
        <div class="empty-state">
          <svg class="radar-icon animate-pulse" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
            <circle cx="12" cy="12" r="10"/>
            <path d="M12 2a15.3 15.3 0 0 1 4 10 15.3 15.3 0 0 1-4 10 15.3 15.3 0 0 1-4-10 15.3 15.3 0 0 1 4-10z"/>
            <path d="M2 12h20"/>
          </svg>
          <p>{t.searchingPeers}</p>
          <span class="subtext">{t.searchingSub}</span>
        </div>
      {:else}
        {#each filteredPeers as peer}
          <button class="contact-card {viewMode === 'chat' && selectedPeer && selectedPeer.username === peer.username ? 'active' : ''}" on:click={() => { selectedPeer = peer; viewMode = 'chat'; }}>
            <div class="avatar initials" style="background: {peer.color || 'linear-gradient(135deg, #005c4b, #202c33)'}; transition: background 0.3s; position: relative;">
              {peer.username[0].toUpperCase()}
              <span class="active-dot" style="position: absolute; bottom: 0; right: 0; width: 8px; height: 8px; border-radius: 50%; background-color: {peer.color || 'var(--accent)'}; border: 1.5px solid var(--bg-secondary);"></span>
            </div>
            <div class="contact-details">
              <div class="contact-row">
                <span class="contact-name" style="display: flex; align-items: center; gap: 5px;">
                  {peer.username}
                  {#if peer.is_wan}
                    <span style="font-size: 0.8rem; filter: drop-shadow(0 0 2px rgba(0, 168, 132, 0.45));" title="İnternet Paylaşımı (WAN)">🌐</span>
                  {/if}
                  {#if peer.peer_id}
                    <span style="font-family: monospace; font-size: 0.64rem; background: rgba(0, 168, 132, 0.12); padding: 1px 5px; border-radius: 4px; color: var(--accent); border: 1px solid rgba(0, 168, 132, 0.2); letter-spacing: 0.2px; font-weight: 600;">{peer.peer_id.substring(3, 7)}</span>
                  {/if}
                </span>
                <span class="contact-time" style="color: {peer.color || 'var(--accent)'}; font-weight: 600; font-size: 0.72rem; max-width: 80px; overflow: hidden; text-overflow: ellipsis; white-space: nowrap;">{peer.status || (currentLang === 'tr' ? "Çevrimiçi" : "Online")}</span>
              </div>
              <div class="contact-row">
                <span class="contact-ip">{t.secureChannel}</span>
                <span class="lock-icon" style="color: var(--accent); font-size: 0.7rem; font-weight: 600; display: flex; align-items: center; gap: 3px;">
                  <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" style="width: 11px; height: 11px;"><rect x="3" y="11" width="18" height="11" rx="2" ry="2"/><path d="M7 11V7a5 5 0 0 1 10 0v4"/></svg>
                  E2EE
                </span>
              </div>
            </div>
          </button>
        {/each}
      {/if}
    </div>
  </aside>

  <!-- RIGHT WINDOW (SPLASH OR ACTIVE CHAT) -->
  <main class="chat-area">
    {#if viewMode === 'transfers'}
      <!-- Transfers Dashboard (filtered to selected peer) -->
      <div class="transfers-dashboard" style="display: flex; flex-direction: column; height: 100%; background: var(--bg-primary);">
        <header style="padding: 20px 24px; border-bottom: 1px solid var(--border-light); display: flex; justify-content: space-between; align-items: center; background: var(--bg-secondary); box-shadow: 0 4px 10px rgba(0,0,0,0.15);">
          <div>
            <h1 style="font-size: 1.2rem; font-weight: 700; color: white; margin: 0; font-family: var(--font-main);">
              {selectedPeer ? selectedPeer.username : t.allTransfersDashboard}
            </h1>
            <p style="font-size: 0.78rem; color: var(--text-secondary); margin: 3px 0 0 0; font-family: var(--font-main);">{t.transferHistorySub}</p>
          </div>
          {#if transfers.length > 0}
            <button on:click={handleClearHistory} class="bubble-btn animate-fade" style="background: rgba(234, 67, 53, 0.08); border-color: rgba(234, 67, 53, 0.15); color: #ff8f8f; padding: 6px 12px; font-weight: 600; display: flex; align-items: center; gap: 5px;" title={t.clearHistory}>
              {t.clearHistoryBtn}
            </button>
          {/if}
        </header>

        <div style="flex: 1; overflow-y: auto; padding: 24px; display: flex; flex-direction: column; gap: 12px;">
          {#if !selectedPeer}
            <div style="display: flex; flex-direction: column; align-items: center; justify-content: center; height: 100%; color: var(--text-secondary); text-align: center; gap: 12px; opacity: 0.55;">
              <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" style="width: 48px; height: 48px; color: var(--accent);"><path d="M17 21v-2a4 4 0 0 0-4-4H5a4 4 0 0 0-4 4v2"/><circle cx="9" cy="7" r="4"/><path d="M23 21v-2a4 4 0 0 0-3-3.87"/><path d="M16 3.13a4 4 0 0 1 0 7.75"/></svg>
              <p style="font-size: 0.92rem; font-weight: 600; margin: 0;">{currentLang === 'tr' ? 'Geçmişi görmek için sol panelden bir arkadaş seçin.' : 'Select a friend from the left to view history.'}</p>
            </div>
          {:else}
            {#each filteredTransfers.filter(tr => !tr.filename.startsWith('[TextBase64]')) as tr}
              <div class="glass-panel animate-slide" style="padding: 14px 18px; border-radius: 12px; border: 1px solid var(--border-light); background: rgba(255,255,255,0.02); display: flex; flex-direction: column; gap: 10px; transition: all 0.25s;">
                <div style="display: flex; justify-content: space-between; align-items: center; gap: 16px;">
                  <div style="display: flex; align-items: center; gap: 12px; overflow: hidden; flex: 1;">
                    <div style="width: 36px; height: 36px; border-radius: 8px; background: {tr.is_sender ? 'rgba(66, 133, 244, 0.08)' : 'rgba(0, 168, 132, 0.08)'}; color: {tr.is_sender ? '#4285f4' : 'var(--accent)'}; display: flex; align-items: center; justify-content: center; font-size: 1.05rem; flex-shrink: 0; border: 1px solid {tr.is_sender ? 'rgba(66, 133, 244, 0.15)' : 'rgba(0, 168, 132, 0.15)'};">
                      {#if tr.is_sender}
                        📤
                      {:else}
                        📥
                      {/if}
                    </div>
                    <div style="display: flex; flex-direction: column; overflow: hidden; text-align: left;">
                      <span style="font-size: 0.88rem; font-weight: 600; color: white; overflow: hidden; text-overflow: ellipsis; white-space: nowrap;" title={tr.filename}>{tr.filename}</span>
                      <span style="font-size: 0.7rem; color: var(--text-secondary); margin-top: 2px;">
                        {formatBytes(tr.filesize)} • {tr.is_sender ? (currentLang === 'tr' ? 'Alıcı' : 'Recipient') : (currentLang === 'tr' ? 'Gönderen' : 'Sender')}: <strong style="color: white; font-weight: 600;">{tr.peer_name}</strong>
                      </span>
                    </div>
                  </div>

                  <div style="display: flex; align-items: center; gap: 8px; flex-shrink: 0;">
                    {#if tr.status !== 'transferring' && tr.status !== 'paused'}
                      <span style="font-size: 0.65rem; font-weight: 700; padding: 4px 8px; border-radius: 10px; text-transform: uppercase; letter-spacing: 0.5px;
                        background: {tr.status === 'completed' ? 'rgba(0, 168, 132, 0.08)' : tr.status === 'declined' ? 'rgba(234, 67, 53, 0.08)' : 'rgba(255, 255, 255, 0.04)'};
                        color: {tr.status === 'completed' ? 'var(--accent)' : tr.status === 'declined' || tr.status === 'failed' ? '#ff8f8f' : 'var(--text-secondary)'};
                        border: 1px solid {tr.status === 'completed' ? 'rgba(0, 168, 132, 0.15)' : tr.status === 'declined' || tr.status === 'failed' ? 'rgba(234, 67, 53, 0.15)' : 'rgba(255, 255, 255, 0.08)'};">
                        {tr.status === 'completed' ? t.statusCompleted : tr.status === 'declined' ? t.statusDeclined : tr.status === 'failed' ? t.statusFailed : tr.status}
                      </span>
                    {/if}
                  </div>
                </div>

                {#if tr.status === 'transferring' || tr.status === 'paused'}
                  <div class="progress-container" style="margin: 0; padding: 0; background: transparent; border: none; width: 100%;">
                    <div class="progress-header" style="display: flex; justify-content: space-between; font-size: 0.72rem; color: var(--text-secondary); margin-bottom: 5px;">
                      <span>{tr.status === 'paused' ? t.statusPaused : t.statusTransferring} %{tr.percent}</span>
                      <span>{tr.status === 'paused' ? '0.0' : tr.speed_mb.toFixed(1)} MB/s</span>
                    </div>
                    <div class="progress-bar-bg" style="height: 5px; background: rgba(255,255,255,0.06); border-radius: 3px; overflow: hidden; width: 100%;">
                      <div class="progress-bar-fill" style="width: {tr.percent}%; height: 100%; background-color: {tr.status === 'paused' ? '#fbbc05' : 'var(--accent)'}; box-shadow: 0 0 8px {tr.status === 'paused' ? '#fbbc05' : 'var(--accent-glow)'}; transition: width 0.1s ease;"></div>
                    </div>
                    <div style="display: flex; justify-content: flex-end; gap: 6px; margin-top: 6px;">
                      {#if tr.is_sender}
                        <button on:click={() => togglePauseResume(tr)} class="bubble-btn" style="background-color: rgba(255,255,255,0.04); font-size: 0.68rem; border-color: rgba(255,255,255,0.08); padding: 3px 8px; border-radius: 4px;">
                          {tr.status === 'paused' ? t.actionResume : t.actionPause}
                        </button>
                      {:else}
                        <button on:click={() => togglePauseResume(tr)} class="bubble-btn" style="background-color: rgba(255,255,255,0.04); font-size: 0.68rem; border-color: rgba(255,255,255,0.08); padding: 3px 8px; border-radius: 4px;">
                          {tr.status === 'paused' ? t.actionResume : t.actionPause}
                        </button>
                        <button on:click={() => cancelTransfer(tr)} class="bubble-btn" style="background-color: rgba(234,67,53,0.08); font-size: 0.68rem; border-color: rgba(234,67,53,0.2); color: #ff8f8f; padding: 3px 8px; border-radius: 4px;">
                          {t.actionCancel}
                        </button>
                      {/if}
                    </div>
                  </div>
                {:else if tr.status === 'completed'}
                  {#if !tr.is_sender && tr.local_path}
                    <div style="display: flex; gap: 6px; justify-content: flex-end; width: 100%;">
                      <button on:click={() => openFile(tr.local_path)} class="bubble-btn open-file-btn" style="padding: 4px 10px; font-size: 0.7rem; border-radius: 4px;">
                        {t.actionOpen}
                      </button>
                      <button on:click={() => openFolder(tr.local_path)} class="bubble-btn folder-btn" title={t.actionShow} style="padding: 4px 10px; font-size: 0.7rem; border-radius: 4px;">
                        {t.actionShow}
                      </button>
                    </div>
                  {/if}
                {/if}
              </div>
            {:else}
              <div style="display: flex; flex-direction: column; align-items: center; justify-content: center; height: 100%; color: var(--text-secondary); text-align: center; gap: 12px; opacity: 0.55;">
                <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" style="width: 48px; height: 48px; color: var(--accent);"><path d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z"/><polyline points="14 2 14 8 20 8"/></svg>
                <p style="font-size: 0.92rem; font-weight: 600; margin: 0;">{t.noTransfersYet}</p>
              </div>
            {/each}
          {/if}
        </div>
      </div>
    {:else if !selectedPeer}
      <!-- Splash Screen -->
      <div class="splash-screen" style="display: flex; justify-content: center; align-items: center; height: 100%; background: radial-gradient(circle at center, rgba(0,168,132,0.03) 0%, rgba(11,20,26,0) 70%);">
        <div class="splash-content glass-panel animate-fade" style="width: 100%; max-width: 520px; padding: 40px; text-align: center; display: flex; flex-direction: column; align-items: center; border: 1px solid rgba(255,255,255,0.06); background: rgba(17, 27, 33, 0.4); box-shadow: 0 20px 50px rgba(0,0,0,0.5); backdrop-filter: blur(20px);">
          
          <!-- Concentric Radar / Shield Visual -->
          <div style="position: relative; width: 120px; height: 120px; margin-bottom: 24px; display: flex; justify-content: center; align-items: center;">
            <div style="position: absolute; width: 120px; height: 120px; border-radius: 50%; border: 1px dashed rgba(0, 168, 132, 0.15); animation: spin 20s linear infinite;"></div>
            <div style="position: absolute; width: 90px; height: 90px; border-radius: 50%; border: 1px solid rgba(0, 168, 132, 0.25); animation: pulse 3s ease-in-out infinite;"></div>
            <div style="position: absolute; width: 60px; height: 60px; border-radius: 50%; background: radial-gradient(circle, rgba(0, 168, 132, 0.15) 0%, transparent 80%);"></div>
            
            <svg class="splash-shield" viewBox="0 0 24 24" fill="none" stroke="var(--accent)" stroke-width="1.5" style="width: 48px; height: 48px; filter: drop-shadow(0 0 12px var(--accent-glow)); z-index: 2;">
              <path d="M12 22C12 22 20 18 20 12V5L12 2L4 5V12C4 18 12 22 12 22Z"/>
              <path d="M9 11L11 13L15 9" stroke="var(--accent)" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round"/>
            </svg>
          </div>

          <h1 style="font-size: 2.2rem; font-weight: 800; letter-spacing: -0.5px; background: linear-gradient(135deg, #ffffff 30%, #00a884 100%); -webkit-background-clip: text; -webkit-text-fill-color: transparent; margin-bottom: 4px; font-family: var(--font-main);">QBFile</h1>
          <p class="splash-title" style="font-size: 1.05rem; font-weight: 600; color: var(--accent); letter-spacing: 1px; text-transform: uppercase; margin-bottom: 16px; font-family: var(--font-main);">{t.splashTitle}</p>
          <p class="splash-desc" style="font-size: 0.9rem; color: var(--text-secondary); line-height: 1.6; margin-bottom: 28px; font-family: var(--font-main);">
            {t.splashDesc}
          </p>

          <div style="display: grid; grid-template-columns: 1fr 1fr; gap: 14px; width: 100%; text-align: left; margin-bottom: 12px;">
            <div style="background: rgba(255,255,255,0.01); border: 1px solid rgba(255,255,255,0.03); padding: 12px 16px; border-radius: 8px;">
              <div style="font-size: 0.85rem; font-weight: 700; color: white; display: flex; align-items: center; gap: 6px; margin-bottom: 4px; font-family: var(--font-main);">
                {t.splashFeature1Title}
              </div>
              <div style="font-size: 0.75rem; color: var(--text-secondary); line-height: 1.4; font-family: var(--font-main);">{t.splashFeature1Desc}</div>
            </div>
            <div style="background: rgba(255,255,255,0.01); border: 1px solid rgba(255,255,255,0.03); padding: 12px 16px; border-radius: 8px;">
              <div style="font-size: 0.85rem; font-weight: 700; color: white; display: flex; align-items: center; gap: 6px; margin-bottom: 4px; font-family: var(--font-main);">
                {t.splashFeature2Title}
              </div>
              <div style="font-size: 0.75rem; color: var(--text-secondary); line-height: 1.4; font-family: var(--font-main);">{t.splashFeature2Desc}</div>
            </div>
          </div>
          
          <div class="security-badge" style="margin-top: 10px; display: inline-flex; align-items: center; gap: 8px; font-size: 0.78rem; font-weight: 600; color: var(--accent); background: rgba(0, 168, 132, 0.08); padding: 8px 16px; border-radius: 20px; border: 1px solid rgba(0, 168, 132, 0.15); font-family: var(--font-main);">
            <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" style="width: 14px; height: 14px;"><rect x="3" y="11" width="18" height="11" rx="2" ry="2"/><path d="M7 11V7a5 5 0 0 1 10 0v4"/></svg>
            {t.splashSecurityBadge}
          </div>
        </div>
      </div>
    {:else}
      <!-- Active Chat Interface -->
      <div class="chat-window" style="position: relative;" on:dragover={handleDragOver} on:dragleave={handleDragLeave} on:drop={handleDrop}>
        
        {#if isDragging}
          <div class="drag-overlay glass-panel animate-fade" style="position: absolute; top: 0; left: 0; width: 100%; height: 100%; display: flex; flex-direction: column; justify-content: center; align-items: center; background-color: rgba(11,20,26,0.92); backdrop-filter: blur(16px); z-index: 100; border: 3px dashed var(--accent); margin: 0; box-shadow: inset 0 0 50px rgba(0, 168, 132, 0.15); transition: all 0.3s;">
            <div style="animation: bounce 1.8s infinite ease-in-out; display: flex; justify-content: center; align-items: center; width: 100px; height: 100px; background: rgba(0, 168, 132, 0.1); border-radius: 50%; margin-bottom: 20px; border: 1px solid rgba(0, 168, 132, 0.25);">
              <svg viewBox="0 0 24 24" fill="none" stroke="var(--accent)" stroke-width="1.8" style="width: 48px; height: 48px;"><path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4"/><polyline points="17 8 12 3 7 8"/><line x1="12" y1="3" x2="12" y2="15"/></svg>
            </div>
            <h2 style="font-size: 1.5rem; font-weight: 800; color: white; letter-spacing: -0.5px; font-family: var(--font-main);">{t.dragOverlayTitle}</h2>
            <p style="font-size: 0.9rem; color: var(--text-secondary); margin-top: 8px; font-family: var(--font-main); max-width: 380px; text-align: center; line-height: 1.5;">{t.dragOverlaySub}</p>
          </div>
        {/if}

        <!-- Chat Header -->
        <header class="chat-header">
          <div class="peer-profile">
            <div class="avatar initials" style="background: {selectedPeer.color || '#00a884'}">
              {selectedPeer.username[0].toUpperCase()}
            </div>
            <div class="peer-meta">
              <h2>{selectedPeer.username}</h2>
              <span class="peer-status">
                {#if selectedPeer.online}
                  <span class="active-dot" style="background-color: {selectedPeer.color || 'var(--accent)'}"></span> {selectedPeer.status || (currentLang === 'tr' ? "Çevrimiçi" : "Online")}
                {:else}
                  <span class="offline-dot"></span> {t.offlineText}
                {/if}
              </span>
            </div>
          </div>
          <div style="display: flex; gap: 10px; align-items: center;">
            <button on:click={() => handleRemoveFriend(selectedPeer)} class="bubble-btn animate-fade" style="background-color: rgba(234, 67, 53, 0.08); border-color: rgba(234, 67, 53, 0.15); color: #ff8f8f; padding: 6px 12px; display: flex; align-items: center; gap: 5px;" title={t.deleteFriendTitle}>
              <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" style="width: 12px; height: 12px;"><path d="M16 21v-2a4 4 0 0 0-4-4H5a4 4 0 0 0-4 4v2"/><circle cx="8.5" cy="7" r="4"/><line x1="23" y1="11" x2="17" y2="11"/></svg>
              {t.deleteFriend}
            </button>
            <button on:click={handleClearHistory} class="bubble-btn animate-fade" style="background-color: rgba(255, 255, 255, 0.05); border-color: rgba(255, 255, 255, 0.1); color: var(--text-secondary); padding: 6px 12px; display: flex; align-items: center; gap: 5px;" title={t.clearHistory}>
              {t.clearHistoryBtn}
            </button>
            <div class="security-lock" title="Elliptic Curve Diffie-Hellman Key Exchange (ECDH)">
              <svg class="header-lock-icon" viewBox="0 0 24 24" fill="none" stroke="#00a884" stroke-width="2"><rect x="3" y="11" width="18" height="11" rx="2" ry="2"/><path d="M7 11V7a5 5 0 0 1 10 0v4"/></svg>
              <span>{t.activeE2EE}</span>
            </div>
          </div>
        </header>

        {#if startupFilePath}
          <div class="glass-panel animate-slide" style="margin: 12px 24px 0 24px; padding: 14px 20px; background: rgba(0, 168, 132, 0.08); border: 1px solid rgba(0, 168, 132, 0.25); border-radius: 10px; display: flex; justify-content: space-between; align-items: center; gap: 16px; box-shadow: 0 4px 15px rgba(0,0,0,0.15); z-index: 10;">
            <div style="display: flex; align-items: center; gap: 12px; overflow: hidden;">
              <span style="font-size: 1.25rem; filter: drop-shadow(0 0 4px rgba(0,168,132,0.5));">⚡</span>
              <div style="display: flex; flex-direction: column; overflow: hidden; text-align: left;">
                <span style="font-size: 0.7rem; color: var(--accent); font-weight: 700; text-transform: uppercase; letter-spacing: 0.6px; line-height: 1.2;">{t.rightClickReady}</span>
                <span style="font-size: 0.82rem; color: white; font-weight: 600; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; margin-top: 2px;" title={startupFilePath}>{startupFilePath.split(/[/\\]/).pop()}</span>
              </div>
            </div>
            <div style="display: flex; gap: 8px; shrink: 0;">
              <button class="bubble-btn" on:click={() => { startupFilePath = ""; ClearStartupFilePath(); }} style="background: rgba(255,255,255,0.05); border-color: rgba(255,255,255,0.08); color: var(--text-secondary); font-size: 0.72rem; padding: 6px 12px; border-radius: 6px; cursor: pointer; transition: all 0.2s;">
                {t.cancelBtn}
              </button>
              <button class="bubble-btn glowing" on:click={handleSendStartupPath} style="background: var(--accent); border-color: var(--accent); color: white; font-size: 0.72rem; padding: 6px 12px; font-weight: 700; border-radius: 6px; cursor: pointer; transition: all 0.2s; box-shadow: 0 0 10px rgba(0,168,132,0.3);">
                {t.btnSendNow}
              </button>
            </div>
          </div>
        {/if}

        <!-- Message/Transfer List -->
        <div class="transfer-history">
          <div class="security-warning glass-panel">
            <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><rect x="3" y="11" width="18" height="11" rx="2" ry="2"/><path d="M7 11V7a5 5 0 0 1 10 0v4"/></svg>
            {t.e2eWarning}
          </div>

          {#if filteredTransfers.length === 0}
            <div class="no-transfers animate-fade">
              <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><path d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z"/><polyline points="14 2 14 8 20 8"/><line x1="16" y1="13" x2="8" y2="13"/><line x1="16" y1="17" x2="8" y2="17"/><polyline points="10 9 9 9 8 9"/></svg>
              <p>{t.emptyHistory}</p>
              <span>{t.emptyHistorySub}</span>
            </div>
          {:else}
            {#each filteredTransfers as tr (tr.id)}
              <div class="bubble-row {tr.is_sender ? 'sender-row' : 'receiver-row'}">
                {#if tr.filename.startsWith("[TextBase64]")}
                  <!-- Encrypted Chat Text Message Bubble -->
                  <div class="bubble {tr.is_sender ? 'bubble-sender' : 'bubble-receiver'} glass-panel animate-slide" style="padding: 12px 16px; border-radius: 12px; max-width: 70%; line-height: 1.4; word-break: break-word; font-family: var(--font-main); font-size: 0.92rem; box-shadow: 0 4px 15px rgba(0,0,0,0.15); border: 1px solid var(--glass-border);">
                    <div style="color: white; font-weight: 500; font-family: var(--font-main);">
                      {decodeBase64Utf8(tr.filename.substring(12, tr.filename.length - 4))}
                    </div>
                    <div style="font-size: 0.65rem; color: rgba(255,255,255,0.45); text-align: right; margin-top: 5px; font-family: var(--font-main); display: flex; align-items: center; justify-content: flex-end; gap: 4px;">
                      <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" style="width: 10px; height: 10px;"><rect x="3" y="11" width="18" height="11" rx="2" ry="2"/><path d="M7 11V7a5 5 0 0 1 10 0v4"/></svg>
                      {tr.is_sender ? 'Gönderildi' : 'Alındı'} • E2EE
                    </div>
                  </div>
                {:else}
                  <!-- Standard File Card Bubble -->
                  <div class="bubble {tr.is_sender ? 'bubble-sender' : 'bubble-receiver'} glass-panel animate-slide">
                    <!-- File Info -->
                    <div class="file-card">
                      <div class="file-icon-bg {getFileIcon(tr.filename)}">
                        {#if getFileIcon(tr.filename) === 'image'}
                          <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><rect x="3" y="3" width="18" height="18" rx="2" ry="2"/><circle cx="8.5" cy="8.5" r="1.5"/><polyline points="21 15 16 10 5 21"/></svg>
                        {:else if getFileIcon(tr.filename) === 'video'}
                          <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><polygon points="23 7 16 12 23 17 23 7"/><rect x="1" y="5" width="15" height="14" rx="2" ry="2"/></svg>
                        {:else if getFileIcon(tr.filename) === 'audio'}
                          <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M9 18V5l12-2v13"/><circle cx="6" cy="18" r="3"/><circle cx="18" cy="16" r="3"/></svg>
                        {:else if getFileIcon(tr.filename) === 'archive'}
                          <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><rect x="3" y="3" width="18" height="18" rx="2" ry="2"/><line x1="12" y1="3" x2="12" y2="21"/><line x1="3" y1="12" x2="21" y2="12"/></svg>
                        {:else if getFileIcon(tr.filename) === 'document'}
                          <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z"/><polyline points="14 2 14 8 20 8"/><line x1="16" y1="13" x2="8" y2="13"/><line x1="16" y1="17" x2="8" y2="17"/><polyline points="10 9 9 9 8 9"/></svg>
                        {:else}
                          <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M13 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V9z"/><polyline points="13 2 13 9 20 9"/></svg>
                        {/if}
                      </div>
                      <div class="file-details">
                        <span class="filename" title={tr.filename}>{tr.filename}</span>
                        <span class="filesize">{formatBytes(tr.filesize)}</span>
                      </div>
                    </div>

                    <!-- Progress Indicator -->
                    {#if tr.status === 'transferring' || tr.status === 'paused'}
                      <div class="progress-container">
                        <div class="progress-header">
                          <span>{tr.status === 'paused' ? t.statusPaused : t.statusTransferring} %{tr.percent}</span>
                          <span>{tr.status === 'paused' ? '0.0' : tr.speed_mb.toFixed(1)} MB/s</span>
                        </div>
                        <div class="progress-bar-bg">
                          <div class="progress-bar-fill" style="width: {tr.percent}%; background-color: {tr.status === 'paused' ? '#fbbc05' : 'var(--accent)'}; box-shadow: 0 0 8px {tr.status === 'paused' ? '#fbbc05' : 'var(--accent-glow)'};"></div>
                        </div>
                        <div style="display: flex; justify-content: flex-end; gap: 6px; margin-top: 8px;">
                          <button on:click={() => togglePauseResume(tr)} class="bubble-btn" style="background-color: rgba(255,255,255,0.06); font-size: 0.7rem; border-color: rgba(255,255,255,0.1);">
                            {tr.status === 'paused' ? t.actionResume : t.actionPause}
                          </button>
                          {#if !tr.is_sender}
                            <button on:click={() => cancelTransfer(tr)} class="bubble-btn" style="background-color: rgba(234,67,53,0.08); font-size: 0.7rem; border-color: rgba(234,67,53,0.2); color: #ff8f8f;">
                              {t.actionCancel}
                            </button>
                          {/if}
                        </div>
                      </div>
                    {:else if tr.status === 'pending'}
                      <div class="status-indicator pending animate-pulse">
                        <span class="dot"></span> {t.statusPending}
                      </div>
                    {:else if tr.status === 'completed'}
                      <div class="status-indicator completed">
                        <div class="completed-meta">
                          <svg class="check-icon" viewBox="0 0 24 24" fill="none" stroke="#00a884" stroke-width="3"><polyline points="20 6 9 17 4 12"/></svg>
                          <span>{t.statusCompleted}</span>
                        </div>
                        
                        {#if !tr.is_sender && tr.local_path}
                          <div class="action-buttons animate-fade">
                            <button on:click={() => openFile(tr.local_path)} class="bubble-btn open-file-btn">
                              {t.actionOpen}
                            </button>
                            <button on:click={() => openFolder(tr.local_path)} class="bubble-btn folder-btn" title={t.actionShow}>
                              {t.actionShow}
                            </button>
                          </div>
                        {/if}
                      </div>
                    {:else if tr.status === 'declined'}
                      <div class="status-indicator declined">
                        <svg class="alert-icon" viewBox="0 0 24 24" fill="none" stroke="#ea4335" stroke-width="2"><circle cx="12" cy="12" r="10"/><line x1="15" y1="9" x2="9" y2="15"/><line x1="9" y1="9" x2="15" y2="15"/></svg>
                        <span>{t.statusDeclined}</span>
                      </div>
                    {:else if tr.status === 'failed'}
                      <div class="status-indicator failed">
                        <svg class="alert-icon" viewBox="0 0 24 24" fill="none" stroke="#ea4335" stroke-width="2"><circle cx="12" cy="12" r="10"/><line x1="12" y1="8" x2="12" y2="12"/><line x1="12" y1="16" x2="12.01" y2="16"/></svg>
                        <span>{t.statusFailed} ({tr.peer_name})</span>
                      </div>
                    {/if}
                  </div>
                {/if}
              </div>
            {/each}
          {/if}
        </div>

        <!-- Premium E2EE Chat Input Footer -->
        <footer class="chat-footer-action" style="flex-direction: column; gap: 12px; padding: 16px 24px; background: var(--bg-secondary); border-top: 1px solid var(--border-light); display: flex;">
          {#if selectedPeer.online}
            <!-- Message Input Line -->
            <div style="display: flex; width: 100%; gap: 12px; align-items: center;">
              <!-- Attachment / Actions Button -->
              <button on:click={triggerMultipleFileSelect} class="glass-panel" style="background: rgba(255,255,255,0.06); border: 1px solid var(--glass-border); color: var(--text-secondary); width: 44px; height: 44px; border-radius: 50%; display: flex; align-items: center; justify-content: center; cursor: pointer; transition: all 0.25s; flex-shrink: 0;" title={t.btnSendFiles}>
                <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" style="width: 20px; height: 20px;"><path d="M21.44 11.05l-9.19 9.19a6 6 0 0 1-8.49-8.49l9.19-9.19a4 4 0 0 1 5.66 5.66l-9.2 9.19a2 2 0 0 1-2.83-2.83l8.49-8.48"/></svg>
              </button>
              
              <button on:click={triggerFolderSelect} class="glass-panel" style="background: rgba(255,255,255,0.06); border: 1px solid var(--glass-border); color: var(--text-secondary); width: 44px; height: 44px; border-radius: 50%; display: flex; align-items: center; justify-content: center; cursor: pointer; transition: all 0.25s; flex-shrink: 0;" title={t.btnSendFolder}>
                <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" style="width: 20px; height: 20px;"><path d="M22 19a2 2 0 0 1-2 2H4a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h5l2 3h9a2 2 0 0 1 2 2z"/></svg>
              </button>

              <!-- Text Message Input -->
              <input 
                type="text" 
                placeholder="E2EE Güvenli mesajınızı yazın..." 
                bind:value={chatMessageInput} 
                on:keydown={handleChatKeyDown}
                style="flex: 1; background: var(--bg-primary); border: 1px solid var(--border-light); border-radius: 24px; padding: 12px 20px; color: var(--text-primary); font-family: var(--font-main); font-size: 0.95rem; outline: none; transition: all 0.25s;"
              />

              <!-- Send Button -->
              <button 
                on:click={handleSendChatMessage} 
                class="glowing" 
                style="background: var(--accent); color: white; width: 44px; height: 44px; border-radius: 50%; border: none; display: flex; align-items: center; justify-content: center; cursor: pointer; transition: all 0.25s; flex-shrink: 0; box-shadow: 0 4px 15px var(--accent-glow);"
                disabled={!chatMessageInput.trim()}
              >
                <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round" style="width: 18px; height: 18px; margin-left: 2px;"><line x1="22" y1="2" x2="11" y2="13"/><polygon points="22 2 15 22 11 13 2 9 22 2"/></svg>
              </button>
            </div>
          {:else}
            <div class="offline-footer-notice">
              {t.offlineNotice}
            </div>
          {/if}
        </footer>
      </div>
    {/if}
  </main>
</div>
{/if}

<!-- INCOMING REQUEST OVERLAY MODAL -->
{#if incomingRequest}
<div class="modal-overlay">
  <div class="modal-card glass-panel animate-slide">
    <div class="modal-header">
      <svg class="modal-lock" viewBox="0 0 24 24" fill="none" stroke="#00a884" stroke-width="2"><rect x="3" y="11" width="18" height="11" rx="2" ry="2"/><path d="M7 11V7a5 5 0 0 1 10 0v4"/></svg>
      <h2>{t.incomingTitle}</h2>
    </div>
    
    <div class="modal-body">
      <div class="sender-alert-avatar">
        {incomingRequest.peer_name[0].toUpperCase()}
      </div>
      <p class="request-desc">
        <strong class="glow-text">{incomingRequest.peer_name}</strong> {t.incomingDesc}
      </p>
      
      <div class="modal-file-card glass-panel">
        <span class="modal-filename">{incomingRequest.filename}</span>
        <span class="modal-filesize">{formatBytes(incomingRequest.filesize)}</span>
      </div>
 
      <div class="save-dir-selector">
        <label for="save-dir">{t.saveFolder}</label>
        <div class="dir-input-wrapper">
          <input type="text" id="save-dir" readonly bind:value={targetSaveDir}/>
          <button class="dir-select-btn" on:click={selectNewSaveDir}>{t.btnChange}</button>
        </div>
      </div>
    </div>
 
    <div class="modal-footer">
      <button class="modal-btn decline-btn" on:click={() => respondToTransfer(false)}>
        {t.declineBtn}
      </button>
      <button class="modal-btn accept-btn glowing" on:click={() => respondToTransfer(true)}>
        {t.acceptBtn}
      </button>
    </div>
  </div>
</div>
{/if}

{#if showSettings}
<div class="modal-overlay" style="z-index: 9999; backdrop-filter: blur(25px); background: rgba(11, 20, 26, 0.75);">
  <div class="modal-card glass-panel animate-slide" style="max-width: 520px; border: 1px solid rgba(255,255,255,0.08); padding: 32px; background: rgba(20, 32, 38, 0.85); box-shadow: 0 25px 60px rgba(0,0,0,0.6); max-height: 90vh; display: flex; flex-direction: column; overflow: hidden; border-radius: 16px;">
    
    <!-- Modal Header -->
    <div class="modal-header" style="justify-content: flex-start; gap: 12px; margin-bottom: 20px; border-bottom: 1px solid rgba(255,255,255,0.06); padding-bottom: 16px; align-items: center;">
      <svg class="modal-lock" viewBox="0 0 24 24" fill="none" stroke="var(--accent)" stroke-width="2.2" style="width: 22px; height: 22px;"><circle cx="12" cy="12" r="3"/><path d="M19.4 15a1.65 1.65 0 0 0 .33 1.82l.06.06a2 2 0 1 1-2.83 2.83l-.06-.06a1.65 1.65 0 0 0-1.82-.33 1.65 1.65 0 0 0-1 1.51V21a2 2 0 0 1-4 0v-.09A1.65 1.65 0 0 0 9 19.4a1.65 1.65 0 0 0-1.82.33l-.06.06a2 2 0 1 1-2.83 2.83l.06-.06a1.65 1.65 0 0 0 .33-1.82 1.65 1.65 0 0 0-1.51-1H3a2 2 0 0 1 0-4h.09A1.65 1.65 0 0 0 4.6 9a1.65 1.65 0 0 0-.33-1.82l-.06-.06a2 2 0 1 1 2.83-2.83l.06.06a1.65 1.65 0 0 0 1.82.33H9a1.65 1.65 0 0 0 1-1.51V3a2 2 0 0 1 4 0v.09a1.65 1.65 0 0 0 1 1.51 1.65 1.65 0 0 0 1.82-.33l.06-.06a2 2 0 1 1 2.83 2.83l-.06.06a1.65 1.65 0 0 0-.33 1.82V9a1.65 1.65 0 0 0 1.51 1H21a2 2 0 0 1 0 4h-.09a1.65 1.65 0 0 0-1.51 1z"/></svg>
      <h2 style="font-size: 1.35rem; font-weight: 800; color: white; margin: 0; font-family: var(--font-main); letter-spacing: -0.3px;">{t.settingsTitle}</h2>
    </div>

    <!-- Scrollable Modal Body -->
    <div class="modal-body" style="flex: 1; overflow-y: auto; padding-right: 8px; display: flex; flex-direction: column; gap: 20px; font-family: var(--font-main);">
      
      <!-- Warning Text -->
      <div style="background: rgba(234, 67, 53, 0.08); border: 1px solid rgba(234, 67, 53, 0.15); border-radius: 10px; padding: 14px 18px; color: #ff8f8f;">
        <div style="font-weight: 700; font-size: 0.85rem; display: flex; align-items: center; gap: 6px; margin-bottom: 6px;">
          {t.noBackupWarn}
        </div>
        <p style="font-size: 0.78rem; line-height: 1.5; color: rgba(255,255,255,0.7); font-weight: 400; margin: 0;">
          {t.settingsInfo}
        </p>
      </div>

      <!-- Language Selection -->
      <div style="display: flex; flex-direction: column; gap: 6px;">
        <label style="font-size: 0.75rem; font-weight: 700; color: var(--text-secondary); text-transform: uppercase; letter-spacing: 0.5px;">{t.langLabel}</label>
        <select bind:value={currentLang} style="background-color: var(--bg-secondary); border: 1px solid var(--glass-border); border-radius: 8px; padding: 10px 14px; color: var(--text-primary); outline: none; font-size: 0.88rem; font-family: var(--font-main); cursor: pointer; transition: border 0.2s;">
          <option value="tr">Türkçe (TR)</option>
          <option value="en">English (EN)</option>
        </select>
      </div>

      <!-- Username Input -->
      <div style="display: flex; flex-direction: column; gap: 6px;">
        <label style="font-size: 0.75rem; font-weight: 700; color: var(--text-secondary); text-transform: uppercase; letter-spacing: 0.5px;">{t.usernameLabel}</label>
        <input type="text" placeholder="Takma ad..." bind:value={tempUsername} style="background-color: var(--bg-secondary); border: 1px solid var(--glass-border); border-radius: 8px; padding: 12px 14px; color: white; outline: none; font-size: 0.88rem; font-family: var(--font-main); transition: border 0.25s;" on:focus={(e) => e.target.style.borderColor = 'var(--accent)'} on:blur={(e) => e.target.style.borderColor = 'var(--glass-border)'}/>
      </div>

      <!-- Accent Color Picker Grid -->
      <div style="display: flex; flex-direction: column; gap: 8px;">
        <label style="font-size: 0.75rem; font-weight: 700; color: var(--text-secondary); text-transform: uppercase; letter-spacing: 0.5px;">{t.colorLabel}</label>
        <div style="display: flex; gap: 10px; flex-wrap: wrap; background: rgba(0,0,0,0.15); padding: 12px; border-radius: 10px; border: 1px solid rgba(255,255,255,0.03);">
          {#each availableColors as col}
            <button style="background-color: {col}; width: 28px; height: 28px; border-radius: 50%; border: {tempColor === col ? '2.5px solid white' : '1px solid rgba(255,255,255,0.1)'}; cursor: pointer; transition: transform 0.2s, box-shadow 0.2s; box-shadow: {tempColor === col ? '0 0 8px ' + col : 'none'};" on:click={() => tempColor = col} on:mouseover={(e) => e.target.style.transform = 'scale(1.18)'} on:mouseleave={(e) => e.target.style.transform = 'scale(1)'}></button>
          {/each}
        </div>
      </div>

      <!-- Status Message Input -->
      <div style="display: flex; flex-direction: column; gap: 6px;">
        <label style="font-size: 0.75rem; font-weight: 700; color: var(--text-secondary); text-transform: uppercase; letter-spacing: 0.5px;">{t.statusLabel}</label>
        <input type="text" placeholder="Durumunuz..." bind:value={tempStatus} style="background-color: var(--bg-secondary); border: 1px solid var(--glass-border); border-radius: 8px; padding: 12px 14px; color: white; outline: none; font-size: 0.88rem; font-family: var(--font-main); transition: border 0.25s;" on:focus={(e) => e.target.style.borderColor = 'var(--accent)'} on:blur={(e) => e.target.style.borderColor = 'var(--glass-border)'}/>
      </div>

      <!-- Matchmaking Server Input -->
      <div style="display: flex; flex-direction: column; gap: 6px;">
        <label style="font-size: 0.75rem; font-weight: 700; color: var(--text-secondary); text-transform: uppercase; letter-spacing: 0.5px;">{t.serverLabel}</label>
        <input type="text" placeholder="örn: 127.0.0.1:12130 veya relay.domain.com:12130" bind:value={tempMatchmakingServer} style="background-color: var(--bg-secondary); border: 1px solid var(--glass-border); border-radius: 8px; padding: 12px 14px; color: white; outline: none; font-size: 0.88rem; font-family: var(--font-main); transition: border 0.25s;" on:focus={(e) => e.target.style.borderColor = 'var(--accent)'} on:blur={(e) => e.target.style.borderColor = 'var(--glass-border)'}/>
      </div>
    </div>

    <!-- Modal Footer Actions -->
    <div class="modal-footer" style="display: flex; gap: 12px; margin-top: 24px; border-top: 1px solid rgba(255,255,255,0.06); padding-top: 18px;">
      <button class="modal-btn decline-btn" on:click={() => showSettings = false} style="flex: 1; padding: 12px; font-weight: 600; font-family: var(--font-main);">
        {t.cancelBtn}
      </button>
      <button class="modal-btn accept-btn glowing" on:click={handleProfileUpdate} style="flex: 1; padding: 12px; font-weight: 700; font-family: var(--font-main); background-color: var(--accent); color: white;">
        {t.saveBtn}
      </button>
    </div>
  </div>
</div>
{/if}

{#if showAddFriendModal}
<div class="modal-overlay" style="z-index: 9999; backdrop-filter: blur(25px); background: rgba(11, 20, 26, 0.75);">
  <div class="modal-card glass-panel animate-slide" style="max-width: 520px; border: 1px solid rgba(255,255,255,0.08); padding: 32px; background: rgba(20, 32, 38, 0.85); box-shadow: 0 25px 60px rgba(0,0,0,0.6); max-height: 90vh; display: flex; flex-direction: column; overflow: hidden; border-radius: 16px;">
    
    <!-- Modal Header -->
    <div class="modal-header" style="justify-content: flex-start; gap: 12px; margin-bottom: 20px; border-bottom: 1px solid rgba(255,255,255,0.06); padding-bottom: 16px; align-items: center;">
      <svg viewBox="0 0 24 24" fill="none" stroke="var(--accent)" stroke-width="2.2" style="width: 22px; height: 22px;"><path d="M16 21v-2a4 4 0 0 0-4-4H5a4 4 0 0 0-4 4v2"/><circle cx="8.5" cy="7" r="4"/><line x1="20" y1="8" x2="20" y2="14"/><line x1="23" y1="11" x2="17" y2="11"/></svg>
      <h2 style="font-size: 1.35rem; font-weight: 800; color: white; margin: 0; font-family: var(--font-main); letter-spacing: -0.3px;">Yeni Arkadaş Ekle</h2>
    </div>

    <!-- Modal Body -->
    <div class="modal-body" style="flex: 1; display: flex; flex-direction: column; gap: 20px; font-family: var(--font-main);">
      
      <!-- Informational guidance -->
      <div style="background: rgba(0, 168, 132, 0.08); border: 1px solid rgba(0, 168, 132, 0.15); border-radius: 10px; padding: 14px 18px; color: #7bf1c3;">
        <div style="font-weight: 700; font-size: 0.85rem; display: flex; align-items: center; gap: 6px; margin-bottom: 6px;">
          Güvenli Arkadaş Ekleme
        </div>
        <p style="font-size: 0.78rem; line-height: 1.5; color: rgba(255,255,255,0.7); font-weight: 400; margin: 0;">
          Arkadaşınızın benzersiz kimliğini (Peer ID veya Public Key Hex) aşağıya girin. Ekleme tamamlandığında, arkadaşınız çevrimiçi olduğu anda otomatik olarak listenizde belirecektir.
        </p>
      </div>

      <!-- Peer ID / Public Key Input -->
      <div style="display: flex; flex-direction: column; gap: 6px;">
        <label style="font-size: 0.75rem; font-weight: 700; color: var(--text-secondary); text-transform: uppercase; letter-spacing: 0.5px;">Arkadaşın Kimliği (Peer ID)</label>
        <input 
          type="text" 
          placeholder="Örn: QB-5F9D-1C4E-7A8B..." 
          bind:value={addFriendInput} 
          style="background-color: var(--bg-secondary); border: 1px solid var(--glass-border); border-radius: 8px; padding: 12px 14px; color: white; outline: none; font-size: 0.88rem; font-family: var(--font-main); transition: border 0.25s;" 
          on:focus={(e) => e.target.style.borderColor = 'var(--accent)'} 
          on:blur={(e) => e.target.style.borderColor = 'var(--glass-border)'}
        />
      </div>
    </div>

    <!-- Modal Footer Actions -->
    <div class="modal-footer" style="display: flex; gap: 12px; margin-top: 24px; border-top: 1px solid rgba(255,255,255,0.06); padding-top: 18px;">
      <button class="modal-btn decline-btn" on:click={() => showAddFriendModal = false} style="flex: 1; padding: 12px; font-weight: 600; font-family: var(--font-main);">
        İptal
      </button>
      <button class="modal-btn accept-btn glowing" on:click={handleAddFriend} style="flex: 1; padding: 12px; font-weight: 700; font-family: var(--font-main); background-color: var(--accent); color: white;" disabled={!addFriendInput.trim()}>
        Arkadaşı Ekle
      </button>
    </div>
  </div>
</div>
{/if}

<style>
  /* Base transitions & animations */
  .animate-fade {
    animation: fadeIn 0.4s ease-out forwards;
  }
  .animate-slide {
    animation: slideIn 0.35s cubic-bezier(0.16, 1, 0.3, 1) forwards;
  }
  .animate-pulse {
    animation: pulseLogo 2.5s infinite;
  }

  @keyframes fadeIn {
    from { opacity: 0; }
    to { opacity: 1; }
  }
  @keyframes slideIn {
    from { opacity: 0; transform: translateY(12px); }
    to { opacity: 1; transform: translateY(0); }
  }
  @keyframes pulseLogo {
    0% { transform: scale(1); filter: drop-shadow(0 0 0 rgba(0, 168, 132, 0)); }
    50% { transform: scale(1.05); filter: drop-shadow(0 0 12px var(--accent-glow)); }
    100% { transform: scale(1); filter: drop-shadow(0 0 0 rgba(0, 168, 132, 0)); }
  }

  @keyframes bounce {
    0%, 100% { transform: translateY(0); }
    50% { transform: translateY(-8px); }
  }

  /* Auth Screen Layout */
  .auth-container {
    height: 100vh;
    width: 100vw;
    display: flex;
    justify-content: center;
    align-items: center;
    background: radial-gradient(circle at center, #111b21 0%, #0b141a 100%);
    padding: 20px;
  }

  .auth-card {
    width: 100%;
    max-width: 440px;
    padding: 40px;
    text-align: center;
  }

  .logo-area {
    margin-bottom: 30px;
  }

  .logo-shield {
    width: 60px;
    height: 60px;
    margin-bottom: 12px;
  }

  .logo-area h1 {
    font-size: 2.2rem;
    font-weight: 700;
    letter-spacing: -0.5px;
    color: var(--text-primary);
  }

  .logo-area .subtitle {
    font-size: 0.9rem;
    color: var(--text-secondary);
    margin-top: 4px;
  }

  .auth-form h2 {
    font-size: 1.4rem;
    font-weight: 600;
    margin-bottom: 8px;
    color: var(--text-primary);
  }

  .form-info {
    font-size: 0.8rem;
    color: var(--text-secondary);
    margin-bottom: 24px;
    line-height: 1.4;
  }

  .input-group {
    text-align: left;
    margin-bottom: 18px;
  }

  .input-group label {
    display: block;
    font-size: 0.8rem;
    font-weight: 500;
    color: var(--text-secondary);
    margin-bottom: 6px;
    text-transform: uppercase;
    letter-spacing: 0.5px;
  }

  .input-group input {
    width: 100%;
    background-color: var(--bg-secondary);
    border: 1px solid var(--glass-border);
    border-radius: 8px;
    padding: 12px 14px;
    color: var(--text-primary);
    font-family: var(--font-main);
    font-size: 0.95rem;
    outline: none;
    transition: all 0.25s ease;
  }

  .input-group input:focus {
    border-color: var(--accent);
    box-shadow: 0 0 10px var(--accent-glow);
  }

  .password-wrapper {
    position: relative;
    width: 100%;
  }

  .eye-btn {
    position: absolute;
    right: 12px;
    top: 50%;
    transform: translateY(-50%);
    background: none;
    border: none;
    color: var(--text-secondary);
    cursor: pointer;
    padding: 4px;
    display: flex;
    align-items: center;
  }

  .eye-btn svg {
    width: 18px;
    height: 18px;
  }

  .eye-btn:hover {
    color: var(--text-primary);
  }

  .alert {
    padding: 10px 14px;
    border-radius: 8px;
    font-size: 0.85rem;
    margin-bottom: 18px;
    text-align: left;
    line-height: 1.4;
  }

  .alert-error {
    background-color: rgba(234, 67, 53, 0.15);
    border: 1px solid rgba(234, 67, 53, 0.3);
    color: #ff8f8f;
  }

  .submit-btn {
    width: 100%;
    background-color: var(--accent);
    color: white;
    border: none;
    border-radius: 8px;
    padding: 14px;
    font-family: var(--font-main);
    font-size: 1rem;
    font-weight: 600;
    cursor: pointer;
    transition: all 0.25s;
    display: flex;
    justify-content: center;
    align-items: center;
    gap: 8px;
  }

  .submit-btn:hover:not(:disabled) {
    transform: translateY(-1px);
    box-shadow: 0 4px 15px var(--accent-glow);
  }

  .submit-btn:active:not(:disabled) {
    transform: translateY(0);
  }

  .submit-btn:disabled {
    opacity: 0.6;
    cursor: not-allowed;
  }

  .glowing {
    position: relative;
    overflow: hidden;
  }

  .glowing::after {
    content: '';
    position: absolute;
    top: -50%;
    left: -50%;
    width: 200%;
    height: 200%;
    background: linear-gradient(
      to bottom right,
      rgba(255, 255, 255, 0) 0%,
      rgba(255, 255, 255, 0.15) 50%,
      rgba(255, 255, 255, 0) 100%
    );
    transform: rotate(45deg);
    transition: all 0.5s;
    animation: shine 4s infinite linear;
  }

  @keyframes shine {
    0% { transform: translate(-30%, -30%) rotate(45deg); }
    100% { transform: translate(30%, 30%) rotate(45deg); }
  }

  .btn-loader {
    width: 16px;
    height: 16px;
    border: 2px solid rgba(255,255,255,0.4);
    border-top-color: white;
    border-radius: 50%;
    animation: spin 1s infinite linear;
  }

  @keyframes spin {
    to { transform: rotate(360deg); }
  }

  /* App Dashboard Layout */
  .app-layout {
    height: 100vh;
    width: 100vw;
    display: flex;
    background-color: var(--bg-primary);
  }

  /* Sidebar Design */
  .sidebar {
    width: 350px;
    min-width: 300px;
    border-right: 1px solid var(--border-light);
    display: flex;
    flex-direction: column;
    background-color: var(--bg-secondary);
  }

  .sidebar-header {
    padding: 16px 20px;
    display: flex;
    justify-content: space-between;
    align-items: center;
    border-bottom: 1px solid var(--border-light);
  }

  .user-profile {
    display: flex;
    align-items: center;
    gap: 12px;
  }

  .avatar {
    width: 42px;
    height: 42px;
    border-radius: 50%;
    display: flex;
    justify-content: center;
    align-items: center;
    font-weight: 700;
    color: white;
    box-shadow: 0 4px 10px rgba(0,0,0,0.15);
  }

  .profile-info h3 {
    font-size: 0.95rem;
    font-weight: 600;
    color: var(--text-primary);
  }

  .status-badge {
    font-size: 0.75rem;
    color: var(--accent);
    display: flex;
    align-items: center;
    gap: 5px;
    margin-top: 2px;
  }

  .logout-btn {
    background: none;
    border: none;
    color: var(--text-secondary);
    cursor: pointer;
    padding: 8px;
    border-radius: 50%;
    transition: all 0.25s;
  }

  .logout-btn:hover {
    color: #ea4335;
    background-color: rgba(234, 67, 53, 0.1);
  }

  .logout-btn svg {
    width: 20px;
    height: 20px;
  }

  .search-container {
    padding: 10px 16px;
  }

  .search-bar {
    width: 100%;
    background-color: var(--bg-primary);
    border: 1px solid var(--border-light);
    border-radius: 8px;
    padding: 8px 12px;
    display: flex;
    align-items: center;
    gap: 10px;
  }

  .search-icon {
    width: 16px;
    height: 16px;
    color: var(--text-secondary);
  }

  .search-bar input {
    background: none;
    border: none;
    color: var(--text-primary);
    outline: none;
    width: 100%;
    font-family: var(--font-main);
    font-size: 0.85rem;
  }

  .contacts-list {
    flex: 1;
    overflow-y: auto;
    padding: 8px 0;
  }

  .empty-state {
    padding: 40px 24px;
    text-align: center;
    color: var(--text-secondary);
  }

  .radar-icon {
    width: 48px;
    height: 48px;
    color: var(--text-secondary);
    margin-bottom: 16px;
  }

  .empty-state p {
    font-size: 0.9rem;
    font-weight: 500;
    color: var(--text-primary);
    margin-bottom: 4px;
  }

  .empty-state .subtext {
    font-size: 0.75rem;
    line-height: 1.4;
    display: block;
  }

  .contact-card {
    width: 100%;
    background: none;
    border: none;
    border-bottom: 1px solid rgba(255,255,255,0.015);
    padding: 14px 20px;
    display: flex;
    align-items: center;
    gap: 14px;
    cursor: pointer;
    text-align: left;
    font-family: var(--font-main);
    transition: all 0.25s cubic-bezier(0.4, 0, 0.2, 1);
    position: relative;
    overflow: hidden;
  }

  .contact-card::before {
    content: '';
    position: absolute;
    left: 0;
    top: 20%;
    height: 60%;
    width: 3px;
    background-color: var(--accent);
    border-radius: 0 4px 4px 0;
    transform: scaleY(0);
    transition: transform 0.2s cubic-bezier(0.4, 0, 0.2, 1);
  }

  .contact-card:hover {
    background-color: rgba(255, 255, 255, 0.02);
  }

  .contact-card.active {
    background-color: rgba(0, 168, 132, 0.06);
  }

  .contact-card.active::before {
    transform: scaleY(1);
  }

  .contact-details {
    flex: 1;
  }

  .contact-row {
    display: flex;
    justify-content: space-between;
    align-items: center;
    margin-bottom: 4px;
  }

  .contact-name {
    font-size: 0.95rem;
    font-weight: 600;
    color: var(--text-primary);
  }

  .contact-time {
    font-size: 0.7rem;
    color: var(--accent);
    font-weight: 500;
  }

  .contact-ip {
    font-size: 0.75rem;
    color: var(--text-secondary);
  }

  .contact-card .lock-icon {
    font-size: 0.7rem;
    color: var(--text-secondary);
    background-color: rgba(255,255,255,0.05);
    padding: 2px 6px;
    border-radius: 4px;
  }

  /* Chat Area Right Panel */
  .chat-area {
    flex: 1;
    display: flex;
    flex-direction: column;
    background-color: var(--bg-chat);
  }

  .splash-screen {
    flex: 1;
    display: flex;
    justify-content: center;
    align-items: center;
    padding: 40px;
    background: radial-gradient(circle at center, #111b21 0%, #0b141a 100%);
  }

  .splash-content {
    max-width: 500px;
    padding: 40px;
    text-align: center;
  }

  .splash-shield {
    width: 72px;
    height: 72px;
    color: var(--text-secondary);
    margin-bottom: 20px;
  }

  .splash-title {
    font-size: 1.6rem;
    font-weight: 700;
    margin-top: 10px;
    color: var(--text-primary);
  }

  .splash-desc {
    font-size: 0.9rem;
    color: var(--text-secondary);
    line-height: 1.5;
    margin: 16px 0 24px;
  }

  .security-badge {
    display: inline-flex;
    align-items: center;
    gap: 8px;
    font-size: 0.8rem;
    color: var(--text-secondary);
    background-color: rgba(255, 255, 255, 0.03);
    padding: 8px 16px;
    border-radius: 20px;
    border: 1px solid var(--border-light);
  }

  .security-badge svg {
    width: 14px;
    height: 14px;
    color: var(--accent);
  }

  /* Chat Window Active View */
  .chat-window {
    flex: 1;
    display: flex;
    flex-direction: column;
    height: 100%;
    overflow: hidden;
  }

  .chat-header {
    padding: 14px 24px;
    background-color: var(--bg-secondary);
    border-bottom: 1px solid var(--border-light);
    display: flex;
    justify-content: space-between;
    align-items: center;
  }

  .peer-profile {
    display: flex;
    align-items: center;
    gap: 12px;
  }

  .peer-meta h2 {
    font-size: 1rem;
    font-weight: 600;
    color: var(--text-primary);
  }

  .peer-status {
    font-size: 0.75rem;
    color: var(--text-secondary);
    display: flex;
    align-items: center;
    gap: 5px;
  }

  .active-dot {
    width: 6px;
    height: 6px;
    background-color: var(--accent);
    border-radius: 50%;
  }

  .offline-dot {
    width: 6px;
    height: 6px;
    background-color: #ea4335;
    border-radius: 50%;
  }

  .security-lock {
    display: flex;
    align-items: center;
    gap: 6px;
    font-size: 0.8rem;
    color: var(--accent);
    background-color: rgba(0, 168, 132, 0.08);
    padding: 6px 12px;
    border-radius: 20px;
    border: 1px solid rgba(0, 168, 132, 0.15);
  }

  .header-lock-icon {
    width: 13px;
    height: 13px;
  }

  .transfer-history {
    flex: 1;
    overflow-y: auto;
    padding: 24px;
    display: flex;
    flex-direction: column;
    gap: 16px;
  }

  .security-warning {
    align-self: center;
    max-width: 500px;
    padding: 12px 16px;
    font-size: 0.75rem;
    color: var(--text-secondary);
    text-align: center;
    display: flex;
    align-items: center;
    gap: 10px;
    margin-bottom: 8px;
    border-radius: 8px;
  }

  .security-warning svg {
    width: 16px;
    height: 16px;
    color: var(--accent);
    flex-shrink: 0;
  }

  .no-transfers {
    flex: 1;
    display: flex;
    flex-direction: column;
    justify-content: center;
    align-items: center;
    color: var(--text-secondary);
    padding: 40px;
    text-align: center;
  }

  .no-transfers svg {
    width: 56px;
    height: 56px;
    color: rgba(255,255,255,0.06);
    margin-bottom: 16px;
  }

  .no-transfers p {
    font-size: 0.95rem;
    font-weight: 500;
    color: var(--text-primary);
    margin-bottom: 4px;
  }

  .no-transfers span {
    font-size: 0.78rem;
    max-width: 320px;
    line-height: 1.4;
  }

  /* Bubble Styling */
  .bubble-row {
    display: flex;
    width: 100%;
  }

  .sender-row {
    justify-content: flex-end;
  }

  .receiver-row {
    justify-content: flex-start;
  }

  .bubble {
    max-width: 380px;
    min-width: 250px;
    padding: 12px;
    border-radius: 12px;
  }

  .bubble-sender {
    background-color: var(--bubble-out);
    border-bottom-right-radius: 2px;
  }

  .bubble-receiver {
    background-color: var(--bubble-in);
    border-bottom-left-radius: 2px;
  }

  /* File Card inside Bubble */
  .file-card {
    display: flex;
    align-items: center;
    gap: 12px;
  }

  .file-icon-bg {
    width: 44px;
    height: 44px;
    border-radius: 8px;
    display: flex;
    justify-content: center;
    align-items: center;
    color: white;
    flex-shrink: 0;
    box-shadow: 0 2px 6px rgba(0,0,0,0.15);
  }

  .file-icon-bg.image { background: linear-gradient(135deg, #34a853, #1e7e34); }
  .file-icon-bg.video { background: linear-gradient(135deg, #ea4335, #bd2130); }
  .file-icon-bg.audio { background: linear-gradient(135deg, #fbbc05, #d39e00); }
  .file-icon-bg.archive { background: linear-gradient(135deg, #9060eb, #6c3dbf); }
  .file-icon-bg.document { background: linear-gradient(135deg, #4285f4, #1b62db); }
  .file-icon-bg.file { background: linear-gradient(135deg, #6c757d, #495057); }

  .file-icon-bg svg {
    width: 22px;
    height: 22px;
  }

  .file-details {
    flex: 1;
    min-width: 0; /* allows text truncation */
    display: flex;
    flex-direction: column;
  }

  .filename {
    font-size: 0.9rem;
    font-weight: 600;
    color: var(--text-primary);
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }

  .filesize {
    font-size: 0.75rem;
    color: var(--text-secondary);
    margin-top: 2px;
  }

  /* Progress inside bubble */
  .progress-container {
    margin-top: 12px;
    padding-top: 10px;
    border-top: 1px solid rgba(255,255,255,0.06);
  }

  .progress-header {
    display: flex;
    justify-content: space-between;
    font-size: 0.75rem;
    color: var(--text-secondary);
    margin-bottom: 6px;
  }

  .progress-bar-bg {
    width: 100%;
    height: 5px;
    background-color: rgba(255,255,255,0.1);
    border-radius: 4px;
    overflow: hidden;
  }

  .progress-bar-fill {
    height: 100%;
    background-color: var(--accent);
    box-shadow: 0 0 8px var(--accent-glow);
    border-radius: 4px;
    transition: width 0.3s ease;
  }

  .status-indicator {
    margin-top: 10px;
    padding-top: 8px;
    border-top: 1px solid rgba(255,255,255,0.05);
    display: flex;
    align-items: center;
    gap: 6px;
    font-size: 0.75rem;
    color: var(--text-secondary);
  }

  .status-indicator.pending .dot {
    width: 6px;
    height: 6px;
    background-color: #fbbc05;
    border-radius: 50%;
  }

  .check-icon {
    width: 14px;
    height: 14px;
    stroke: var(--accent);
  }

  .completed-meta {
    display: flex;
    align-items: center;
    gap: 6px;
    flex: 1;
  }

  .action-buttons {
    display: flex;
    gap: 6px;
  }

  .bubble-btn {
    background-color: rgba(255,255,255,0.08);
    color: var(--text-primary);
    border: 1px solid rgba(255,255,255,0.04);
    border-radius: 4px;
    padding: 4px 10px;
    font-family: var(--font-main);
    font-size: 0.72rem;
    font-weight: 500;
    cursor: pointer;
    transition: all 0.2s;
  }

  .bubble-btn:hover {
    background-color: var(--accent);
    border-color: var(--accent);
  }

  .alert-icon {
    width: 14px;
    height: 14px;
  }

  .status-indicator.declined span {
    color: #ff8f8f;
  }

  .status-indicator.failed span {
    color: #ff8f8f;
  }

  /* Chat Footer Button */
  .chat-footer-action {
    padding: 16px 24px;
    background-color: var(--bg-secondary);
    border-top: 1px solid var(--border-light);
    display: flex;
    justify-content: center;
  }

  .send-file-button {
    width: 100%;
    max-width: 400px;
    background-color: var(--accent);
    color: white;
    border: none;
    border-radius: 24px;
    padding: 12px 24px;
    font-family: var(--font-main);
    font-size: 0.95rem;
    font-weight: 600;
    cursor: pointer;
    display: flex;
    justify-content: center;
    align-items: center;
    gap: 8px;
    transition: all 0.25s;
  }

  .send-file-button:hover {
    transform: translateY(-1px);
    box-shadow: 0 4px 15px var(--accent-glow);
  }

  .send-file-button svg {
    width: 18px;
    height: 18px;
  }

  .offline-footer-notice {
    font-size: 0.85rem;
    color: var(--text-secondary);
    background-color: rgba(234, 67, 53, 0.05);
    border: 1px solid rgba(234, 67, 53, 0.1);
    padding: 10px 20px;
    border-radius: 20px;
    text-align: center;
    width: 100%;
    max-width: 400px;
  }

  /* Modal Overlay styling */
  .modal-overlay {
    position: fixed;
    top: 0;
    left: 0;
    width: 100vw;
    height: 100vh;
    background-color: rgba(0,0,0,0.6);
    backdrop-filter: blur(8px);
    display: flex;
    justify-content: center;
    align-items: center;
    z-index: 1000;
    padding: 20px;
  }

  .modal-card {
    width: 100%;
    max-width: 460px;
    padding: 30px;
    text-align: center;
  }

  .modal-header {
    display: flex;
    align-items: center;
    justify-content: center;
    gap: 10px;
    margin-bottom: 20px;
  }

  .modal-lock {
    width: 24px;
    height: 24px;
  }

  .modal-header h2 {
    font-size: 1.3rem;
    font-weight: 700;
    color: var(--text-primary);
  }

  .sender-alert-avatar {
    width: 54px;
    height: 54px;
    border-radius: 50%;
    background: linear-gradient(135deg, #00a884, #005c4b);
    display: inline-flex;
    justify-content: center;
    align-items: center;
    font-size: 1.4rem;
    font-weight: 700;
    color: white;
    margin-bottom: 12px;
    box-shadow: 0 4px 10px rgba(0,0,0,0.15);
  }

  .request-desc {
    font-size: 0.95rem;
    color: var(--text-secondary);
    line-height: 1.4;
  }

  .modal-file-card {
    margin: 18px 0;
    padding: 14px 20px;
    display: flex;
    flex-direction: column;
    gap: 4px;
    text-align: center;
    background-color: rgba(255,255,255,0.02);
  }

  .modal-filename {
    font-size: 1rem;
    font-weight: 600;
    color: var(--text-primary);
    word-break: break-all;
  }

  .modal-filesize {
    font-size: 0.8rem;
    color: var(--accent);
    font-weight: 500;
  }

  .save-dir-selector {
    text-align: left;
    margin-bottom: 24px;
  }

  .save-dir-selector label {
    display: block;
    font-size: 0.78rem;
    font-weight: 500;
    color: var(--text-secondary);
    margin-bottom: 6px;
    text-transform: uppercase;
    letter-spacing: 0.5px;
  }

  .dir-input-wrapper {
    display: flex;
    gap: 8px;
  }

  .dir-input-wrapper input {
    flex: 1;
    background-color: var(--bg-secondary);
    border: 1px solid var(--glass-border);
    border-radius: 6px;
    padding: 10px 12px;
    color: var(--text-primary);
    font-family: var(--font-main);
    font-size: 0.82rem;
    outline: none;
  }

  .dir-select-btn {
    background-color: var(--bg-panel);
    border: 1px solid var(--glass-border);
    color: var(--text-primary);
    border-radius: 6px;
    padding: 0 14px;
    font-family: var(--font-main);
    font-size: 0.85rem;
    font-weight: 500;
    cursor: pointer;
    transition: all 0.2s;
  }

  .dir-select-btn:hover {
    background-color: var(--bg-hover);
  }

  .modal-footer {
    display: flex;
    gap: 12px;
  }

  .modal-btn {
    flex: 1;
    padding: 12px;
    border-radius: 8px;
    font-family: var(--font-main);
    font-size: 0.95rem;
    font-weight: 600;
    cursor: pointer;
    transition: all 0.2s;
    border: none;
  }

  .decline-btn {
    background-color: rgba(234, 67, 53, 0.12);
    color: #ff8f8f;
    border: 1px solid rgba(234, 67, 53, 0.2);
  }

  .decline-btn:hover {
    background-color: rgba(234, 67, 53, 0.2);
  }

  .accept-btn {
    background-color: var(--accent);
    color: white;
  }

  .accept-btn:hover {
    box-shadow: 0 4px 15px var(--accent-glow);
  }
</style>
