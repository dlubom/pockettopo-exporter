package main

import (
	"fmt"
	"os"
	"reflect"
	"strings"
	"syscall"
	"testing"
	"time"
	"unsafe"
)

func TestWindowsMenuNativeRegistrationRoundtrip(t *testing.T) {
	path := fmt.Sprintf(`Software\Classes\SystemFileAssociations\.top\shell\PocketTopoExporterTest-%d-%d`, os.Getpid(), time.Now().UnixNano())
	first := `C:\Program Files\Żółć $ apostrophe's & test\exporter.exe`
	second := `D:\Moved portable folder\exporter.exe`
	if err := installMenuAt(path, first); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := uninstallMenuAt(path); err != nil {
			t.Error(err)
		}
	})
	got, exists, err := readRegistryTree(path)
	if err != nil || !exists {
		t.Fatal("registered menu absent", exists, err)
	}
	if len(got) != 20 || len(got["shell"]) != 0 || got["shell\\08dxfside\\command"][""] != `"C:\Program Files\Żółć $ apostrophe's & test\exporter.exe" shell-export "%1" --format dxf --view side` {
		t.Fatalf("registered menu shape/arguments: %#v", got)
	}
	if err := installMenuAt(path, second); err != nil {
		t.Fatal("reinstall", err)
	}
	got, _, err = readRegistryTree(path)
	if err != nil || !strings.HasPrefix(got["shell\\01text\\command"][""], `"D:\Moved portable folder\exporter.exe"`) {
		t.Fatal("moved executable not registered", got, err)
	}
	if err := uninstallMenuAt(path); err != nil {
		t.Fatal(err)
	}
	if _, exists, err := readRegistryTree(path); err != nil || exists {
		t.Fatal("uninstall retained owned menu", exists, err)
	}
	if err := uninstallMenuAt(path); err != nil {
		t.Fatal("idempotent uninstall", err)
	}
}

func TestWindowsMenuCommandNativeArgumentParsing(t *testing.T) {
	executable := `C:\spaces Żółć $ apostrophe's & test\exporter.exe`
	command, err := syscall.UTF16PtrFromString(menuCommand(executable, "dxf", "side"))
	if err != nil {
		t.Fatal(err)
	}
	var count int32
	argv, err := syscall.CommandLineToArgv(command, &count)
	if err != nil {
		t.Fatal(err)
	}
	defer syscall.LocalFree(syscall.Handle(unsafe.Pointer(argv)))
	got := []string{}
	for _, arg := range argv[:count] {
		got = append(got, syscall.UTF16ToString(arg[:]))
	}
	want := []string{executable, "shell-export", "%1", "--format", "dxf", "--view", "side"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("native parsed arguments: %#v", got)
	}
}

func TestWindowsMenuPercentExecutableRefusalDoesNotWriteRegistry(t *testing.T) {
	path := fmt.Sprintf(`Software\Classes\SystemFileAssociations\.top\shell\PocketTopoExporterTest-%d-%d`, os.Getpid(), time.Now().UnixNano())
	for _, token := range []string{"%1", "%L", "%2", "bare%"} {
		if err := installMenuAt(path, `C:\portable `+token+`\exporter.exe`); err == nil || !strings.Contains(err.Error(), "must not contain %") {
			t.Fatalf("unsafe absent-key install accepted: %s, %v", token, err)
		}
		if _, exists, err := readRegistryTree(path); err != nil || exists {
			t.Fatal("rejected install created registry contents", exists, err)
		}
	}
	if err := installMenuAt(path, `C:\safe folder\exporter.exe`); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := uninstallMenuAt(path); err != nil {
			t.Error(err)
		}
	})
	before, _, err := readRegistryTree(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := installMenuAt(path, `D:\unsafe %1 folder\exporter.exe`); err == nil || !strings.Contains(err.Error(), "must not contain %") {
		t.Fatal("unsafe reinstall accepted", err)
	}
	after, _, err := readRegistryTree(path)
	if err != nil || !reflect.DeepEqual(before, after) {
		t.Fatal("unsafe reinstall changed registry", err, after)
	}
}

func TestWindowsMenuRefusesForeignContents(t *testing.T) {
	for _, foreign := range []string{"value", "subkey", "command"} {
		t.Run(foreign, func(t *testing.T) {
			path := fmt.Sprintf(`Software\Classes\SystemFileAssociations\.top\shell\PocketTopoExporterTest-%d-%d`, os.Getpid(), time.Now().UnixNano())
			if err := installMenuAt(path, `C:\exporter.exe`); err != nil {
				t.Fatal(err)
			}
			node, name, value := "", "foreign", "keep"
			if foreign == "subkey" {
				node = "foreign"
			}
			if foreign == "command" {
				node = `shell\01text\command`
				name = ""
				value = `"C:\other.exe" "%1"`
			}
			h, _, err := createRegistryKey(path + map[bool]string{true: "\\" + node, false: ""}[node != ""])
			if err != nil {
				t.Fatal(err)
			}
			if err := setRegistryString(h, name, value); err != nil {
				t.Fatal(err)
			}
			syscall.RegCloseKey(h)
			before, _, err := readRegistryTree(path)
			if err != nil {
				t.Fatal(err)
			}
			if err := installMenuAt(path, `D:\exporter.exe`); err == nil {
				t.Fatal("reinstall accepted foreign contents")
			}
			if err := uninstallMenuAt(path); err == nil {
				t.Fatal("uninstall accepted foreign contents")
			}
			after, _, err := readRegistryTree(path)
			if err != nil || !reflect.DeepEqual(before, after) {
				t.Fatal("foreign contents changed", after, err)
			}
			// Remove only our deliberately added test content, restoring ownership
			// before exercising the production scoped uninstaller.
			h, err = openRegistryKey(path + map[bool]string{true: "\\" + node, false: ""}[node != ""])
			if err != nil {
				t.Fatal(err)
			}
			if foreign == "command" {
				err = setRegistryString(h, "", `"C:\exporter.exe" shell-export "%1" --format text`)
			} else {
				err = deleteRegistryValue(h, name)
			}
			syscall.RegCloseKey(h)
			if err != nil {
				t.Fatal(err)
			}
			if foreign == "subkey" {
				if err := deleteRegistryKey(path + "\\foreign"); err != nil {
					t.Fatal(err)
				}
			}
			if err := uninstallMenuAt(path); err != nil {
				t.Fatal(err)
			}
		})
	}
}
