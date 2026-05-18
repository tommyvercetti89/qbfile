# QBFile - Çöpçatan & Kör Güvenli Aktarım (Relay) Sunucusu Kurulum Rehberi

Bu klasör, QBFile uygulamasının internet üzerinden (LAN dışı) iletişim kurmasını sağlayan yönlendirici sunucu dosyalarını içerir. 

Sunucunun tek görevi; cihazları birbiriyle tanıştırmak (matchmaking) ve arkasından gelen uçtan uca şifreli dosya veri paketlerini (relay) kör bir şekilde (veri içeriğini görmeden) hedefe iletmektir.

---

## 📂 Dosya Yapısı

*   `qbfile_server_linux_arm64` ➡️ **Oracle Cloud ARM64 (aarch64)**, Raspberry Pi veya diğer ARM tabanlı sunucular için çalıştırılabilir dosya.
*   `qbfile_server_linux_amd64` ➡️ Standart Intel/AMD (x86_64) Linux sunucular için çalıştırılabilir dosya.
*   `qbfile_server_windows.exe` ➡️ Windows Server işletim sistemleri için çalıştırılabilir dosya.
*   `main.go` ➡️ Sunucunun Go kaynak kodu.

---

## 🐧 1. Linux Sunucularda Kurulum ve Çalıştırma (Önerilen)

En popüler Linux sunucularda (Ubuntu / Debian) kurulumu en profesyonel şekilde tamamlamak için aşağıdaki adımları izleyin. Sunucuda **Go kurulu olmasına gerek yoktur**.

### Adım 1: Doğru Dosyayı Sunucuya Yükleyin
Sunucunuzun işlemci mimarisine uygun olan dosyayı sunucunuza yükleyin (Örn: `/opt/qbfile/` klasörüne):
*   **Oracle ARM / Ampere Sunucular İçin**: `qbfile_server_linux_arm64` dosyasını yükleyin ve ismini kolaylık için `qbfile_server_linux` yapın.
*   **Standart Intel/AMD Sunucular İçin**: `qbfile_server_linux_amd64` dosyasını yükleyin ve ismini kolaylık için `qbfile_server_linux` yapın.

### Adım 2: Çalıştırma Yetkisi Verin
Terminal üzerinden dosyanın bulunduğu klasöre giderek çalıştırma iznini verin:
```bash
chmod +x qbfile_server_linux
```

### Adım 3: Servis Olarak Yapılandırın (Ömür Boyu Arka Planda Kesintisiz Çalıştırma)
Terminal kapatıldığında sunucunun kapanmaması ve sunucu yeniden başlatıldığında otomatik olarak devreye girmesi için bir Linux Servisi oluşturacağız.

1. Şu komutla yeni bir servis dosyası açın:
   ```bash
   sudo nano /etc/systemd/system/qbfile-server.service
   ```

2. Açılan editöre aşağıdaki şablonu yapıştırın (dosya yolunu kendi yüklediğiniz yere göre güncelleyin):
   ```ini
   [Unit]
   Description=QBFile P2P Matchmaking and Relay Server
   After=network.target

   [Service]
   Type=simple
   User=root
   WorkingDirectory=/opt/qbfile
   ExecStart=/opt/qbfile/qbfile_server_linux
   Restart=always
   RestartSec=5

   [Install]
   WantedBy=multi-user.target
   ```

3. `Ctrl + O` ardından `Enter` ile kaydedip, `Ctrl + X` ile editörden çıkın.

4. Servisi aktif edin ve başlatın:
   ```bash
   sudo systemctl daemon-reload
   sudo systemctl enable qbfile-server
   sudo systemctl start qbfile-server
   ```

5. Servisin durumunu kontrol edin:
   ```bash
   sudo systemctl status qbfile-server
   ```

Artık sunucunuz arka planda ömür boyu çalışacaktır!

---

## 🪟 2. Windows Sunucularda Kurulum ve Çalıştırma

1. `qbfile_server_windows.exe` dosyasını sunucunuza kopyalayın.
2. PowerShell veya Komut İstemi (CMD) açarak dosyanın bulunduğu konuma gidin.
3. Çalıştırmak için şu komutu yazın:
   ```cmd
   .\qbfile_server_windows.exe
   ```
*(Sürekli çalışması için uygulamayı bir Windows Servisi haline getirebilir veya arka planda açık bırakabilirsiniz.)*

---

## 🔒 3. Güvenlik Duvarı (Firewall) Ayarları

Sunucunun istemcilerle haberleşebilmesi için **TCP 12130** portunun dış dünyaya açık olması gerekmektedir.

### Ubuntu (UFW) Güvenlik Duvarı kullanıyorsanız:
```bash
sudo ufw allow 12130/tcp
sudo ufw reload
```

### AWS / DigitalOcean / Azure Kullanıyorsanız:
Sunucu sağlayıcınızın yönetim paneline (Security Groups / Firewalls) giderek **Inbound (Gelen)** kurallarına şu kuralı ekleyin:
*   **Port**: `12130`
*   **Protokol**: `TCP`
*   **Kaynak (Source)**: `0.0.0.0/0` (Herkese Açık)

---

## ⚙️ Uygulamayı Dağıtırken Yapılması Gerekenler
Kullanıcılarınızın bu sunucuya otomatik olarak bağlanması için istemci kodundaki (`wan_client.go`) varsayılan sunucu adresini kendi sunucu IP'niz ile değiştirip derlemeniz yeterlidir:

```go
// Default WAN Matchmaking server settings
const (
	DefaultWANServer = "SENIN_SUNUCU_IP_ADRESIN:12130"
)
```
