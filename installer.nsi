Name "Beshence Bank"

OutFile "build\windows\BeshenceBankSetup.exe"

InstallDir "$PROGRAMFILES\Beshence\Bank"

Page directory
Page instfiles

Section
    SetOutPath $INSTDIR
    File "build\windows\bank.exe"

    WriteUninstaller "$INSTDIR\uninstall.exe"

    WriteRegStr HKLM \
        "Software\Microsoft\Windows\CurrentVersion\Uninstall\Beshence Bank" \
        "DisplayName" \
        "Beshence Bank"

    WriteRegStr HKLM \
        "Software\Microsoft\Windows\CurrentVersion\Uninstall\Beshence Bank" \
        "UninstallString" \
        "$INSTDIR\uninstall.exe"

    WriteRegStr HKLM \
        "Software\Microsoft\Windows\CurrentVersion\Uninstall\Beshence Bank" \
        "InstallLocation" \
        "$INSTDIR"

    WriteRegStr HKLM \
        "Software\Microsoft\Windows\CurrentVersion\Uninstall\Beshence Bank" \
        "Publisher" \
        "Beshence"

    WriteRegStr HKLM \
        "Software\Microsoft\Windows\CurrentVersion\Uninstall\Beshence Bank" \
        "DisplayVersion" \
        "1.0.0"

    CreateShortcut \
        "$DESKTOP\Beshence Bank.lnk" \
        "$INSTDIR\bank.exe"

    CreateShortcut \
        "$SMPROGRAMS\Beshence Bank.lnk" \
        "$INSTDIR\bank.exe"
SectionEnd

Section "Uninstall"
    Delete "$INSTDIR\bank.exe"
    Delete "$INSTDIR\uninstall.exe"

    Delete "$DESKTOP\Beshence Bank.lnk"
    Delete "$SMPROGRAMS\Beshence Bank.lnk"

    DeleteRegKey HKLM \
        "Software\Microsoft\Windows\CurrentVersion\Uninstall\Beshence Bank"

    RMDir /r "$INSTDIR"
SectionEnd