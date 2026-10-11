// P04d: invoke original DataSet.Read/Write and side Drawing.Read, never Load.
// All mutations are independent literal streams or disposable fixture copies.
using System;
using System.IO;
using System.Reflection;
using System.Windows.Forms;
using System.Runtime.Serialization;
class ReferenceDocumentProbe {
 static BindingFlags f=BindingFlags.Public|BindingFlags.NonPublic|BindingFlags.Instance|BindingFlags.Static;
 static Assembly a; static Type d,m,r,ds;
 static object Map(){return FormatterServices.GetUninitializedObject(m);}
 static object Drawing(bool side){return Activator.CreateInstance(d,f,null,new object[]{Map(),side},null);}
 static object Dataset(){a.GetType("PocketTopo.Trip").GetMethod("Reset",f).Invoke(null,null);object survey=Activator.CreateInstance(a.GetType("PocketTopo.Survey"),f,null,new object[]{new BindingSource(),new BindingSource(),new EventHandler(delegate{})},null);return Activator.CreateInstance(ds,f,null,new object[]{survey,Map(),Drawing(false),Drawing(true)},null);}
 static byte[] Literal(int color,int records){
  MemoryStream s=new MemoryStream();BinaryWriter w=new BinaryWriter(s);
  w.Write(new byte[]{84,111,112,3});w.Write(0);w.Write(0);w.Write(0);
  for(int i=0;i<2;i++){w.Write(0);w.Write(0);w.Write(500);}w.Write((byte)0);
  w.Write(-66051);w.Write(16777217);w.Write(-501);
  for(int i=0;i<records;i++){
   if(i%2==0){w.Write((byte)1);w.Write(i%3);for(int j=0;j<i%3;j++){w.Write(-66051-j);w.Write(16777217+j);}w.Write((byte)color);}
   else{w.Write((byte)3);w.Write(int.MinValue);w.Write(int.MaxValue);w.Write((uint)0x800fffff);w.Write(-2);}
  }w.Write((byte)0);return s.ToArray();
 }
 static void Run(string name,byte[] b,int sideStart,string dir){
  string path=Path.Combine(dir,"probe.top");File.WriteAllBytes(path,b);
  object dataset=Dataset();bool accepted=(bool)ds.GetMethod("Read",f).Invoke(dataset,new object[]{path,false});
  MemoryStream s=new MemoryStream(b);s.Position=sideStart;
  object fr=Activator.CreateInstance(r,f,null,new object[]{new BinaryReader(s),3},null);object side=Drawing(true);
  try{d.GetMethod("Read",f).Invoke(side,new object[]{fr});
   object e=d.GetField("elems",f).GetValue(side);int n=0;string values="";
   while(e!=null){Type t=e.GetType();if(n++>0)values+="|";
    if(t.Name=="Polygon"){Array pts=(Array)t.GetField("points",f).GetValue(e);values+="P:"+pts.Length;foreach(object p in pts){Type pt=p.GetType();values+=":"+pt.GetProperty("X").GetValue(p,null)+","+pt.GetProperty("Y").GetValue(p,null);}}
    else{object p=t.GetField("center",f).GetValue(e);Type pt=p.GetType();object id=t.GetField("num",f).GetValue(e);values+="X:"+pt.GetProperty("X").GetValue(p,null)+","+pt.GetProperty("Y").GetValue(p,null)+","+id.GetType().GetField("value",f).GetValue(id)+","+t.GetField("direction",f).GetValue(e);}
    e=a.GetType("PocketTopo.Element").GetField("next",f).GetValue(e);
   }
   Console.WriteLine(name+" accepted="+accepted+" records="+n+" values="+values+" consumed="+s.Position+" tail="+(s.Length-s.Position));
  }catch(TargetInvocationException ex){if(!(ex.InnerException is EndOfStreamException))throw;Console.WriteLine(name+" accepted="+accepted+" error=EndOfStreamException consumed="+s.Position);}
 }
 static int Main(string[] args){try{
  a=Assembly.LoadFrom(args[0]);d=a.GetType("PocketTopo.Drawing");m=a.GetType("PocketTopo.Mapping");r=a.GetType("PocketTopo.FileReader");ds=a.GetType("PocketTopo.DataSet");
  string dir=args[1];Directory.CreateDirectory(dir);
  foreach(int n in new int[]{0,1,4,17})Run("mixed="+n,Literal(255,n),41,dir);
  for(int color=0;color<256;color++)Run("color="+color,Literal(color,3),41,dir);
  byte[] core=Literal(128,4);
  foreach(byte[] tail in new byte[][]{new byte[0],new byte[]{0},new byte[]{0,0},new byte[]{0,0,0},new byte[]{0,0,0,0},new byte[]{1,0,0,0},new byte[]{0,0,0,0,255}}){byte[] b=new byte[core.Length+tail.Length];Array.Copy(core,b,core.Length);Array.Copy(tail,0,b,core.Length,tail.Length);Run("tail="+BitConverter.ToString(tail),b,41,dir);}
  for(int cut=41;cut<core.Length;cut++){byte[] b=new byte[cut];Array.Copy(core,b,cut);Run("cut="+cut,b,41,dir);}
  foreach(string file in new string[]{"api-references.top","api-drawings.top"})Run(file,File.ReadAllBytes(Path.Combine(args[2],file)),file=="api-references.top"?231:405,dir);
  foreach(byte marker in new byte[]{2,255}){byte[] b=Literal(0,0);b[b.Length-1]=marker;Array.Resize(ref b,b.Length+1);Run("unknown="+marker,b,41,dir);}
  object empty=Dataset();string output=Path.Combine(dir,"writer.top");ds.GetMethod("Write",f).Invoke(empty,new object[]{output});byte[] written=File.ReadAllBytes(output);Console.WriteLine("writer size="+written.Length+" bytes="+BitConverter.ToString(written));
  return 0;
 }catch(Exception ex){Console.Error.WriteLine(ex);return 1;}}
}
