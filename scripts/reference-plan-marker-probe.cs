// Optional P04c1 evidence: original table/Mapping.Read methods, then exactly
// one BinaryReader.ReadByte. No Drawing.Read, payload, side mapping or TOP writes.
using System;
using System.IO;
using System.Reflection;
using System.Runtime.Serialization;
using System.Globalization;

class ReferencePlanMarkerProbe
{
    static BindingFlags binding = BindingFlags.Public | BindingFlags.NonPublic |
        BindingFlags.Instance | BindingFlags.Static;
    static Type mapping, fileReader, trip, station, reference, id, location;

    static void ReadMapping(object reader)
    {
        object value = FormatterServices.GetUninitializedObject(mapping);
        mapping.GetMethod("Read", binding).Invoke(value, new object[] { reader });
    }

    static void Fixture(string name, string path, int markerStart, byte expected, int mode)
    {
        byte[] bytes = File.ReadAllBytes(path);
        // Literal offsets are pinned helper/format evidence, never Go output.
        MemoryStream stream = new MemoryStream(bytes, 0,
            mode == 0 ? bytes.Length : markerStart + (mode == 1 ? 1 : 0));
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
        ReadMapping(reader);
        if (stream.Position != markerStart - 12) throw new Exception("Unexpected native plan offset");
        ReadMapping(reader);
        if (stream.Position != markerStart) throw new Exception("Unexpected native marker offset");
        BinaryReader raw = (BinaryReader)fileReader.GetField("r", binding).GetValue(reader);
        if (mode == 2)
        {
            try { raw.ReadByte(); }
            catch (EndOfStreamException)
            {
                if (stream.Position != markerStart) throw new Exception("Failed marker read advanced stream");
                Console.WriteLine(name + " marker_start=" + markerStart +
                    " missing=EndOfStreamException consumed=" + stream.Position + " tail=0");
                return;
            }
            throw new Exception("Missing marker unexpectedly succeeded");
        }
        byte marker = raw.ReadByte();
        if (marker != expected || stream.Position != markerStart + 1)
            throw new Exception("Unexpected marker or consumed position");
        Console.WriteLine(name + " marker_start=" + markerStart + " marker=" + marker +
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
            Fixture("references-fixture", args[1], 230, 0, 0);
            Fixture("drawings-fixture", args[2], 146, 1, 0);
            Fixture("references-prefix", args[1], 230, 0, 1);
            Fixture("drawings-prefix", args[2], 146, 1, 1);
            Fixture("references-missing", args[1], 230, 0, 2);
            Fixture("drawings-missing", args[2], 146, 1, 2);
        }
    }

    static int Main(string[] args)
    {
        try { Run(args); return 0; }
        catch (Exception error) { Console.Error.WriteLine(error); return 1; }
    }
}
