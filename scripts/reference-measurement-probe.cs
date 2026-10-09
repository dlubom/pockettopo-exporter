// Optional evidence probe: original PocketTopo 1.372 Station.Read/.NET 2.0.
// No Go reader, GUI, native file writer, references or drawings are invoked.
using System;
using System.IO;
using System.Reflection;
using System.Text;

class ReferenceMeasurementProbe
{
    static BindingFlags binding = BindingFlags.Public | BindingFlags.NonPublic |
        BindingFlags.Instance | BindingFlags.Static;
    static Type station, trip, fileReader;

    static byte[] Record(int distance, short tripIndex, byte flags, byte[] comment)
    {
        MemoryStream stream = new MemoryStream();
        BinaryWriter writer = new BinaryWriter(stream);
        writer.Write(0x800fffffu);
        writer.Write(0x80100000u);
        writer.Write(distance);
        writer.Write(short.MinValue);
        writer.Write(short.MaxValue);
        writer.Write(flags);
        writer.Write((byte)255);
        writer.Write(tripIndex);
        if (comment != null) writer.Write(comment);
        return stream.ToArray();
    }

    static object Reader(MemoryStream stream)
    {
        return Activator.CreateInstance(fileReader, binding, null,
            new object[] { new BinaryReader(stream), 3 }, null);
    }

    static object Field(object value, string name)
    {
        return value.GetType().GetField(name, binding).GetValue(value);
    }

    static void Read(string name, MemoryStream stream, object reader, int tripOffset)
    {
        object value = Activator.CreateInstance(station);
        try
        {
            station.GetMethod("Read", binding).Invoke(value, new object[] { reader, tripOffset });
            string comment = (string)Field(value, "comment");
            Console.WriteLine(name + " ACCEPT position=" + stream.Position +
                " from=" + Field(Field(value, "from"), "value") +
                " to=" + Field(Field(value, "to"), "value") +
                " distance=" + Field(value, "dist") + " azimuth=" + Field(value, "azimuth") +
                " inclination=" + Field(value, "incl") +
                " flags=" + Convert.ToByte(Field(value, "flags")) + " roll=" + Field(value, "roll") +
                " trip=" + Field(value, "trip") + " comment_base64=" +
                (comment == null ? "ABSENT" : Convert.ToBase64String(Encoding.UTF8.GetBytes(comment))));
        }
        catch (TargetInvocationException error)
        {
            Console.WriteLine(name + " REJECT " + error.InnerException.GetType().FullName +
                " position=" + stream.Position);
        }
    }

    static void Probe(string name, byte[] data, int tripOffset)
    {
        MemoryStream stream = new MemoryStream(data);
        Read(name, stream, Reader(stream), tripOffset);
    }

    static void Main(string[] args)
    {
        Assembly assembly = Assembly.LoadFrom(args[0]);
        station = assembly.GetType("PocketTopo.Station", true);
        trip = assembly.GetType("PocketTopo.Trip", true);
        fileReader = assembly.GetType("PocketTopo.FileReader", true);
        Console.WriteLine("assembly=" + assembly.GetName().Version + " runtime=" + Environment.Version);
        Type flagType = station.GetNestedType("Flags", binding);
        foreach (string name in Enum.GetNames(flagType))
            Console.WriteLine("flag " + name + "=" + Convert.ToByte(Enum.Parse(flagType, name)));
        for (int flags = 0; flags < 256; flags++)
            Probe("flags-" + flags, Record(-1, -1, (byte)flags,
                (flags & 2) == 0 ? null : new byte[] { 0 }), 0);
        foreach (int distance in new int[] { int.MinValue, -1, 0, 1, int.MaxValue })
            Probe("distance-" + distance, Record(distance, 0, 0, null), 0);
        foreach (short index in new short[] { short.MinValue, -2, -1, 0, 1, 2, short.MaxValue })
        {
            Probe("trip-zero-offset-" + index, Record(1, index, 0, null), 0);
            Probe("trip-offset-3-" + index, Record(1, index, 0, null), 3);
        }
        Probe("utf8-invalid", Record(1, 0, 2, new byte[] { 3, 65, 255, 66 }), 0);
        Probe("length-nonminimal-zero", Record(1, 0, 2, new byte[] { 128, 128, 128, 128, 0 }), 0);
        Probe("length-fifth-byte-16", Record(1, 0, 2, new byte[] { 128, 128, 128, 128, 16 }), 0);
        // Start at byte 4 to let native Trip.ReadList establish the table boundary.
        byte[] fixture = File.ReadAllBytes(args[1]);
        MemoryStream stream = new MemoryStream(fixture);
        stream.Position = 4;
        object reader = Reader(stream);
        trip.GetMethod("Reset", binding).Invoke(null, null);
        int tripOffset = (int)trip.GetMethod("ReadList", binding).Invoke(null, new object[] { reader });
        int count = new BinaryReader(stream).ReadInt32();
        Console.WriteLine("fixture count=" + count + " start=" + stream.Position + " tripOffset=" + tripOffset);
        for (int i = 0; i < count; i++) Read("fixture[" + i + "]", stream, reader, tripOffset);
    }
}
