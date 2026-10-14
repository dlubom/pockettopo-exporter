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

Add-Type @'
using System;
using System.Text;
using System.Collections.Generic;
using System.Runtime.InteropServices;
public static class ExplorerMenuProbe {
 public delegate bool EnumProc(IntPtr h, IntPtr unused);
 [DllImport("user32.dll")] static extern bool EnumWindows(EnumProc cb, IntPtr arg);
 [DllImport("user32.dll", CharSet=CharSet.Unicode)] static extern int GetClassName(IntPtr h, StringBuilder b, int n);
 [DllImport("user32.dll")] public static extern bool IsWindowVisible(IntPtr h);
 [DllImport("user32.dll")] public static extern bool SetForegroundWindow(IntPtr h);
 [DllImport("user32.dll")] public static extern uint GetWindowThreadProcessId(IntPtr h, out uint pid);
 [DllImport("user32.dll")] public static extern IntPtr SendMessage(IntPtr h, uint msg, IntPtr w, IntPtr l);
 [DllImport("user32.dll")] public static extern int GetMenuItemCount(IntPtr h);
 [DllImport("user32.dll", CharSet=CharSet.Unicode)] static extern int GetMenuString(IntPtr h, uint item, StringBuilder b, int n, uint flags);
 [DllImport("user32.dll")] public static extern IntPtr GetSubMenu(IntPtr h, int item);
 [StructLayout(LayoutKind.Sequential)] public struct Rect { public int left,top,right,bottom; }
 [DllImport("user32.dll")] static extern bool GetMenuItemRect(IntPtr window, IntPtr menu, uint item, out Rect r);
 [DllImport("user32.dll")] static extern bool SetCursorPos(int x, int y);
 [DllImport("user32.dll")] static extern void mouse_event(uint flags,uint x,uint y,uint data,UIntPtr extra);
 [DllImport("user32.dll")] public static extern void keybd_event(byte key,byte scan,uint flags,UIntPtr extra);
 public static string Class(IntPtr h) {var b=new StringBuilder(256);GetClassName(h,b,b.Capacity);return b.ToString();}
 public static IntPtr[] VisibleMenus() {var a=new List<IntPtr>();EnumWindows((h,x)=>{if(IsWindowVisible(h)&&Class(h)=="#32768")a.Add(h);return true;},IntPtr.Zero);return a.ToArray();}
 public static string MenuText(IntPtr menu,int item) {var b=new StringBuilder(2048);GetMenuString(menu,(uint)item,b,b.Capacity,0x400);return b.ToString().Replace("&","");}
 public static void Hover(IntPtr menu,int item,bool click) {Rect r;if(!GetMenuItemRect(IntPtr.Zero,menu,(uint)item,out r))throw new Exception("Cannot locate actual popup item");if(!SetCursorPos((r.left+r.right)/2,(r.top+r.bottom)/2))throw new Exception("Cannot move pointer on interactive desktop");if(click){mouse_event(2,0,0,0,UIntPtr.Zero);mouse_event(4,0,0,0,UIntPtr.Zero);}}
 public static void ClassicMenu() {keybd_event(0x10,0,0,UIntPtr.Zero);keybd_event(0x79,0,0,UIntPtr.Zero);keybd_event(0x79,0,2,UIntPtr.Zero);keybd_event(0x10,0,2,UIntPtr.Zero);}
 [DllImport("kernel32.dll")] static extern bool FreeConsole();
 [DllImport("kernel32.dll")] static extern bool AttachConsole(uint pid);
 [DllImport("kernel32.dll", CharSet=CharSet.Unicode, SetLastError=true)] static extern IntPtr CreateFile(string name,uint access,uint share,IntPtr sec,uint mode,uint flags,IntPtr template);
 [DllImport("kernel32.dll")] static extern bool CloseHandle(IntPtr h);
 [StructLayout(LayoutKind.Sequential)] struct Coord {public short x,y; public Coord(short a,short b){x=a;y=b;}}
 [StructLayout(LayoutKind.Sequential)] struct ConsoleInfo {public Coord size,cursor;public ushort attributes;public short left,top,right,bottom;public Coord maximum;}
 [DllImport("kernel32.dll")] static extern bool GetConsoleScreenBufferInfo(IntPtr h,out ConsoleInfo info);
 [DllImport("kernel32.dll", CharSet=CharSet.Unicode)] static extern bool ReadConsoleOutputCharacter(IntPtr h,StringBuilder b,uint n,Coord origin,out uint read);
 [StructLayout(LayoutKind.Explicit, CharSet=CharSet.Unicode, Size=20)] struct InputRecord {
  [FieldOffset(0)] public ushort type; [FieldOffset(4)] public int down; [FieldOffset(8)] public ushort repeat; [FieldOffset(10)] public ushort key; [FieldOffset(12)] public ushort scan; [FieldOffset(14)] public char character; [FieldOffset(16)] public uint control;
 }
 [DllImport("kernel32.dll", CharSet=CharSet.Unicode)] static extern bool WriteConsoleInput(IntPtr h,InputRecord[] inputs,uint n,out uint written);
 public static string ConsoleText(uint pid,bool enter) {
  FreeConsole();
  if(!AttachConsole(pid)){AttachConsole(0xffffffff);throw new Exception("Cannot attach to Explorer-launched console");}
  IntPtr output=IntPtr.Zero,input=IntPtr.Zero;
  try {
   output=CreateFile("CONOUT$",0x80000000,3,IntPtr.Zero,3,0,IntPtr.Zero);
   ConsoleInfo info;if(!GetConsoleScreenBufferInfo(output,out info))throw new Exception("No readable native console buffer");
   uint n=(uint)(info.size.x*(info.cursor.y+1));if(n>1000000)throw new Exception("Unexpected console output size");
   var b=new StringBuilder((int)n);uint read;if(!ReadConsoleOutputCharacter(output,b,n,new Coord(0,0),out read))throw new Exception("Cannot read visible console status");
   if(enter){input=CreateFile("CONIN$",0x40000000,3,IntPtr.Zero,3,0,IntPtr.Zero);var records=new InputRecord[]{new InputRecord{type=1,down=1,repeat=1,key=13,character='\r'},new InputRecord{type=1,down=0,repeat=1,key=13,character='\r'}};uint written;if(!WriteConsoleInput(input,records,2,out written)||written!=2)throw new Exception("Cannot deliver Enter to native console");}
   return b.ToString();
  } finally {if(output!=IntPtr.Zero)CloseHandle(output);if(input!=IntPtr.Zero)CloseHandle(input);FreeConsole();AttachConsole(0xffffffff);}
 }
}
'@

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
    foreach ($window in [ExplorerMenuProbe]::VisibleMenus()) {
        $menu = [ExplorerMenuProbe]::SendMessage($window,0x01e1,[IntPtr]::Zero,[IntPtr]::Zero) # MN_GETHMENU
        for ($i=0; $i -lt [ExplorerMenuProbe]::GetMenuItemCount($menu); $i++) {
            if ([ExplorerMenuProbe]::MenuText($menu,$i) -eq $Label) { return @{window=$window;menu=$menu;index=$i} }
        }
    }
    return $null
}

function Invoke-ExplorerExport([string]$InputFile,[string]$Name,[string]$ExpectedOutput,[bool]$Reject) {
    $item=$script:explorerWindow.Document.Folder.ParseName([IO.Path]::GetFileName($InputFile))
    if ($null -eq $item) { throw "Explorer cannot select validation file" }
    $script:explorerWindow.Document.SelectItem($item,1+4+8+16) # select, deselect others, ensure visible, focus
    if (![ExplorerMenuProbe]::SetForegroundWindow([IntPtr]$script:explorerWindow.HWND)) { throw "Cannot focus the actual Explorer window" }
    Start-Sleep -Milliseconds 300
    [ExplorerMenuProbe]::ClassicMenu()
    $deadline=[DateTime]::UtcNow.AddSeconds(10)
    $root=$null
    while ([DateTime]::UtcNow -lt $deadline -and $null -eq $root) { $root=Find-PopupItem 'PocketTopo Exporter'; Start-Sleep -Milliseconds 100 }
    if ($null -eq $root) { throw "PocketTopo Exporter absent from actual Explorer classic popup" }
    [ExplorerMenuProbe]::Hover($root.menu,$root.index,$false)
    $deadline=[DateTime]::UtcNow.AddSeconds(10)
    $action=$null
    while ([DateTime]::UtcNow -lt $deadline -and $null -eq $action) { $action=Find-PopupItem 'Export as Text'; Start-Sleep -Milliseconds 100 }
    if ($null -eq $action) { throw "Explorer did not open the registered submenu" }
    $labels=@()
    for ($i=0; $i -lt [ExplorerMenuProbe]::GetMenuItemCount($action.menu); $i++) { $labels += [ExplorerMenuProbe]::MenuText($action.menu,$i) }
    $expected=@('Export as Text','Export as Survex','Export as Walls','Export as Therion interchange','Export as VisualTopo','Export as Toporobot','Export as DXF plan','Export as DXF side','Export as DXF 3D')
    if (@(Compare-Object $expected $labels).Count -ne 0 -or $labels.Count -ne 9) { throw "Actual Explorer submenu differs from all nine required entries: $labels" }
    $script:evidence.menu_entries=$labels
    [IO.File]::WriteAllText((Join-Path $Output "$Name.menu.txt"),($labels -join "`n"),[Text.Encoding]::UTF8)
    [ExplorerMenuProbe]::Hover($action.menu,$action.index,$true)
    $process=$null
    $deadline=[DateTime]::UtcNow.AddSeconds(15)
    while ([DateTime]::UtcNow -lt $deadline -and $null -eq $process) {
        $launches=@(Get-CimInstance Win32_Process | Where-Object { $_.ExecutablePath -eq $script:portableBinary -and !$script:ownedProcesses.Contains([int]$_.ProcessId) })
        if ($launches.Count -eq 1) { $process=Get-Process -Id $launches[0].ProcessId; $script:ownedProcesses.Add($process.Id) }
        if ($launches.Count -gt 1) { throw "Ambiguous Explorer-launched process" }
        Start-Sleep -Milliseconds 100
    }
    if ($null -eq $process) { throw "Explorer menu did not launch the registered executable" }
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
    $null=[ExplorerMenuProbe]::ConsoleText([uint32]$process.Id,$true)
    if (!$process.WaitForExit(10000)) { throw "Console did not close after Enter" }
    $expectedCode=0
    if ($Reject) { $expectedCode=1 }
    if ($process.ExitCode -ne $expectedCode) { throw "Explorer command returned incorrect status" }
    $script:evidence.cases += @{ name=$Name; actual_explorer_menu=$true; exit_code=$process.ExitCode; console_stayed_until_enter=$true; output=$ExpectedOutput }
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
    throw
} finally {
    foreach ($processId in $ownedProcesses) { $p=Get-Process -Id $processId -ErrorAction SilentlyContinue; if ($null -ne $p -and !$p.HasExited) { Stop-Process -Id $processId -Force } }
    if ($null -ne $explorerWindow) { $explorerWindow.Quit() }
    if ($installed) { try { Invoke-CLI @('windows-menu','uninstall') 'cleanup-uninstall' } catch { $evidence.cleanup_failure=$_.Exception.Message } }
    [IO.File]::WriteAllText((Join-Path $Output 'report.json'),($evidence | ConvertTo-Json -Depth 8),[Text.Encoding]::UTF8)
}
