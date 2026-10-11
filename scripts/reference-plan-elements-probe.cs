// P04c15: original Drawing.Read dispatches Polygon.Read and XSection.Read.
// Literal streams and fixture copies only; no TOP writes, GUI or exporters.
using System;
using System.IO;
using System.Reflection;
using System.Runtime.Serialization;
using System.Globalization;
class ReferencePlanElementsProbe {
 static BindingFlags flags=BindingFlags.Public|BindingFlags.NonPublic|BindingFlags.Instance|BindingFlags.Static;
 static Type drawing,mapping,reader,polygon,xsection,element;
 static string Describe(object e) {
  if(e.GetType()==xsection){
   object center=xsection.GetField("center",flags).GetValue(e);Type pt=center.GetType();
   object id=xsection.GetField("num",flags).GetValue(e);
   return "X:"+pt.GetProperty("X").GetValue(center,null)+","+pt.GetProperty("Y").GetValue(center,null)+","+id.GetType().GetField("value",flags).GetValue(id)+","+xsection.GetField("direction",flags).GetValue(e);
  }
  if(e.GetType()!=polygon)throw new Exception("Unexpected element");
  Array pts=(Array)polygon.GetField("points",flags).GetValue(e);string row="P:";
  for(int i=0;i<pts.Length;i++){object p=pts.GetValue(i);Type pt=p.GetType();if(i>0)row+=";";row+=pt.GetProperty("X").GetValue(p,null)+","+pt.GetProperty("Y").GetValue(p,null);}
  return row;
 }
 static void Native(string name,byte[] bytes,int available){
  MemoryStream s=new MemoryStream(bytes,0,available);
  object r=Activator.CreateInstance(reader,flags,null,new object[]{new BinaryReader(s),3},null);
  object map=FormatterServices.GetUninitializedObject(mapping);
  object d=Activator.CreateInstance(drawing,flags,null,new object[]{map,false},null);
  try{
   drawing.GetMethod("Read",flags).Invoke(d,new object[]{r});
   object e=drawing.GetField("elems",flags).GetValue(d);string rows="";int n=0;
   while(e!=null){if(n>0)rows+="|";rows+=Describe(e);n++;if(n>1000)throw new Exception("Probe ceiling");e=element.GetField("next",flags).GetValue(e);}
   Console.WriteLine(name+" records="+n+" values="+rows+" consumed="+s.Position+" tail="+(s.Length-s.Position));
  }catch(TargetInvocationException ex){if(!(ex.InnerException is EndOfStreamException))throw;Console.WriteLine(name+" error=EndOfStreamException consumed="+s.Position+" tail="+(s.Length-s.Position));}
 }
 static byte[] Literal(int n,int x,int y,uint id,int direction,bool tail){
  MemoryStream s=new MemoryStream();BinaryWriter w=new BinaryWriter(s);w.Write(-100);w.Write(200);w.Write(500);
  for(int i=0;i<n;i++){if(i%3==0){w.Write((byte)1);w.Write(1);w.Write(i*66051);w.Write(-i*16777217);w.Write((byte)(i*73));}else{w.Write((byte)3);w.Write(x);w.Write(y);w.Write(id);w.Write(direction);}}
  w.Write((byte)0);if(tail){w.Write((byte)255);w.Write((byte)128);}return s.ToArray();
 }
 static int Main(string[] args){try{
  System.Threading.Thread.CurrentThread.CurrentCulture=CultureInfo.InvariantCulture;
  Assembly a=Assembly.LoadFrom(args[0]);drawing=a.GetType("PocketTopo.Drawing",true);mapping=a.GetType("PocketTopo.Mapping",true);reader=a.GetType("PocketTopo.FileReader",true);polygon=a.GetType("PocketTopo.Polygon",true);xsection=a.GetType("PocketTopo.XSection",true);element=a.GetType("PocketTopo.Element",true);
  byte[] fixture=File.ReadAllBytes(args[1]);byte[] plan=new byte[fixture.Length-134];Array.Copy(fixture,134,plan,0,plan.Length);
  Native("fixture-full",plan,plan.Length);Native("fixture-exact",plan,271);Native("fixture-missing-stop",plan,270);
  foreach(int n in new int[]{0,1,2,3,4,17,257}){byte[] b=Literal(n,-66051,16777217,0x800fffff,-1,true);Native("many="+n+"-tail",b,b.Length);Native("many="+n+"-exact",b,b.Length-2);}
  foreach(int v in new int[]{int.MinValue,-16777217,-2,-1,0,66051,65535,65536,int.MaxValue})foreach(uint id in new uint[]{0,0x7fffffff,0x80000000,0x800fffff,0x80100000,0xffffffff}){
   byte[] b=Literal(3,v,~v,id,v,true);string name="raw="+v+","+id;Native(name+"-tail",b,b.Length);Native(name+"-exact",b,b.Length-2);
  }
  byte[] cuts=Literal(7,-66051,16777217,0xffffffff,-2,false);for(int n=12;n<cuts.Length;n++)Native("cut="+n,cuts,n);
  // Native unknown markers are skipped with zero payload bytes; deliberately
  // unlike Go's successful bounded stop. These rows document the difference.
  foreach(byte m in new byte[]{2,255})Native("native-unknown="+m,new byte[]{0,0,0,0,0,0,0,0,0,0,0,0,m,0,255,128},16);
  return 0;
 }catch(Exception e){Console.Error.WriteLine(e);return 1;}}
}
