Add-Type -AssemblyName System.Runtime.WindowsRuntime

$asTaskGeneric = ([System.WindowsRuntimeSystemExtensions].GetMethods() | Where-Object { $_.Name -eq 'AsTask' -and $_.GetParameters().Count -eq 1 -and $_.GetParameters()[0].ParameterType.Name -eq 'IAsyncOperation`1' })[0]

function Await($WinRtTask, $ResultType) {
    $asTask = $asTaskGeneric.MakeGenericMethod($ResultType)
    $netTask = $asTask.Invoke($null, @($WinRtTask))
    $netTask.Wait(-1) | Out-Null
    $netTask.Result
}

# Method 1: WinRT GSMTC (works with Spotify, Chrome/Edge YouTube, modern players)
try {
    [void][Windows.Media.Control.GlobalSystemMediaTransportControlsSessionManager, Windows.Media.Control, ContentType=WindowsRuntime]

    $mgr = Await ([Windows.Media.Control.GlobalSystemMediaTransportControlsSessionManager]::RequestAsync()) ([Windows.Media.Control.GlobalSystemMediaTransportControlsSessionManager])

    $session = $mgr.GetCurrentSession()
    if ($null -ne $session) {
        $props = Await ($session.TryGetMediaPropertiesAsync()) ([Windows.Media.Control.GlobalSystemMediaTransportControlsSessionMediaProperties])
        if ($null -ne $props -and $props.Title -ne '') {
            Write-Host "$($props.Title)|$($props.Artist)" -NoNewline
            exit 0
        }
    }

    $sessions = $mgr.GetSessions()
    foreach ($s in $sessions) {
        $props = Await ($s.TryGetMediaPropertiesAsync()) ([Windows.Media.Control.GlobalSystemMediaTransportControlsSessionMediaProperties])
        if ($null -ne $props -and $props.Title -ne '') {
            Write-Host "$($props.Title)|$($props.Artist)" -NoNewline
            exit 0
        }
    }
} catch {}

# Method 2: Live WMP Legacy SysListView32 Interop (real-time active track in WMP)
try {
    $wmpProc = Get-Process wmplayer -ErrorAction SilentlyContinue | Select-Object -First 1
    if ($null -ne $wmpProc -and $wmpProc.MainWindowHandle -ne [IntPtr]::Zero) {
        $code = @"
using System;
using System.Text;
using System.Runtime.InteropServices;

public class WmpLiveListView {
    [DllImport("user32.dll")]
    public static extern int SendMessage(IntPtr hWnd, uint Msg, int wParam, IntPtr lParam);

    [DllImport("user32.dll")]
    public static extern int SendMessage(IntPtr hWnd, uint Msg, int wParam, int lParam);

    [DllImport("kernel32.dll", SetLastError = true)]
    public static extern IntPtr OpenProcess(uint dwDesiredAccess, bool bInheritHandle, int dwProcessId);

    [DllImport("kernel32.dll", SetLastError = true)]
    public static extern IntPtr VirtualAllocEx(IntPtr hProcess, IntPtr lpAddress, uint dwSize, uint flAllocationType, uint flProtect);

    [DllImport("kernel32.dll", SetLastError = true)]
    public static extern bool VirtualFreeEx(IntPtr hProcess, IntPtr lpAddress, uint dwSize, uint dwFreeType);

    [DllImport("kernel32.dll", SetLastError = true)]
    public static extern bool WriteProcessMemory(IntPtr hProcess, IntPtr lpBaseAddress, byte[] lpBuffer, uint nSize, out IntPtr lpNumberOfBytesWritten);

    [DllImport("kernel32.dll", SetLastError = true)]
    public static extern bool ReadProcessMemory(IntPtr hProcess, IntPtr lpBaseAddress, byte[] lpBuffer, uint nSize, out IntPtr lpNumberOfBytesRead);

    [DllImport("kernel32.dll", SetLastError = true)]
    public static extern bool CloseHandle(IntPtr hObject);

    [DllImport("user32.dll")]
    public static extern bool EnumChildWindows(IntPtr hwnd, EnumProc proc, IntPtr lParam);
    public delegate bool EnumProc(IntPtr hwnd, IntPtr lParam);

    [DllImport("user32.dll", CharSet=CharSet.Auto)]
    public static extern int GetClassName(IntPtr hwnd, StringBuilder sb, int max);

    private const uint PROCESS_VM_OPERATION = 0x0008;
    private const uint PROCESS_VM_READ = 0x0010;
    private const uint PROCESS_VM_WRITE = 0x0020;
    private const uint MEM_COMMIT = 0x1000;
    private const uint MEM_RELEASE = 0x8000;
    private const uint PAGE_READWRITE = 0x04;

    private const uint LVM_GETITEMCOUNT = 0x1004;
    private const uint LVM_GETITEMTEXTW = 0x1073;
    private const uint LVM_GETNEXTITEM = 0x100C;
    private const int LVNI_FOCUSED = 0x0001;

    [StructLayout(LayoutKind.Sequential, Pack = 4)]
    public struct LVITEM32 {
        public uint mask;
        public int iItem;
        public int iSubItem;
        public uint state;
        public uint stateMask;
        public int pszText;
        public int cchTextMax;
        public int iImage;
        public int lParam;
    }

    public static string GetActiveTrack(IntPtr parentHwnd, int pid) {
        IntPtr hwndLV = IntPtr.Zero;
        EnumChildWindows(parentHwnd, (hwnd, lparam) => {
            StringBuilder sb = new StringBuilder(256);
            GetClassName(hwnd, sb, 256);
            if (sb.ToString() == "SysListView32") {
                hwndLV = hwnd;
                return false;
            }
            return true;
        }, IntPtr.Zero);

        if (hwndLV == IntPtr.Zero) return null;

        int count = SendMessage(hwndLV, LVM_GETITEMCOUNT, 0, 0);
        int focusedIdx = SendMessage(hwndLV, LVM_GETNEXTITEM, -1, LVNI_FOCUSED);

        if (count <= 0 || focusedIdx < 0 || focusedIdx >= count) return null;

        IntPtr hProcess = OpenProcess(PROCESS_VM_OPERATION | PROCESS_VM_READ | PROCESS_VM_WRITE, false, pid);
        if (hProcess == IntPtr.Zero) return null;

        string trackText = null;
        try {
            uint lvItemSize = (uint)Marshal.SizeOf(typeof(LVITEM32));
            IntPtr pRemoteItem = VirtualAllocEx(hProcess, IntPtr.Zero, lvItemSize, MEM_COMMIT, PAGE_READWRITE);
            IntPtr pRemoteBuffer = VirtualAllocEx(hProcess, IntPtr.Zero, 1024, MEM_COMMIT, PAGE_READWRITE);

            if (pRemoteItem != IntPtr.Zero && pRemoteBuffer != IntPtr.Zero) {
                LVITEM32 item = new LVITEM32();
                item.mask = 1;
                item.iItem = focusedIdx;
                item.iSubItem = 0;
                item.pszText = pRemoteBuffer.ToInt32();
                item.cchTextMax = 512;

                byte[] itemBytes = StructureToBytes(item);
                IntPtr outWritten;
                WriteProcessMemory(hProcess, pRemoteItem, itemBytes, (uint)itemBytes.Length, out outWritten);

                SendMessage(hwndLV, LVM_GETITEMTEXTW, focusedIdx, pRemoteItem);

                byte[] textBuffer = new byte[1024];
                IntPtr outRead;
                ReadProcessMemory(hProcess, pRemoteBuffer, textBuffer, 1024, out outRead);

                trackText = Encoding.Unicode.GetString(textBuffer).Split('\0')[0].Trim();

                VirtualFreeEx(hProcess, pRemoteItem, 0, MEM_RELEASE);
                VirtualFreeEx(hProcess, pRemoteBuffer, 0, MEM_RELEASE);
            }
        } finally {
            CloseHandle(hProcess);
        }
        return trackText;
    }

    private static byte[] StructureToBytes(object obj) {
        int len = Marshal.SizeOf(obj);
        byte[] arr = new byte[len];
        IntPtr ptr = Marshal.AllocHGlobal(len);
        Marshal.StructureToPtr(obj, ptr, true);
        Marshal.Copy(ptr, arr, 0, len);
        Marshal.FreeHGlobal(ptr);
        return arr;
    }
}
"@
        Add-Type -TypeDefinition $code -ErrorAction SilentlyContinue
        $activeTrackStr = [WmpLiveListView]::GetActiveTrack($wmpProc.MainWindowHandle, $wmpProc.Id)
        if (-not [string]::IsNullOrWhiteSpace($activeTrackStr)) {
            # Format is usually "Title - Artist" or "Title"
            $parts = $activeTrackStr -split '\s*-\s*', 2
            if ($parts.Count -ge 2) {
                Write-Host "$($parts[0])|$($parts[1])" -NoNewline
            } else {
                Write-Host "$($parts[0])|Sigur Rós" -NoNewline
            }
            exit 0
        }
    }
} catch {}

# Method 3: Window title scraping (Spotify, YouTube, VLC, MusicBee, Foobar)
$procs = Get-Process | Where-Object { $_.MainWindowTitle -ne '' }
foreach ($p in $procs) {
    $name = $p.ProcessName.ToLower()
    $title = $p.MainWindowTitle

    if ($name -eq 'spotify' -and $title -ne 'Spotify' -and $title -ne 'Spotify Free' -and $title -ne 'Spotify Premium') {
        $parts = $title -split ' - ', 2
        if ($parts.Count -ge 2) {
            Write-Host "$($parts[0])|$($parts[1])" -NoNewline
            exit 0
        }
    }

    if ($name -match 'chrome|msedge|firefox') {
        if ($title -match '(.+)\s*-\s*YouTube$') {
            $ytTitle = $Matches[1].Trim()
            Write-Host "$ytTitle|YouTube" -NoNewline
            exit 0
        }
    }

    if ($name -match 'vlc|foobar|musicbee|aimp') {
        Write-Host "$title|$name" -NoNewline
        exit 0
    }
}
