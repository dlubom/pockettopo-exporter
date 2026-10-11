// P04c16: original Drawing.Read establishes the terminated plan boundary,
// then original Mapping.Read consumes only the side mapping. No TOP writes.
using System;
using System.IO;
using System.Reflection;
using System.Runtime.Serialization;
using System.Globalization;
class ReferenceSideMappingProbe {
 static BindingFlags flags=BindingFlags.Public|BindingFlags.NonPublic|BindingFlags.Instance|BindingFlags.Static;
 static Type drawing,mapping,reader;
 static void Native(string name,byte[] bytes,int start,int available) {
  MemoryStream s=new MemoryStream(bytes,0,available);s.Position=start;
  object r=Activator.CreateInstance(reader,flags,null,new object[]{new BinaryReader(s),3},null);
  object planMap=FormatterServices.GetUninitializedObject(mapping);
  object d=Activator.CreateInstance(drawing,flags,null,new object[]{planMap,false},null);
  drawing.GetMethod("Read",flags).Invoke(d,new object[]{r});
  long sideStart=s.Position;
  object side=FormatterServices.GetUninitializedObject(mapping);
  int pixels=(int)mapping.GetField("PixPerMm",flags).GetValue(null);
  try {
   mapping.GetMethod("Read",flags).Invoke(side,new object[]{r});
   long end=s.Position;s.Position=sideStart;BinaryReader raw=new BinaryReader(s);
   int x=raw.ReadInt32(),y=raw.ReadInt32(),scale=raw.ReadInt32();s.Position=end;
   Console.WriteLine(name+" start="+sideStart+" raw="+x+","+y+","+scale+" native="+mapping.GetField("x0",flags).GetValue(side)+","+mapping.GetField("y0",flags).GetValue(side)+","+mapping.GetField("scaleFact",flags).GetValue(side)+" pixels="+pixels+" consumed="+end+" tail="+(s.Length-end));
  } catch(TargetInvocationException ex) {
   if(!(ex.InnerException is EndOfStreamException))throw;
   Console.WriteLine(name+" start="+sideStart+" error=EndOfStreamException pixels="+pixels+" consumed="+s.Position+" tail="+(s.Length-s.Position));
  }
 }
 static byte[] Literal(int n,int x,int y,int scale) {
  MemoryStream s=new MemoryStream();BinaryWriter w=new BinaryWriter(s);
  w.Write(-100);w.Write(200);w.Write(500);
  for(int i=0;i<n;i++){
   if(i%2==0){w.Write((byte)1);w.Write(1);w.Write(-i);w.Write(i+1);w.Write((byte)255);}
   else {w.Write((byte)3);w.Write(-i);w.Write(i+1);w.Write((uint)0x800fffff);w.Write(-2);}
  }
  w.Write((byte)0);w.Write(x);w.Write(y);w.Write(scale);w.Write((byte)255);w.Write((byte)128);return s.ToArray();
 }
 static int Main(string[] args) {try {
  System.Threading.Thread.CurrentThread.CurrentCulture=CultureInfo.InvariantCulture;
  Assembly a=Assembly.LoadFrom(args[0]);drawing=a.GetType("PocketTopo.Drawing",true);mapping=a.GetType("PocketTopo.Mapping",true);reader=a.GetType("PocketTopo.FileReader",true);
  for(int mode=0;mode<2;mode++) {
   if(mode==1)mapping.GetMethod("SetVga",flags).Invoke(null,null);
   foreach(int n in new int[]{0,1,4,17}) {
    byte[] b=Literal(n,-66051,16777217,-501);Native("many="+n+"-tail",b,0,b.Length);Native("many="+n+"-exact",b,0,b.Length-2);
    for(int cut=b.Length-14;cut<b.Length-2;cut++)Native("many="+n+"-cut="+(cut-(b.Length-14)),b,0,cut);
   }
   foreach(int field in new int[]{0,1,2})foreach(int value in new int[]{int.MinValue,-16777217,-501,-11,-5,-1,0,1,4,5,9,10,501,16777217,int.MaxValue}) {
    int[] v=new int[]{-66051,16777217,-501};v[field]=value;byte[] b=Literal(4,v[0],v[1],v[2]);Native("field="+field+" value="+value,b,0,b.Length);
   }
   byte[] refs=File.ReadAllBytes(args[1]),fixture=File.ReadAllBytes(args[2]);
   Native("references-full",refs,218,refs.Length);Native("references-exact",refs,218,243);
   Native("drawings-full",fixture,134,fixture.Length);Native("drawings-exact",fixture,134,417);
  }
  return 0;
 }catch(Exception e){Console.Error.WriteLine(e);return 1;}}
}
