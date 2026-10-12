// P09 native oracle: literal fixture plus original assembly Read/Load/CloseLoops/WriteExport.
// No recovered exporter or geometry code is compiled.
using System; using System.IO; using System.Reflection; using System.Globalization;
using System.Threading; using System.Windows.Forms; using System.Drawing;
using System.Runtime.Serialization;
class ReferenceTherionProbe {
 static BindingFlags f=BindingFlags.Public|BindingFlags.NonPublic|BindingFlags.Instance|BindingFlags.Static;
 static Assembly a; static Type sv,ds,dr,mp,tr; static string root;
 static object Field(object o,string n){return o.GetType().GetField(n,f).GetValue(o);}
 static byte[] Literal(){using(MemoryStream m=new MemoryStream()){BinaryWriter w=new BinaryWriter(m);w.Write(new byte[]{84,111,112,3});w.Write(3);
  foreach(short decl in new short[]{1234,short.MinValue,-16384}){w.Write(new DateTime(2024,2,29,23,59,59).Ticks);w.Write("ignored \"comment\" 終");w.Write(decl);}
  int[,] shots={{65536,65537,1000,100,200,2,0},{65536,65537,2000,16384,300,1,1},{65537,65536,3000,-32768,-400,0,2},{65537,65538,4321,-12345,-1000,1,1},{65538,unchecked((int)0x80000000),2345,32767,12345,0,2},{65538,65539,9876,1000,-1000,16,0},{65539,unchecked((int)0x80000000),500,8192,-4096,8,2},{65539,unchecked((int)0x80000000),600,-8192,4096,0,2},{65539,unchecked((int)0x80000000),0,0,0,0,0},{65540,65541,1000,0,0,0,0},{65549,65549,1000,16384,8192,0,0},{unchecked((int)0x80000000),65543,1000,0,0,0,0},{unchecked((int)0x80000000),65543,2000,16384,0,0,0},{65550,65551,1000,0,0,0,3},{65539,65536,500,16384,8192,0,-1},{unchecked((int)0x80000000),65543,1000,0,0,0,0},{65536,65537,5000,16384,0,0,32767}};
  w.Write(shots.GetLength(0));for(int i=0;i<shots.GetLength(0);i++){w.Write((uint)shots[i,0]);w.Write((uint)shots[i,1]);w.Write(shots[i,2]);w.Write((short)shots[i,3]);w.Write((short)shots[i,4]);w.Write((byte)shots[i,5]);w.Write((byte)255);w.Write((short)shots[i,6]);if((shots[i,5]&2)!=0)w.Write("comment");}
  w.Write(3);foreach(uint id in new uint[]{65536,65536,0x80000000}){w.Write(id);w.Write(1005L);w.Write(-1005L);w.Write(125);w.Write("ref");}
  w.Write(123);w.Write(-456);w.Write(-500);
  for(int side=0;side<2;side++){w.Write(999);w.Write(-999);w.Write(1000);
   for(int color=0;color<256;color++){w.Write((byte)1);w.Write(color%3);for(int j=0;j<color%3;j++){w.Write(j==0?int.MinValue:1005+color);w.Write(j==0?int.MaxValue:-1005-color);}w.Write((byte)color);}
   foreach(int dir in new int[]{-1,0,8192,65536,int.MinValue,int.MaxValue}){w.Write((byte)3);w.Write(int.MaxValue);w.Write(int.MinValue);w.Write((uint)65539);w.Write(dir);}
   foreach(uint id in new uint[]{65536,65537,65538,65540,0x80000000,0x80100000,12345}){w.Write((byte)3);w.Write(1005);w.Write(-1005);w.Write(id);w.Write(-1);}
   w.Write((byte)0);
  }w.Write(0);return m.ToArray();}}
 static byte[] Extreme(){using(MemoryStream m=new MemoryStream()){BinaryWriter w=new BinaryWriter(m);w.Write(new byte[]{84,111,112,3});w.Write(0);w.Write(0);w.Write(4);foreach(uint id in new uint[]{0x7fffffff,0x80000001,65536,0x80100000}){w.Write(id);w.Write((long)int.MaxValue);w.Write((long)int.MinValue);w.Write(int.MinValue);w.Write("");}for(int i=0;i<2;i++){w.Write(0);w.Write(0);w.Write(500);}w.Write((byte)1);w.Write(1);w.Write(0);w.Write(int.MinValue);w.Write((byte)1);w.Write((byte)0);w.Write(0);w.Write(0);w.Write(500);w.Write((byte)0);w.Write(0);return m.ToArray();}}
 static byte[] ContextTop(bool current,bool empty){using(MemoryStream m=new MemoryStream()){BinaryWriter w=new BinaryWriter(m);w.Write(new byte[]{84,111,112,3});w.Write(1);w.Write(0L);w.Write("");w.Write((short)4096);
 int[,] rows=current?(empty?new int[0,5]:new int[,]{{unchecked((int)0x80000000),65549,500,0,0},{65537,unchecked((int)0x80000000),200,0,8192},{65537,unchecked((int)0x80000000),300,16384,0},{65537,65538,1000,16384,0}}):new int[,]{{65536,65537,1000,0,8192},{65537,unchecked((int)0x80000000),100,16384,0}};
 w.Write(rows.GetLength(0));for(int i=0;i<rows.GetLength(0);i++){w.Write((uint)rows[i,0]);w.Write((uint)rows[i,1]);w.Write(rows[i,2]);w.Write((short)rows[i,3]);w.Write((short)rows[i,4]);w.Write((byte)0);w.Write((byte)0);w.Write((short)0);}w.Write(current?0:2);if(!current){w.Write((uint)65535);w.Write(90000L);w.Write(195000L);w.Write(299000);w.Write("");w.Write((uint)65536);w.Write(100000L);w.Write(200000L);w.Write(300000);w.Write("");}
 for(int i=0;i<2;i++){w.Write(0);w.Write(0);w.Write(500);}w.Write((byte)1);w.Write(2);w.Write(3005);w.Write(-4005);w.Write(-3005);w.Write(4005);w.Write((byte)5);foreach(int direction in new int[]{-1,8192}){w.Write((byte)3);w.Write(1005);w.Write(-1005);w.Write((uint)65537);w.Write(direction);}w.Write((byte)0);w.Write(0);w.Write(0);w.Write(500);w.Write((byte)1);w.Write(1);w.Write(5005);w.Write(6005);w.Write((byte)4);w.Write((byte)3);w.Write(2005);w.Write(-2005);w.Write((uint)65537);w.Write(-1);w.Write((byte)0);w.Write(0);return m.ToArray();}}
 static byte[] BoundaryTop(bool current){using(MemoryStream m=new MemoryStream()){BinaryWriter w=new BinaryWriter(m);w.Write(new byte[]{84,111,112,3});w.Write(1);w.Write(0L);w.Write("");w.Write((short)0);w.Write(1);w.Write((uint)(current?65537:65536));w.Write((uint)(current?65536:65537));w.Write(current?2000:1000);w.Write((short)(current?-32768:0));w.Write((short)0);w.Write((byte)0);w.Write((byte)0);w.Write((short)0);w.Write(0);for(int i=0;i<2;i++){w.Write(0);w.Write(0);w.Write(500);}w.Write((byte)0);w.Write(0);w.Write(0);w.Write(500);w.Write((byte)0);w.Write(0);return m.ToArray();}}
 static void Run(string name,string path,bool context,bool close){tr.GetMethod("Reset",f).Invoke(null,null);
  object survey=Activator.CreateInstance(sv,f,null,new object[]{new BindingSource(),new BindingSource(),new EventHandler(delegate{})},null);
  object map=FormatterServices.GetUninitializedObject(mp);object plan=Activator.CreateInstance(dr,f,null,new object[]{map,false},null),side=Activator.CreateInstance(dr,f,null,new object[]{FormatterServices.GetUninitializedObject(mp),true},null);
  object data=Activator.CreateInstance(ds,f,null,new object[]{survey,map,plan,side},null);
  a.GetType("PocketTopo.XSection").GetField("survey",f).SetValue(null,survey);
  a.GetType("PocketTopo.Config").GetField("actual",f).SetValue(null,path);
  bool ok=(bool)ds.GetMethod(context?"Load":"Read",f).Invoke(data,context?new object[]{path}:new object[]{path,false});if(!ok)throw new Exception("read "+name);
  string output=Path.Combine(root,name+(close?"-closed":"")+".txt");
  try{sv.GetMethod("CloseLoops",f).Invoke(survey,new object[]{close,false});}catch(TargetInvocationException e){Console.WriteLine(name+(close?"-closed":"")+" error="+e.InnerException.GetType().Name);return;}
  a.GetType("PocketTopo.Therion").GetMethod("WriteExport",f).Invoke(null,new object[]{output,survey,plan,side});
  if(!File.Exists(output))throw new Exception("no export "+name);Console.WriteLine(name+(close?"-closed":"")+" bytes="+File.ReadAllBytes(output).Length);
 }
 static int Main(string[] args){try{Thread.CurrentThread.CurrentCulture=CultureInfo.InvariantCulture;Thread.CurrentThread.CurrentUICulture=CultureInfo.InvariantCulture;
  a=Assembly.LoadFrom(args[0]);sv=a.GetType("PocketTopo.Survey");ds=a.GetType("PocketTopo.DataSet");dr=a.GetType("PocketTopo.Drawing");mp=a.GetType("PocketTopo.Mapping");tr=a.GetType("PocketTopo.Trip");root=args[1];Directory.CreateDirectory(root);
  // Native Read maps raw colors to these distinct static pen identities.
  Type form=a.GetType("PocketTopo.MainForm");foreach(string color in new string[]{"black","gray","brown","blue","green","red","orange"})form.GetField(color,f).SetValue(null,new Pen(Color.Black));
  string literal=Path.Combine(root,"features.top");File.WriteAllBytes(literal,Literal());
  string extreme=Path.Combine(root,"extreme.top");File.WriteAllBytes(extreme,Extreme());
  foreach(bool close in new bool[]{false,true}){
   foreach(bool empty in new bool[]{false,true}){string n=empty?"empty-current-drawing":"template-rays";string dir=Path.Combine(root,n);Directory.CreateDirectory(dir);File.WriteAllBytes(Path.Combine(dir,"a.top"),ContextTop(false,false));string cp=Path.Combine(dir,"current.top");File.WriteAllBytes(cp,ContextTop(true,empty));Run("nativecontext-"+n,cp,true,close);}
   string bd=Path.Combine(root,"boundary-data");Directory.CreateDirectory(bd);File.WriteAllBytes(Path.Combine(bd,"a.top"),BoundaryTop(false));string bc=Path.Combine(bd,"current.top");File.WriteAllBytes(bc,BoundaryTop(true));Run("nativecontext-boundary-data",bc,true,close);
   Run("features",literal,false,close);Run("extreme",extreme,false,close);
   foreach(string path in Directory.GetFiles(Path.Combine(args[2],"internal/processing/testdata/geometry"),"*.top")){string name=Path.GetFileNameWithoutExtension(path);Run("geometry-"+name,path,false,close);}
   foreach(string dir in Directory.GetDirectories(Path.Combine(args[2],"internal/processing/testdata/context"))){string path=Path.Combine(dir,"current.top");if(File.Exists(path))Run("context-"+Path.GetFileName(dir),path,true,close);}
   foreach(string name in new string[]{"empty","mixed","flags","numbers","rounding","api-references","api-drawings"})Run("text-"+name,Path.Combine(args[2],(name.StartsWith("api-")?"internal/top/testdata/":"internal/exporttext/testdata/")+name+".top"),false,close);
  }return 0;
 }catch(Exception e){Console.Error.WriteLine(e);return 1;}}
}
