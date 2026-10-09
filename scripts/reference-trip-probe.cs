// Optional evidence probe for the original PocketTopo 1.372 assembly/.NET 2.0.
// Uses native Trip.ReadList/Read, never the Go reader, GUI, or file writer.
using System;
using System.IO;
using System.Reflection;
using System.Text;

class ReferenceTripProbe
{
    static BindingFlags flags = BindingFlags.Public | BindingFlags.NonPublic |
        BindingFlags.Instance | BindingFlags.Static;
    static Type trip, fileReader;

    static byte[] Record(long ticks, byte[] encodedComment, short declination)
    {
        MemoryStream stream = new MemoryStream();
        BinaryWriter writer = new BinaryWriter(stream);
        writer.Write(1);
        writer.Write(ticks);
        writer.Write(encodedComment);
        writer.Write(declination);
        return stream.ToArray();
    }

    static void Probe(string name, byte[] data)
    {
        MemoryStream stream = new MemoryStream(data);
        BinaryReader reader = new BinaryReader(stream);
        trip.GetMethod("Reset", flags).Invoke(null, null);
        object nativeReader = Activator.CreateInstance(fileReader, flags, null,
            new object[] { reader, 3 }, null);
        try
        {
            trip.GetMethod("ReadList", flags).Invoke(null, new object[] { nativeReader });
            Console.Write(name + " ACCEPT position=" + stream.Position);
            for (int i = 0; ; i++)
            {
                object value = trip.GetMethod("ByIndex", flags).Invoke(null, new object[] { i });
                if (value == null) break;
                DateTime date = (DateTime)trip.GetField("date", flags).GetValue(value);
                string comment = (string)trip.GetProperty("Comment", flags).GetValue(value, null);
                object angle = trip.GetField("declCorr", flags).GetValue(value);
                Console.Write(" trip[" + i + "] ticks=" + date.Ticks + " auto=" + trip.GetField("automatic", flags).GetValue(value) +
                    " declCorr=" + angle.GetType().GetField("value", flags).GetValue(angle) +
                    " comment_base64=" + Convert.ToBase64String(Encoding.UTF8.GetBytes(comment)));
            }
            Console.WriteLine();
        }
        catch (TargetInvocationException error)
        {
            Console.WriteLine(name + " REJECT " + error.InnerException.GetType().FullName +
                " position=" + stream.Position);
        }
    }

    static void Main(string[] args)
    {
        Assembly assembly = Assembly.LoadFrom(args[0]);
        trip = assembly.GetType("PocketTopo.Trip", true);
        fileReader = assembly.GetType("PocketTopo.FileReader", true);
        Console.WriteLine("assembly=" + assembly.GetName().Version + " runtime=" + Environment.Version);
        long max = DateTime.MaxValue.Ticks;
        Console.WriteLine("DateTime_ticks_range=" + DateTime.MinValue.Ticks + ".." + max);
        Probe("ticks-min", Record(0, new byte[] { 0 }, 0));
        Probe("ticks-max-auto", Record(max, new byte[] { 0 }, -32768));
        Probe("ticks-negative", Record(-1, new byte[] { 0 }, 0));
        Probe("ticks-max-plus-one", Record(max + 1, new byte[] { 0 }, 0));
        Probe("utf8-invalid-interior", Record(0, new byte[] { 3, 65, 255, 66 }, -1));
        Probe("utf8-incomplete", Record(0, new byte[] { 2, 226, 130 }, 0));
        Probe("length-nonminimal-zero", Record(0, new byte[] { 128, 128, 128, 128, 0 }, 0));
        Probe("length-fifth-byte-16", Record(0, new byte[] { 128, 128, 128, 128, 16 }, 0));
        Probe("length-negative", Record(0, new byte[] { 128, 128, 128, 128, 8 }, 0));
        Probe("length-six-bytes", Record(0, new byte[] { 128, 128, 128, 128, 128, 0 }, 0));
        Probe("negative-count", new byte[] { 255, 255, 255, 255 });
        byte[] fixture = File.ReadAllBytes(args[1]);
        byte[] body = new byte[fixture.Length - 4];
        Array.Copy(fixture, 4, body, 0, body.Length);
        Probe("native-fixture", body);
    }
}
