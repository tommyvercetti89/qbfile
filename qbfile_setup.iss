; ----------------------------------------------------------------------------------
; QBFile - Professional Windows Installer Setup Script
; Created by Antigravity AI
; Target: Inno Setup 6.0+
; ----------------------------------------------------------------------------------

#define MyAppName "QBFile"
#define MyAppVersion "1.0.0"
#define MyAppPublisher "QBFile Team"
#define MyAppExeName "qbfile.exe"
#define MyAppIconPath "build\windows\icon.ico"

[Setup]
; Unique Cryptographic App ID (ensures clean upgrades/re-installs without duplicates)
AppId={{5D0B2E1D-D65D-4D3B-A87F-E65C47FF5A68}
AppName={#MyAppName}
AppVersion={#MyAppVersion}
AppPublisher={#MyAppPublisher}
DefaultDirName={autopf}\{#MyAppName}
DisableProgramGroupPage=yes
AppendDefaultDirName=yes

; Output Configuration
OutputDir=build\bin
OutputBaseFilename=qbfile_installer_x64
SetupIconFile={#MyAppIconPath}
UninstallDisplayIcon={app}\{#MyAppExeName}

; Design & Compression Settings (Sleek Modern Setup style with LZMA2 ultra compression)
WizardStyle=modern
Compression=lzma2/max
SolidCompression=yes
ArchitecturesAllowed=x64
ArchitecturesInstallIn64BitMode=x64

; Privileges (Requires Admin to modify Program Files and Registry HKCR keys)
PrivilegesRequired=admin

[Languages]
Name: "english"; MessagesFile: "compiler:Default.isl"
Name: "turkish"; MessagesFile: "compiler:Languages\Turkish.isl"

[Tasks]
Name: "desktopicon"; Description: "{cm:CreateDesktopIcon}"; GroupDescription: "{cm:AdditionalIcons}"; Flags: unchecked

[Files]
; Compile the main high-performance compiled binary
Source: "build\bin\qbfile.exe"; DestDir: "{app}"; Flags: ignoreversion

[Icons]
Name: "{autoprograms}\{#MyAppName}"; Filename: "{app}\{#MyAppExeName}"
Name: "{userdesktop}\{#MyAppName}"; Filename: "{app}\{#MyAppExeName}"; Tasks: desktopicon

[Run]
Filename: "{app}\{#MyAppExeName}"; Description: "{cm:LaunchProgram,{#StringChange(MyAppName, '&', '&&')}}"; Flags: nowait postinstall skipifsilent

[Registry]
; ==================================================================================
; WINDOWS CONTEXT MENU (RIGHT-CLICK) INTEGRATION
; ==================================================================================

; --- RIGHT-CLICK MENU FOR ALL FILES ---
Root: HKCR; Subkey: "*\shell\QBFile"; ValueType: string; ValueName: ""; ValueData: "QBFile ile Gönder"; Flags: uninsdeletekey; Languages: turkish
Root: HKCR; Subkey: "*\shell\QBFile"; ValueType: string; ValueName: ""; ValueData: "Send with QBFile"; Flags: uninsdeletekey; Languages: english
Root: HKCR; Subkey: "*\shell\QBFile"; ValueType: string; ValueName: "Icon"; ValueData: "{app}\{#MyAppExeName}"; Flags: uninsdeletekey
Root: HKCR; Subkey: "*\shell\QBFile\command"; ValueType: string; ValueName: ""; ValueData: """{app}\{#MyAppExeName}"" ""%1"""; Flags: uninsdeletekey

; --- RIGHT-CLICK MENU FOR DIRECTORIES / FOLDERS ---
Root: HKCR; Subkey: "Directory\shell\QBFile"; ValueType: string; ValueName: ""; ValueData: "QBFile ile Gönder"; Flags: uninsdeletekey; Languages: turkish
Root: HKCR; Subkey: "Directory\shell\QBFile"; ValueType: string; ValueName: ""; ValueData: "Send with QBFile"; Flags: uninsdeletekey; Languages: english
Root: HKCR; Subkey: "Directory\shell\QBFile"; ValueType: string; ValueName: "Icon"; ValueData: "{app}\{#MyAppExeName}"; Flags: uninsdeletekey
Root: HKCR; Subkey: "Directory\shell\QBFile\command"; ValueType: string; ValueName: ""; ValueData: """{app}\{#MyAppExeName}"" ""%1"""; Flags: uninsdeletekey
