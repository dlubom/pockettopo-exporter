// P15: original assembly only; independently reads the authored P15 TOP fixtures. Empty placement dictionaries isolate EvalData's
// grouping from SetupShot/SetupXsect; no Update, templates or closure are run.
using System;
using System.IO;
using System.Reflection;
using System.Globalization;
using System.Threading;
using System.Windows.Forms;
using System.Runtime.Serialization;
class ReferenceSRVProbe {
 static BindingFlags f=BindingFlags.Public|BindingFlags.NonPublic|BindingFlags.Instance|BindingFlags.Static;
 static Type survey,ds,drawing,mapping,trip,station,angle;
 static object Angle(int value){return Activator.CreateInstance(angle,new object[]{value});}
 static int[] Rect(int len,int value){object[] args={len,Angle(value),0,0};angle.GetMethod("ToRect",f).Invoke(null,args);return new int[]{(int)args[2],(int)args[3]};}
 static int Field(object s,string name){return (int)station.GetField(name,f).GetValue(s);}
 static uint Bits(object v){return BitConverter.ToUInt32(BitConverter.GetBytes((float)v),0);}
 static void Run(string name,string output){
  trip.GetMethod("Reset",f).Invoke(null,null);
  object surveyObj=Activator.CreateInstance(survey,f,null,new object[]{new BindingSource(),new BindingSource(),new EventHandler(delegate{})},null);
  object map=FormatterServices.GetUninitializedObject(mapping);
  object data=Activator.CreateInstance(ds,f,null,new object[]{surveyObj,map,Activator.CreateInstance(drawing,f,null,new object[]{map,false},null),Activator.CreateInstance(drawing,f,null,new object[]{FormatterServices.GetUninitializedObject(mapping),true},null)},null);
  string path=name;name=Path.GetFileNameWithoutExtension(path);
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
  for(int i=2;i<args.Length;i++)Run(args[i],args[1]);
  return 0;
 }catch(Exception e){Console.Error.WriteLine(e);return 1;}}
}
