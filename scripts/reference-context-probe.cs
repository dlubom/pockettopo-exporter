// P08: literal tables, original assembly Read/SetBoundary/Update, no rebuilt logic.
using System; using System.IO; using System.Reflection; using System.Globalization;
using System.Threading; using System.Windows.Forms; using System.Runtime.Serialization;
class ReferenceContextProbe {
 static BindingFlags f=BindingFlags.Public|BindingFlags.NonPublic|BindingFlags.Instance|BindingFlags.Static;
 static Type sv,ds,st,rf,tr,dr,mp; static int boundary,rb;
 static object Field(object o,string n){return o.GetType().GetField(n,f).GetValue(o);}
 static int I(object o,string n){return (int)Field(o,n);} static long L(object o,string n){return (long)Field(o,n);}
 static int ID(object o,string n){return (int)Field(Field(o,n),"value");}
 static string Key(object o){if(o==null)return "none";int i=I(o,"idx");bool r=rf.IsInstanceOfType(o);int b=r?rb:boundary;return (i<b?"template:":"")+(r?"reference:":"group:")+(i<b?i:i-b);}
 // shots: from,to,length,azimuth,inclination,flags,trip; refs: id,e,n,h.
 static byte[] Top(int[,] shots,int[,] refs,int decl,bool tablesOnly){using(MemoryStream m=new MemoryStream()){BinaryWriter w=new BinaryWriter(m);w.Write(new byte[]{84,111,112,3});w.Write(decl==99999?0:1);if(decl!=99999){w.Write(0L);w.Write("");w.Write((short)decl);}w.Write(shots.GetLength(0));for(int i=0;i<shots.GetLength(0);i++){w.Write((uint)shots[i,0]);w.Write((uint)shots[i,1]);w.Write(shots[i,2]);w.Write((short)shots[i,3]);w.Write((short)shots[i,4]);w.Write((byte)shots[i,5]);w.Write((byte)0);w.Write((short)shots[i,6]);}w.Write(refs.GetLength(0));for(int i=0;i<refs.GetLength(0);i++){w.Write((uint)refs[i,0]);w.Write((long)refs[i,1]);w.Write((long)refs[i,2]);w.Write(refs[i,3]);w.Write("");}if(!tablesOnly){for(int j=0;j<2;j++){w.Write(0);w.Write(0);w.Write(500);}w.Write((byte)0);w.Write(0);w.Write(0);w.Write(500);w.Write((byte)0);w.Write(0);}else w.Write((byte)255);return m.ToArray();}}
 static int[,] empty=new int[0,7], noRefs=new int[0,4];
 static void Write(string dir,string name,int[,] shots,int[,] refs,int decl,bool tables){File.WriteAllBytes(Path.Combine(dir,name),Top(shots,refs,decl,tables));}
 static void Case(string root,string name){string dir=Path.Combine(root,name);Directory.CreateDirectory(dir);
  int[,] current={{65537,65538,1000,0,0,0,-1}};int[,] refs=noRefs;int decl=99999;
  if(name=="empty"||name=="emptytemplate"||name=="emptytemplate-current"){if(name!="emptytemplate-current")current=empty;if(name!="empty")Write(dir,"a.top",empty,noRefs,99999,true);}
  else if(name=="reference-base"){Write(dir,"a.top",empty,new int[,]{{65536,100000,200000,300000},{65537,101000,201000,299000}},99999,true);refs=new int[,]{{65537,105000,206000,307000}};}
  else if(name=="actual-priority"){Write(dir,"a.top",new int[,]{{65536,65537,1000,0,0,0,-1}},noRefs,99999,true);refs=new int[,]{{65537,5000,6000,7000}};current=new int[,]{{65539,65537,1000,16384,0,0,-1},{65537,65538,1000,0,0,0,-1}};}
  else if(name=="crossfile"||name=="declination"){Write(dir,"b.top",new int[,]{{65537,65536,3000,-32768,0,1,name=="declination"?0:-1}},noRefs,name=="declination"?8192:99999,true);Write(dir,"a.top",new int[,]{{65536,65537,1000,0,0,0,name=="declination"?0:-1}},noRefs,name=="declination"?4096:99999,true);}
  else if(name=="template-loop"){Write(dir,"a.top",new int[,]{{65536,65537,1000,0,0,0,-1},{65537,65538,1000,16384,0,0,-1},{65538,65536,1400,-24576,0,0,-1}},noRefs,99999,true);current=new int[,]{{65536,65539,1000,0,0,0,-1},{65539,65538,1000,16384,0,0,-1}};}
  else if(name=="synthetic-before-reference"){Write(dir,"a.top",new int[,]{{65536,65537,1000,0,0,0,-1}},noRefs,99999,true);refs=new int[,]{{65538,9000,8000,7000}};}
  else if(name=="projection"){Write(dir,"a.top",new int[,]{{65536,65537,1000,8192,0,1,-1}},noRefs,99999,true);current=new int[,]{{65537,65538,1000,16384,0,16,-1},{65537,unchecked((int)0x80000000),500,0,0,0,-1}};}
  else if(name=="inclined-loop"){Write(dir,"a.top",new int[,]{{65536,65537,1000,0,8192,0,-1}},noRefs,99999,true);current=new int[,]{{65537,65538,1000,16384,4096,0,-1},{65538,65539,1000,0,0,0,-1},{65539,65537,1400,-24576,-4096,0,-1}};}
  else {Write(dir,"a.top",new int[,]{{65536,65537,1000,0,0,0,-1}},noRefs,99999,true);if(name=="boundary")current=new int[,]{{65536,65537,3000,0,0,0,-1}};}
  Write(dir,"current.top",current,refs,decl,false);
  foreach(bool close in new bool[]{false,true}){
   tr.GetMethod("Reset",f).Invoke(null,null);object survey=Activator.CreateInstance(sv,f,null,new object[]{new BindingSource(),new BindingSource(),new EventHandler(delegate{})},null);object map=FormatterServices.GetUninitializedObject(mp);object data=Activator.CreateInstance(ds,f,null,new object[]{survey,map,Activator.CreateInstance(dr,f,null,new object[]{map,false},null),Activator.CreateInstance(dr,f,null,new object[]{FormatterServices.GetUninitializedObject(mp),true},null)},null);
   // Set the backing field directly: Config.actualFile's setter writes registry.
   sv.Assembly.GetType("PocketTopo.Config").GetField("actual",f).SetValue(null,Path.Combine(dir,"current.top"));
   if(!(bool)ds.GetMethod("Load",f).Invoke(data,new object[]{Path.Combine(dir,"current.top")}))throw new Exception("native Load");
   boundary=I(survey,"dataBoundary");rb=I(survey,"refBoundary");
   using(StreamWriter w=new StreamWriter(Path.Combine(root,name+(close?"-closed":"")+".tsv"))){try{sv.GetMethod("CloseLoops",f).Invoke(survey,new object[]{close,false});}catch(TargetInvocationException e){w.WriteLine("error\t"+e.InnerException.GetType().Name);continue;}
    w.WriteLine("base\t"+Key(Field(survey,"refBase")));BindingSource rs=(BindingSource)Field(survey,"references"), rows=(BindingSource)Field(survey,"database");
    // Synthetic roots are not in the reference table. Emit in phase order.
    for(int phase=0;phase<2;phase++){for(int i=phase==0?0:rb;i<(phase==0?rb:rs.Count);i++)if(ID(rs[i],"num")!=-1)Node(w,rs[i]);object tail=Field(survey,phase==0?"template":"actual");if(tail!=null){while(Field(tail,"next")!=null)tail=Field(tail,"next");if(I(tail,"idx")==-1)Node(w,tail);}for(int i=phase==0?0:boundary;i<(phase==0?boundary:rows.Count);i++)if(I(rows[i],"idx")==i)Node(w,rows[i]);}
    object loop=Field(survey,"loops");while(loop!=null){object a=Field(loop,"end1"),b=Field(loop,"end2");w.WriteLine("loop\t{0}\t{1}\t{2}\t{3}\t{4}\t{5}",Key(a),Key(b),unchecked(I(a,"x")-I(b,"x")),unchecked(I(a,"y")-I(b,"y")),unchecked(I(a,"h")-I(b,"h")),unchecked(I(a,"d")-I(b,"d")));loop=Field(loop,"next");}
   }
  }Console.WriteLine(name+" OK");
 }
 static void Node(StreamWriter w,object n){string status=rf.IsInstanceOfType(n)?"placed":ID(n,"from")==-1?"undefined_from":Field(n,"start")!=null?"placed":I(n,"dist")==0&&ID(n,"to")==-1?"zero_splay":"unplaced";w.WriteLine("{0}\t{1}\t{2}\t{3}\t{4}\t{5}\t{6}\t{7}\t{8}\t{9}\t{10}\t{11}\t{12}\t{13}\t{14}",Key(n),status,ID(n,"num"),Key(Field(n,"start")),I(n,"x"),I(n,"y"),I(n,"h"),I(n,"d"),I(n,"dx"),I(n,"dy"),I(n,"dh"),I(n,"dd"),L(n,"ex"),L(n,"ey"),L(n,"eh"));}
 static void Pattern(string root){string dir=Path.Combine(root,"pattern");Directory.CreateDirectory(dir);foreach(string n in new string[]{"a.top","b.TOP","c.topx","d.top.bak"})File.WriteAllText(Path.Combine(dir,n),"");Directory.CreateDirectory(Path.Combine(dir,"nested.top"));string[] names=Directory.GetFiles(dir,"*.top");Array.Sort(names);using(StreamWriter w=new StreamWriter(Path.Combine(root,"native-pattern.tsv")))foreach(string n in names)w.WriteLine(Path.GetFileName(n));}
 static int Main(string[] args){try{Thread.CurrentThread.CurrentCulture=CultureInfo.InvariantCulture;Assembly a=Assembly.LoadFrom(args[0]);sv=a.GetType("PocketTopo.Survey");ds=a.GetType("PocketTopo.DataSet");st=a.GetType("PocketTopo.Station");rf=a.GetType("PocketTopo.Reference");tr=a.GetType("PocketTopo.Trip");dr=a.GetType("PocketTopo.Drawing");mp=a.GetType("PocketTopo.Mapping");Directory.CreateDirectory(args[1]);Pattern(args[1]);foreach(string n in new string[]{"empty","emptytemplate","emptytemplate-current","attach","crossfile","declination","boundary","reference-base","actual-priority","template-loop","synthetic-before-reference","projection","inclined-loop"})Case(args[1],n);return 0;}catch(Exception e){Console.Error.WriteLine(e);return 1;}}
}
