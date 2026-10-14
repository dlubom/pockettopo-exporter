package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"syscall"
	"unsafe"
)

var getConsoleMode = syscall.NewLazyDLL("kernel32.dll").NewProc("GetConsoleMode")
var shellChangeNotify = syscall.NewLazyDLL("shell32.dll").NewProc("SHChangeNotify")

func runDesktop(args []string, stdout, stderr io.Writer) (int, bool) {
	if len(args) == 0 {
		return 0, false
	}
	switch args[0] {
	case "shell-export":
		code := runShellExport(args, stdout, stderr)
		// Redirected stdin is not interactive. Never make automation wait for
		// input, and never consume a pipe containing unrelated data.
		var mode uint32
		console, _, _ := getConsoleMode.Call(os.Stdin.Fd(), uintptr(unsafe.Pointer(&mode)))
		if console != 0 {
			if _, err := fmt.Fprint(stdout, "Press Enter to close...\n"); err != nil {
				fmt.Fprintf(stderr, "Cannot write console prompt: %v\n", err)
				return 1, true
			}
			if _, err := bufio.NewReader(os.Stdin).ReadString('\n'); err != nil {
				fmt.Fprintf(stderr, "Cannot read console input: %v\n", err)
				code = 1
			}
		}
		return code, true
	case "windows-menu":
		if len(args) != 2 || (args[1] != "install" && args[1] != "uninstall") {
			fmt.Fprint(stderr, desktopUsage)
			return 1, true
		}
		var err error
		if args[1] == "install" {
			var executable string
			executable, err = os.Executable()
			if err == nil {
				executable, err = filepath.Abs(executable)
			}
			if err == nil {
				err = installMenuAt(menuRegistryPath, executable)
			}
		} else {
			err = uninstallMenuAt(menuRegistryPath)
		}
		if err != nil {
			fmt.Fprintf(stderr, "Cannot %s Windows menu: %v\n", args[1], err)
			return 1, true
		}
		// SHCNE_ASSOCCHANGED, SHCNF_IDLIST refreshes associations without
		// restarting Explorer or changing the system's default application.
		shellChangeNotify.Call(0x08000000, 0, 0, 0)
		if _, err := fmt.Fprintf(stdout, "Windows menu %s complete for the current user.\n", args[1]); err != nil {
			fmt.Fprintf(stderr, "Cannot write menu status: %v\n", err)
			return 1, true
		}
		return 0, true
	}
	return 0, false
}
