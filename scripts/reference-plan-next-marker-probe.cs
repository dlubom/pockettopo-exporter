// Optional P04c5 evidence: original table/Mapping.Read methods, then raw
// signed X/Y pairs and one color byte and exactly one next marker. No Polygon.Read, GUI or TOP writes.
using System;
using System.IO;
using System.Reflection;
using System.Runtime.Serialization;
using System.Globalization;

class ReferencePlanNextMarkerProbe
{
    static BindingFlags binding = BindingFlags.Public | BindingFlags.NonPublic |
        BindingFlags.Instance | BindingFlags.Static;
    static Type mapping, fileReader, trip, station, reference, id, location;

    static void ReadMapping(object reader)
    {
        object value = FormatterServices.GetUninitializedObject(mapping);
        mapping.GetMethod("Read", binding).Invoke(value, new object[] { reader });
    }

    static void Fixture(string name, string path, int markerBytes, bool full)
    {
        const int markerStart = 146;
        byte[] bytes = File.ReadAllBytes(path);
        // Literal offsets are pinned helper/format evidence, never Go output.
        MemoryStream stream = new MemoryStream(bytes, 0,
            full ? bytes.Length : 176 + markerBytes);
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
        ReadPoints(raw, stream, new int[,] {{-6000,-2000},{-5500,-1000},{-4500,-1500}}, 151);
        if (raw.ReadByte() != 1 || stream.Position != 176) throw new Exception("Unexpected marker/span");
        ReadMarker(name, raw, stream, 1, 176, markerBytes != 0 || full);
    }

    static void ReadPoints(BinaryReader raw, MemoryStream stream, int[,] expected, int start)
    {
        for (int i = 0; i < expected.GetLength(0); i++)
        {
            int x = raw.ReadInt32();
            int y = raw.ReadInt32();
            if (x != expected[i,0] || y != expected[i,1]) throw new Exception("Unexpected coordinate");
        }
        if (stream.Position != start + 8 * expected.GetLength(0)) throw new Exception("Unexpected point end");
    }

    static void ReadMarker(string name, BinaryReader raw, MemoryStream stream, int expected, int start, bool present)
    {
        if (stream.Position != start) throw new Exception("Unexpected marker start");
        try
        {
            byte marker = raw.ReadByte();
            if (!present || marker != expected || stream.Position != start + 1) throw new Exception("Unexpected marker/span");
            Console.WriteLine(name + " marker=" + marker + " start=" + start + " consumed=" + stream.Position + " tail=" + (stream.Length-stream.Position));
        }
        catch (EndOfStreamException)
        {
            if (present || stream.Position != start) throw new Exception("Unexpected missing marker position");
            Console.WriteLine(name + " start=" + start + " error=EndOfStreamException consumed=" + stream.Position + " tail=0");
        }
    }

    static void Literal(byte[] bytes, int[,] expected, int marker, bool tail)
    {
        MemoryStream stream = new MemoryStream();
        stream.Write(bytes, 0, bytes.Length);
        stream.WriteByte((byte)(marker < 0 ? 129 : 255-marker));
        if (marker >= 0) stream.WriteByte((byte)marker);
        if (tail) { stream.WriteByte(255); stream.WriteByte(128); }
        stream.Position = 0;
        object reader = Activator.CreateInstance(fileReader, binding, null,
            new object[] { new BinaryReader(stream), 3 }, null);
        BinaryReader raw = (BinaryReader)fileReader.GetField("r", binding).GetValue(reader);
        if (raw.ReadByte() != 1 || raw.ReadInt32() != expected.GetLength(0)) throw new Exception("Unexpected literal count");
        ReadPoints(raw, stream, expected, 5);
        int color = raw.ReadByte();
        if (color != (marker < 0 ? 129 : 255-marker)) throw new Exception("Unexpected literal color");
        ReadMarker("literal points=" + expected.GetLength(0) + " color=" + color, raw, stream, marker, bytes.Length+1, marker >= 0);
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
            Fixture("drawings-fixture",args[1],1,true);
            Fixture("drawings-prefix",args[1],1,false);
            Fixture("drawings-missing",args[1],0,false);
        }
        // Literal bytes and coordinates are independent of the Go implementation.
        byte[][] bytes = {
            new byte[] {1,0,0,0,0},
            new byte[] {1,1,0,0,0,3,2,1,0,253,253,254,255},
            new byte[] {1,3,0,0,0,0,0,0,128,255,255,255,127,255,255,255,255,0,0,0,0,1,0,0,1,255,255,255,254}
        };
        int[][,] points = {
            new int[0,2],
            new int[,] {{66051,-66051}},
            new int[,] {{Int32.MinValue,Int32.MaxValue},{-1,0},{16777217,-16777217}}
        };
        for (int i=0; i<bytes.Length; i++)
        {
            Literal(bytes[i],points[i],-1,false);
            for (int marker=0; marker<256; marker++)
                for (int tail=0; tail<2; tail++) Literal(bytes[i],points[i],marker,tail!=0);
        }
    }

    static int Main(string[] args)
    {
        try { Run(args); return 0; }
        catch (Exception error) { Console.Error.WriteLine(error); return 1; }
    }
}
