// Optional evidence: original Mapping.Read/Write on literal bytes and native
// fixtures. No Go reader, drawing read, GUI, TOP writer or archive changes.
using System;
using System.IO;
using System.Reflection;
using System.Runtime.Serialization;
using System.Globalization;

class ReferenceMappingProbe
{
    static BindingFlags binding = BindingFlags.Public | BindingFlags.NonPublic |
        BindingFlags.Instance | BindingFlags.Static;
    static Type mapping, fileReader, fileWriter, trip, station, reference, id, location;

    static object Field(object value, string name)
    {
        return value.GetType().GetField(name, binding).GetValue(value);
    }

    static object Reader(MemoryStream stream)
    {
        return Activator.CreateInstance(fileReader, binding, null,
            new object[] { new BinaryReader(stream), 3 }, null);
    }

    static byte[] Record(int x, int y, int scale)
    {
        MemoryStream stream = new MemoryStream();
        BinaryWriter writer = new BinaryWriter(stream);
        writer.Write(x); writer.Write(y); writer.Write(scale);
        return stream.ToArray();
    }

    static void Read(string name, MemoryStream stream, object reader)
    {
        long start = stream.Position;
        BinaryReader raw = new BinaryReader(stream);
        int x = raw.ReadInt32(), y = raw.ReadInt32(), scale = raw.ReadInt32();
        stream.Position = start;
        // Read/Write access only these scalar fields; avoid the UI constructor.
        object value = FormatterServices.GetUninitializedObject(mapping);
        mapping.GetMethod("Read", binding).Invoke(value, new object[] { reader });
        MemoryStream rewritten = new MemoryStream();
        object writer = Activator.CreateInstance(fileWriter, binding, null,
            new object[] { new BinaryWriter(rewritten) }, null);
        mapping.GetMethod("Write", binding).Invoke(value, new object[] { writer });
        rewritten.Position = 8;
        Console.WriteLine(name + " start=" + start + " consumed=" + stream.Position +
            " raw_x0=" + x + " raw_y0=" + y + " stored_scale=" + scale +
            " native_x0=" + Field(value, "x0") + " native_y0=" + Field(value, "y0") +
            " PixPerMm=" + mapping.GetField("PixPerMm", binding).GetValue(null) +
            " native_scaleFact=" + Field(value, "scaleFact") +
            " rewritten_scale=" + new BinaryReader(rewritten).ReadInt32());
    }

    static void Probe(string name, byte[] bytes)
    {
        MemoryStream stream = new MemoryStream(bytes);
        Read(name, stream, Reader(stream));
    }

    static void Fixture(string name, string path)
    {
        MemoryStream stream = new MemoryStream(File.ReadAllBytes(path));
        stream.Position = 4;
        object reader = Reader(stream);
        trip.GetMethod("Reset", binding).Invoke(null, null);
        int tripOffset = (int)trip.GetMethod("ReadList", binding).Invoke(null, new object[] { reader });
        int count = new BinaryReader(stream).ReadInt32();
        for (int i = 0; i < count; i++)
            station.GetMethod("Read", binding, null, new Type[] { fileReader, typeof(int) }, null)
                .Invoke(Activator.CreateInstance(station), new object[] { reader, tripOffset });
        count = new BinaryReader(stream).ReadInt32();
        for (int i = 0; i < count; i++)
        {
            object value = Activator.CreateInstance(reference, binding, null,
                new object[] { Activator.CreateInstance(id), Activator.CreateInstance(location) }, null);
            reference.GetMethod("Read", binding, null, new Type[] { fileReader }, null)
                .Invoke(value, new object[] { reader });
        }
        Read(name, stream, reader);
        Console.WriteLine(name + " tail=" + (stream.Length - stream.Position));
    }

    static void Run(string[] args)
    {
        System.Threading.Thread.CurrentThread.CurrentCulture = CultureInfo.InvariantCulture;
        Assembly assembly = Assembly.LoadFrom(args[0]);
        mapping = assembly.GetType("PocketTopo.Mapping", true);
        fileReader = assembly.GetType("PocketTopo.FileReader", true);
        fileWriter = assembly.GetType("PocketTopo.FileWriter", true);
        trip = assembly.GetType("PocketTopo.Trip", true);
        station = assembly.GetType("PocketTopo.Station", true);
        reference = assembly.GetType("PocketTopo.Reference", true);
        id = assembly.GetType("PocketTopo.ID", true);
        location = assembly.GetType("PocketTopo.MetricLocation", true);
        Console.WriteLine("assembly=" + assembly.GetName().Version + " runtime=" + Environment.Version);
        for (int mode = 0; mode < 2; mode++)
        {
            if (mode == 1) mapping.GetMethod("SetVga", binding).Invoke(null, null);
            foreach (int scale in new int[] { int.MinValue, -501, -11, -10, -6, -5, -4, -1, 0, 1, 4, 5, 6, 9, 10, 11, 499, 500, 501, int.MaxValue })
                Probe("scale-" + scale, Record(-1234, 5678, scale));
            foreach (int origin in new int[] { int.MinValue, -1, 0, 1, int.MaxValue })
                Probe("origin-" + origin, Record(origin, origin, 501));
            Probe("signed-endian", new byte[] { 1, 2, 3, 0x84, 8, 7, 6, 5, 0xfe, 0xfd, 0xfc, 0xfb });
            Fixture("references-fixture", args[1]);
            Fixture("drawings-fixture", args[2]);
        }
    }

    static int Main(string[] args)
    {
        try { Run(args); return 0; }
        catch (Exception error) { Console.Error.WriteLine(error); return 1; }
    }
}
