// P06: original assembly only. Empty placement dictionaries isolate EvalData's
// grouping from SetupShot/SetupXsect; no Update, templates or closure are run.
using System;
using System.IO;
using System.Reflection;
using System.Globalization;
using System.Threading;
using System.Windows.Forms;
using System.Runtime.Serialization;
class ReferenceGroupingProbe {
 static BindingFlags f=BindingFlags.Public|BindingFlags.NonPublic|BindingFlags.Instance|BindingFlags.Static;
 static Type survey,ds,drawing,mapping,trip,station,angle;
 static object Angle(int value){return Activator.CreateInstance(angle,new object[]{value});}
 static int[] Rect(int len,int value){object[] args={len,Angle(value),0,0};angle.GetMethod("ToRect",f).Invoke(null,args);return new int[]{(int)args[2],(int)args[3]};}
 static int Field(object s,string name){return (int)station.GetField(name,f).GetValue(s);}
 static uint Bits(object v){return BitConverter.ToUInt32(BitConverter.GetBytes((float)v),0);}
 static void Shot(BinaryWriter w,uint from,uint to,int dist,int az,int inc,int flags,int idx){
  w.Write(from);w.Write(to);w.Write(dist);w.Write(unchecked((short)az));w.Write(unchecked((short)inc));w.Write((byte)flags);w.Write((byte)173);w.Write((short)idx);if((flags&2)!=0)w.Write("row");
 }
 static byte[] Literal(string name){
  MemoryStream shots=new MemoryStream();BinaryWriter s=new BinaryWriter(shots);int count=0;
  if(name=="repeats"){
   Shot(s,65536,65537,1000,100,200,2,0);Shot(s,65536,65537,2000,16384,300,3,1);Shot(s,65537,65536,3000,-32768,-400,10,2);count+=3;
   Shot(s,65537,65538,1234,-1,-2,1,3);count++;
   Shot(s,65536,65537,5000,16384,0,0,-1);count++;
   Shot(s,65538,65539,1001,8192,-8192,1,0);Shot(s,65539,65538,1002,40960,8192,0,0);count+=2;
   for(int i=0;i<5;i++){Shot(s,i%2==0?65540U:65541U,i%2==0?65541U:65540U,1000+i,i%2==0?1:32769,i%2==0?-1:1,i==0?4:255,i%4);count++;}
  } else if(name=="boundaries"){
   Shot(s,65536,65537,1000000,-182,0,0,-1);Shot(s,65536,65537,1000000,182,0,0,-2);count+=2;
   Shot(s,65537,65538,1000,0,0,0,0);Shot(s,65537,65538,1000,32768,0,0,0);count+=2;
   Shot(s,65538,65539,0,0,0,0,1);count++;
   Shot(s,65539,65540,1000,0,16384,0,2);count++;
   Shot(s,65540,65541,-5,-16384,-32768,1,32767);count++;
   Shot(s,65541,65542,int.MaxValue,16384,0,0,-1);Shot(s,65541,65542,int.MaxValue,16384,0,0,-1);count+=2;
   Shot(s,65542,65543,int.MinValue,-16384,0,0,-1);count++;
   Shot(s,65543,65544,1,32768,0,0,-1);Shot(s,65543,65544,2,32768,0,0,-1);count+=2;
   Shot(s,65544,65545,2000,16384,0,0,1);count++;
   Shot(s,65545,65546,1000,100,200,0,4);count++;
  } else if(name=="identities"){
   Shot(s,0x80000000,65537,1000,0,0,0,-1);Shot(s,0x800fffff,65537,1000,0,0,0,-1);count+=2;
   Shot(s,65536,0x80000000,1000,0,0,0,-1);Shot(s,65536,0x800fffff,2000,0,0,0,-1);count+=2;
   Shot(s,0x800ffffe,0x800ffffd,1000,0,0,0,-1);Shot(s,0x800ffffd,0x800ffffe,2000,32768,0,0,-1);count+=2;
   Shot(s,65536,65536,1000,0,0,0,-1);Shot(s,65536,65536,2000,0,0,0,-1);count+=2;
   Shot(s,65536,65537,1000,0,0,0,-1);Shot(s,0x80000000,65537,1000,0,0,0,-1);Shot(s,65536,65537,2000,0,0,0,-1);count+=3;
   Shot(s,0x80200000,65537,1000,16384,0,0,-1);Shot(s,0x000fffff,65537,3000,16384,0,0,-1);count+=2;
  } else if(name=="flags"){
   for(int i=0;i<256;i++){uint from=(uint)(65536+i*2);Shot(s,from,from+1,12345,12345,-12345,i,-1);Shot(s,from+1,from,23456,-20423,12345,255-i,-1);count+=2;}
  } else if(name=="random"){
   ulong state=0x123456789abcdef0UL;
   for(int i=0;i<512;i++){
    state=unchecked(state*6364136223846793005UL+1442695040888963407UL);
    uint from=(uint)(65536+(i/4)*2),to=from+1;bool rev=(state&1)!=0;
    Shot(s,rev?to:from,rev?from:to,unchecked((int)state),unchecked((short)(state>>32)),unchecked((short)(state>>48)),(byte)(state>>24),(short)(i%6-2));count++;
   }
  }
  MemoryStream stream=new MemoryStream();BinaryWriter w=new BinaryWriter(stream);
  w.Write(new byte[]{84,111,112,3});w.Write(4);
  foreach(short decl in new short[]{1234,short.MinValue,-16384,32767}){w.Write(0L);w.Write("");w.Write(decl);}
  w.Write(count);w.Write(shots.ToArray());w.Write(0);
  for(int i=0;i<2;i++){w.Write(0);w.Write(0);w.Write(500);}w.Write((byte)0);w.Write(0);w.Write(0);w.Write(500);w.Write((byte)0);w.Write(0);return stream.ToArray();
 }
 static void Run(string name,string output){
  trip.GetMethod("Reset",f).Invoke(null,null);
  object surveyObj=Activator.CreateInstance(survey,f,null,new object[]{new BindingSource(),new BindingSource(),new EventHandler(delegate{})},null);
  object map=FormatterServices.GetUninitializedObject(mapping);
  object data=Activator.CreateInstance(ds,f,null,new object[]{surveyObj,map,Activator.CreateInstance(drawing,f,null,new object[]{map,false},null),Activator.CreateInstance(drawing,f,null,new object[]{FormatterServices.GetUninitializedObject(mapping),true},null)},null);
  string path=Path.Combine(output,name+".top");File.WriteAllBytes(path,Literal(name));
  if(!(bool)ds.GetMethod("Read",f).Invoke(data,new object[]{path,false}))throw new Exception("native read rejected");
  trip.GetMethod("Evaluate",f).Invoke(null,new object[]{null});
  BindingSource rows=(BindingSource)survey.GetField("database",f).GetValue(surveyObj);
  using(StreamWriter w=new StreamWriter(Path.Combine(output,name+".tsv"))){
   for(int i=0;i<rows.Count;i++){
    object row=rows[i];station.GetField("idx",f).SetValue(row,-99);
    int dist=Field(row,"dist"),incl=(short)station.GetField("incl",f).GetValue(row);
    object az=station.GetMethod("RealAzimuth",f).Invoke(row,null);int real=(int)angle.GetField("value",f).GetValue(az);
    int[] v=Rect(dist,incl),xy=Rect(v[1],real);
    w.WriteLine("row\t{0}\t{1}\t{2}\t{3}\t{4}\t{5}",i,unchecked((short)real),xy[0],xy[1],v[0],v[1]);
   }
   survey.GetMethod("EvalData",f).Invoke(surveyObj,new object[]{0,rows.Count,true});
   for(int i=0;i<rows.Count;i++){
    object row=rows[i];if(Field(row,"idx")!=i)continue;
    int end=i+1;while(end<rows.Count && Field(rows[end],"idx")==-99)end++;
    int from=(int)station.GetField("from",f).GetValue(row).GetType().GetField("value",f).GetValue(station.GetField("from",f).GetValue(row));
    string kind=from==-1?"undefined_from":((int)station.GetField("to",f).GetValue(row).GetType().GetField("value",f).GetValue(station.GetField("to",f).GetValue(row))==-1?"splay":"leg");
    object[] fw={0f,0f,0f,false},rev={0f,0f,0f,true};station.GetMethod("GetAvDirection",f).Invoke(row,fw);station.GetMethod("GetAvDirection",f).Invoke(row,rev);
    w.WriteLine("group\t{0}\t{1}\t{2}\t{3}\t{4}\t{5}\t{6}\t{7:X8}\t{8:X8}\t{9:X8}\t{10:X8}\t{11:X8}\t{12:X8}",i,end,kind,Field(row,"dx"),Field(row,"dy"),Field(row,"dh"),Field(row,"dd"),Bits(fw[0]),Bits(fw[1]),Bits(fw[2]),Bits(rev[0]),Bits(rev[1]),Bits(rev[2]));
   }
  }
  Console.WriteLine(name+" rows="+rows.Count+" isolated_EvalData=True");
 }
 static int Main(string[] args){try{
  Thread.CurrentThread.CurrentCulture=CultureInfo.InvariantCulture;Thread.CurrentThread.CurrentUICulture=CultureInfo.InvariantCulture;
  Assembly a=Assembly.LoadFrom(args[0]);survey=a.GetType("PocketTopo.Survey");ds=a.GetType("PocketTopo.DataSet");drawing=a.GetType("PocketTopo.Drawing");mapping=a.GetType("PocketTopo.Mapping");trip=a.GetType("PocketTopo.Trip");station=a.GetType("PocketTopo.Station");angle=a.GetType("PocketTopo.Angle");
  Directory.CreateDirectory(args[1]);Rect(1,0);
  using(BinaryWriter w=new BinaryWriter(File.Create(Path.Combine(args[1],"sine-table.bin")))){foreach(int v in (int[])angle.GetField("sinTab",f).GetValue(null))w.Write(v);}
  using(StreamWriter w=new StreamWriter(Path.Combine(args[1],"rect.tsv"))){foreach(int len in new int[]{0,1,-1,5,-5,1000,-1000,16777216,-16777216,int.MaxValue,int.MinValue})foreach(int ang in new int[]{0,1,-1,8192,-8192,16383,16384,16385,-16384,-16385,32767,-32768,65536,65537,int.MaxValue,int.MinValue}){int[] v=Rect(len,ang);w.WriteLine("{0}\t{1}\t{2}\t{3}",len,ang,v[0],v[1]);}}
  foreach(string name in new string[]{"empty","repeats","boundaries","identities","flags","random"})Run(name,args[1]);
  return 0;
 }catch(Exception e){Console.Error.WriteLine(e);return 1;}}
}
