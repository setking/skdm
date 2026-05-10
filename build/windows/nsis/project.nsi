Unicode true

####
## Please note: Template replacements don't work in this file. They are provided with default defines like
## mentioned underneath.
## If the keyword is not defined, "wails_tools.nsh" will populate them.
## If they are defined here, "wails_tools.nsh" will not touch them. This allows you to use this project.nsi manually
## from outside of Wails for debugging and development of the installer.
##
## For development first make a wails nsis build to populate the "wails_tools.nsh":
## > wails build --target windows/amd64 --nsis
## Then you can call makensis on this file with specifying the path to your binary:
## For a AMD64 only installer:
## > makensis -DARG_WAILS_AMD64_BINARY=..\..\bin\app.exe
## For a ARM64 only installer:
## > makensis -DARG_WAILS_ARM64_BINARY=..\..\bin\app.exe
## For a installer with both architectures:
## > makensis -DARG_WAILS_AMD64_BINARY=..\..\bin\app-amd64.exe -DARG_WAILS_ARM64_BINARY=..\..\bin\app-arm64.exe
####
## The following information overrides the defaults in wails_tools.nsh
####
!define INFO_COMPANYNAME    "SKDM"
!define INFO_PRODUCTNAME    "SKDM"
!define INFO_PRODUCTVERSION "0.2.0"
!define INFO_COPYRIGHT      "(c) 2026, SKDM"
###
## 用户级安装，无需管理员权限，安装到 %LOCALAPPDATA%\Programs
!define REQUEST_EXECUTION_LEVEL "user"
####
## Include the wails tools
####
!include "wails_tools.nsh"

# The version information for this two must consist of 4 parts
VIProductVersion "${INFO_PRODUCTVERSION}.0"
VIFileVersion    "${INFO_PRODUCTVERSION}.0"

VIAddVersionKey "CompanyName"     "${INFO_COMPANYNAME}"
VIAddVersionKey "FileDescription" "${INFO_PRODUCTNAME} Installer"
VIAddVersionKey "ProductVersion"  "${INFO_PRODUCTVERSION}"
VIAddVersionKey "FileVersion"     "${INFO_PRODUCTVERSION}"
VIAddVersionKey "LegalCopyright"  "${INFO_COPYRIGHT}"
VIAddVersionKey "ProductName"     "${INFO_PRODUCTNAME}"

# Enable HiDPI support. https://nsis.sourceforge.io/Reference/ManifestDPIAware
ManifestDPIAware true

!include "MUI.nsh"

!define MUI_ICON "..\icon.ico"
!define MUI_UNICON "..\icon.ico"
# !define MUI_WELCOMEFINISHPAGE_BITMAP "resources\leftimage.bmp" #Include this to add a bitmap on the left side of the Welcome Page. Must be a size of 164x314
!define MUI_FINISHPAGE_NOAUTOCLOSE # Wait on the INSTFILES page so the user can take a look into the details of the installation steps
!define MUI_ABORTWARNING # This will warn the user if they exit from the installer.

!define MUI_PAGE_CUSTOMFUNCTION_PRE SkipIfElevated
!insertmacro MUI_PAGE_WELCOME # Welcome to the installer page.
# !insertmacro MUI_PAGE_LICENSE "resources\eula.txt" # Adds a EULA page to the installer
!define MUI_PAGE_CUSTOMFUNCTION_PRE SkipIfElevated
!insertmacro MUI_PAGE_DIRECTORY # In which folder install page.
!insertmacro MUI_PAGE_INSTFILES # Installing page.
!insertmacro MUI_PAGE_FINISH # Finished installation page.

!insertmacro MUI_UNPAGE_INSTFILES # Uninstalling page

!insertmacro MUI_LANGUAGE "English" # Set the Language of the installer

## The following two statements can be used to sign the installer and the uninstaller. The path to the binaries are provided in %1
#!uninstfinalize 'signtool --file "%1"'
#!finalize 'signtool --file "%1"'

Name "${INFO_PRODUCTNAME}"
OutFile "..\..\..\bin\${INFO_PROJECTNAME}-${ARCH}-installer.exe" # Name of the installer's file.
InstallDir "$LOCALAPPDATA\Programs\${INFO_COMPANYNAME}\${INFO_PRODUCTNAME}" # 用户级安装到 %LOCALAPPDATA%\Programs，无需管理员权限
ShowInstDetails show # This will always show the installation details.

Var IsElevated
Var SkipPages

Function .onInit
    !insertmacro wails.checkArchitecture

    ; 检查是否是提权后重新启动的实例（携带 /ELEVATED 参数）
    ${GetParameters} $R0
    ${GetOptions} $R0 "/ELEVATED" $R1
    ${IfNot} ${Errors}
        StrCpy $IsElevated "1"
        ; 从临时注册表恢复用户选择的安装目录
        ReadRegStr $INSTDIR HKCU "Software\${INFO_COMPANYNAME}\InstallerTemp" "InstallDir"
        DeleteRegKey HKCU "Software\${INFO_COMPANYNAME}\InstallerTemp"
    ${Else}
        ; 非提权实例：尝试从上次安装记录恢复安装目录
        SetRegView 64
        ReadRegStr $0 HKCU "Software\Microsoft\Windows\CurrentVersion\Uninstall\${UNINST_KEY_NAME}" "UninstallString"
        ${If} $0 != ""
            ${GetParent} $0 $INSTDIR
        ${EndIf}
    ${EndIf}
FunctionEnd

## 提权模式下跳过欢迎页和目录选择页，直接进入安装。
Function SkipIfElevated
    ${If} $IsElevated == "1"
        Abort
    ${EndIf}
FunctionEnd

## 校验用户选择的安装目录是否可写。
## 如果目录需要管理员权限，询问用户是否以管理员身份重新启动安装程序。
Function .onVerifyInstDir
    CreateDirectory "$INSTDIR"
    ClearErrors
    FileOpen $0 "$INSTDIR\.skdm_install_test" w
    IfErrors cant_write can_write

    cant_write:
        MessageBox MB_ICONQUESTION|MB_YESNO|MB_DEFBUTTON2 \
            "The selected directory requires administrator privileges to write to.$\n$\nWould you like to restart the installer with administrator privileges?$\n$\nSelect 'No' to return and choose a different directory." \
            IDYES request_elevation IDNO stay

    request_elevation:
        ; 将用户选择的目录写入临时注册表，供提权后的实例读取
        SetRegView 64
        WriteRegStr HKCU "Software\${INFO_COMPANYNAME}\InstallerTemp" "InstallDir" "$INSTDIR"
        ; 通过 ShellExecuteW + "runas" 触发 UAC 提权重启
        System::Call 'shell32::ShellExecuteW(i $HWNDPARENT, w "runas", w "$EXEPATH", w "/ELEVATED", w "$EXEDIR", i 1) i.r0'
        ; 非提权实例直接退出
        Quit

    stay:
        Abort

    can_write:
        FileClose $0
        Delete "$INSTDIR\.skdm_install_test"
FunctionEnd

Section
    !insertmacro wails.setShellContext

    !insertmacro wails.webview2runtime

    SetOutPath $INSTDIR
    
    !insertmacro wails.files

    CreateShortcut "$SMPROGRAMS\${INFO_PRODUCTNAME}.lnk" "$INSTDIR\${PRODUCT_EXECUTABLE}"
    CreateShortCut "$DESKTOP\${INFO_PRODUCTNAME}.lnk" "$INSTDIR\${PRODUCT_EXECUTABLE}"

    !insertmacro wails.associateFiles
    !insertmacro wails.associateCustomProtocols
    
    ; 用户级卸载注册表（写入 HKCU 而非 HKLM）
    WriteUninstaller "$INSTDIR\uninstall.exe"
    SetRegView 64
    WriteRegStr HKCU "Software\Microsoft\Windows\CurrentVersion\Uninstall\${UNINST_KEY_NAME}" "Publisher" "${INFO_COMPANYNAME}"
    WriteRegStr HKCU "Software\Microsoft\Windows\CurrentVersion\Uninstall\${UNINST_KEY_NAME}" "DisplayName" "${INFO_PRODUCTNAME}"
    WriteRegStr HKCU "Software\Microsoft\Windows\CurrentVersion\Uninstall\${UNINST_KEY_NAME}" "DisplayVersion" "${INFO_PRODUCTVERSION}"
    WriteRegStr HKCU "Software\Microsoft\Windows\CurrentVersion\Uninstall\${UNINST_KEY_NAME}" "DisplayIcon" "$INSTDIR\${PRODUCT_EXECUTABLE}"
    WriteRegStr HKCU "Software\Microsoft\Windows\CurrentVersion\Uninstall\${UNINST_KEY_NAME}" "UninstallString" "$\"$INSTDIR\uninstall.exe$\""
    WriteRegStr HKCU "Software\Microsoft\Windows\CurrentVersion\Uninstall\${UNINST_KEY_NAME}" "QuietUninstallString" "$\"$INSTDIR\uninstall.exe$\" /S"
    ${GetSize} "$INSTDIR" "/S=0K" $0 $1 $2
    IntFmt $0 "0x%08X" $0
    WriteRegDWORD HKCU "Software\Microsoft\Windows\CurrentVersion\Uninstall\${UNINST_KEY_NAME}" "EstimatedSize" "$0"
SectionEnd

Section "uninstall"
    !insertmacro wails.setShellContext

    RMDir /r "$AppData\${PRODUCT_EXECUTABLE}" # Remove the WebView2 DataPath

    RMDir /r $INSTDIR

    Delete "$SMPROGRAMS\${INFO_PRODUCTNAME}.lnk"
    Delete "$DESKTOP\${INFO_PRODUCTNAME}.lnk"

    !insertmacro wails.unassociateFiles
    !insertmacro wails.unassociateCustomProtocols

    Delete "$INSTDIR\uninstall.exe"
    SetRegView 64
    DeleteRegKey HKCU "Software\Microsoft\Windows\CurrentVersion\Uninstall\${UNINST_KEY_NAME}"
SectionEnd
