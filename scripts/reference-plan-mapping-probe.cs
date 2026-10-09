// Optional P04b evidence: original Mapping.Read at native table-derived
// overview/plan offsets, stopping before any element marker. No TOP writes.
using System;
using System.IO;
using System.Reflection;
using System.Runtime.Serialization;
using System.Globalization;

class ReferencePlanMappingProbe
{
    static BindingFlags binding = BindingFlags.Public | BindingFlags.NonPublic |
        BindingFlags.Instance | BindingFlags.Static;
    static Type mapping, fileReader, trip, station, reference, id, location;

    static void Read(string name, MemoryStream stream, object reader)
    {
        long start = stream.Position;
        BinaryReader raw = new BinaryReader(stream);
        int x = raw.ReadInt32(), y = raw.ReadInt32(), scale = raw.ReadInt32();
        stream.Position = start;
        object value = FormatterServices.GetUninitializedObject(mapping);
        mapping.GetMethod("Read", binding).Invoke(value, new object[] { reader });
        Console.WriteLine(name + " start=" + start + " consumed=" + stream.Position +
            " raw_x0=" + x + " raw_y0=" + y + " stored_scale=" + scale +
            " native_x0=" + mapping.GetField("x0", binding).GetValue(value) +
            " native_y0=" + mapping.GetField("y0", binding).GetValue(value) +
            " PixPerMm=" + mapping.GetField("PixPerMm", binding).GetValue(null) +
            " native_scaleFact=" + mapping.GetField("scaleFact", binding).GetValue(value));
    }

    static void Fixture(string name, string path, int planStart, bool prefixOnly)
    {
        byte[] bytes = File.ReadAllBytes(path);
        // Literal offsets are pinned helper/format evidence, never Go output.
        MemoryStream stream = new MemoryStream(bytes, 0,
            prefixOnly ? planStart + 12 : bytes.Length);
        stream.Position = 4;
        object reader = Activator.CreateInstance(fileReader, binding, null,
            new object[] { new BinaryReader(stream), 3 }, null);
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
        Read(name + "-overview", stream, reader);
        if (stream.Position != planStart) throw new Exception("Unexpected native plan offset");
        Read(name + "-plan", stream, reader);
        if (stream.Position != planStart + 12) throw new Exception("Unexpected native plan end");
        Console.WriteLine(name + " tail=" + (stream.Length - stream.Position));
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
            Fixture("references-fixture", args[1], 218, false);
            Fixture("drawings-fixture", args[2], 134, false);
            Fixture("references-prefix", args[1], 218, true);
            Fixture("drawings-prefix", args[2], 134, true);
        }
    }

    static int Main(string[] args)
    {
        try { Run(args); return 0; }
        catch (Exception error) { Console.Error.WriteLine(error); return 1; }
    }
}
