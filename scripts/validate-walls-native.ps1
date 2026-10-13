# Run on an isolated Windows Actions runner, against the pinned original Walls.
# Win32 messages target only the newly launched validation process.
param([Parameter(Mandatory=$true)][string]$Binary,
      [Parameter(Mandatory=$true)][string]$Walls,
      [string]$Output = "build/walls-native")
$ErrorActionPreference = "Stop"
$Binary = (Resolve-Path $Binary).Path
$Walls = (Resolve-Path $Walls).Path
$Output = [IO.Path]::GetFullPath($Output)
if (Test-Path $Output) { throw "Native validation output must be new" }
New-Item -ItemType Directory $Output | Out-Null
Add-Type @'
using System;
using System.Text;
using System.Collections.Generic;
using System.Runtime.InteropServices;
public static class WallsWindows {
 public delegate bool EnumProc(IntPtr h, IntPtr unused);
 [DllImport("user32.dll")] public static extern bool EnumWindows(EnumProc cb, IntPtr arg);
 [DllImport("user32.dll")] public static extern bool EnumChildWindows(IntPtr h, EnumProc cb, IntPtr arg);
 [DllImport("user32.dll")] public static extern uint GetWindowThreadProcessId(IntPtr h, out uint pid);
 [DllImport("user32.dll", CharSet=CharSet.Unicode)] public static extern int GetWindowText(IntPtr h, StringBuilder b, int n);
 [DllImport("user32.dll", CharSet=CharSet.Unicode)] public static extern int GetClassName(IntPtr h, StringBuilder b, int n);
 [DllImport("user32.dll")] public static extern IntPtr GetWindow(IntPtr h, uint command);
 [DllImport("user32.dll")] public static extern int GetDlgCtrlID(IntPtr h);
 [DllImport("user32.dll")] public static extern bool IsWindowEnabled(IntPtr h);
 [DllImport("user32.dll")] public static extern bool IsWindowVisible(IntPtr h);
 [DllImport("user32.dll")] public static extern bool PostMessage(IntPtr h, uint msg, IntPtr w, IntPtr l);
 public static string Text(IntPtr h) { var b=new StringBuilder(2048); GetWindowText(h,b,b.Capacity);return b.ToString(); }
 public static string Class(IntPtr h) { var b=new StringBuilder(256); GetClassName(h,b,b.Capacity);return b.ToString(); }
 public static IntPtr[] Owned(int pid) { var a=new List<IntPtr>();EnumWindows((h,x)=>{uint p;GetWindowThreadProcessId(h,out p);if(p==pid)a.Add(h);return true;},IntPtr.Zero);return a.ToArray(); }
 public static string Describe(IntPtr h) { var a=new List<string>();a.Add(Class(h)+" | "+Text(h));EnumChildWindows(h,(c,x)=>{var t=Text(c);if(t.Length>0)a.Add(Class(c)+" | "+t);return true;},IntPtr.Zero);return String.Join("\n",a); }
 public static bool Completed(int pid) {
  int done=0;
  foreach(var h in Owned(pid)) EnumChildWindows(h,(c,x)=>{
   if(GetDlgCtrlID(c)==2 && Class(c)=="Button" && Text(c)=="Abort" && !IsWindowEnabled(c) && !IsWindowVisible(c)) done++;
   return true;
  },IntPtr.Zero);
  return done==1;
 }
}
'@

function Compile-Case([string]$Name, [bool]$Reject) {
    $caseDir = Join-Path $Output $Name
    New-Item -ItemType Directory $caseDir | Out-Null
    if ($Reject) {
        $srv = Join-Path $caseDir "survey.srv"
        [IO.File]::WriteAllText($srv, "#Units Reset Meters Order=DAV A=D V=D Case=Mixed`r`n1.0 1.1 1.000 0.000000 100.000000`r`n", [Text.Encoding]::ASCII)
    } else {
        & $Binary export "internal/exportwalls/testdata/$Name.top" --format srv --output (Join-Path $caseDir "export")
        if ($LASTEXITCODE -ne 0) { throw "Export failed: $Name" }
        Copy-Item (Join-Path $caseDir "export/survey.srv") (Join-Path $caseDir "survey.srv")
    }
    $project = Join-Path $caseDir "p15.wpj"
    [IO.File]::WriteAllText($project, ".BOOK P15 validation`r`n.NAME p15`r`n.STATUS 2`r`n.SURVEY P15 measurements`r`n.NAME survey`r`n.STATUS 0`r`n.ENDBOOK`r`n", [Text.Encoding]::ASCII)
    $process = Start-Process -FilePath $Walls -ArgumentList ('"'+$project+'"') -PassThru
    try {
        $null = $process.WaitForInputIdle(15000)
        $sent = $false
        $deadline = [DateTime]::UtcNow.AddSeconds(45)
        $states = [Collections.Generic.List[string]]::new()
        while ([DateTime]::UtcNow -lt $deadline) {
            $process.Refresh()
            if ($process.HasExited) { throw "Walls exited before validation" }
            $windows = [WallsWindows]::Owned($process.Id)
            foreach ($window in $windows) {
                $description = [WallsWindows]::Describe($window)
                if (!$states.Contains($description)) { $states.Add($description) }
                # Only this known startup notice is dismissed. Error dialogs are
                # evidence and are never accepted to manufacture success.
                if ([WallsWindows]::Text($window) -eq "Tip of the Day") {
                    $null = [WallsWindows]::PostMessage($window, 0x111, [IntPtr]2, [IntPtr]::Zero)
                }
            }
            $main = @($windows | Where-Object {
                [WallsWindows]::GetWindow($_,4) -eq [IntPtr]::Zero -and
                [WallsWindows]::Class($_).StartsWith("Afx:") -and
                [WallsWindows]::Text($_) -match "Walls"
            })
            $modal = @($windows | Where-Object { [WallsWindows]::Class($_) -eq "#32770" -and [WallsWindows]::Text($_) -ne "Tip of the Day" })
            if (!$sent -and $main.Count -eq 1 -and $modal.Count -eq 0) {
                # ID_PRJ_COMPILEITEM performs the program's own normal UI path.
                if (![WallsWindows]::PostMessage($main[0],0x111,[IntPtr]32810,[IntPtr]::Zero)) { throw "Cannot dispatch Walls compile" }
                $sent = $true
            }
            $success = @(Get-ChildItem $caseDir -Recurse -Filter *.ntn)
            if ($Reject -and $sent -and $modal.Count -gt 0) { break }
            if (!$Reject -and $success.Count -eq 1 -and [WallsWindows]::Completed($process.Id)) { break }
            Start-Sleep -Milliseconds 250
        }
        [IO.File]::WriteAllText((Join-Path $caseDir "windows.txt"), ($states -join "`n---`n"), [Text.Encoding]::UTF8)
        if (!$sent) { throw "No ready Walls project window: $Name" }
        $success = @(Get-ChildItem $caseDir -Recurse -Filter *.ntn)
        if ($Reject) {
            if ($success.Count -ne 0 -or ($states -join "`n") -notmatch '(?i)inclination|vertical angle') { throw "Native negative control did not reject inclination" }
        } else {
            if ($success.Count -ne 1 -or ![WallsWindows]::Completed($process.Id)) { throw "Walls did not complete original compilation: $Name" }
            foreach ($extension in @("ntv","nts","ntp")) {
                if (@(Get-ChildItem $caseDir -Recurse -Filter "*.$extension").Count -ne 1) { throw "Missing native $extension result: $Name" }
            }
            # File creation precedes adjustment completion. The native app must
            # close its DBF handles normally; the Python gate checks completed
            # headers and the final survey summary, not file existence alone.
            if (!$process.CloseMainWindow() -or !$process.WaitForExit(10000)) {
                throw "Walls did not close normally: $Name"
            }
        }
    } finally {
        # Shut down only our own disposable process, releasing native DBF locks.
        if (!$process.HasExited) { Stop-Process -Id $process.Id -Force; $process.WaitForExit() }
    }
}
foreach ($name in @("features","mixed","ties","aliases")) { Compile-Case $name $false }
Compile-Case "negative-control" $true
& python scripts/validate-walls-ntv.py --directory $Output --walls $Walls --report (Join-Path $Output "report.json")
if ($LASTEXITCODE -ne 0) { throw "Native Walls vector validation failed" }
