// Optional evidence: original PocketTopo Reference.Read on literal bytes and
// preserved native fixtures. No Go code, native writer, GUI or drawings.
using System;
using System.IO;
using System.Reflection;
using System.Text;
using System.Globalization;

class ReferenceReferenceProbe
{
    static BindingFlags binding = BindingFlags.Public | BindingFlags.NonPublic |
        BindingFlags.Instance | BindingFlags.Static;
    static Type reference, id, location, fileReader, trip, station;

    static object Field(object value, string name)
    {
        return value.GetType().GetField(name, binding).GetValue(value);
    }

    static object Reader(MemoryStream stream)
    {
        return Activator.CreateInstance(fileReader, binding, null,
            new object[] { new BinaryReader(stream), 3 }, null);
    }

    static byte[] Record(uint rawID, long east, long north, int altitude, byte[] comment)
    {
        MemoryStream stream = new MemoryStream();
        BinaryWriter writer = new BinaryWriter(stream);
        writer.Write(rawID); writer.Write(east); writer.Write(north); writer.Write(altitude);
        writer.Write(comment);
        return stream.ToArray();
    }

    static void Read(string name, MemoryStream stream, object reader)
    {
        object value = Activator.CreateInstance(reference, binding, null,
            new object[] { Activator.CreateInstance(id), Activator.CreateInstance(location) }, null);
        try
        {
            reference.GetMethod("Read", binding, null, new Type[] { fileReader }, null)
                .Invoke(value, new object[] { reader });
            string comment = (string)Field(value, "comment");
            object loc = Field(value, "loc");
            Console.WriteLine(name + " ACCEPT position=" + stream.Position +
                " id=" + Field(Field(value, "num"), "value") +
                " east=" + Field(loc, "e") + " north=" + Field(loc, "n") +
                " altitude=" + Field(loc, "alt") + " comment_base64=" +
                (comment == null ? "NULL" : Convert.ToBase64String(Encoding.UTF8.GetBytes(comment))));
        }
        catch (TargetInvocationException error)
        {
            Console.WriteLine(name + " REJECT " + error.InnerException.GetType().FullName +
                " position=" + stream.Position);
        }
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
        {
            object measurement = Activator.CreateInstance(station);
            station.GetMethod("Read", binding).Invoke(measurement, new object[] { reader, tripOffset });
        }
        long countStart = stream.Position;
        int references = new BinaryReader(stream).ReadInt32();
        Console.WriteLine(name + " count=" + references + " countStart=" + countStart + " start=" + stream.Position);
        for (int i = 0; i < references; i++) Read(name + "[" + i + "]", stream, reader);
        Console.WriteLine(name + " consumed=" + stream.Position + " tail=" + (stream.Length-stream.Position));
    }

    static void Main(string[] args)
    {
        System.Threading.Thread.CurrentThread.CurrentCulture = CultureInfo.InvariantCulture;
        Assembly assembly = Assembly.LoadFrom(args[0]);
        reference = assembly.GetType("PocketTopo.Reference", true);
        id = assembly.GetType("PocketTopo.ID", true);
        location = assembly.GetType("PocketTopo.MetricLocation", true);
        fileReader = assembly.GetType("PocketTopo.FileReader", true);
        trip = assembly.GetType("PocketTopo.Trip", true);
        station = assembly.GetType("PocketTopo.Station", true);
        Console.WriteLine("assembly=" + assembly.GetName().Version + " runtime=" + Environment.Version);
        foreach (long coordinate in new long[] { long.MinValue, -9007199254740993L, -1, 0, 1, 9007199254740993L, long.MaxValue })
            Probe("coordinates-" + coordinate, Record(0x800fffffu, coordinate, coordinate, -1, new byte[] { 0 }));
        foreach (int altitude in new int[] { int.MinValue, -1, 0, 1, int.MaxValue })
            Probe("altitude-" + altitude, Record(0x80000000u, -1, 1, altitude, new byte[] { 0 }));
        foreach (uint rawID in new uint[] { 1, 0x80100002u, 0x800fffffu, 0x80000000u, 0x80100000u })
            Probe("id-" + rawID.ToString("x8"), Record(rawID, 1, -1, 1, new byte[] { 0 }));
        Probe("signed-endian", new byte[] {
            0xff, 0xff, 0x0f, 0x80,
            1, 2, 3, 4, 5, 6, 7, 0x88, 8, 7, 6, 5, 4, 3, 2, 1,
            0xfe, 0xff, 0xff, 0xff, 3, 65, 0xc4, 0x85 });
        Probe("length-nonminimal-zero", Record(1, 1, -1, 1, new byte[] { 128, 128, 128, 128, 0 }));
        Probe("length-nonminimal-one", Record(1, 1, -1, 1, new byte[] { 129, 0, 65 }));
        Probe("utf8-invalid", Record(1, 1, -1, 1, new byte[] { 3, 65, 255, 66 }));
        Probe("length-fifth-byte-16", Record(1, 1, -1, 1, new byte[] { 128, 128, 128, 128, 16 }));
        Probe("length-negative", Record(1, 1, -1, 1, new byte[] { 128, 128, 128, 128, 8 }));
        Probe("length-six-bytes", Record(1, 1, -1, 1, new byte[] { 128, 128, 128, 128, 128, 0 }));
        Fixture("zero-fixture", args[1]);
        Fixture("references-fixture", args[2]);
    }
}
