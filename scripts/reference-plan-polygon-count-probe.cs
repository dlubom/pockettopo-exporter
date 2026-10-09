// Optional P04c2 evidence: original table/Mapping.Read methods, then marker 1
// and one BinaryReader.ReadInt32. No Polygon.Read, points, color or TOP writes.
using System;
using System.IO;
using System.Reflection;
using System.Runtime.Serialization;
using System.Globalization;

class ReferencePlanPolygonCountProbe
{
    static BindingFlags binding = BindingFlags.Public | BindingFlags.NonPublic |
        BindingFlags.Instance | BindingFlags.Static;
    static Type mapping, fileReader, trip, station, reference, id, location;

    static void ReadMapping(object reader)
    {
        object value = FormatterServices.GetUninitializedObject(mapping);
        mapping.GetMethod("Read", binding).Invoke(value, new object[] { reader });
    }

    static void Fixture(string name, string path, int countBytes, bool full)
    {
        const int markerStart = 146;
        byte[] bytes = File.ReadAllBytes(path);
        // Literal offsets are pinned helper/format evidence, never Go output.
        MemoryStream stream = new MemoryStream(bytes, 0,
            full ? bytes.Length : markerStart + 1 + countBytes);
        stream.Position = 4;
        object reader = Activator.CreateInstance(fileReader, binding, null,
            new object[] { new BinaryReader(stream), 3 }, null);
        trip.GetMethod("Reset", binding).Invoke(null, null);
        int tripOffset = (int)trip.GetMethod("ReadList", binding).Invoke(null, new object[] { reader });
        int tableCount = new BinaryReader(stream).ReadInt32();
        for (int i = 0; i < tableCount; i++)
            station.GetMethod("Read", binding, null, new Type[] { fileReader, typeof(int) }, null)
                .Invoke(Activator.CreateInstance(station), new object[] { reader, tripOffset });
        tableCount = new BinaryReader(stream).ReadInt32();
        for (int i = 0; i < tableCount; i++)
        {
            object value = Activator.CreateInstance(reference, binding, null,
                new object[] { Activator.CreateInstance(id), Activator.CreateInstance(location) }, null);
            reference.GetMethod("Read", binding, null, new Type[] { fileReader }, null)
                .Invoke(value, new object[] { reader });
        }
        ReadMapping(reader);
        if (stream.Position != markerStart - 12) throw new Exception("Unexpected native plan offset");
        ReadMapping(reader);
        if (stream.Position != markerStart) throw new Exception("Unexpected native marker offset");
        BinaryReader raw = (BinaryReader)fileReader.GetField("r", binding).GetValue(reader);
        if (raw.ReadByte() != 1 || stream.Position != 147)
            throw new Exception("Unexpected native Polygon marker");
        try
        {
            int count = raw.ReadInt32();
            if (!full && countBytes < 4) throw new Exception("Incomplete count succeeded");
            if (count != 3 || stream.Position != 151) throw new Exception("Unexpected native count/span");
            Console.WriteLine(name + " count_start=147 count=" + count + " consumed=" +
                stream.Position + " tail=" + (stream.Length - stream.Position));
        }
        catch (EndOfStreamException)
        {
            if (full || countBytes == 4 || stream.Position != 147 + countBytes)
                throw new Exception("Unexpected native truncated count position");
            Console.WriteLine(name + " count_start=147 available=" + countBytes +
                " error=EndOfStreamException consumed=" + stream.Position + " tail=0");
        }
    }

    static void Literal(byte[] countBytes, int expected, bool tail)
    {
        MemoryStream stream = new MemoryStream();
        stream.WriteByte(1);
        stream.Write(countBytes, 0, countBytes.Length);
        if (tail) { stream.WriteByte(255); stream.WriteByte(128); }
        stream.Position = 0;
        object reader = Activator.CreateInstance(fileReader, binding, null,
            new object[] { new BinaryReader(stream), 3 }, null);
        BinaryReader raw = (BinaryReader)fileReader.GetField("r", binding).GetValue(reader);
        if (raw.ReadByte() != 1) throw new Exception("Unexpected literal marker");
        int count = raw.ReadInt32();
        if (count != expected || stream.Position != 5) throw new Exception("Unexpected literal count/span");
        Console.WriteLine("literal bytes=" + BitConverter.ToString(countBytes) + " count=" + count +
            " consumed=" + stream.Position + " tail=" + (stream.Length - stream.Position));
    }

    static void Run(string[] args)
    {
        System.Threading.Thread.CurrentThread.CurrentCulture = CultureInfo.InvariantCulture;
        Assembly assembly = Assembly.LoadFrom(args[0]);
        mapping = assembly.GetType("PocketTopo.Mapping", true);
        fileReader = assembly.GetType("PocketTopo.FileReader", true);
        trip = assembly.GetType("PocketTopo.Trip", true);
        station = assembly.GetType("PocketTopo.Station", true);
        reference = assembly.GetType("PocketTopo.Reference", true);
        id = assembly.GetType("PocketTopo.ID", true);
        location = assembly.GetType("PocketTopo.MetricLocation", true);
        Console.WriteLine("assembly=" + assembly.GetName().Version + " runtime=" + Environment.Version);
        for (int mode = 0; mode < 2; mode++)
        {
            if (mode == 1) mapping.GetMethod("SetVga", binding).Invoke(null, null);
            Console.WriteLine("PixPerMm=" + mapping.GetField("PixPerMm", binding).GetValue(null));
            Fixture("drawings-fixture", args[1], 4, true);
            Fixture("drawings-prefix", args[1], 4, false);
            for (int available = 0; available < 4; available++)
                Fixture("drawings-truncated", args[1], available, false);
        }
        // Literal little-endian bytes, independent of Go and BinaryWriter.
        byte[][] bytes = new byte[][] {
            new byte[] {0,0,0,0}, new byte[] {1,0,0,0}, new byte[] {3,0,0,0},
            new byte[] {3,2,1,0}, new byte[] {64,66,15,0}, new byte[] {65,66,15,0},
            new byte[] {255,255,255,127}, new byte[] {255,255,255,255},
            new byte[] {0,0,0,128} };
        int[] expected = new int[] {0,1,3,66051,1000000,1000001,Int32.MaxValue,-1,Int32.MinValue};
        for (int i = 0; i < bytes.Length; i++)
        {
            Literal(bytes[i], expected[i], false);
            Literal(bytes[i], expected[i], true);
        }
    }

    static int Main(string[] args)
    {
        try { Run(args); return 0; }
        catch (Exception error) { Console.Error.WriteLine(error); return 1; }
    }
}
