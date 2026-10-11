// P07: literal TOP streams and reflection into the original 1.372 assembly.
// No recompiled PocketTopo logic, templates or directory loading.
using System;
using System.IO;
using System.Reflection;
using System.Globalization;
using System.Threading;
using System.Windows.Forms;
using System.Runtime.Serialization;
class ReferenceGeometryProbe {
 static BindingFlags f=BindingFlags.Public|BindingFlags.NonPublic|BindingFlags.Instance|BindingFlags.Static;
 static Type survey,ds,drawing,mapping,trip,station,angle,reference,projection;
 static StreamWriter links;
 static int I(object s,string n){return (int)station.GetField(n,f).GetValue(s);}
 static long L(object s,string n){return (long)station.GetField(n,f).GetValue(s);}
 static int ID(object s,string n){object id=station.GetField(n,f).GetValue(s);return (int)id.GetType().GetField("value",f).GetValue(id);}
 static void Shot(BinaryWriter w,uint a,uint b,int len,int az,int inc,int flags){w.Write(a);w.Write(b);w.Write(len);w.Write((short)az);w.Write((short)inc);w.Write((byte)flags);w.Write((byte)0);w.Write((short)-1);if((flags&2)!=0)w.Write("source");}
 static void Ref(BinaryWriter w,uint a,long e,long n,int h){w.Write(a);w.Write(e);w.Write(n);w.Write(h);w.Write("fix");}
 static byte[] Literal(string name){
  MemoryStream sh=new MemoryStream(),rf=new MemoryStream();BinaryWriter s=new BinaryWriter(sh),r=new BinaryWriter(rf);int ns=0,nr=0;
  if(name=="chain"||name=="projected"||name=="splays"){
   Shot(s,65536,0x80000000,500,8192,0,0);ns++;
   Shot(s,65536,65537,1000,0,0,1);ns++;
   Shot(s,65537,65538,2000,8192,4096,name=="projected"?16:0);ns++;
   Shot(s,65537,0x800fffff,800,16384,8192,0);ns++;
   Shot(s,65538,65539,500,16384,0,0);ns++;
   Shot(s,65537,0x80000000,700,-8192,-4096,1);ns++;
   if(name=="splays"){Shot(s,65537,65540,1000,-16384,0,1);Shot(s,65537,0x80000000,0,0,0,0);ns+=2;}
  }else if(name=="references"){
   Ref(r,0x80000000,999,999,999);Ref(r,65536,long.MaxValue,long.MinValue,int.MaxValue);Ref(r,65537,long.MinValue,long.MaxValue,int.MinValue);Ref(r,65536,123,456,789);Ref(r,0x800ffffe,-999,888,-777);nr=5;
   Shot(s,65536,65538,1000,16384,0,0);Shot(s,65537,65539,1000,0,0,0);Shot(s,0x800ffffe,65540,1000,0,0,0);ns=3;
  }else if(name=="delayed"){
   Shot(s,65539,65538,1000,16384,0,0);Shot(s,65538,65537,1000,16384,0,0);Shot(s,65536,65537,1000,0,0,0);Shot(s,65550,65551,1000,0,0,0);Shot(s,65539,0x80000000,500,0,0,0);Shot(s,0x80000000,65536,1,0,0,0);ns=6;
  }else if(name=="loop"||name=="intersecting"||name=="fixedloop"){
   if(name=="fixedloop"){Ref(r,65536,100000,200000,300000);Ref(r,65539,102000,200000,300000);nr=2;}
   Shot(s,65536,65537,1000,0,0,0);Shot(s,65537,65538,1000,16384,0,0);Shot(s,65538,65536,1400,-24576,0,0);ns=3;
   if(name=="intersecting"||name=="fixedloop"){Shot(s,65537,65539,1600,8192,1000,0);Shot(s,65539,65538,1000,24576,-1000,0);ns+=2;}
  }else if(name=="zero"){
   Shot(s,65536,65537,0,0,0,0);Shot(s,65537,65536,0,16384,0,0);Shot(s,65536,65536,0,0,0,0);Shot(s,65536,0x80000000,1000,0,0,0);ns=4;
  }else if(name=="failure"){
   Shot(s,65536,65537,1000,0,0,16);ns=1;
  }else if(name=="phase-priority"){
   Ref(r,65536,0,0,0);nr=1;
   Shot(s,65539,65538,1000,8192,4096,0);Shot(s,65539,65537,1000,0,0,0);Shot(s,65536,65537,1000,0,0,0);Shot(s,65536,65538,1000,16384,0,0);Shot(s,65550,65539,500,8192,1000,1);ns=5;
  }else if(name=="priority"){
   Ref(r,65537,0,0,0);nr=1;
   Shot(s,65538,65537,1000,8192,4096,0);Shot(s,65536,65538,2000,16384,0,0);Shot(s,65537,65536,1000,0,0,0);ns=3;
  }else if(name=="common-cleanup"){
   int[,] pairs={{0,1},{1,2},{1,3},{3,4},{0,5},{4,5},{2,4}};
   for(int i=0;i<7;i++)Shot(s,(uint)(65536+pairs[i,0]),(uint)(65536+pairs[i,1]),1000,0,0,0);ns=7;
  }else if(name=="reversed"){
   Ref(r,65537,0,0,0);nr=1;
   Shot(s,65536,65537,1000,8192,4096,0);Shot(s,65536,0x80000000,500,16384,-8192,0);ns=2;
  }else if(name=="ordering"){
   Ref(r,65536,0,0,0);nr=1;
   Shot(s,65537,65539,1000,0,0,0);Shot(s,65538,65539,1000,16384,0,0);Shot(s,65536,65537,1000,0,0,0);Shot(s,65536,65538,1000,16384,0,0);ns=4;
  }else if(name=="repeatloop"){
   Shot(s,65536,65537,900,0,0,0);Shot(s,65537,65536,1100,-32768,0,1);Shot(s,65537,65538,1100,16384,0,0);Shot(s,65538,65536,1300,-24576,0,0);ns=4;
  }else if(name=="projection-chain"){
   Shot(s,65536,65537,1000,8192,0,0);Shot(s,65537,65538,0,0,0,0);Shot(s,65538,65539,2000,16384,0,16);Shot(s,65539,65540,1000,24576,0,16);Shot(s,65540,0x80000000,700,0,0,0);Shot(s,65540,65541,1000,0,0,0);ns=6;
  }else if(name=="arithmetic"){
   Shot(s,65536,65537,1000,0,0,0);Shot(s,65537,65538,int.MinValue,0,0,16);ns=2;
  }else if(name=="undefined-only"){
   Shot(s,0x80000000,65536,1000,0,0,0);Shot(s,0x800fffff,65537,1000,0,0,0);ns=2;
  }else if(name=="markfail"){
   Shot(s,65536,65537,1000,0,0,32);Shot(s,65537,65538,1000,16384,0,0);Shot(s,65538,65536,1400,-24576,0,0);ns=3;
  }else if(name=="identities"){
   Ref(r,0x80200000,0,0,0);Ref(r,0x000fffff,1000,1000,1000);nr=2;
   Shot(s,0x000fffff,65536,1000,0,0,0);Shot(s,65536,65537,1000,0,0,4);Shot(s,65536,65537,2000,0,0,0);Shot(s,65537,0x800ffffe,1000,16384,0,0);ns=4;
  }
  MemoryStream b=new MemoryStream();BinaryWriter w=new BinaryWriter(b);w.Write(new byte[]{84,111,112,3});w.Write(0);w.Write(ns);w.Write(sh.ToArray());w.Write(nr);w.Write(rf.ToArray());for(int i=0;i<2;i++){w.Write(0);w.Write(0);w.Write(500);}w.Write((byte)0);w.Write(0);w.Write(0);w.Write(500);w.Write((byte)0);w.Write(0);return b.ToArray();
 }
 static string Key(object n){if(n==null)return "none";return (reference.IsInstanceOfType(n)?"reference":"group")+":"+I(n,"idx");}
 static void WriteNode(StreamWriter w,object n,string status){object start=station.GetField("start",f).GetValue(n);w.WriteLine("{0}\t{1}\t{2}\t{3}\t{4}\t{5}\t{6}\t{7}\t{8}\t{9}\t{10}\t{11}\t{12}\t{13}\t{14}",Key(n),status,ID(n,"num"),Key(start),I(n,"x"),I(n,"y"),I(n,"h"),I(n,"d"),I(n,"dx"),I(n,"dy"),I(n,"dh"),I(n,"dd"),L(n,"ex"),L(n,"ey"),L(n,"eh"));}
 static void Run(string name,string output,bool close){
  trip.GetMethod("Reset",f).Invoke(null,null);object sv=Activator.CreateInstance(survey,f,null,new object[]{new BindingSource(),new BindingSource(),new EventHandler(delegate{})},null);object map=FormatterServices.GetUninitializedObject(mapping);object data=Activator.CreateInstance(ds,f,null,new object[]{sv,map,Activator.CreateInstance(drawing,f,null,new object[]{map,false},null),Activator.CreateInstance(drawing,f,null,new object[]{FormatterServices.GetUninitializedObject(mapping),true},null)},null);
  string path=Path.Combine(output,name+".top");File.WriteAllBytes(path,Literal(name));if(!(bool)ds.GetMethod("Read",f).Invoke(data,new object[]{path,false}))throw new Exception("native rejected");
  using(StreamWriter w=new StreamWriter(Path.Combine(output,name+(close?"-closed":"")+".tsv"))){
   try{survey.GetMethod("CloseLoops",f).Invoke(sv,new object[]{close,false});}catch(TargetInvocationException e){w.WriteLine("error\t"+e.InnerException.GetType().Name);Console.WriteLine(name+" close="+close+" "+e.InnerException.GetType().Name);return;}
   object b=survey.GetField("refBase",f).GetValue(sv);w.WriteLine("base\t"+Key(b));
   BindingSource refs=(BindingSource)survey.GetField("references",f).GetValue(sv),rows=(BindingSource)survey.GetField("database",f).GetValue(sv);
   for(int i=0;i<refs.Count;i++){object n=refs[i];if(ID(n,"num")!=-1)WriteNode(w,n,"placed");}
   object actual=survey.GetField("actual",f).GetValue(sv);if(actual!=null){object tail=actual;while(station.GetField("next",f).GetValue(tail)!=null)tail=station.GetField("next",f).GetValue(tail);if(I(tail,"idx")==-1)WriteNode(w,tail,"placed");}
   for(int i=0;i<rows.Count;i++){object n=rows[i];if(I(n,"idx")!=i)continue;string status=ID(n,"from")==-1?"undefined_from":(station.GetField("start",f).GetValue(n)!=null?"placed":(I(n,"dist")==0&&ID(n,"to")==-1?"zero_splay":"unplaced"));WriteNode(w,n,status);}
   object loops=survey.GetField("loops",f).GetValue(sv);int count=0;while(loops!=null){object end1=loops.GetType().GetField("end1",f).GetValue(loops),end2=loops.GetType().GetField("end2",f).GetValue(loops);links.WriteLine("{0}\t{1}\t{2}\t{3}\t{4}\t{5}\t{6}\t{7}",name,close,Key(end1),Key(end2),unchecked(I(end1,"x")-I(end2,"x")),unchecked(I(end1,"y")-I(end2,"y")),unchecked(I(end1,"h")-I(end2,"h")),unchecked(I(end1,"d")-I(end2,"d")));count++;loops=loops.GetType().GetField("next",f).GetValue(loops);}w.WriteLine("loops\t"+count);
  }Console.WriteLine(name+" close="+close+" OK");
 }
 static int Main(string[] args){try{
  Thread.CurrentThread.CurrentCulture=CultureInfo.InvariantCulture;Assembly a=Assembly.LoadFrom(args[0]);survey=a.GetType("PocketTopo.Survey");ds=a.GetType("PocketTopo.DataSet");drawing=a.GetType("PocketTopo.Drawing");mapping=a.GetType("PocketTopo.Mapping");trip=a.GetType("PocketTopo.Trip");station=a.GetType("PocketTopo.Station");angle=a.GetType("PocketTopo.Angle");reference=a.GetType("PocketTopo.Reference");projection=a.GetType("PocketTopo.Projection");Directory.CreateDirectory(args[1]);
  using(StreamWriter w=new StreamWriter(Path.Combine(args[1],"projection.tsv"))){foreach(int a1 in new int[]{int.MinValue,-32768,-8192,0,8192,16384,32767})foreach(int a2 in new int[]{int.MinValue,-32768,-8192,0,8192,16384,32767})foreach(int az in new int[]{-32768,-16384,-8192,-1,0,1,4096,8192,16384,32767})for(int flips=0;flips<4;flips++){object[] v={Activator.CreateInstance(angle,new object[]{a1}),Activator.CreateInstance(angle,new object[]{a2}),12345,Activator.CreateInstance(angle,new object[]{az}),(flips&1)!=0,(flips&2)!=0};object result=projection.GetMethod("SideProjection",f).Invoke(null,v);w.WriteLine("{0}\t{1}\t{2}\t{3}\t{4}",a1,a2,az,flips,result);}}
  links=new StreamWriter(Path.Combine(args[1],"loops.tsv"));
  foreach(string name in new string[]{"empty","chain","projected","splays","references","delayed","loop","intersecting","fixedloop","zero","failure","identities","repeatloop","projection-chain","arithmetic","undefined-only","markfail","ordering","reversed","common-cleanup","priority","phase-priority"}){Run(name,args[1],false);Run(name,args[1],true);}links.Close();return 0;
 }catch(Exception e){Console.Error.WriteLine(e);return 1;}}
}
