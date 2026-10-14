# Native, isolated-runner acceptance: actual Explorer classic popup, commands,
# interactive console, repeated export, readable failure and scoped uninstall.
param([Parameter(Mandatory=$true)][string]$Binary,
      [string]$Output = "build/windows-menu")
$ErrorActionPreference = "Stop"
$Binary = (Resolve-Path -LiteralPath $Binary).Path
$Output = [IO.Path]::GetFullPath($Output)
if (Test-Path -LiteralPath $Output) { throw "Windows menu validation output must be new" }
New-Item -ItemType Directory -Path $Output | Out-Null
$menuKey = 'Software\Classes\SystemFileAssociations\.top\shell\PocketTopoExporter'
$existing = [Microsoft.Win32.Registry]::CurrentUser.OpenSubKey($menuKey)
if ($null -ne $existing) { $existing.Dispose(); throw "Isolated validation requires no preexisting PocketTopo Exporter menu" }
$evidence = [ordered]@{ schema_version=1; platform=[Runtime.InteropServices.RuntimeInformation]::OSDescription; architecture=[Runtime.InteropServices.RuntimeInformation]::OSArchitecture.ToString(); binary_sha256=(Get-FileHash -LiteralPath $Binary -Algorithm SHA256).Hash.ToLowerInvariant(); explorer_session=$false; menu_entries=@(); cases=@(); uninstall=$false; validation_ok=$false }
$installed = $false
$explorerWindow = $null
$ownedProcesses = [Collections.Generic.List[int]]::new()
$script:currentCase = 'setup'
$script:currentInputFile = ''
$hiddenPrivacyWindows = [Collections.Generic.List[System.IntPtr]]::new()
$hiddenPrivacyCoreWindows = [Collections.Generic.List[object]]::new()
$hiddenPrivacyBackdrops = [Collections.Generic.List[object]]::new()

Add-Type @'
using System;
using System.Text;
using System.Collections.Generic;
using System.Runtime.InteropServices;
public static class ExplorerMenuProbe {
 public delegate bool EnumProc(IntPtr h, IntPtr unused);
 [DllImport("user32.dll")] static extern bool EnumWindows(EnumProc cb, IntPtr arg);
 [DllImport("user32.dll")] static extern bool EnumChildWindows(IntPtr h, EnumProc cb, IntPtr arg);
 [DllImport("user32.dll", CharSet=CharSet.Unicode)] static extern int GetClassName(IntPtr h, StringBuilder b, int n);
 [DllImport("user32.dll", CharSet=CharSet.Unicode)] static extern int GetWindowText(IntPtr h, StringBuilder b, int n);
 [DllImport("user32.dll")] public static extern bool IsWindowVisible(IntPtr h);
 [DllImport("user32.dll")] static extern bool IsWindow(IntPtr h);
 [DllImport("user32.dll")] static extern bool IsIconic(IntPtr h);
 [DllImport("user32.dll")] static extern IntPtr GetShellWindow();
 [DllImport("user32.dll")] static extern IntPtr GetAncestor(IntPtr h,uint flags);
 [DllImport("user32.dll")] static extern IntPtr GetWindow(IntPtr h,uint command);
 [DllImport("user32.dll")] static extern IntPtr GetParent(IntPtr h);
 [DllImport("user32.dll", SetLastError=true)] static extern bool ShowWindowAsync(IntPtr h,int command);
 [DllImport("user32.dll")] public static extern bool SetForegroundWindow(IntPtr h);
 [DllImport("user32.dll")] public static extern IntPtr GetForegroundWindow();
 [DllImport("user32.dll")] public static extern int GetSystemMetrics(int index);
 [DllImport("user32.dll")] public static extern uint GetWindowThreadProcessId(IntPtr h, out uint pid);
 [DllImport("user32.dll")] public static extern IntPtr SendMessage(IntPtr h, uint msg, IntPtr w, IntPtr l);
 [DllImport("user32.dll", CharSet=CharSet.Unicode, SetLastError=true)] public static extern bool PostMessage(IntPtr h, uint msg, IntPtr w, IntPtr l);
 [DllImport("user32.dll")] public static extern int GetMenuItemCount(IntPtr h);
 [DllImport("user32.dll", CharSet=CharSet.Unicode)] static extern int GetMenuString(IntPtr h, uint item, StringBuilder b, int n, uint flags);
 [DllImport("user32.dll")] public static extern IntPtr GetSubMenu(IntPtr h, int item);
 [StructLayout(LayoutKind.Sequential)] public struct Rect { public int left,top,right,bottom; }
 [StructLayout(LayoutKind.Sequential)] struct Point {public int x,y;public Point(int a,int b){x=a;y=b;}}
 [DllImport("user32.dll")] static extern IntPtr WindowFromPoint(Point point);
 [DllImport("user32.dll")] static extern bool GetCursorPos(out Point point);
 [DllImport("user32.dll", SetLastError=true)] static extern IntPtr SetThreadDpiAwarenessContext(IntPtr context);
 [DllImport("user32.dll")] static extern bool GetMenuItemRect(IntPtr window, IntPtr menu, uint item, out Rect r);
 [DllImport("user32.dll")] static extern bool SetCursorPos(int x, int y);
 [DllImport("user32.dll")] static extern void mouse_event(uint flags,uint x,uint y,uint data,UIntPtr extra);
 [DllImport("user32.dll")] static extern void keybd_event(byte key,byte scan,uint flags,UIntPtr extra);
 public static string Class(IntPtr h) {var b=new StringBuilder(256);GetClassName(h,b,b.Capacity);return b.ToString();}
 public static string WindowDescription(IntPtr h) {uint pid;uint tid=GetWindowThreadProcessId(h,out pid);var b=new StringBuilder(2048);GetWindowText(h,b,b.Capacity);return "hwnd="+h.ToInt64()+" class="+Class(h)+" pid="+pid+" tid="+tid+" visible="+IsWindowVisible(h)+" title="+b.ToString();}
 public static string WindowRelations(IntPtr h) {return "parent="+WindowDescription(GetParent(h))+" | owner="+WindowDescription(GetWindow(h,4))+" | root="+WindowDescription(GetAncestor(h,2))+" | root_owner="+WindowDescription(GetAncestor(h,3));}
 public static string[] VisibleWindows() {var a=new List<string>();EnumWindows((h,x)=>{if(IsWindowVisible(h))a.Add(WindowDescription(h));return true;},IntPtr.Zero);return a.ToArray();}
 static bool IsDesktopPrivacyWindow(IntPtr h) {uint shellOwner,owner;GetWindowThreadProcessId(GetShellWindow(),out shellOwner);GetWindowThreadProcessId(h,out owner);return shellOwner!=0&&owner==shellOwner&&Class(h)=="Shell_OOBEProxy";}
 public static IntPtr[] VisiblePrivacyWindows() {var a=new List<IntPtr>();EnumWindows((h,x)=>{if(IsDesktopPrivacyWindow(h)&&IsWindowVisible(h)&&!IsIconic(h))a.Add(h);return true;},IntPtr.Zero);return a.ToArray();}
 public static void HidePrivacyWindow(IntPtr h) {if(!IsDesktopPrivacyWindow(h))throw new Exception("Refusing to hide any window other than current desktop Explorer Shell_OOBEProxy");if(!ShowWindowAsync(h,0))throw new Exception("Cannot hide disposable-runner privacy overlay: "+Marshal.GetLastWin32Error());}
 public static void RestorePrivacyWindow(IntPtr h) {if(!IsWindow(h))return;if(!IsDesktopPrivacyWindow(h))throw new Exception("Privacy overlay handle changed; refusing to show a foreign window");if(!ShowWindowAsync(h,8))throw new Exception("Cannot restore disposable-runner privacy overlay visibility: "+Marshal.GetLastWin32Error());}
 public static bool PrivacyWindowsRestored(IntPtr[] windows) {foreach(var h in windows){if(IsWindow(h)&&(!IsDesktopPrivacyWindow(h)||!IsWindowVisible(h)))return false;}return true;}
 static bool IsPrivacyCoreWindow(IntPtr h,uint expectedOwner) {uint owner;GetWindowThreadProcessId(h,out owner);var text=new StringBuilder(256);GetWindowText(h,text,text.Capacity);return owner==expectedOwner&&Class(h)=="Windows.UI.Core.CoreWindow"&&text.ToString()=="Microsoft account";}
 public static IntPtr ForegroundPrivacyCoreWindow() {IntPtr h=GetForegroundWindow();uint owner;GetWindowThreadProcessId(h,out owner);return IsWindowVisible(h)&&IsPrivacyCoreWindow(h,owner)?h:IntPtr.Zero;}
 public static void HidePrivacyCoreWindow(IntPtr h,uint owner) {if(!IsPrivacyCoreWindow(h,owner))throw new Exception("Verified privacy CoreWindow identity changed before hide");if(!ShowWindowAsync(h,0))throw new Exception("Cannot hide verified disposable-runner privacy CoreWindow: "+Marshal.GetLastWin32Error());}
 public static void RestorePrivacyCoreWindow(IntPtr h,uint owner) {if(!IsWindow(h))return;if(!IsPrivacyCoreWindow(h,owner))throw new Exception("Privacy CoreWindow identity changed; refusing to show a foreign window");if(!ShowWindowAsync(h,8))throw new Exception("Cannot restore verified disposable-runner privacy CoreWindow: "+Marshal.GetLastWin32Error());}
 public static bool PrivacyCoreWindowHidden(IntPtr h,uint owner) {return !IsWindow(h)||(IsPrivacyCoreWindow(h,owner)&&!IsWindowVisible(h));}
 public static bool PrivacyCoreWindowRestored(IntPtr h,uint owner) {return !IsWindow(h)||(IsPrivacyCoreWindow(h,owner)&&IsWindowVisible(h));}
 static bool IsPrivacyBackdrop(IntPtr h,uint expectedOwner) {uint shellOwner,owner;GetWindowThreadProcessId(GetShellWindow(),out shellOwner);GetWindowThreadProcessId(h,out owner);var text=new StringBuilder(256);GetWindowText(h,text,text.Capacity);return shellOwner!=0&&owner==shellOwner&&owner==expectedOwner&&Class(h)=="UserOOBEWindowClass"&&text.Length==0;}
 public static IntPtr PrivacyBackdropAtPointer() {Point point;if(!GetCursorPos(out point))return IntPtr.Zero;IntPtr h=WindowFromPoint(point);uint owner;GetWindowThreadProcessId(h,out owner);return IsWindowVisible(h)&&IsPrivacyBackdrop(h,owner)?h:IntPtr.Zero;}
 public static void HidePrivacyBackdrop(IntPtr h,uint owner) {if(!IsPrivacyBackdrop(h,owner))throw new Exception("Verified desktop OOBE backdrop identity changed before hide");if(!ShowWindowAsync(h,0))throw new Exception("Cannot hide verified disposable-runner OOBE backdrop: "+Marshal.GetLastWin32Error());}
 public static void RestorePrivacyBackdrop(IntPtr h,uint owner) {if(!IsWindow(h))return;if(!IsPrivacyBackdrop(h,owner))throw new Exception("OOBE backdrop identity changed; refusing to show a foreign window");if(!ShowWindowAsync(h,8))throw new Exception("Cannot restore verified disposable-runner OOBE backdrop: "+Marshal.GetLastWin32Error());}
 public static bool PrivacyBackdropHidden(IntPtr h,uint owner) {return !IsWindow(h)||(IsPrivacyBackdrop(h,owner)&&!IsWindowVisible(h));}
 public static bool PrivacyBackdropRestored(IntPtr h,uint owner) {return !IsWindow(h)||(IsPrivacyBackdrop(h,owner)&&IsWindowVisible(h));}
 public static string CursorDescription() {Point point;if(!GetCursorPos(out point))return "Cursor unavailable";return "x="+point.x+" y="+point.y+" hit="+WindowDescription(WindowFromPoint(point));}
 public static string[] ChildWindows(IntPtr parent) {var a=new List<string>();EnumChildWindows(parent,(h,x)=>{a.Add(WindowDescription(h));return a.Count<512;},IntPtr.Zero);return a.ToArray();}
 public static IntPtr FileView(IntPtr explorer) {IntPtr view=IntPtr.Zero;EnumChildWindows(explorer,(h,x)=>{if(IsWindowVisible(h)&&Class(h)=="SHELLDLL_DefView"){view=h;return false;}return true;},IntPtr.Zero);if(view==IntPtr.Zero)throw new Exception("No visible native Explorer file view (SHELLDLL_DefView)");return view;}
 [StructLayout(LayoutKind.Sequential)] struct GuiInfo {public uint size,flags;public IntPtr active,focus,capture,menuOwner,moveSize,caret;public Rect caretRect;}
 [DllImport("user32.dll", SetLastError=true)] static extern bool GetGUIThreadInfo(uint thread,ref GuiInfo info);
 public static string FocusDescription(IntPtr owner) {uint pid;uint tid=GetWindowThreadProcessId(owner,out pid);var info=new GuiInfo{size=(uint)Marshal.SizeOf<GuiInfo>()};if(!GetGUIThreadInfo(tid,ref info))return "GetGUIThreadInfo unavailable: "+Marshal.GetLastWin32Error();return "active="+WindowDescription(info.active)+" | focus="+WindowDescription(info.focus)+" | menuOwner="+WindowDescription(info.menuOwner)+" flags="+info.flags;}
 [DllImport("kernel32.dll")] static extern uint GetCurrentThreadId();
 [DllImport("user32.dll")] static extern IntPtr GetProcessWindowStation();
 [DllImport("user32.dll")] static extern IntPtr GetThreadDesktop(uint thread);
 [DllImport("user32.dll", SetLastError=true)] static extern IntPtr OpenInputDesktop(uint flags,bool inherit,uint access);
 [DllImport("user32.dll")] static extern bool CloseDesktop(IntPtr desktop);
 [DllImport("user32.dll", CharSet=CharSet.Unicode, SetLastError=true)] static extern bool GetUserObjectInformation(IntPtr handle,int index,StringBuilder data,uint bytes,out uint needed);
 static string ObjectName(IntPtr handle) {var b=new StringBuilder(256);uint needed;if(!GetUserObjectInformation(handle,2,b,512,out needed))return "unavailable: "+Marshal.GetLastWin32Error();return b.ToString();}
 public static string DesktopDescription() {string result="window_station="+ObjectName(GetProcessWindowStation())+" thread_desktop="+ObjectName(GetThreadDesktop(GetCurrentThreadId()));IntPtr input=OpenInputDesktop(0,false,1);if(input==IntPtr.Zero)return result+" input_desktop_unavailable="+Marshal.GetLastWin32Error();try{return result+" input_desktop="+ObjectName(input);}finally{CloseDesktop(input);}}
 public static IntPtr[] VisibleMenus() {var a=new List<IntPtr>();EnumWindows((h,x)=>{if(IsWindowVisible(h)&&Class(h)=="#32768")a.Add(h);return true;},IntPtr.Zero);return a.ToArray();}
 public static IntPtr[] ModernMenus(IntPtr explorer) {uint owner;GetWindowThreadProcessId(explorer,out owner);var a=new List<IntPtr>();EnumWindows((h,x)=>{uint pid;GetWindowThreadProcessId(h,out pid);if(pid==owner&&IsWindowVisible(h)&&Class(h)=="Microsoft.UI.Content.PopupWindowSiteBridge")a.Add(h);return true;},IntPtr.Zero);return a.ToArray();}
 public static void ModernMoreOptions(IntPtr explorer) {
  if(ModernMenus(explorer).Length==0)throw new Exception("No actual Explorer-owned modern popup for Show more options");
  uint owner,foreground;GetWindowThreadProcessId(explorer,out owner);GetWindowThreadProcessId(GetForegroundWindow(),out foreground);
  if(owner!=foreground)throw new Exception("Explorer is not foreground for Show more options keytip");
  // The English native-runner popup exposes W beside Show more options, as
  // captured by first-export.modern-popup.desktop.png. The resulting classic
  // popup remains mandatory; a key injection alone never counts as success.
  keybd_event(0x57,0,0,UIntPtr.Zero);keybd_event(0x57,0,2,UIntPtr.Zero);
 }
 public static string MenuText(IntPtr menu,int item) {var b=new StringBuilder(2048);GetMenuString(menu,(uint)item,b,b.Capacity,0x400);return b.ToString().Replace("&","");}
 public static string Hover(IntPtr popup,IntPtr menu,int item,bool click) {
  IntPtr previous=SetThreadDpiAwarenessContext(new IntPtr(-4));
  if(previous==IntPtr.Zero)throw new Exception("Cannot use physical screen coordinates for menu pointer: "+Marshal.GetLastWin32Error());
  try {
   Rect r;if(!GetMenuItemRect(IntPtr.Zero,menu,(uint)item,out r))throw new Exception("Cannot locate actual popup item");
   var point=new Point((r.left+r.right)/2,(r.top+r.bottom)/2);
   if(!SetCursorPos(point.x,point.y))throw new Exception("Cannot move pointer on interactive desktop");
   IntPtr hit=WindowFromPoint(point);
   string evidence="item="+item+" rect="+r.left+","+r.top+","+r.right+","+r.bottom+" target="+point.x+","+point.y+" | "+CursorDescription();
   if(hit!=popup)throw new Exception("Actual popup is covered at its menu item; "+evidence+" expected="+WindowDescription(popup));
   if(click){mouse_event(2,0,0,0,UIntPtr.Zero);mouse_event(4,0,0,0,UIntPtr.Zero);}
   return evidence;
  } finally {SetThreadDpiAwarenessContext(previous);}
 }
 [DllImport("kernel32.dll")] static extern bool FreeConsole();
 [DllImport("kernel32.dll")] static extern bool AttachConsole(uint pid);
 [DllImport("kernel32.dll", CharSet=CharSet.Unicode, SetLastError=true)] static extern IntPtr CreateFile(string name,uint access,uint share,IntPtr sec,uint mode,uint flags,IntPtr template);
 [DllImport("kernel32.dll")] static extern bool CloseHandle(IntPtr h);
 [DllImport("kernel32.dll", SetLastError=true)] static extern IntPtr OpenProcess(uint access,bool inherit,uint pid);
 [DllImport("kernel32.dll", SetLastError=true)] static extern uint WaitForSingleObject(IntPtr h,uint milliseconds);
 [DllImport("kernel32.dll", SetLastError=true)] static extern bool GetExitCodeProcess(IntPtr h,out uint code);
 [DllImport("kernel32.dll", CharSet=CharSet.Unicode, ExactSpelling=true)] static extern int GetPackageFamilyName(IntPtr process,ref uint length,StringBuilder name);
 [DllImport("kernel32.dll", CharSet=CharSet.Unicode, ExactSpelling=true)] static extern int GetApplicationUserModelId(IntPtr process,ref uint length,StringBuilder name);
 delegate int PackageQuery(IntPtr process,ref uint length,StringBuilder name);
 static string QueryPackageString(IntPtr process,PackageQuery query) {uint length=0;int code=query(process,ref length,null);if(code!=122)return "unavailable: native code "+code;if(length==0||length>4096)throw new Exception("Unexpected packaged process identity length");var name=new StringBuilder((int)length);code=query(process,ref length,name);return code==0?name.ToString():"unavailable: native code "+code;}
 public static string ProcessPackageFamily(IntPtr process) {return QueryPackageString(process,GetPackageFamilyName);}
 public static string ProcessApplicationId(IntPtr process) {return QueryPackageString(process,GetApplicationUserModelId);}
 public static IntPtr PinProcess(uint pid) {var handle=OpenProcess(0x00101000,false,pid);if(handle==IntPtr.Zero)throw new Exception("Cannot retain launched process handle: "+Marshal.GetLastWin32Error());return handle;}
 public static uint ProcessExitCode(IntPtr handle) {uint code;if(!GetExitCodeProcess(handle,out code))throw new Exception("Cannot read native process exit code: "+Marshal.GetLastWin32Error());return code;}
 public static void ReleaseProcess(IntPtr handle) {CloseHandle(handle);}
 [StructLayout(LayoutKind.Sequential)] struct Coord {public short x,y; public Coord(short a,short b){x=a;y=b;}}
 [StructLayout(LayoutKind.Sequential)] struct ConsoleInfo {public Coord size,cursor;public ushort attributes;public short left,top,right,bottom;public Coord maximum;}
 [DllImport("kernel32.dll")] static extern bool GetConsoleScreenBufferInfo(IntPtr h,out ConsoleInfo info);
 [DllImport("kernel32.dll", CharSet=CharSet.Unicode)] static extern bool ReadConsoleOutputCharacter(IntPtr h,StringBuilder b,uint n,Coord origin,out uint read);
 [StructLayout(LayoutKind.Explicit, CharSet=CharSet.Unicode, Size=20)] struct InputRecord {
  [FieldOffset(0)] public ushort type; [FieldOffset(4)] public int down; [FieldOffset(8)] public ushort repeat; [FieldOffset(10)] public ushort key; [FieldOffset(12)] public ushort scan; [FieldOffset(14)] public char character; [FieldOffset(16)] public uint control;
 }
 [DllImport("kernel32.dll", CharSet=CharSet.Unicode)] static extern bool WriteConsoleInput(IntPtr h,InputRecord[] inputs,uint n,out uint written);
 static string ReadConsoleBuffer(IntPtr output) {ConsoleInfo info;if(!GetConsoleScreenBufferInfo(output,out info))throw new Exception("No readable native console buffer");uint n=(uint)(info.size.x*(info.cursor.y+1));if(n>1000000)throw new Exception("Unexpected console output size");var b=new StringBuilder((int)n);uint read;if(!ReadConsoleOutputCharacter(output,b,n,new Coord(0,0),out read))throw new Exception("Cannot read visible console status");return b.ToString();}
 static void EnterConsole(IntPtr input) {var records=new InputRecord[]{new InputRecord{type=1,down=1,repeat=1,key=13,character='\r'},new InputRecord{type=1,down=0,repeat=1,key=13,character='\r'}};uint written;if(!WriteConsoleInput(input,records,2,out written)||written!=2)throw new Exception("Cannot deliver Enter to native console");}
 public class ExitObservation {public uint WaitResult {get;set;}public uint ExitCode {get;set;}public string ConsoleAfterEnter {get;set;}}
 public static ExitObservation EnterAndObserveExit(uint pid,IntPtr process,uint milliseconds) {
  FreeConsole();
  if(!AttachConsole(pid)){AttachConsole(0xffffffff);throw new Exception("Cannot attach to Explorer-launched console before Enter");}
  IntPtr output=IntPtr.Zero,input=IntPtr.Zero;
  try {
   output=CreateFile("CONOUT$",0x80000000,3,IntPtr.Zero,3,0,IntPtr.Zero);
   input=CreateFile("CONIN$",0x40000000,3,IntPtr.Zero,3,0,IntPtr.Zero);
   EnterConsole(input);
   uint wait=WaitForSingleObject(process,milliseconds);
   if(wait==0xffffffff)throw new Exception("Native process wait failed: "+Marshal.GetLastWin32Error());
   // This probe remains attached until after the process ends, preserving the
   // console buffer so any final input/read failure stays available as evidence.
   return new ExitObservation{WaitResult=wait,ExitCode=ProcessExitCode(process),ConsoleAfterEnter=ReadConsoleBuffer(output)};
  } finally {if(output!=IntPtr.Zero)CloseHandle(output);if(input!=IntPtr.Zero)CloseHandle(input);FreeConsole();AttachConsole(0xffffffff);}
 }
 public static string ConsoleText(uint pid,bool enter) {
  FreeConsole();
  if(!AttachConsole(pid)){AttachConsole(0xffffffff);throw new Exception("Cannot attach to Explorer-launched console");}
  IntPtr output=IntPtr.Zero,input=IntPtr.Zero;
  try {
   output=CreateFile("CONOUT$",0x80000000,3,IntPtr.Zero,3,0,IntPtr.Zero);
   string text=ReadConsoleBuffer(output);
   if(enter){input=CreateFile("CONIN$",0x40000000,3,IntPtr.Zero,3,0,IntPtr.Zero);EnterConsole(input);}
   return text;
  } finally {if(output!=IntPtr.Zero)CloseHandle(output);if(input!=IntPtr.Zero)CloseHandle(input);FreeConsole();AttachConsole(0xffffffff);}
 }
}
'@

function Get-ProcessOwnerSid([int]$ProcessId) {
    $nativeProcess=Get-CimInstance Win32_Process -Filter "ProcessId=$ProcessId"
    if ($null -eq $nativeProcess) { return 'process already exited' }
    $owner=Invoke-CimMethod -InputObject $nativeProcess -MethodName GetOwnerSid
    if ($owner.ReturnValue -ne 0) { return "GetOwnerSid unavailable: $($owner.ReturnValue)" }
    return $owner.Sid
}

function Read-MenuRegistry([Microsoft.Win32.RegistryKey]$Root,[string]$Path) {
    $tree=[ordered]@{}
    $key=$Root.OpenSubKey($Path)
    if ($null -eq $key) { return @{exists=$false} }
    try {
        foreach ($name in $key.GetValueNames()) {
            $tree[$name]=@{kind=$key.GetValueKind($name).ToString();value=$key.GetValue($name)}
        }
        foreach ($name in $key.GetSubKeyNames()) {
            $tree['key:'+ $name]=Read-MenuRegistry $Root ($Path+'\'+$name)
        }
    } finally { $key.Dispose() }
    return @{exists=$true;contents=$tree}
}

function Get-ExplorerSelection {
    $result=[ordered]@{selected_items=@();focused_item=$null;item_verbs=@()}
    if ($null -eq $script:explorerWindow) { return $result }
    $document=$script:explorerWindow.Document
    $selected=$document.SelectedItems()
    for ($i=0;$i -lt $selected.Count;$i++) {
        $item=$selected.Item($i)
        $result.selected_items += @{name=$item.Name;path=$item.Path}
    }
    $focused=$document.FocusedItem
    if ($null -ne $focused) { $result.focused_item=@{name=$focused.Name;path=$focused.Path} }
    if ($script:currentInputFile -ne '') {
        $item=$document.Folder.ParseName([IO.Path]::GetFileName($script:currentInputFile))
        if ($null -ne $item) {
            $verbs=$item.Verbs()
            for ($i=0;$i -lt $verbs.Count;$i++) { $result.item_verbs += $verbs.Item($i).Name }
        }
    }
    return $result
}

function Save-ExplorerState([string]$Name) {
    $state=[ordered]@{utc=[DateTime]::UtcNow.ToString('o');expected_input=$script:currentInputFile;session_id=(Get-Process -Id $PID).SessionId;pwsh_sid=[Security.Principal.WindowsIdentity]::GetCurrent().User.Value;user_interactive=[Environment]::UserInteractive;desktop=[ExplorerMenuProbe]::DesktopDescription();foreground=[ExplorerMenuProbe]::WindowDescription([ExplorerMenuProbe]::GetForegroundWindow());cursor=[ExplorerMenuProbe]::CursorDescription();visible_windows=@([ExplorerMenuProbe]::VisibleWindows());popups=@();diagnostic_errors=@()}
    if ($null -ne $script:explorerWindow) {
        $window=[IntPtr]$script:explorerWindow.HWND
        [uint32]$ownerId=0
        $null=[ExplorerMenuProbe]::GetWindowThreadProcessId($window,[ref]$ownerId)
        $state.explorer_window=[ExplorerMenuProbe]::WindowDescription($window)
        $state.explorer_focus=[ExplorerMenuProbe]::FocusDescription($window)
        $state.explorer_children=@([ExplorerMenuProbe]::ChildWindows($window))
        try { $state.explorer_sid=Get-ProcessOwnerSid ([int]$ownerId) } catch { $state.diagnostic_errors += $_.Exception.Message }
        try { $state.selection=Get-ExplorerSelection } catch { $state.diagnostic_errors += $_.Exception.Message }
    }
    foreach ($window in [ExplorerMenuProbe]::VisibleMenus()) {
        $menu=[ExplorerMenuProbe]::SendMessage($window,0x01e1,[IntPtr]::Zero,[IntPtr]::Zero)
        $items=@()
        for ($i=0;$i -lt [ExplorerMenuProbe]::GetMenuItemCount($menu);$i++) {
            $items += @{index=$i;label=[ExplorerMenuProbe]::MenuText($menu,$i);submenu=[ExplorerMenuProbe]::GetSubMenu($menu,$i).ToInt64()}
        }
        $state.popups += @{window=[ExplorerMenuProbe]::WindowDescription($window);menu_handle=$menu.ToInt64();items=$items}
    }
    try {
        $state.current_user_registry=Read-MenuRegistry ([Microsoft.Win32.Registry]::CurrentUser) $menuKey
        $state.merged_classes_registry=Read-MenuRegistry ([Microsoft.Win32.Registry]::ClassesRoot) 'SystemFileAssociations\.top\shell\PocketTopoExporter'
    } catch { $state.diagnostic_errors += $_.Exception.Message }
    # A screenshot supplements native menu/window state; unavailable screenshot
    # support is recorded and never silently substituted for popup acceptance.
    $bitmap=$null
    $graphics=$null
    try {
        Add-Type -AssemblyName System.Drawing
        $x=[ExplorerMenuProbe]::GetSystemMetrics(76)
        $y=[ExplorerMenuProbe]::GetSystemMetrics(77)
        $width=[ExplorerMenuProbe]::GetSystemMetrics(78)
        $height=[ExplorerMenuProbe]::GetSystemMetrics(79)
        if ($width -le 0 -or $height -le 0 -or [long]$width*$height -gt 50000000) { throw 'No bounded visible desktop for screenshot' }
        $bitmap=[Drawing.Bitmap]::new($width,$height)
        $graphics=[Drawing.Graphics]::FromImage($bitmap)
        $graphics.CopyFromScreen($x,$y,0,0,$bitmap.Size)
        $bitmap.Save((Join-Path $Output "$Name.desktop.png"),[Drawing.Imaging.ImageFormat]::Png)
        $state.screenshot="$Name.desktop.png"
    } catch { $state.screenshot_unavailable=$_.Exception.Message }
    finally {
        if ($null -ne $graphics) { $graphics.Dispose() }
        if ($null -ne $bitmap) { $bitmap.Dispose() }
    }
    [IO.File]::WriteAllText((Join-Path $Output "$Name.state.json"),($state | ConvertTo-Json -Depth 20),[Text.Encoding]::UTF8)
}

function Prepare-HostedRunnerDesktop {
    $overlays=@([ExplorerMenuProbe]::VisiblePrivacyWindows())
    $script:evidence.runner_desktop_preparation=@{applied=$false;overlay_count=$overlays.Count;visibility_restored=$false}
    if ($overlays.Count -eq 0) { return }
    Save-ExplorerState 'runner-privacy.before'
    # This is isolated hosted-runner preparation, never an installer action.
    # Temporarily hide only the known first-login overlay; do not accept privacy
    # choices, modify policies, close applications or terminate shell processes.
    if ($env:GITHUB_ACTIONS -ne 'true' -or $env:RUNNER_ENVIRONMENT -ne 'github-hosted') {
        throw 'First-login privacy overlay blocks actual Explorer UI; automatic preparation is restricted to GitHub-hosted disposable runners'
    }
    $currentSid=[Security.Principal.WindowsIdentity]::GetCurrent().User.Value
    $currentSession=(Get-Process -Id $PID).SessionId
    # EnumWindows deliberately omits some packaged-app windows. The actual
    # foreground privacy CoreWindow can therefore survive hiding its shell
    # proxy. Prove the exact Windows package and desktop identity before touching
    # it; a Microsoft account title or CoreWindow class alone is insufficient.
    $coreWindow=[ExplorerMenuProbe]::ForegroundPrivacyCoreWindow()
    if ($coreWindow -ne [IntPtr]::Zero) {
        [uint32]$coreOwner=0
        $null=[ExplorerMenuProbe]::GetWindowThreadProcessId($coreWindow,[ref]$coreOwner)
        $coreHandle=[ExplorerMenuProbe]::PinProcess($coreOwner)
        $keepHandle=$false
        try {
            $nativeProcess=Get-CimInstance Win32_Process -Filter "ProcessId=$coreOwner"
            if ($null -eq $nativeProcess) { throw 'Privacy CoreWindow process disappeared before identity verification' }
            $packageFamily=[ExplorerMenuProbe]::ProcessPackageFamily($coreHandle)
            $applicationId=[ExplorerMenuProbe]::ProcessApplicationId($coreHandle)
            $coreSid=Get-ProcessOwnerSid ([int]$coreOwner)
            $coreIdentity=[ordered]@{window=[ExplorerMenuProbe]::WindowDescription($coreWindow);relations=[ExplorerMenuProbe]::WindowRelations($coreWindow);process_id=$coreOwner;process_name=$nativeProcess.Name;executable_path=$nativeProcess.ExecutablePath;parent_process_id=$nativeProcess.ParentProcessId;session_id=$nativeProcess.SessionId;sid=$coreSid;package_family=$packageFamily;application_user_model_id=$applicationId;contemporaneous_desktop_privacy_proxies=@($overlays | ForEach-Object { [ExplorerMenuProbe]::WindowDescription($_) })}
            [IO.File]::WriteAllText((Join-Path $Output 'runner-privacy.core-identity.json'),($coreIdentity | ConvertTo-Json -Depth 8),[Text.Encoding]::UTF8)
            if ($packageFamily -ne 'Microsoft.Windows.CloudExperienceHost_cw5n1h2txyewy' -or
                $nativeProcess.SessionId -ne $currentSession -or $coreSid -ne $currentSid) {
                throw "Refusing privacy CoreWindow without exact CloudExperienceHost package and current desktop identity: $packageFamily"
            }
            $script:hiddenPrivacyCoreWindows.Add(@{window=$coreWindow;owner=$coreOwner;process_handle=$coreHandle})
            $keepHandle=$true
            $script:evidence.runner_desktop_preparation.core_identity=$coreIdentity
        } finally {
            if (!$keepHandle) { [ExplorerMenuProbe]::ReleaseProcess($coreHandle) }
        }
    }
    foreach ($window in $overlays) {
        [uint32]$ownerId=0
        $null=[ExplorerMenuProbe]::GetWindowThreadProcessId($window,[ref]$ownerId)
        $owner=Get-Process -Id $ownerId
        if ($owner.ProcessName -ne 'explorer' -or $owner.SessionId -ne $currentSession -or
            (Get-ProcessOwnerSid ([int]$ownerId)) -ne $currentSid) {
            throw 'Refusing preparation of a privacy overlay outside the current hosted-runner desktop identity'
        }
        $script:hiddenPrivacyWindows.Add($window)
        [ExplorerMenuProbe]::HidePrivacyWindow($window)
    }
    foreach ($entry in $hiddenPrivacyCoreWindows) {
        [ExplorerMenuProbe]::HidePrivacyCoreWindow($entry.window,$entry.owner)
        $deadline=[DateTime]::UtcNow.AddSeconds(10)
        while ([DateTime]::UtcNow -lt $deadline -and ![ExplorerMenuProbe]::PrivacyCoreWindowHidden($entry.window,$entry.owner)) { Start-Sleep -Milliseconds 100 }
        if (![ExplorerMenuProbe]::PrivacyCoreWindowHidden($entry.window,$entry.owner)) { throw 'Verified CloudExperienceHost CoreWindow remains visible; actual Explorer UI acceptance is blocked' }
    }
    # The verified first-login renderer can leave an opaque desktop-owned OOBE
    # backdrop. Discover only the exact window currently intercepting the cursor,
    # and only after the contemporaneous proxy + packaged renderer were proved.
    if ($hiddenPrivacyCoreWindows.Count -gt 0) {
        $backdrop=[ExplorerMenuProbe]::PrivacyBackdropAtPointer()
        if ($backdrop -ne [IntPtr]::Zero) {
            [uint32]$backdropOwner=0
            $null=[ExplorerMenuProbe]::GetWindowThreadProcessId($backdrop,[ref]$backdropOwner)
            $owner=Get-Process -Id $backdropOwner
            $backdropSid=Get-ProcessOwnerSid ([int]$backdropOwner)
            $nativeOwner=Get-CimInstance Win32_Process -Filter "ProcessId=$backdropOwner"
            $backdropIdentity=[ordered]@{window=[ExplorerMenuProbe]::WindowDescription($backdrop);relations=[ExplorerMenuProbe]::WindowRelations($backdrop);process_id=$backdropOwner;process_name=$owner.ProcessName;executable_path=$nativeOwner.ExecutablePath;session_id=$owner.SessionId;sid=$backdropSid;verified_cloud_experience_host_context=$true}
            [IO.File]::WriteAllText((Join-Path $Output 'runner-privacy.backdrop-identity.json'),($backdropIdentity | ConvertTo-Json -Depth 8),[Text.Encoding]::UTF8)
            if ($owner.ProcessName -ne 'explorer' -or $owner.SessionId -ne $currentSession -or $backdropSid -ne $currentSid) {
                throw 'Refusing an OOBE backdrop outside the verified current desktop Explorer identity'
            }
            Save-ExplorerState 'runner-privacy.backdrop.before'
            $script:hiddenPrivacyBackdrops.Add(@{window=$backdrop;owner=$backdropOwner})
            [ExplorerMenuProbe]::HidePrivacyBackdrop($backdrop,$backdropOwner)
            $deadline=[DateTime]::UtcNow.AddSeconds(10)
            while ([DateTime]::UtcNow -lt $deadline -and ![ExplorerMenuProbe]::PrivacyBackdropHidden($backdrop,$backdropOwner)) { Start-Sleep -Milliseconds 100 }
            if (![ExplorerMenuProbe]::PrivacyBackdropHidden($backdrop,$backdropOwner)) { throw 'Verified OOBE backdrop remains visible; actual Explorer UI acceptance is blocked' }
            $script:evidence.runner_desktop_preparation.backdrop_identity=$backdropIdentity
        }
    }
    $deadline=[DateTime]::UtcNow.AddSeconds(10)
    while ([DateTime]::UtcNow -lt $deadline -and @([ExplorerMenuProbe]::VisiblePrivacyWindows()).Count -gt 0) { Start-Sleep -Milliseconds 100 }
    Save-ExplorerState 'runner-privacy.after'
    if (@([ExplorerMenuProbe]::VisiblePrivacyWindows()).Count -gt 0) { throw 'Hosted-runner privacy overlay remains visible; actual Explorer UI acceptance is blocked' }
    $script:evidence.runner_desktop_preparation.applied=$true
}

function Invoke-CLI([string[]]$Arguments, [string]$Name, [int]$Expected=0) {
    $info = [Diagnostics.ProcessStartInfo]::new($script:portableBinary)
    $info.UseShellExecute = $false
    $info.RedirectStandardOutput = $true
    $info.RedirectStandardError = $true
    $info.RedirectStandardInput = $true
    foreach ($argument in $Arguments) { $info.ArgumentList.Add($argument) }
    $process = [Diagnostics.Process]::Start($info)
    $process.StandardInput.Close()
    $stdoutTask = $process.StandardOutput.ReadToEndAsync()
    $stderrTask = $process.StandardError.ReadToEndAsync()
    if (!$process.WaitForExit(30000)) { $process.Kill(); throw "CLI timed out: $Name" }
    $stdout = $stdoutTask.GetAwaiter().GetResult()
    $stderr = $stderrTask.GetAwaiter().GetResult()
    [IO.File]::WriteAllText((Join-Path $Output "$Name.stdout.txt"),$stdout,[Text.Encoding]::UTF8)
    [IO.File]::WriteAllText((Join-Path $Output "$Name.stderr.txt"),$stderr,[Text.Encoding]::UTF8)
    if ($process.ExitCode -ne $Expected) { throw "CLI status $($process.ExitCode), expected ${Expected}: $Name $stderr" }
    if (($stdout+$stderr).Contains('Press Enter')) { throw "Redirected CLI unexpectedly prompted: $Name" }
}

function Open-ExplorerFolder([string]$Directory) {
    # The COM object must represent an actual visible Explorer window, owned by
    # explorer.exe in our session, rather than a detached Shell namespace object.
    $script:shell = New-Object -ComObject Shell.Application
    $script:shell.Open($Directory)
    $deadline = [DateTime]::UtcNow.AddSeconds(30)
    while ([DateTime]::UtcNow -lt $deadline) {
        foreach ($window in @($script:shell.Windows())) {
            try {
                if ([IO.Path]::GetFullPath($window.Document.Folder.Self.Path) -eq $Directory -and
                    [ExplorerMenuProbe]::IsWindowVisible([IntPtr]$window.HWND)) {
                    [uint32]$owner=0
                    $null=[ExplorerMenuProbe]::GetWindowThreadProcessId([IntPtr]$window.HWND,[ref]$owner)
                    $p=Get-Process -Id $owner
                    if ($p.ProcessName -eq 'explorer' -and $p.SessionId -eq (Get-Process -Id $PID).SessionId) { return $window }
                }
            } catch { }
        }
        Start-Sleep -Milliseconds 250
    }
    throw "No usable interactive Explorer session. Registry readback does not satisfy menu acceptance."
}

function Find-PopupItem([string]$Label) {
    [uint32]$explorerOwner=0
    $null=[ExplorerMenuProbe]::GetWindowThreadProcessId([IntPtr]$script:explorerWindow.HWND,[ref]$explorerOwner)
    foreach ($window in [ExplorerMenuProbe]::VisibleMenus()) {
        [uint32]$popupOwner=0
        $null=[ExplorerMenuProbe]::GetWindowThreadProcessId($window,[ref]$popupOwner)
        if ($popupOwner -ne $explorerOwner) { continue }
        $menu = [ExplorerMenuProbe]::SendMessage($window,0x01e1,[IntPtr]::Zero,[IntPtr]::Zero) # MN_GETHMENU
        for ($i=0; $i -lt [ExplorerMenuProbe]::GetMenuItemCount($menu); $i++) {
            if ([ExplorerMenuProbe]::MenuText($menu,$i) -eq $Label) { return @{window=$window;menu=$menu;index=$i} }
        }
    }
    return $null
}

function Invoke-ExplorerExport([string]$InputFile,[string]$Name,[string]$ExpectedOutput,[bool]$Reject) {
    $script:currentCase=$Name
    $script:currentInputFile=$InputFile
    $item=$script:explorerWindow.Document.Folder.ParseName([IO.Path]::GetFileName($InputFile))
    if ($null -eq $item) { throw "Explorer cannot select validation file" }
    $script:explorerWindow.Document.SelectItem($item,1+4+8+16) # select, deselect others, ensure visible, focus
    if (![ExplorerMenuProbe]::SetForegroundWindow([IntPtr]$script:explorerWindow.HWND)) { throw "Cannot focus the actual Explorer window" }
    $deadline=[DateTime]::UtcNow.AddSeconds(10)
    $selected=$false
    while ([DateTime]::UtcNow -lt $deadline) {
        $selection=Get-ExplorerSelection
        if ($selection.selected_items.Count -eq 1 -and
            $selection.selected_items[0].path -eq $InputFile -and
            $selection.focused_item.path -eq $InputFile) { $selected=$true; break }
        Start-Sleep -Milliseconds 100
    }
    Save-ExplorerState "$Name.selected"
    if (!$selected) { throw 'Explorer did not confirm exactly the intended selected and focused TOP file' }
    $view=[ExplorerMenuProbe]::FileView([IntPtr]$script:explorerWindow.HWND)
    # WM_CONTEXTMENU with (-1,-1) requests the keyboard-positioned context menu
    # for the current selection. Dispatch to the actual native file view rather
    # than sending keys to whichever address/navigation control owns focus.
    # https://learn.microsoft.com/en-us/windows/win32/menurc/wm-contextmenu
    if (![ExplorerMenuProbe]::PostMessage($view,0x007b,$view,[IntPtr](-1))) {
        throw 'Cannot dispatch selected-file context menu to the actual Explorer view'
    }
    $deadline=[DateTime]::UtcNow.AddSeconds(10)
    $root=$null
    $modern=$false
    while ([DateTime]::UtcNow -lt $deadline -and $null -eq $root) {
        $root=Find-PopupItem 'PocketTopo Exporter'
        if ($null -ne $root) { break }
        if (@([ExplorerMenuProbe]::ModernMenus([IntPtr]$script:explorerWindow.HWND)).Count -gt 0) { $modern=$true; break }
        Start-Sleep -Milliseconds 100
    }
    if ($modern) {
        Save-ExplorerState "$Name.modern-popup"
        if (![Globalization.CultureInfo]::CurrentUICulture.Name.StartsWith('en', [StringComparison]::OrdinalIgnoreCase)) {
            throw 'Show more options keytip probe requires an English Windows runner UI; modern popup captured'
        }
        [ExplorerMenuProbe]::ModernMoreOptions([IntPtr]$script:explorerWindow.HWND)
        $deadline=[DateTime]::UtcNow.AddSeconds(10)
        while ([DateTime]::UtcNow -lt $deadline -and $null -eq $root) { $root=Find-PopupItem 'PocketTopo Exporter'; Start-Sleep -Milliseconds 100 }
    }
    Save-ExplorerState "$Name.popup"
    if ($null -eq $root) {
        if (@([ExplorerMenuProbe]::VisibleMenus()).Count -eq 0) { throw 'Explorer produced no visible native classic popup; diagnostic state and screenshot were captured' }
        throw "PocketTopo Exporter absent from actual Explorer classic popup; all popup labels captured"
    }
    $hoverEvidence=[ExplorerMenuProbe]::Hover($root.window,$root.menu,$root.index,$false)
    [IO.File]::WriteAllText((Join-Path $Output "$Name.hover.txt"),$hoverEvidence,[Text.Encoding]::UTF8)
    $deadline=[DateTime]::UtcNow.AddSeconds(10)
    $action=$null
    while ([DateTime]::UtcNow -lt $deadline -and $null -eq $action) { $action=Find-PopupItem 'Export as Text'; Start-Sleep -Milliseconds 100 }
    Save-ExplorerState "$Name.submenu"
    if ($null -eq $action) { throw "Explorer did not open the registered submenu" }
    $labels=@()
    for ($i=0; $i -lt [ExplorerMenuProbe]::GetMenuItemCount($action.menu); $i++) { $labels += [ExplorerMenuProbe]::MenuText($action.menu,$i) }
    $expected=@('Export as Text','Export as Survex','Export as Walls','Export as Therion interchange','Export as VisualTopo','Export as Toporobot','Export as DXF plan','Export as DXF side','Export as DXF 3D')
    if (@(Compare-Object $expected $labels).Count -ne 0 -or $labels.Count -ne 9) { throw "Actual Explorer submenu differs from all nine required entries: $labels" }
    $script:evidence.menu_entries=$labels
    [IO.File]::WriteAllText((Join-Path $Output "$Name.menu.txt"),($labels -join "`n"),[Text.Encoding]::UTF8)
    $clickEvidence=[ExplorerMenuProbe]::Hover($action.window,$action.menu,$action.index,$true)
    [IO.File]::WriteAllText((Join-Path $Output "$Name.click.txt"),$clickEvidence,[Text.Encoding]::UTF8)
    $process=$null
    $deadline=[DateTime]::UtcNow.AddSeconds(15)
    while ([DateTime]::UtcNow -lt $deadline -and $null -eq $process) {
        $launches=@(Get-CimInstance Win32_Process | Where-Object { $_.ExecutablePath -eq $script:portableBinary -and !$script:ownedProcesses.Contains([int]$_.ProcessId) })
        if ($launches.Count -eq 1) { $process=Get-Process -Id $launches[0].ProcessId; $script:ownedProcesses.Add($process.Id) }
        if ($launches.Count -gt 1) { throw "Ambiguous Explorer-launched process" }
        Start-Sleep -Milliseconds 100
    }
    if ($null -eq $process) { throw "Explorer menu did not launch the registered executable" }
    $script:evidence.last_launched_process=@{id=$process.Id;session_id=$process.SessionId;sid=(Get-ProcessOwnerSid $process.Id)}
    $text=''
    $deadline=[DateTime]::UtcNow.AddSeconds(20)
    while ([DateTime]::UtcNow -lt $deadline) {
        $process.Refresh()
        if ($process.HasExited) { throw "Interactive command exited before Enter; failure would not remain readable" }
        $text=[ExplorerMenuProbe]::ConsoleText([uint32]$process.Id,$false)
        if ($text.Contains('Press Enter to close')) { break }
        Start-Sleep -Milliseconds 100
    }
    [IO.File]::WriteAllText((Join-Path $Output "$Name.console.txt"),$text,[Text.Encoding]::UTF8)
    if (!$text.Contains('Press Enter to close')) { throw "No visible interactive status prompt" }
    if ($Reject) {
        if (!$text.Contains('Cannot export:') -or (Test-Path -LiteralPath $ExpectedOutput)) { throw "Actual menu failure was unreadable or published partial output" }
    } else {
        if (!$text.Contains($ExpectedOutput) -or !$text.Contains('report.json')) { throw "Visible console omitted output/report location" }
        foreach ($file in @('survey.txt','report.json')) { if (!(Test-Path -LiteralPath (Join-Path $ExpectedOutput $file))) { throw "Menu export incomplete" } }
    }
    Start-Sleep -Milliseconds 300
    $process.Refresh()
    if ($process.HasExited) { throw "Console did not remain open before Enter" }
    $expectedCode=0
    if ($Reject) { $expectedCode=1 }
    # Get-Process attaches to an existing process. Its ExitCode can be unavailable
    # after the process exits unless a live handle was retained. Keep a native
    # QUERY_LIMITED_INFORMATION|SYNCHRONIZE handle before delivering Enter.
    $nativeHandle=[ExplorerMenuProbe]::PinProcess([uint32]$process.Id)
    $exitState=[ordered]@{name=$Name;process_id=$process.Id;retained_handle=$nativeHandle.ToInt64();expected_exit_code=$expectedCode;wait_timeout_ms=10000}
    try {
        $exitState.native_code_before_enter=[ExplorerMenuProbe]::ProcessExitCode($nativeHandle)
        if ($exitState.native_code_before_enter -ne 259) { throw 'Native process ended before Enter' }
        $observation=[ExplorerMenuProbe]::EnterAndObserveExit([uint32]$process.Id,$nativeHandle,10000)
        $exitState.native_wait_result=$observation.WaitResult
        $exitState.native_exit_code=$observation.ExitCode
        $exitState.native_exit_code_type=$observation.ExitCode.GetType().FullName
        [IO.File]::WriteAllText((Join-Path $Output "$Name.after-enter.console.txt"),$observation.ConsoleAfterEnter,[Text.Encoding]::UTF8)
        try {
            $process.Refresh()
            $exitState.powershell_has_exited=$process.HasExited
            $cachedCode=$process.ExitCode
            $exitState.powershell_exit_code=$cachedCode
            $exitState.powershell_exit_code_type='null'
            if ($null -ne $cachedCode) { $exitState.powershell_exit_code_type=$cachedCode.GetType().FullName }
        } catch { $exitState.powershell_status_unavailable=$_.Exception.Message }
        if ($observation.WaitResult -ne 0 -or $observation.ExitCode -eq 259) {
            throw "Console did not close after Enter: native wait=$($observation.WaitResult), exit=$($observation.ExitCode)"
        }
        if ($observation.ExitCode -ne $expectedCode) {
            throw "Explorer command returned native exit $($observation.ExitCode) ($($exitState.native_exit_code_type)), expected $expectedCode"
        }
        $script:evidence.cases += @{ name=$Name; actual_explorer_menu=$true; show_more_options_used=$modern; exit_code=$observation.ExitCode; exit_code_source='GetExitCodeProcess retained native handle'; console_stayed_until_enter=$true; output=$ExpectedOutput }
    } catch {
        $exitState.failure=$_.Exception.Message
        throw
    } finally {
        [ExplorerMenuProbe]::ReleaseProcess($nativeHandle)
        [IO.File]::WriteAllText((Join-Path $Output "$Name.exit.json"),($exitState | ConvertTo-Json -Depth 8),[Text.Encoding]::UTF8)
    }
}

try {
    $work=Join-Path $Output "portable spaces Żółć `$ apostrophe's & test"
    New-Item -ItemType Directory -Path $work | Out-Null
    $script:portableBinary=Join-Path $work 'pockettopo-exporter.exe'
    Copy-Item -LiteralPath $Binary -Destination $script:portableBinary
    $sourceDirectory=Join-Path $work "source spaces Żółć %1 %L bare% `$ apostrophe's & test"
    New-Item -ItemType Directory -Path $sourceDirectory | Out-Null
    $inputFile=Join-Path $sourceDirectory "cave spaces Żółć %1 %L bare% `$ apostrophe's & test.top"
    Copy-Item -LiteralPath 'internal/top/testdata/api-drawings.top' -Destination $inputFile
    $inputHash=(Get-FileHash -LiteralPath $inputFile -Algorithm SHA256).Hash
    $base=[IO.Path]::Combine($sourceDirectory,[IO.Path]::GetFileNameWithoutExtension($inputFile)+'-text')
    $defaultTop=[Microsoft.Win32.Registry]::CurrentUser.OpenSubKey('Software\Classes\.top')
    $defaultBefore=$null
    if ($null -ne $defaultTop) { $defaultBefore=$defaultTop.GetValue(''); $defaultTop.Dispose() }
    Invoke-CLI @('windows-menu','install') 'install'
    $installed=$true
    Invoke-CLI @('windows-menu','install') 'reinstall'
    $key=[Microsoft.Win32.Registry]::CurrentUser.OpenSubKey($menuKey)
    if ($null -eq $key -or $key.GetValue('PocketTopoExporterExecutable') -ne $script:portableBinary -or $key.GetValueKind('Icon') -ne [Microsoft.Win32.RegistryValueKind]::String) { throw "Registry identity/string readback failed" }
    $key.Dispose()
    Prepare-HostedRunnerDesktop
    $script:explorerWindow=Open-ExplorerFolder $sourceDirectory
    $evidence.explorer_session=$true
    Invoke-ExplorerExport $inputFile 'first-export' $base $false
    $firstHashes=@{}
    foreach ($file in @('survey.txt','report.json')) { $firstHashes[$file]=(Get-FileHash -LiteralPath (Join-Path $base $file) -Algorithm SHA256).Hash }
    Invoke-ExplorerExport $inputFile 'second-export' ($base+'-2') $false
    foreach ($file in @('survey.txt','report.json')) { if ((Get-FileHash -LiteralPath (Join-Path $base $file) -Algorithm SHA256).Hash -ne $firstHashes[$file]) { throw "Repeated export overwrote existing output" } }
    # Check independent native bytes in addition to visible success/file existence.
    $nativeText=[IO.File]::ReadAllText('internal/exporttext/testdata/api-drawings.txt',[Text.Encoding]::UTF8)
    $nativeTitle="api-drawings   (m, 360)`r`n"
    if (!$nativeText.StartsWith($nativeTitle)) { throw "Unexpected independent native golden header" }
    # The native format includes the source basename in its first line. Only
    # that known title changes for the hostile-path case; all other bytes stay
    # exactly the independently captured original-program expectation.
    $expectedText=[IO.Path]::GetFileNameWithoutExtension($inputFile)+"   (m, 360)`r`n"+$nativeText.Substring($nativeTitle.Length)
    $expectedBytes=[Text.UTF8Encoding]::new($false).GetBytes($expectedText)
    $actualBytes=[IO.File]::ReadAllBytes((Join-Path $base 'survey.txt'))
    if ([Convert]::ToBase64String($actualBytes) -ne [Convert]::ToBase64String($expectedBytes)) { throw "Explorer text export differs from native golden" }
    $bad=Join-Path $sourceDirectory 'malformed.top'
    [IO.File]::WriteAllText($bad,'not a TOP',[Text.Encoding]::ASCII)
    Invoke-ExplorerExport $bad 'readable-error' (Join-Path $sourceDirectory 'malformed-text') $true
    Invoke-CLI @('shell-export',$bad,'--format','text') 'redirected-error' 1
    if ((Get-FileHash -LiteralPath $inputFile -Algorithm SHA256).Hash -ne $inputHash) { throw "Menu changed source bytes" }
    Invoke-CLI @('windows-menu','uninstall') 'uninstall'
    $installed=$false
    Invoke-CLI @('windows-menu','uninstall') 'uninstall-again'
    $left=[Microsoft.Win32.Registry]::CurrentUser.OpenSubKey($menuKey)
    if ($null -ne $left) { $left.Dispose();throw "Scoped uninstall left the owned key" }
    $defaultTop=[Microsoft.Win32.Registry]::CurrentUser.OpenSubKey('Software\Classes\.top')
    $defaultAfter=$null
    if ($null -ne $defaultTop) { $defaultAfter=$defaultTop.GetValue(''); $defaultTop.Dispose() }
    if ($defaultBefore -ne $defaultAfter) { throw "Menu changed default TOP application" }
    $evidence.uninstall=$true
    $evidence.validation_ok=$true
} catch {
    $evidence.failure=$_.Exception.Message
    try { Save-ExplorerState "$script:currentCase.failure" } catch { $evidence.diagnostic_failure=$_.Exception.Message }
    throw
} finally {
    $cleanupFailures=[Collections.Generic.List[string]]::new()
    $restoreFailures=[Collections.Generic.List[string]]::new()
    try {
        foreach ($processId in $ownedProcesses) {
            try {
                $p=Get-Process -Id $processId -ErrorAction SilentlyContinue
                if ($null -ne $p -and !$p.HasExited) { Stop-Process -Id $processId -Force }
            } catch { $cleanupFailures.Add("Owned process ${processId}: $($_.Exception.Message)") }
        }
        if ($null -ne $explorerWindow) {
            try { $explorerWindow.Quit() }
            catch { $cleanupFailures.Add("Owned Explorer window: $($_.Exception.Message)") }
            finally { $script:explorerWindow=$null }
        }
        if ($installed) {
            try { Invoke-CLI @('windows-menu','uninstall') 'cleanup-uninstall' }
            catch { $cleanupFailures.Add("Menu uninstall: $($_.Exception.Message)") }
        }
    } finally {
        # Always attempt every tracked overlay, even if an unrelated cleanup or
        # another window's restoration fails. An asynchronous show request is
        # insufficient: verify visibility before claiming restoration.
        foreach ($window in $hiddenPrivacyWindows) {
            try { [ExplorerMenuProbe]::RestorePrivacyWindow($window) }
            catch { $restoreFailures.Add("Overlay $($window.ToInt64()): $($_.Exception.Message)") }
        }
        foreach ($entry in $hiddenPrivacyBackdrops) {
            try {
                [ExplorerMenuProbe]::RestorePrivacyBackdrop($entry.window,$entry.owner)
                $deadline=[DateTime]::UtcNow.AddSeconds(10)
                while ([DateTime]::UtcNow -lt $deadline -and ![ExplorerMenuProbe]::PrivacyBackdropRestored($entry.window,$entry.owner)) { Start-Sleep -Milliseconds 100 }
                if (![ExplorerMenuProbe]::PrivacyBackdropRestored($entry.window,$entry.owner)) { throw 'Verified OOBE backdrop did not regain its visibility' }
            } catch { $restoreFailures.Add("Backdrop $($entry.window.ToInt64()): $($_.Exception.Message)") }
        }
        foreach ($entry in $hiddenPrivacyCoreWindows) {
            try {
                [ExplorerMenuProbe]::RestorePrivacyCoreWindow($entry.window,$entry.owner)
                $deadline=[DateTime]::UtcNow.AddSeconds(10)
                while ([DateTime]::UtcNow -lt $deadline -and ![ExplorerMenuProbe]::PrivacyCoreWindowRestored($entry.window,$entry.owner)) { Start-Sleep -Milliseconds 100 }
                if (![ExplorerMenuProbe]::PrivacyCoreWindowRestored($entry.window,$entry.owner)) { throw 'Verified privacy CoreWindow did not regain its visibility' }
            } catch { $restoreFailures.Add("Core overlay $($entry.window.ToInt64()): $($_.Exception.Message)") }
            finally { [ExplorerMenuProbe]::ReleaseProcess($entry.process_handle) }
        }
        if ($hiddenPrivacyWindows.Count -gt 0 -or $hiddenPrivacyCoreWindows.Count -gt 0 -or $hiddenPrivacyBackdrops.Count -gt 0) {
            try {
                $deadline=[DateTime]::UtcNow.AddSeconds(10)
                while ([DateTime]::UtcNow -lt $deadline -and ![ExplorerMenuProbe]::PrivacyWindowsRestored($hiddenPrivacyWindows.ToArray())) { Start-Sleep -Milliseconds 100 }
                if (![ExplorerMenuProbe]::PrivacyWindowsRestored($hiddenPrivacyWindows.ToArray())) { throw 'Original privacy-overlay windows did not regain their visibility' }
                if ($restoreFailures.Count -eq 0) { $evidence.runner_desktop_preparation.visibility_restored=$true }
                Save-ExplorerState 'runner-privacy.restored'
            } catch { $restoreFailures.Add($_.Exception.Message) }
        }
    }
    if ($cleanupFailures.Count -gt 0 -or $restoreFailures.Count -gt 0) {
        $evidence.cleanup_failures=$cleanupFailures.ToArray()
        $evidence.runner_desktop_restore_failures=$restoreFailures.ToArray()
        $evidence.validation_ok=$false
    }
    [IO.File]::WriteAllText((Join-Path $Output 'report.json'),($evidence | ConvertTo-Json -Depth 8),[Text.Encoding]::UTF8)
    if ($cleanupFailures.Count -gt 0 -or $restoreFailures.Count -gt 0) { throw 'Windows menu validation cleanup failed; see report.json for every recorded failure' }
}
