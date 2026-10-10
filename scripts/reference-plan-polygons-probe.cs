// P04c14: execute original Drawing.Read/Polygon.Read on bounded plan streams.
// Separately exercise the explicitly safer stop policy with original FileReader's
// BinaryReader. No TOP writes, GUI instance, exporters or Go-derived expectations.
using System;
using System.IO;
using System.Reflection;
using System.Runtime.Serialization;
using System.Globalization;

class ReferencePlanPolygonsProbe
{
    static BindingFlags flags = BindingFlags.Public | BindingFlags.NonPublic | BindingFlags.Instance | BindingFlags.Static;
    static Type drawing, mapping, readerType, polygon, element;
    static object Reader(MemoryStream stream) {
        return Activator.CreateInstance(readerType, flags, null, new object[] { new BinaryReader(stream), 3 }, null);
    }
    static byte[] Literal(int n, int color, byte stop, bool tail) {
        MemoryStream s = new MemoryStream();
        BinaryWriter w = new BinaryWriter(s);
        w.Write(-100); w.Write(200); w.Write(500);
        for (int i=0; i<n; i++) {
            w.Write((byte)1); w.Write(i%4);
            for(int j=0;j<i%4;j++) {w.Write(i*66051+j);w.Write(-(i*16777217+j));}
            w.Write((byte)(color<0 ? i*73 : color));
        }
        w.Write(stop);
        if(tail) {w.Write((byte)255);w.Write((byte)128);}
        return s.ToArray();
    }
    static string Points(object value) {
        Array pts=(Array)polygon.GetField("points",flags).GetValue(value);
        string result="";
        for(int j=0;j<pts.Length;j++) {
            object p=pts.GetValue(j); Type pt=p.GetType();
            if(j>0)result+=";";
            result+=pt.GetProperty("X").GetValue(p,null)+","+pt.GetProperty("Y").GetValue(p,null);
        }
        return result;
    }
    static void Native(string name, byte[] bytes, int available) {
        MemoryStream s=new MemoryStream(bytes,0,available);
        object r=Reader(s);
        object map=FormatterServices.GetUninitializedObject(mapping);
        object d=Activator.CreateInstance(drawing,flags,null,new object[]{map,false},null);
        try {
            drawing.GetMethod("Read",flags).Invoke(d,new object[]{r});
            object e=drawing.GetField("elems",flags).GetValue(d);
            string rows="";int n=0;
            while(e!=null) {
                if(e.GetType()!=polygon)throw new Exception("Unexpected native element");
                if(n>0)rows+="|";
                rows+=Points(e);n++;
                e=element.GetField("next",flags).GetValue(e);
            }
            Console.WriteLine(name+" native polygons="+n+" points="+rows+" consumed="+s.Position+" tail="+(s.Length-s.Position));
        } catch(TargetInvocationException ex) {
            if(!(ex.InnerException is EndOfStreamException))throw;
            Console.WriteLine(name+" native error=EndOfStreamException consumed="+s.Position+" tail="+(s.Length-s.Position));
        }
    }
    static void Bounded(string name, byte[] bytes, int available, int start) {
        MemoryStream s=new MemoryStream(bytes,0,available);s.Position=start;
        object r=Reader(s);BinaryReader raw=(BinaryReader)readerType.GetField("r",flags).GetValue(r);
        mapping.GetMethod("Read",flags).Invoke(FormatterServices.GetUninitializedObject(mapping),new object[]{r});
        int n=0,total=0;string rows="";
        try {
            while(true) {
                long markerStart=s.Position;byte marker=raw.ReadByte();
                if(marker!=1) {
                    Console.WriteLine(name+" bounded polygons="+n+" total="+total+" records="+rows+" marker="+marker+" start="+markerStart+" consumed="+s.Position+" tail="+(s.Length-s.Position));return;
                }
                int count=raw.ReadInt32();if(count<0||count>1000||n>=1000)throw new Exception("Probe bound exceeded");
                string pts="";
                for(int j=0;j<count;j++){int x=raw.ReadInt32(),y=raw.ReadInt32();if(j>0)pts+=";";pts+=x+","+y;}
                byte color=raw.ReadByte();
                if(n>0)rows+="|";
                rows+=markerStart+":"+count+":"+pts+":"+color+":"+s.Position;
                total+=count;n++;
            }
        }catch(EndOfStreamException){Console.WriteLine(name+" bounded error=EndOfStreamException consumed="+s.Position+" tail="+(s.Length-s.Position));}
    }
    static void Run(string[] args) {
        System.Threading.Thread.CurrentThread.CurrentCulture=CultureInfo.InvariantCulture;
        Assembly a=Assembly.LoadFrom(args[0]);
        drawing=a.GetType("PocketTopo.Drawing",true);mapping=a.GetType("PocketTopo.Mapping",true);
        readerType=a.GetType("PocketTopo.FileReader",true);polygon=a.GetType("PocketTopo.Polygon",true);element=a.GetType("PocketTopo.Element",true);
        byte[] fixture=File.ReadAllBytes(args[1]);
        Bounded("fixture-full",fixture,fixture.Length,134);
        Bounded("fixture-exact",fixture,371,134);
        Bounded("fixture-missing-stop",fixture,370,134);
        // Independent boundary test: replace XSection marker with 0 in a copy.
        // Execute native Drawing.Read and confirm it cannot reach side/payload.
        byte[] plan=new byte[fixture.Length-134];Array.Copy(fixture,134,plan,0,plan.Length);plan[236]=0;
        Native("fixture-terminated-full",plan,plan.Length);
        Native("fixture-terminated-exact",plan,237);
        foreach(int n in new int[]{0,1,4,17,257}) {
            byte[] b=Literal(n,-1,0,true);
            Native("literal-n="+n+"-tail",b,b.Length);
            Native("literal-n="+n+"-exact",b,b.Length-2);
            Bounded("literal-n="+n,b,b.Length,0);
        }
        for(int c=0;c<256;c++) {
            byte[] b=Literal(4,c,0,true);
            Native("color="+c,b,b.Length);Bounded("color="+c,b,b.Length,0);
        }
        for(int m=0;m<256;m++)if(m!=1) {
            byte[] b=Literal(5,-1,(byte)m,true);Bounded("stop="+m,b,b.Length,0);
        }
        byte[] cuts=Literal(5,-1,0,false);
        for(int n=12;n<cuts.Length;n++) {
            Native("cut="+n,cuts,n);Bounded("cut="+n,cuts,n,0);
        }
    }
    static int Main(string[] args) {try{Run(args);return 0;}catch(Exception e){Console.Error.WriteLine(e);return 1;}}
}
