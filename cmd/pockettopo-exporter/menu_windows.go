package main

import (
	"errors"
	"fmt"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"syscall"
	"unicode/utf16"
	"unsafe"
)

const menuRegistryPath = `Software\Classes\SystemFileAssociations\.top\shell\PocketTopoExporter`
const menuOwner = "pockettopo-exporter/windows-menu/1"

var registryDLL = syscall.NewLazyDLL("advapi32.dll")
var regCreateKey = registryDLL.NewProc("RegCreateKeyExW")
var regSetValue = registryDLL.NewProc("RegSetValueExW")
var regEnumValue = registryDLL.NewProc("RegEnumValueW")
var regDeleteKey = registryDLL.NewProc("RegDeleteKeyW")
var regDeleteValue = registryDLL.NewProc("RegDeleteValueW")

// Every value and subkey below the named menu belongs to the application.
// Existing contents must match the complete installed shape before changing
// anything. Additional/changed values or subkeys are foreign and block removal.
type registryTree map[string]map[string]string

func menuTree(executable string) registryTree {
	tree := registryTree{
		"":      {"MUIVerb": "PocketTopo Exporter", "SubCommands": "", "Icon": quoteWindowsArg(executable), "MultiSelectModel": "Single", "PocketTopoExporterOwner": menuOwner, "PocketTopoExporterExecutable": executable},
		"shell": {},
	}
	for _, entry := range menuEntries {
		path := "shell\\" + entry.key
		tree[path] = map[string]string{"MUIVerb": entry.label, "MultiSelectModel": "Single"}
		tree[path+"\\command"] = map[string]string{"": menuCommand(executable, entry.format, entry.view)}
	}
	return tree
}

func ownedMenu(tree registryTree) bool {
	executable := tree[""]["PocketTopoExporterExecutable"]
	return tree[""]["PocketTopoExporterOwner"] == menuOwner && filepath.IsAbs(executable) && reflect.DeepEqual(tree, menuTree(executable))
}

func installMenuAt(path, executable string) error {
	if err := menuExecutablePathError(executable); err != nil {
		return err
	}
	if !filepath.IsAbs(executable) || strings.ContainsRune(executable, 0) {
		return errors.New("the registered executable must have an absolute path")
	}
	current, exists, err := readRegistryTree(path)
	if err != nil {
		return err
	}
	if exists && !ownedMenu(current) {
		return errors.New("refusing to change a menu containing foreign or incomplete contents")
	}
	expected := menuTree(executable)
	root, created, err := createRegistryKey(path)
	if err != nil {
		return err
	}
	syscall.RegCloseKey(root)
	if !exists && !created {
		return errors.New("menu appeared during installation; refusing to change it")
	}
	if err := writeRegistryTree(path, expected); err != nil {
		if created {
			// Only a newly created subtree containing a subset of the intended
			// values is safe to roll back. Foreign contents are retained.
			partial, _, readErr := readRegistryTree(path)
			if readErr == nil && treeSubset(partial, expected) {
				if cleanupErr := removeRegistryTree(path, partial); cleanupErr != nil {
					return fmt.Errorf("%w; incomplete menu cleanup: %v", err, cleanupErr)
				}
			}
		}
		return err
	}
	readback, _, err := readRegistryTree(path)
	if err != nil || !reflect.DeepEqual(readback, expected) {
		return fmt.Errorf("menu registration readback failed: %v", err)
	}
	return nil
}

func uninstallMenuAt(path string) error {
	tree, exists, err := readRegistryTree(path)
	if err != nil || !exists {
		return err
	}
	if !ownedMenu(tree) {
		return errors.New("refusing to remove a menu containing foreign or incomplete contents")
	}
	return removeRegistryTree(path, tree)
}

func treeSubset(actual, expected registryTree) bool {
	for path, values := range actual {
		wanted, exists := expected[path]
		if !exists {
			return false
		}
		for name, value := range values {
			if got, exists := wanted[name]; !exists || got != value {
				return false
			}
		}
	}
	return true
}

func writeRegistryTree(path string, tree registryTree) error {
	paths := sortedTreePaths(tree)
	for _, node := range paths {
		key, _, err := createRegistryKey(registryPath(path, node))
		if err != nil {
			return err
		}
		names := make([]string, 0, len(tree[node]))
		for name := range tree[node] {
			names = append(names, name)
		}
		sort.Strings(names)
		for _, name := range names {
			if err = setRegistryString(key, name, tree[node][name]); err != nil {
				break
			}
		}
		syscall.RegCloseKey(key)
		if err != nil {
			return err
		}
	}
	return nil
}

func removeRegistryTree(path string, tree registryTree) error {
	// Recheck immediately before removing. Never use RegDeleteTree, which
	// would recursively discard additional foreign contents without checking.
	current, _, err := readRegistryTree(path)
	if err != nil || !reflect.DeepEqual(current, tree) {
		return fmt.Errorf("menu changed before removal; foreign contents retained: %v", err)
	}
	paths := sortedTreePaths(tree)
	for i := len(paths) - 1; i >= 0; i-- {
		if err := deleteRegistryKey(registryPath(path, paths[i])); err != nil {
			return err
		}
	}
	return nil
}

func sortedTreePaths(tree registryTree) []string {
	paths := make([]string, 0, len(tree))
	for path := range tree {
		paths = append(paths, path)
	}
	sort.Strings(paths)
	return paths
}

func registryPath(root, node string) string {
	if node == "" {
		return root
	}
	return root + "\\" + node
}

func openRegistryKey(path string) (syscall.Handle, error) {
	p, err := syscall.UTF16PtrFromString(path)
	if err != nil {
		return 0, err
	}
	var key syscall.Handle
	err = syscall.RegOpenKeyEx(syscall.HKEY_CURRENT_USER, p, 0, syscall.KEY_READ|syscall.KEY_WRITE, &key)
	return key, err
}

func createRegistryKey(path string) (syscall.Handle, bool, error) {
	p, err := syscall.UTF16PtrFromString(path)
	if err != nil {
		return 0, false, err
	}
	var key syscall.Handle
	var disposition uint32
	code, _, _ := regCreateKey.Call(syscall.HKEY_CURRENT_USER, uintptr(unsafe.Pointer(p)), 0, 0, 0, syscall.KEY_READ|syscall.KEY_WRITE, 0, uintptr(unsafe.Pointer(&key)), uintptr(unsafe.Pointer(&disposition)))
	return key, disposition == 1, registryError(code)
}

func setRegistryString(key syscall.Handle, name, value string) error {
	n, err := syscall.UTF16PtrFromString(name)
	if err != nil {
		return err
	}
	v, err := syscall.UTF16FromString(value)
	if err != nil {
		return err
	}
	code, _, _ := regSetValue.Call(uintptr(key), uintptr(unsafe.Pointer(n)), 0, syscall.REG_SZ, uintptr(unsafe.Pointer(&v[0])), uintptr(len(v)*2))
	return registryError(code)
}

func deleteRegistryValue(key syscall.Handle, name string) error {
	n, err := syscall.UTF16PtrFromString(name)
	if err != nil {
		return err
	}
	code, _, _ := regDeleteValue.Call(uintptr(key), uintptr(unsafe.Pointer(n)))
	return registryError(code)
}

func deleteRegistryKey(path string) error {
	p, err := syscall.UTF16PtrFromString(path)
	if err != nil {
		return err
	}
	code, _, _ := regDeleteKey.Call(syscall.HKEY_CURRENT_USER, uintptr(unsafe.Pointer(p)))
	return registryError(code)
}

func registryError(code uintptr) error {
	if code == 0 {
		return nil
	}
	return syscall.Errno(code)
}

func readRegistryTree(path string) (registryTree, bool, error) {
	tree := registryTree{}
	key, err := openRegistryKey(path)
	if errors.Is(err, syscall.ERROR_FILE_NOT_FOUND) {
		return tree, false, nil
	}
	if err != nil {
		return nil, false, err
	}
	defer syscall.RegCloseKey(key)
	err = readRegistryNode(key, "", tree)
	return tree, true, err
}

func readRegistryNode(key syscall.Handle, node string, tree registryTree) error {
	// This menu has 20 keys and at most 6 values per key. A larger/unbounded
	// subtree is foreign; reject it without enumerating an arbitrary registry.
	if len(tree) >= 64 {
		return errors.New("menu contains too many foreign keys")
	}
	values := map[string]string{}
	for i := uint32(0); ; i++ {
		if i >= 64 {
			return errors.New("menu contains too many foreign values")
		}
		name := make([]uint16, 32768)
		nameLen := uint32(len(name))
		var kind uint32
		data := make([]uint16, 32768)
		dataLen := uint32(len(data) * 2)
		code, _, _ := regEnumValue.Call(uintptr(key), uintptr(i), uintptr(unsafe.Pointer(&name[0])), uintptr(unsafe.Pointer(&nameLen)), 0, uintptr(unsafe.Pointer(&kind)), uintptr(unsafe.Pointer(&data[0])), uintptr(unsafe.Pointer(&dataLen)))
		if code == 259 { // ERROR_NO_MORE_ITEMS
			break
		}
		if code != 0 {
			return registryError(code)
		}
		if kind != syscall.REG_SZ || dataLen < 2 || dataLen%2 != 0 || data[dataLen/2-1] != 0 {
			return errors.New("menu contains a foreign non-string registry value")
		}
		value := data[:dataLen/2-1]
		for _, r := range value {
			if r == 0 {
				return errors.New("menu contains a foreign embedded-NUL value")
			}
		}
		values[string(utf16.Decode(name[:nameLen]))] = string(utf16.Decode(value))
	}
	tree[node] = values
	for i := uint32(0); ; i++ {
		name := make([]uint16, 256)
		nameLen := uint32(len(name))
		err := syscall.RegEnumKeyEx(key, i, &name[0], &nameLen, nil, nil, nil, nil)
		if errors.Is(err, syscall.Errno(259)) {
			break
		}
		if err != nil {
			return err
		}
		var child syscall.Handle
		if err := syscall.RegOpenKeyEx(key, &name[0], 0, syscall.KEY_READ, &child); err != nil {
			return err
		}
		childNode := string(utf16.Decode(name[:nameLen]))
		if node != "" {
			childNode = node + "\\" + childNode
		}
		err = readRegistryNode(child, childNode, tree)
		syscall.RegCloseKey(child)
		if err != nil {
			return err
		}
	}
	return nil
}
