// Optional P04c3 evidence: original table/Mapping.Read methods, then raw
// signed X/Y pairs. No Polygon.Read, color, large allocations or TOP writes.
using System;
using System.IO;
using System.Reflection;
using System.Runtime.Serialization;
using System.Globalization;

class ReferencePlanPolygonPointsProbe
{
    static BindingFlags binding = BindingFlags.Public | BindingFlags.NonPublic |
        BindingFlags.Instance | BindingFlags.Static;
    static Type mapping, fileReader, trip, station, reference, id, location;

    static void ReadMapping(object reader)
    {
        object value = FormatterServices.GetUninitializedObject(mapping);
        mapping.GetMethod("Read", binding).Invoke(value, new object[] { reader });
    }

    static void Fixture(string name, string path, int pointBytes, bool full)
    {
        const int markerStart = 146;
        byte[] bytes = File.ReadAllBytes(path);
        // Literal offsets are pinned helper/format evidence, never Go output.
        MemoryStream stream = new MemoryStream(bytes, 0,
            full ? bytes.Length : 151 + pointBytes);
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
        int count = raw.ReadInt32();
        if (count != 3 || stream.Position != 151) throw new Exception("Unexpected count/span");
        ReadPoints(name, raw, stream, new int[,] {{-6000,-2000},{-5500,-1000},{-4500,-1500}}, 151, pointBytes, full);
    }

    static void ReadPoints(string name, BinaryReader raw, MemoryStream stream,
        int[,] expected, int start, int available, bool full)
    {
        string values = "";
        try
        {
            for (int i = 0; i < expected.GetLength(0); i++)
            {
                int x = raw.ReadInt32();
                int y = raw.ReadInt32();
                if (x != expected[i,0] || y != expected[i,1]) throw new Exception("Unexpected coordinate");
                if (i > 0) values += ";";
                values += x + "," + y;
            }
            if (stream.Position != start + 8 * expected.GetLength(0)) throw new Exception("Unexpected point end");
            Console.WriteLine(name + " points=" + values + " start=" + start + " consumed=" + stream.Position + " tail=" + (stream.Length-stream.Position));
        }
        catch (EndOfStreamException)
        {
            if (full || available >= 8 * expected.GetLength(0) || stream.Position != start + available)
                throw new Exception("Unexpected truncated point position");
            Console.WriteLine(name + " start=" + start + " available=" + available + " error=EndOfStreamException consumed=" + stream.Position + " tail=0");
        }
    }

    static void Literal(byte[] bytes, int[,] expected, bool tail)
    {
        MemoryStream stream = new MemoryStream();
        stream.Write(bytes, 0, bytes.Length);
        if (tail) { stream.WriteByte(255); stream.WriteByte(128); }
        stream.Position = 0;
        object reader = Activator.CreateInstance(fileReader, binding, null,
            new object[] { new BinaryReader(stream), 3 }, null);
        BinaryReader raw = (BinaryReader)fileReader.GetField("r", binding).GetValue(reader);
        if (raw.ReadByte() != 1 || raw.ReadInt32() != expected.GetLength(0)) throw new Exception("Unexpected literal count");
        ReadPoints("literal", raw, stream, expected, 5, bytes.Length-5, true);
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
        for (int mode=0; mode<2; mode++)
        {
            if (mode==1) mapping.GetMethod("SetVga", binding).Invoke(null,null);
            Console.WriteLine("PixPerMm=" + mapping.GetField("PixPerMm",binding).GetValue(null));
            Fixture("drawings-fixture",args[1],24,true);
            Fixture("drawings-prefix",args[1],24,false);
            for (int available=0; available<24; available++) Fixture("drawings-truncated",args[1],available,false);
        }
        for (int tail=0; tail<2; tail++)
        {
            Literal(new byte[] {1,0,0,0,0}, new int[0,2], tail!=0);
            Literal(new byte[] {1,1,0,0,0,3,2,1,0,253,253,254,255}, new int[,] {{66051,-66051}}, tail!=0);
            Literal(new byte[] {1,3,0,0,0,0,0,0,128,255,255,255,127,255,255,255,255,0,0,0,0,1,0,0,1,255,255,255,254},
                new int[,] {{Int32.MinValue,Int32.MaxValue},{-1,0},{16777217,-16777217}}, tail!=0);
        }
    }

    static int Main(string[] args)
    {
        try { Run(args); return 0; }
        catch (Exception error) { Console.Error.WriteLine(error); return 1; }
    }
}
