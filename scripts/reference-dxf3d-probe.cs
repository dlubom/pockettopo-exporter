// P13 oracle: invoke the unchanged original assembly, never recovered code.
using System; using System.IO; using System.Reflection; using System.Globalization;
using System.Threading; using System.Windows.Forms; using System.Runtime.Serialization;
class ReferenceDXF3DProbe {
 class CaptureLimit:Exception{}
 class LimitedWriter:StringWriter {
  int lines,limit;public LimitedWriter():this(2800){}public LimitedWriter(int limit):base(CultureInfo.InvariantCulture){this.limit=limit;NewLine="\r\n";}
  void Count(){if(++lines>limit)throw new CaptureLimit();}
  public override void WriteLine(int value){Count();base.WriteLine(value);}
  public override void WriteLine(string value){Count();base.WriteLine(value);}
 }
 static BindingFlags f=BindingFlags.Public|BindingFlags.NonPublic|BindingFlags.Instance|BindingFlags.Static;
 static Assembly a; static Type sv,ds,dr,mp,tr,td,st; static string root,repo; static StreamWriter manifest;
 static object sequenceWriter;static LimitedWriter sequenceOutput;static StringWriter sequenceInput=new StringWriter();static string sequenceStatus="complete";
 static object Field(object o,string n){return o.GetType().GetField(n,f).GetValue(o);}
 static void Set(object o,string n,object v){o.GetType().GetField(n,f).SetValue(o,v);}
 static object Station(int x,int y,int h){object s=FormatterServices.GetUninitializedObject(st);Set(s,"x",x);Set(s,"y",y);Set(s,"h",h);Set(s,"dist",1);return s;}
 static string Bounded(ThreadStart action){string status="complete";Thread t=new Thread(delegate(){try{action();}catch(TargetInvocationException e){status=e.InnerException.GetType().Name;}catch(Exception e){status=e.GetType().Name;}});t.Start();if(!t.Join(2000)){t.Abort();t.Join();status="timeout";}return status;}
 static void Model(string name,int[,] pts){
  // First two rows are endpoints; further rows attach to endpoint1 in order.
  string p=Path.Combine(root,name+".points.tsv");using(StreamWriter w=new StreamWriter(p)){for(int i=0;i<pts.GetLength(0);i++)w.WriteLine(pts[i,0]+"\t"+pts[i,1]+"\t"+pts[i,2]);}
  object s0=Station(pts[0,0],pts[0,1],pts[0,2]),s1=Station(pts[1,0],pts[1,1],pts[1,2]),prev=null;
  for(int i=2;i<pts.GetLength(0);i++){object s=Station(pts[i,0],pts[i,1],pts[i,2]);if(prev==null)Set(s1,"cross",s);else Set(prev,"next",s);prev=s;}
  StringWriter w0=new LimitedWriter();object writer=Activator.CreateInstance(td,f,null,new object[]{w0},null);
  string status=Bounded(delegate(){td.GetMethod("WriteStretch",f).Invoke(writer,new object[]{s0,s1});});
  File.WriteAllText(Path.Combine(root,name+".entities"),w0.ToString(),new System.Text.UTF8Encoding(false));
  manifest.WriteLine(name+"\tmodel\t"+name+".points.tsv\tFalse\tFalse\t"+status+"\t"+name+".entities");manifest.Flush();Console.WriteLine(name+" "+status+" bytes="+w0.ToString().Length);
  if(sequenceWriter!=null&&status=="complete"&&sequenceStatus=="complete"){sequenceInput.WriteLine(name+".points.tsv");sequenceStatus=Bounded(delegate(){td.GetMethod("WriteStretch",f).Invoke(sequenceWriter,new object[]{s0,s1});});}
 }
 static void Run(string name,string path,bool context,bool close){
  tr.GetMethod("Reset",f).Invoke(null,null);
  object survey=Activator.CreateInstance(sv,f,null,new object[]{new BindingSource(),new BindingSource(),new EventHandler(delegate{})},null);
  object map=FormatterServices.GetUninitializedObject(mp),plan=Activator.CreateInstance(dr,f,null,new object[]{map,false},null),side=Activator.CreateInstance(dr,f,null,new object[]{FormatterServices.GetUninitializedObject(mp),true},null);
  object data=Activator.CreateInstance(ds,f,null,new object[]{survey,map,plan,side},null);
  a.GetType("PocketTopo.XSection").GetField("survey",f).SetValue(null,survey);
  a.GetType("PocketTopo.Config").GetField("actual",f).SetValue(null,path);
  if(!(bool)ds.GetMethod(context?"Load":"Read",f).Invoke(data,context?new object[]{path}:new object[]{path,false}))throw new Exception("read "+name);
  string output=Path.Combine(root,name+".dxf"),status="complete";
  try{sv.GetMethod("CloseLoops",f).Invoke(survey,new object[]{close,false});}catch(TargetInvocationException e){status=e.InnerException.GetType().Name;}
  if(status=="complete")status=Bounded(delegate(){td.GetMethod("WriteDXF",f).Invoke(null,new object[]{output,survey,false});});
  if(!File.Exists(output))File.WriteAllBytes(output,new byte[0]);
  if(status=="complete"&&!File.ReadAllText(output).EndsWith("0\r\nEOF\r\n"))status="partial";
  manifest.WriteLine(name+"\ttop\t"+path.Replace(repo+"\\", "").Replace("\\", "/")+"\t"+context+"\t"+close+"\t"+status+"\t"+name+".dxf");manifest.Flush();Console.WriteLine(name+" "+status+" bytes="+new FileInfo(output).Length);
 }
 static byte[] Semantics(bool zeroCross){using(MemoryStream m=new MemoryStream()){BinaryWriter w=new BinaryWriter(m);w.Write(new byte[]{84,111,112,3});w.Write(0);
  int[,] rows={{65536,65537,0,0,0,0},{65537,65538,1005,0,0,0},{65536,unchecked((int)0x80000000),500,16384,8192,4},{65538,65539,1100,16384,0,0},{65539,65540,900,0,0,4},{65538,unchecked((int)0x80000000),800,0,-8192,8},{65540,65541,900,0,0,0}};
  w.Write(rows.GetLength(0)+(zeroCross?1:0));for(int i=0;i<rows.GetLength(0);i++){w.Write((uint)rows[i,0]);w.Write((uint)rows[i,1]);w.Write(rows[i,2]);w.Write((short)rows[i,3]);w.Write((short)rows[i,4]);w.Write((byte)rows[i,5]);w.Write((byte)137);w.Write((short)-1);}
  if(zeroCross){w.Write((uint)65537);w.Write((uint)0x80000000);w.Write(600);w.Write((short)-16384);w.Write((short)8192);w.Write((byte)0);w.Write((byte)237);w.Write((short)-1);}
  w.Write(2);foreach(uint id in new uint[]{65536,65538}){w.Write(id);w.Write(100005L+(id-65536)*500);w.Write(200005L+(id-65536)*500);w.Write(300005);w.Write("reference");}
  for(int i=0;i<2;i++){w.Write(0);w.Write(0);w.Write(500);}w.Write((byte)0);w.Write(0);w.Write(0);w.Write(500);w.Write((byte)0);w.Write(0);return m.ToArray();}}
 static void SortCases(){
  uint[][] cases={new uint[]{65537,65537,65536,65537,65536,65537,65536},new uint[]{0x7fffffff,0x80000001,0x80000000,0x80100000,65536,0},new uint[]{0x80000001,0x7fffffff},new uint[]{0x7fffffff,0x80000001}};
  for(int c=0;c<24;c++){uint[] raw=c<cases.Length?cases[c]:new uint[5+c*3];if(c>=cases.Length)for(int i=0;i<raw.Length;i++)raw[i]=65536+(uint)((i*1372+c)%5);
   Array array=Array.CreateInstance(st,raw.Length);StringWriter input=new StringWriter();for(int i=0;i<raw.Length;i++){int n=(int)raw[i];if(n==int.MinValue)n=-1;else if(n<0)n+=2146435071;object s=Station(0,0,0);Set(s,"num",Activator.CreateInstance(a.GetType("PocketTopo.ID"),f,null,new object[]{n},null));Set(s,"idx",i);array.SetValue(s,i);input.WriteLine(raw[i]);}
   string status="complete";try{foreach(MethodInfo method in typeof(Array).GetMethods(BindingFlags.Public|BindingFlags.Static)){if(method.Name=="Sort"&&method.IsGenericMethodDefinition&&method.GetParameters().Length==1){method.MakeGenericMethod(st).Invoke(null,new object[]{array});break;}}}catch(TargetInvocationException e){status=e.InnerException.GetType().Name;}
   string name="sort"+c;File.WriteAllText(Path.Combine(root,name+".input"),input.ToString());using(StreamWriter w=new StreamWriter(Path.Combine(root,name+".output"))){foreach(object s in array)w.WriteLine(Field(s,"idx"));}
   manifest.WriteLine(name+"\tsort\t"+name+".input\tFalse\tFalse\t"+status+"\t"+name+".output");Console.WriteLine(name+" "+status);
  }
 }
 public static int Main(string[] args){try{
  Thread.CurrentThread.CurrentCulture=CultureInfo.InvariantCulture;Thread.CurrentThread.CurrentUICulture=CultureInfo.InvariantCulture;
  a=Assembly.LoadFrom(args[0]);sv=a.GetType("PocketTopo.Survey");ds=a.GetType("PocketTopo.DataSet");dr=a.GetType("PocketTopo.Drawing");mp=a.GetType("PocketTopo.Mapping");tr=a.GetType("PocketTopo.Trip");td=a.GetType("PocketTopo.ThreeD");st=a.GetType("PocketTopo.Station");root=args[1];repo=args[2];Directory.CreateDirectory(root);
  manifest=new StreamWriter(Path.Combine(root,"cases.tsv"));manifest.WriteLine("name\tkind\tinput\tcontext\tclose\tstatus\toutput");
  if(args.Length>3&&args[3]=="sort"){SortCases();manifest.Close();return 0;}
  if(args.Length>3&&args[3]=="sequence"){sequenceOutput=new LimitedWriter(1000000);sequenceWriter=Activator.CreateInstance(td,f,null,new object[]{sequenceOutput},null);}
  Model("two",new int[,]{{-1,17,-33},{1005,-1005,17}});
  Model("three",new int[,]{{0,0,0},{1000,0,0},{500,500,500}});
  Model("coincident",new int[,]{{0,0,0},{0,0,0},{0,0,0},{0,0,0}});
  Model("collinear",new int[,]{{0,0,0},{1000,0,0},{500,0,0},{2000,0,0}});
  Model("coplanar",new int[,]{{0,0,0},{1000,0,0},{1000,1000,0},{0,1000,0}});
  Model("tetrahedron",new int[,]{{0,0,0},{1000,0,0},{0,1000,0},{0,0,1000}});
  Model("duplicates",new int[,]{{0,0,0},{1000,0,0},{0,1000,0},{0,0,1000},{0,1000,0}});
  Model("quantization",new int[,]{{-17,15,-15},{15,-17,17},{-15,-15,15},{17,17,-17},{1,-1,0}});
  Model("overflow",new int[,]{{int.MinValue,int.MaxValue,int.MinValue},{int.MaxValue,int.MinValue,int.MaxValue},{0,0,0},{1000000,-2000000,3000000},{-3000000,2000000,-1000000}});
  Random random=new Random(1372);for(int c=0;c<50;c++){int[,] pts=new int[4+c%8,3];for(int i=0;i<pts.GetLength(0);i++)for(int j=0;j<3;j++)pts[i,j]=random.Next(-200000,200000);Model("random"+c,pts);}
  if(sequenceWriter!=null){File.WriteAllText(Path.Combine(root,"sequence.input"),sequenceInput.ToString());File.WriteAllText(Path.Combine(root,"sequence.entities"),sequenceOutput.ToString(),new System.Text.UTF8Encoding(false));manifest.WriteLine("sequence\tsequence\tsequence.input\tFalse\tFalse\t"+sequenceStatus+"\tsequence.entities");Console.WriteLine("sequence "+sequenceStatus+" bytes="+sequenceOutput.ToString().Length);manifest.Close();return 0;}
  if(args.Length>3&&args[3]=="semantics"){
   foreach(bool cross in new bool[]{false,true}){string path=Path.Combine(root,cross?"zero-cross.top":"fallback.top");File.WriteAllBytes(path,Semantics(cross));foreach(bool close in new bool[]{false,true})Run(Path.GetFileNameWithoutExtension(path)+(close?"-closed":""),path,false,close);}manifest.Close();return 0;
  }
  foreach(bool close in new bool[]{false,true}){
   foreach(string path in Directory.GetFiles(Path.Combine(repo,"internal/processing/testdata/geometry"),"*.top"))Run("geometry-"+Path.GetFileNameWithoutExtension(path)+(close?"-closed":""),path,false,close);
   foreach(string dir in Directory.GetDirectories(Path.Combine(repo,"internal/processing/testdata/context"))){string path=Path.Combine(dir,"current.top");if(File.Exists(path))Run("context-"+Path.GetFileName(dir)+(close?"-closed":""),path,true,close);}
   foreach(string n in new string[]{"features","zero-polygons","extreme"})Run(n+(close?"-closed":""),Path.Combine(repo,"internal/exportdxf/testdata/"+n+".top"),false,close);
   Run("template-rays"+(close?"-closed":""),Path.Combine(repo,"internal/exportdxf/testdata/template-rays/current.top"),true,close);
  }
  manifest.Close();return 0;
 }catch(Exception e){Console.Error.WriteLine(e);return 1;}}
}
