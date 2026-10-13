// P12 oracle: only this reflection harness is compiled, never recovered code.
using System; using System.IO; using System.Reflection; using System.Globalization;
using System.Threading; using System.Windows.Forms; using System.Drawing;
using System.Runtime.Serialization; using System.Text; using System.Text.RegularExpressions;
class ReferenceToporobotProbe {
 static BindingFlags f=BindingFlags.Public|BindingFlags.NonPublic|BindingFlags.Instance|BindingFlags.Static;
 static Assembly a; static string root,repo; static StreamWriter cases;
 static string StripClock(string s){return Regex.Replace(s,@"(?m)^    -4[^\r\n]*", "CLOCK");}
 static byte[] Fixture(string kind){using(MemoryStream m=new MemoryStream()){BinaryWriter w=new BinaryWriter(m);w.Write(new byte[]{84,111,112,3});w.Write(4);
  int[] years={1919,1920,2019,2020};for(int i=0;i<4;i++){w.Write(new DateTime(years[i],12,31).Ticks);w.Write(kind=="comments"?"\r\nA\ud83d\ude00B Name"+new string('é',70)+"\ud83d\ude00\r\nSecond"+new string('x',70)+"\r\nTail\r\n"+new string('y',280):"Same Name\r\nSecond Name\r\ntrip tail");w.Write((short)(i==3?-32768:1234+i*100));}
  int[,] rows=kind=="topology"?new int[,]{{65536,65537,1000,0,4096,2,0},{65537,65538,1000,8192,-4096,2,1},{65538,131072,2000,16384,0,2,2},{131072,131073,1000,0,0,2,3},{131073,131075,1000,-8192,0,2,2},{131075,65536,1000,-16384,0,2,1}}:new int[,]{{65536,65537,1000,0,4096,2,0},{65537,65538,1000,8192,-4096,2,1},{65537,int.MinValue,1000,-16384,0,0,2},{65537,int.MinValue,1099,-16384,0,0,2},{65537,int.MinValue,1100,-16384,0,0,2},{65537,int.MinValue,1101,-16384,0,0,2},{65537,int.MinValue,300000,16384,0,0,3},{65537,int.MinValue,1005,8192,16384,0,1},{65537,int.MinValue,1005,8192,-16384,0,1}};
  if(kind=="terminal")rows=new int[,]{{65536,65537,1000,0,0,0,0},{65537,65538,1000,0,0,0,1},{65539,131072,1000,-16384,0,0,2},{65540,131072,1000,-8192,0,0,3}};
  w.Write(rows.GetLength(0));for(int i=0;i<rows.GetLength(0);i++){for(int j=0;j<2;j++)w.Write((uint)rows[i,j]);w.Write(rows[i,2]);w.Write((short)rows[i,3]);w.Write((short)rows[i,4]);w.Write((byte)rows[i,5]);w.Write((byte)77);w.Write((short)rows[i,6]);if((rows[i,5]&2)!=0)w.Write(kind=="comments"?"  " +new string('a',54)+"\ud83d\ude00é\r\n"+new string('b',55)+"\n"+new string('c',280)+"  ":"series/name row "+i);}
  w.Write(kind=="terminal"?2:1);w.Write((uint)65536);w.Write(1005L);w.Write(-1005L);w.Write(1005);w.Write(" café\r\n"+new string('r',270));if(kind=="terminal"){w.Write((uint)131072);w.Write(2000L);w.Write(3000L);w.Write(4000);w.Write("second");}w.Write(0);w.Write(0);w.Write(500);for(int j=0;j<2;j++){w.Write(0);w.Write(0);w.Write(500);w.Write((byte)0);}w.Write(0);return m.ToArray();}}
 static void Run(string name,string rel,bool context,bool close,int circle,int code,int trip,int bits,string merge){
  string path=Path.Combine(repo,rel.Replace('/',Path.DirectorySeparatorChar));
  Type sv=a.GetType("PocketTopo.Survey"),ds=a.GetType("PocketTopo.DataSet"),dr=a.GetType("PocketTopo.Drawing"),mp=a.GetType("PocketTopo.Mapping"),tr=a.GetType("PocketTopo.Trip");
  tr.GetMethod("Reset",f).Invoke(null,null);a.GetType("PocketTopo.Config").GetField("angle",f).SetValue(null,circle);
  object survey=Activator.CreateInstance(sv,f,null,new object[]{new BindingSource(),new BindingSource(),new EventHandler(delegate{})},null);
  object map=FormatterServices.GetUninitializedObject(mp);object plan=Activator.CreateInstance(dr,f,null,new object[]{map,false},null),side=Activator.CreateInstance(dr,f,null,new object[]{FormatterServices.GetUninitializedObject(mp),true},null);
  object data=Activator.CreateInstance(ds,f,null,new object[]{survey,map,plan,side},null);
  a.GetType("PocketTopo.XSection").GetField("survey",f).SetValue(null,survey);
  a.GetType("PocketTopo.Config").GetField("actual",f).SetValue(null,path);
  if(!(bool)ds.GetMethod(context?"Load":"Read",f).Invoke(data,context?new object[]{path}:new object[]{path,false}))throw new Exception("read "+name);
  string dir=Path.Combine(root,name);Directory.CreateDirectory(dir);string output=Path.Combine(dir,"survey.text");
  string status="ok";
  try{
   sv.GetMethod("CloseLoops",f).Invoke(survey,new object[]{close,false});
   object main=FormatterServices.GetUninitializedObject(a.GetType("PocketTopo.MainForm"));main.GetType().GetField("dataSet",f).SetValue(main,data);a.GetType("PocketTopo.Exporter").GetField("main",f).SetValue(null,main);
   Type exp=a.GetType("PocketTopo.ToporobotExp"),model=a.GetType("PocketTopo.Toporobot");object exporter=Activator.CreateInstance(exp,true);
   Encoding enc=(Encoding)Activator.CreateInstance(a.GetType("PocketTopo.MacEncoding"),true);
   string inname=merge==""?null:Path.Combine(root,merge);
   // The public path swallows exceptions; invoke AddToporobot separately to verify success.
   object obj=inname==null?Activator.CreateInstance(model):model.GetMethod("Read",f).Invoke(null,new object[]{new StreamReader(inname,enc)});
   exp.GetMethod("AddToporobot",f).Invoke(exporter,new object[]{obj,inname==null?"survey":"",survey,code,trip,(bits&1)!=0,(bits&2)!=0,(bits&4)!=0,(bits&8)!=0});
   exp.GetMethod("WriteExport",f).Invoke(exporter,new object[]{inname,output,code,trip,(bits&1)!=0,(bits&2)!=0,(bits&4)!=0,(bits&8)!=0});
   using(MemoryStream m=new MemoryStream()){using(StreamWriter w=new StreamWriter(m,enc)){model.GetMethod("Write",f).Invoke(obj,new object[]{w});w.Flush();if(StripClock(enc.GetString(m.ToArray()))!=StripClock(enc.GetString(File.ReadAllBytes(output))))throw new Exception("partial export "+name);}}
  }catch(TargetInvocationException e){status=e.InnerException.GetType().Name;Console.WriteLine(name+" error="+status);}
  cases.WriteLine(name+"\t"+rel+"\t"+context+"\t"+close+"\t"+circle+"\t"+code+"\t"+trip+"\t"+bits+"\t"+merge+"\t"+status);cases.Flush();
  if(status=="ok")Console.WriteLine(name+" bytes="+File.ReadAllBytes(output).Length);
 }
 static int Main(string[] args){try{
  Thread.CurrentThread.CurrentCulture=CultureInfo.InvariantCulture;Thread.CurrentThread.CurrentUICulture=CultureInfo.InvariantCulture;
  a=Assembly.LoadFrom(args[0]);root=args[1];repo=args[2];Directory.CreateDirectory(root);
  Type form=a.GetType("PocketTopo.MainForm");foreach(string color in new string[]{"black","gray","brown","blue","green","red","orange"})form.GetField(color,f).SetValue(null,new Pen(Color.Black));
  Encoding mac=(Encoding)Activator.CreateInstance(a.GetType("PocketTopo.MacEncoding"),true);byte[] enc=new byte[65536];for(int i=0;i<65536;i++)enc[i]=mac.GetBytes(new char[]{(char)i})[0];File.WriteAllBytes(Path.Combine(root,"mac.bin"),enc);
  cases=new StreamWriter(Path.Combine(root,"cases.tsv"));
  if(args.Length>3&&args[3]=="boundaries"){
   string seedDir=Path.Combine(root,"boundary-seed");Directory.CreateDirectory(seedDir);
   File.WriteAllBytes(Path.Combine(seedDir,"survey.text"),mac.GetBytes("     1    -2   1   1   1 S\r\n     1    -1   1   1   1       1       0       1       1       1       3       0\r\n     1     0   1   1   1    0.00    0.00    0.00    1.00    2.00    3.00    4.00\r\n"));
   foreach(string n in new string[]{"cross1","zero-next","positive-next","zero-splay"})Run("boundary-"+n,"internal/exporttoporobot/testdata/boundary/"+n+".top",false,false,360,1,1,4,"");
   foreach(string n in new string[]{"zero-next","positive-next"})Run("boundary-merge-"+n,"internal/exporttoporobot/testdata/boundary/"+n+".top",false,false,360,1,1,4,"boundary-seed/survey.text");
   cases.Close();Console.WriteLine("P12 native boundary capture complete");return 0;
  }
  foreach(int circle in new int[]{360,400}){
   foreach(bool close in new bool[]{false,true}){
    string prefix="c"+circle+(close?"-closed-":"-");
    foreach(string n in new string[]{"features","extreme","dimensions","duplicates17","duplicates32","duplicates65"})Run(prefix+n,"internal/exportvisualtopo/testdata/"+n+".top",false,close,circle,1,1,4,"");
    foreach(string p in Directory.GetFiles(Path.Combine(repo,"internal/processing/testdata/geometry"),"*.top"))Run(prefix+"geometry-"+Path.GetFileNameWithoutExtension(p),"internal/processing/testdata/geometry/"+Path.GetFileName(p),false,close,circle,1,1,4,"");
    foreach(string p in Directory.GetDirectories(Path.Combine(repo,"internal/processing/testdata/context")))if(File.Exists(Path.Combine(p,"current.top")))Run(prefix+"context-"+Path.GetFileName(p),"internal/processing/testdata/context/"+Path.GetFileName(p)+"/current.top",true,close,circle,1,1,4,"");
    foreach(string n in new string[]{"empty","mixed","flags","numbers","rounding","api-references","api-drawings"})Run(prefix+"text-"+n,(n.StartsWith("api-")?"internal/top/testdata/":"internal/exporttext/testdata/")+n+".top",false,close,circle,1,1,4,"");
   }
   for(int bits=0;bits<16;bits++)Run("options-"+circle+"-"+bits,"internal/exportvisualtopo/testdata/dimensions.top",false,false,circle,7,9,bits,"");
  }
  foreach(int idx in new int[]{2,9999,10000,100000})Run("padding-"+idx,"internal/exportvisualtopo/testdata/dimensions.top",false,false,360,idx,idx,4,"");
  Run("merge-base","internal/exportvisualtopo/testdata/dimensions.top",false,false,360,1,1,4,"");
  for(int bits=0;bits<16;bits++)Run("merge-"+bits,"internal/exportvisualtopo/testdata/dimensions.top",false,false,400,7,9,bits,"merge-base/survey.text");
  string extra=Path.Combine(repo,"internal/exporttoporobot/testdata");Directory.CreateDirectory(extra);
  foreach(string kind in new string[]{"topology","comments","dedup","terminal"}){File.WriteAllBytes(Path.Combine(extra,kind+".top"),Fixture(kind));for(int bits=0;bits<16;bits++)foreach(int circle in new int[]{360,400})Run("extra-"+kind+"-"+circle+"-"+bits,"internal/exporttoporobot/testdata/"+kind+".top",false,false,circle,1,1,bits,"");}
  cases.Close();Console.WriteLine("P12 native capture complete");return 0;
 }catch(Exception e){Console.Error.WriteLine(e);return 1;}}
}
