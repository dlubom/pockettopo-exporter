// P04c17: original Drawing.Read and Mapping.Read establish the side marker.
// ReadByte is the exact original Drawing.Read marker operation. No TOP writes.
using System;
using System.IO;
using System.Reflection;
using System.Runtime.Serialization;
class ReferenceSideMarkerProbe {
 static BindingFlags flags=BindingFlags.Public|BindingFlags.NonPublic|BindingFlags.Instance|BindingFlags.Static;
 static Type drawing,mapping,reader;
 static void Native(string name,byte[] bytes,int start,int available) {
  MemoryStream s=new MemoryStream(bytes,0,available);s.Position=start;
  BinaryReader br=new BinaryReader(s);
  object r=Activator.CreateInstance(reader,flags,null,new object[]{br,3},null);
  object d=Activator.CreateInstance(drawing,flags,null,new object[]{FormatterServices.GetUninitializedObject(mapping),false},null);
  drawing.GetMethod("Read",flags).Invoke(d,new object[]{r});
  mapping.GetMethod("Read",flags).Invoke(FormatterServices.GetUninitializedObject(mapping),new object[]{r});
  long markerStart=s.Position;
  try {
   byte marker=br.ReadByte();
   Console.WriteLine(name+" start="+markerStart+" marker="+marker+" consumed="+s.Position+" tail="+(s.Length-s.Position));
  }catch(EndOfStreamException){Console.WriteLine(name+" start="+markerStart+" error=EndOfStreamException consumed="+s.Position+" tail="+(s.Length-s.Position));}
 }
 static byte[] Literal(int n,byte marker) {
  MemoryStream s=new MemoryStream();BinaryWriter w=new BinaryWriter(s);
  w.Write(-100);w.Write(200);w.Write(500);
  for(int i=0;i<n;i++) {
   if(i%2==0){w.Write((byte)1);w.Write(1);w.Write(-i);w.Write(i+1);w.Write((byte)255);}
   else {w.Write((byte)3);w.Write(-i);w.Write(i+1);w.Write((uint)0x800fffff);w.Write(-2);}
  }
  w.Write((byte)0);w.Write(-66051);w.Write(16777217);w.Write(-501);
  w.Write(marker);w.Write((byte)255);w.Write((byte)128);return s.ToArray();
 }
 static int Main(string[] args) {try {
  Assembly a=Assembly.LoadFrom(args[0]);drawing=a.GetType("PocketTopo.Drawing",true);mapping=a.GetType("PocketTopo.Mapping",true);reader=a.GetType("PocketTopo.FileReader",true);
  foreach(int n in new int[]{0,4,17}) {
   for(int m=0;m<256;m++){byte[] b=Literal(n,(byte)m);Native("n="+n+" marker="+m,b,0,b.Length);}
   byte[] missing=Literal(n,0);Native("n="+n+" missing",missing,0,missing.Length-3);
  }
  byte[] refs=File.ReadAllBytes(args[1]),fixture=File.ReadAllBytes(args[2]);
  Native("references-full",refs,218,refs.Length);Native("references-exact",refs,218,244);
  Native("drawings-full",fixture,134,fixture.Length);Native("drawings-exact",fixture,134,418);
  return 0;
 }catch(Exception e){Console.Error.WriteLine(e);return 1;}}
}
