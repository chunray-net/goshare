@echo off
rem Build the GoShare Fyne client as a runnable Windows GUI exe.
rem
rem Notes:
rem   Fyne requires cgo (OpenGL/GLFW), so CGO_ENABLED must be 1 and a
rem   64-bit MinGW-w64 gcc must be on PATH.
rem
rem   The strip flags (-s -w) are REQUIRED with the MinGW/binutils 2.30
rem   shipped in this environment. Go 1.26 emits DWARF 5 debug info
rem   (.debug_rnglists/.debug_loclists/.debug_addr); that old GNU ld lays
rem   those sections out at VirtualAddress 0xFFC00000 (beyond
rem   SizeOfImage), so Windows refuses to load the exe with the error
rem   "%1 is not a valid Win32 application". Stripping the debug info
rem   avoids it. (Alternative: upgrade MinGW-w64/binutils to >= 2.35.)
rem
rem   -H=windowsgui builds a GUI subsystem exe, so double-clicking does
rem   not open a console window. Note: stdout/stderr are then invisible;
rem   use "go run ." or rebuild without -H=windowsgui when debugging.
setlocal
cd /d "%~dp0"

set CGO_ENABLED=1
go build -ldflags "-s -w -H=windowsgui" -o client.exe .
if errorlevel 1 (
    echo [ERROR] build failed
    exit /b 1
)

echo [OK] built %~dp0client.exe
endlocal
