// Optional P04c13 evidence: original table/Mapping.Read methods, then scalar
// first/second Polygon fields, following marker/count, third X/Y and third color and exactly one next marker byte.
// No Drawing.Read/Polygon.Read, dispatch, geometry, GUI or TOP writes.
using System;
using System.IO;
using System.Reflection;
using System.Runtime.Serialization;
using System.Globalization;

class ReferencePlanThirdNextMarkerProbe
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
            full ? bytes.Length : 236 + markerBytes);
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
        ReadPoints("first-fixture", raw, stream, new int[,] {{-6000,-2000},{-5500,-1000},{-4500,-1500}}, 151, 24, true, false);
        if(raw.ReadByte()!=1 || raw.ReadByte()!=1 || raw.ReadInt32()!=3 || stream.Position!=181) throw new Exception("Unexpected color/second marker/count");
        ReadPoints("second-fixture", raw, stream, new int[,] {{-4000,-2000},{-3500,-1000},{-2500,-1500}}, 181, 24, true, false);
        if(raw.ReadByte()!=2 || raw.ReadByte()!=1 || raw.ReadInt32()!=3 || stream.Position!=211) throw new Exception("Unexpected second color/third marker/count");
        ReadPoints(name, raw, stream, new int[,] {{-2000,-2000},{-1500,-1000},{-500,-1500}}, 211, 24, true, false);
        if(raw.ReadByte()!=3 || stream.Position!=236) throw new Exception("Unexpected third color/span");
        ReadMarker(name,raw,stream,236,1,markerBytes>0 || full);
    }

    static void ReadPoints(string name, BinaryReader raw, MemoryStream stream,
        int[,] expected, int start, int available, bool full, bool emit)
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
            if(emit) Console.WriteLine(name + " points=" + values + " start=" + start + " consumed=" + stream.Position + " tail=" + (stream.Length-stream.Position));
        }
        catch (EndOfStreamException)
        {
            if (full || available >= 8 * expected.GetLength(0) || stream.Position != start + available)
                throw new Exception("Unexpected truncated point position");
            Console.WriteLine(name + " start=" + start + " available=" + available + " error=EndOfStreamException consumed=" + stream.Position + " tail=0");
        }
    }

    static void ReadMarker(string name, BinaryReader raw, MemoryStream stream, int start, int expected, bool present)
    {
        if (stream.Position != start) throw new Exception("Unexpected color start");
        try
        {
            byte color = raw.ReadByte();
            if (!present || color != expected || stream.Position != start+1) throw new Exception("Unexpected color/end");
            Console.WriteLine(name+" marker="+color+" start="+start+" consumed="+stream.Position+" tail="+(stream.Length-stream.Position));
        }
        catch (EndOfStreamException)
        {
            if (present || stream.Position != start) throw new Exception("Unexpected missing color position");
            Console.WriteLine(name+" start="+start+" error=EndOfStreamException consumed="+stream.Position+" tail=0");
        }
    }

    static void Literal(byte[] first, int[,] firstPoints, byte[] second, int[,] secondPoints, byte[] third, int[,] expected, int color, bool tail)
    {
        MemoryStream stream = new MemoryStream();
        stream.Write(first,0,first.Length);
        stream.WriteByte(129);
        stream.Write(second,0,second.Length);
        stream.WriteByte(255);
        stream.Write(third,0,third.Length);
        stream.WriteByte(254);
        if(color>=0) stream.WriteByte((byte)color);
        if (tail) { stream.WriteByte(255); stream.WriteByte(128); }
        stream.Position = 0;
        object reader = Activator.CreateInstance(fileReader, binding, null,
            new object[] { new BinaryReader(stream), 3 }, null);
        BinaryReader raw = (BinaryReader)fileReader.GetField("r", binding).GetValue(reader);
        if(raw.ReadByte()!=1 || raw.ReadInt32()!=firstPoints.GetLength(0)) throw new Exception("Unexpected first literal count");
        ReadPoints("first-literal",raw,stream,firstPoints,5,first.Length-5,true,false);
        if(raw.ReadByte()!=129 || raw.ReadByte()!=1 || raw.ReadInt32()!=secondPoints.GetLength(0)) throw new Exception("Unexpected second literal marker/count");
        ReadPoints("second-literal",raw,stream,secondPoints,first.Length+6,second.Length-5,true,false);
        if(raw.ReadByte()!=255 || raw.ReadByte()!=1 || raw.ReadInt32()!=expected.GetLength(0)) throw new Exception("Unexpected third literal marker/count");
        ReadPoints("third-literal",raw,stream,expected,first.Length+second.Length+7,third.Length-5,true,false);
        if(raw.ReadByte()!=254) throw new Exception("Unexpected third literal color");
        ReadMarker("literal first="+firstPoints.GetLength(0)+" second="+secondPoints.GetLength(0)+" third="+expected.GetLength(0),raw,stream,first.Length+second.Length+third.Length+3,color,color>=0);
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
        byte[][] bytes = {
            new byte[] {1,0,0,0,0},
            new byte[] {1,1,0,0,0,3,2,1,0,253,253,254,255},
            new byte[] {1,3,0,0,0,0,0,0,128,255,255,255,127,255,255,255,255,0,0,0,0,1,0,0,1,255,255,255,254}
        };
        int[][,] points = {new int[0,2],new int[,] {{66051,-66051}},new int[,] {{Int32.MinValue,Int32.MaxValue},{-1,0},{16777217,-16777217}}};
        for(int first=0;first<bytes.Length;first++)
            for(int second=0;second<bytes.Length;second++)
                for(int third=0;third<bytes.Length;third++) {
                    Literal(bytes[first],points[first],bytes[second],points[second],bytes[third],points[third],-1,false);
                    for(int color=0;color<256;color++)
                        for(int tail=0;tail<2;tail++) Literal(bytes[first],points[first],bytes[second],points[second],bytes[third],points[third],color,tail!=0);
                }
    }

    static int Main(string[] args)
    {
        try { Run(args); return 0; }
        catch (Exception error) { Console.Error.WriteLine(error); return 1; }
    }
}
