#define MyVersion "0.1.0-beta.1"
#define MyName "AussieTV"

[Setup]
AppId={{1A8E8A50-4D3D-4C9D-BE1B-9D2BFD982E66}
AppName={#MyName}
AppVersion={#MyVersion}
AppVerName=AussieTV {#MyVersion}
AppPublisher=AussieTV contributors
AppPublisherURL=https://github.com/zidanerick/AussieTV
AppSupportURL=https://github.com/zidanerick/AussieTV/issues
DefaultDirName={localappdata}\Programs\AussieTV
DefaultGroupName=AussieTV
DisableProgramGroupPage=yes
PrivilegesRequired=lowest
UninstallDisplayIcon={app}\AussieTV.exe
SetupIconFile=..\assets\AussieTV.ico
OutputDir=..\dist\release
OutputBaseFilename=AussieTV-Setup-v{#MyVersion}-win-x64
Compression=lzma2
SolidCompression=yes
WizardStyle=modern
MinVersion=10.0
ArchitecturesAllowed=x64compatible
CloseApplications=yes
RestartApplications=no
LicenseFile=..\LICENSE

[Files]
Source: "..\dist\windows\AussieTV.exe"; DestDir: "{app}"; Flags: ignoreversion
Source: "..\dist\windows\AussieTV-Diagnostics.exe"; DestDir: "{app}"; Flags: ignoreversion
Source: "..\README.txt"; DestDir: "{app}"; Flags: ignoreversion
Source: "..\config\teletext.json"; DestDir: "{app}"; DestName: "teletext.json"; Flags: onlyifdoesntexist
Source: "..\tools\ffmpeg\fetch-windows.ps1"; DestDir: "{app}\tools\ffmpeg"; Flags: ignoreversion
Source: "..\UPDATE-FFmpeg.cmd"; DestDir: "{app}"; Flags: ignoreversion
Source: "..\ffmpeg\NOTICE.txt"; DestDir: "{app}\ffmpeg"; Flags: ignoreversion
Source: "..\ffmpeg\GPL-3.0.txt"; DestDir: "{app}\ffmpeg"; Flags: ignoreversion

[Tasks]
Name: "ffmpeg"; Description: "Download or update FFmpeg for experimental MPEG-TS and Teletext (internet required)"; Flags: checkedonce
Name: "desktopicon"; Description: "Create a desktop shortcut"; GroupDescription: "Shortcuts:"; Flags: unchecked

[Icons]
Name: "{group}\AussieTV"; Filename: "{app}\AussieTV.exe"
Name: "{group}\Update FFmpeg"; Filename: "{app}\UPDATE-FFmpeg.cmd"; WorkingDir: "{app}"
Name: "{group}\AussieTV diagnostics"; Filename: "{app}\AussieTV-Diagnostics.exe"
Name: "{autodesktop}\AussieTV"; Filename: "{app}\AussieTV.exe"; Tasks: desktopicon

[Run]
Filename: "{app}\AussieTV.exe"; Description: "Launch AussieTV"; Flags: nowait postinstall skipifsilent unchecked

[Code]
var
  FFmpegPage: TInputDirWizardPage;
  DefaultFFmpegDir: String;

procedure InitializeWizard();
begin
  FFmpegPage := CreateInputDirPage(wpSelectTasks,
    'FFmpeg location',
    'Choose where FFmpeg should be installed or updated.',
    'Optional FFmpeg is downloaded over HTTPS and checked against its published SHA-256.' + #13#10 +
    'Leave this folder as-is for a self-contained AussieTV installation.' + #13#10 +
    'You can update FFmpeg later from the Start Menu.',
    False, '');
  FFmpegPage.Add('FFmpeg folder:');
  DefaultFFmpegDir := ExpandConstant('{app}\ffmpeg');
  FFmpegPage.Values[0] := DefaultFFmpegDir;
end;

function ShouldSkipPage(PageID: Integer): Boolean;
begin
  Result := (PageID = FFmpegPage.ID) and not WizardIsTaskSelected('ffmpeg');
end;

procedure CurPageChanged(CurPageID: Integer);
begin
  if CurPageID = FFmpegPage.ID then begin
    // Follow the user's chosen app directory until the FFmpeg folder is customized.
    if FFmpegPage.Values[0] = DefaultFFmpegDir then begin
      DefaultFFmpegDir := ExpandConstant('{app}\ffmpeg');
      FFmpegPage.Values[0] := DefaultFFmpegDir;
    end;
  end;
end;

procedure CurStepChanged(CurStep: TSetupStep);
var
  ResultCode: Integer;
  Params: String;
  Destination: String;
begin
  if (CurStep <> ssPostInstall) or not WizardIsTaskSelected('ffmpeg') then Exit;
  Destination := FFmpegPage.Values[0];
  if Destination = '' then Destination := ExpandConstant('{app}\ffmpeg');
  Params := '-NoProfile -ExecutionPolicy Bypass -File "' + ExpandConstant('{app}\tools\ffmpeg\fetch-windows.ps1') +
    '" -Destination "' + Destination + '" -AppRoot "' + ExpandConstant('{app}') + '"';
  if not Exec(ExpandConstant('{sys}\WindowsPowerShell\v1.0\powershell.exe'), Params,
      ExpandConstant('{app}'), SW_SHOW, ewWaitUntilTerminated, ResultCode) or (ResultCode <> 0) then
    MsgBox('AussieTV is installed, but FFmpeg could not be downloaded or verified.' + #13#10 +
      'Use Start Menu > AussieTV > Update FFmpeg to retry. Check your network connection.',
      mbError, MB_OK);
end;
