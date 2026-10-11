// P05 independent fixtures and goldens: invoke the original 1.372 assembly.
// No recovered implementation is compiled; DataSet.Read is isolated, not Load.
using System;
using System.IO;
using System.Reflection;
using System.Globalization;
using System.Threading;
using System.Windows.Forms;
using System.Runtime.Serialization;
class ReferenceTextProbe {
 static BindingFlags f=BindingFlags.Public|BindingFlags.NonPublic|BindingFlags.Instance|BindingFlags.Static;
 static Assembly a; static Type survey, ds, drawing, mapping, trip, station, config;
 static object Dataset(out object s){
  trip.GetMethod("Reset",f).Invoke(null,null);
  s=Activator.CreateInstance(survey,f,null,new object[]{new BindingSource(),new BindingSource(),new EventHandler(delegate{})},null);
  object map=FormatterServices.GetUninitializedObject(mapping);
  object plan=Activator.CreateInstance(drawing,f,null,new object[]{map,false},null);
  object side=Activator.CreateInstance(drawing,f,null,new object[]{FormatterServices.GetUninitializedObject(mapping),true},null);
  return Activator.CreateInstance(ds,f,null,new object[]{s,map,plan,side},null);
 }
 static void Shot(BinaryWriter w,uint from,uint to,int dist,short az,short inc,byte flags,short idx,string comment){
  w.Write(from);w.Write(to);w.Write(dist);w.Write(az);w.Write(inc);w.Write(flags);w.Write((byte)255);w.Write(idx);
  if((flags&2)!=0)w.Write(comment);
 }
 static void Ref(BinaryWriter w,uint id,long e,long n,int h,string comment){w.Write(id);w.Write(e);w.Write(n);w.Write(h);w.Write(comment);}
 static byte[] Literal(string name){
  MemoryStream stream=new MemoryStream();BinaryWriter w=new BinaryWriter(stream);
  w.Write(new byte[]{84,111,112,3});
  if(name=="empty"){w.Write(0);w.Write(0);w.Write(0);}
  else if(name=="numbers") {
   w.Write(0);w.Write(512);ulong state=0x123456789abcdef0UL;
   for(int i=0;i<512;i++){state=unchecked(state*6364136223846793005UL+1442695040888963407UL);Shot(w,0x80000000,0x80000000,unchecked((int)state),unchecked((short)(state>>32)),unchecked((short)(state>>48)),0,-1,"");}
   w.Write(512);
   for(int i=0;i<512;i++){state=unchecked(state*6364136223846793005UL+1442695040888963407UL);long v=unchecked((long)state);Ref(w,0x80000000,v,unchecked(-v),unchecked((int)(state>>16)),"");}
  }
  else {
   w.Write(3);
   w.Write(new DateTime(2024,2,29,23,59,59).Ticks);w.Write("Zażółć \"quote\" \\path\r\nnext\nline");w.Write((short)1234);
   w.Write(0L);w.Write("");w.Write(short.MinValue);
   w.Write(DateTime.MaxValue.Ticks);w.Write("終 🦇");w.Write((short)-16384);
   if(name=="rounding"){
    w.Write(22);
    int[] values=new int[]{0,5,-5,15,-15,125,-125,1005,-1005,1015,-1015,2005,-2005,2147483647,-2147483647,int.MinValue,999999999,-999999999,123456789,-123456789,1,-1};
    foreach(int v in values)Shot(w,0x80000000,0x800fffff,v,(short)(v&65535),(short)(v&65535),2,32767,"");
   } else if(name=="flags"){
    w.Write(256);for(int i=0;i<256;i++)Shot(w,0x80000000,(uint)(i+65536),1005,(short)(i*257),(short)(-i*127),(byte)i,(short)(i%5-2),"x\"\\\r\n終");
   } else if(name=="update-error"){
    w.Write(1);Shot(w,65536,65537,1000,0,0,16,0,"");
   } else {
    w.Write(11);
    Shot(w,65536,65537,1000,100,200,2,0,"first");
    Shot(w,65536,65537,2000,16384,300,3,1,"repeat");
    Shot(w,65537,65536,3000,-32768,-400,10,2,"reverse");
    Shot(w,65537,65538,4321,-12345,-1000,1,-1,"");
    Shot(w,65538,0x80000000,2345,32767,12345,2,-2,"");
    Shot(w,65538,65539,9876,1000,-1000,16,0,"");
    Shot(w,0x80000000,0x800fffff,int.MinValue,-1,-32768,255,32767,"orphan");
    Shot(w,0x800fffff,0x800ffffe,-5,0,0,8,3,"");
    Shot(w,65536,65537,5000,16384,0,0,0,"");
    Shot(w,0x800fff00,0x80100000,1005,1,-1,0,-1,"");
    Shot(w,65539,65536,0,0,0,4,0,"");
   }
   w.Write(5);
   Ref(w,65536,1005,-1005,15,"ref\"\\\r\nline");
   Ref(w,0x80000000,long.MinValue,long.MinValue,int.MinValue,"");
   Ref(w,0x800fffff,long.MaxValue,-long.MaxValue,int.MaxValue,"extremes");
   Ref(w,0x800fff00,123456789012345L,-123456789012345L,-125,"");
   Ref(w,65536,5,-5,125,"duplicate");
  }
  for(int i=0;i<2;i++){w.Write(0);w.Write(0);w.Write(500);}w.Write((byte)0);
  w.Write(0);w.Write(0);w.Write(500);w.Write((byte)0);w.Write(0);return stream.ToArray();
 }
 static string WrittenFields(object s){
  StringWriter w=new StringWriter(CultureInfo.InvariantCulture);trip.GetMethod("WriteText",f).Invoke(null,new object[]{w});w.WriteLine();
  BindingSource refs=(BindingSource)survey.GetField("references",f).GetValue(s);
  foreach(object r in refs)r.GetType().GetMethod("WriteText",f,null,new Type[]{typeof(TextWriter)},null).Invoke(r,new object[]{w});
  w.WriteLine();BindingSource rows=(BindingSource)survey.GetField("database",f).GetValue(s);
  foreach(object row in rows)station.GetMethod("WriteText",f).Invoke(row,new object[]{w,0});return w.ToString();
 }
 static void Run(string name,byte[] bytes,string output){
  string path=Path.Combine(output,name+".top");File.WriteAllBytes(path,bytes);
  object s;object data=Dataset(out s);if(!(bool)ds.GetMethod("Read",f).Invoke(data,new object[]{path,false}))throw new Exception("native read rejected "+name);
  string before=WrittenFields(s);bool updated=true;
  try{survey.GetMethod("Update",f).Invoke(s,null);}catch(TargetInvocationException ex){updated=false;Console.WriteLine(name+" Update error="+ex.InnerException.GetType().Name);}
  string after=WrittenFields(s);if(before!=after)throw new Exception("Update changes text fields: "+name);
  Console.WriteLine(name+" Update="+updated+" text_fields_unchanged="+(before==after)+" rows="+((BindingSource)survey.GetField("database",f).GetValue(s)).Count);
  BindingSource rows=(BindingSource)survey.GetField("database",f).GetValue(s);
  if(updated && rows.Count>0)Console.WriteLine(name+" first_derived_dx="+station.GetField("dx",f).GetValue(rows[0])+" first_derived_dy="+station.GetField("dy",f).GetValue(rows[0]));
  config.GetField("actual",f).SetValue(null,path);
  string export=Path.Combine(output,name+".txt");a.GetType("PocketTopo.Exporter").GetMethod("WriteText",f).Invoke(null,new object[]{export,s});
  if(!File.Exists(export))throw new Exception("native output missing");
  Console.WriteLine(name+" bytes="+File.ReadAllBytes(export).Length);
 }
 static int Main(string[] args){try{
  Thread.CurrentThread.CurrentCulture=CultureInfo.InvariantCulture;Thread.CurrentThread.CurrentUICulture=CultureInfo.InvariantCulture;
  a=Assembly.LoadFrom(args[0]);survey=a.GetType("PocketTopo.Survey");ds=a.GetType("PocketTopo.DataSet");drawing=a.GetType("PocketTopo.Drawing");mapping=a.GetType("PocketTopo.Mapping");trip=a.GetType("PocketTopo.Trip");station=a.GetType("PocketTopo.Station");config=a.GetType("PocketTopo.Config");
  config.GetField("metric",f).SetValue(null,true);config.GetField("angle",f).SetValue(null,360);
  string output=args[1];Directory.CreateDirectory(output);
  foreach(string name in new string[]{"empty","mixed","rounding","flags","numbers","update-error"})Run(name,Literal(name),output);
  foreach(string name in new string[]{"api-references","api-drawings"})Run(name,File.ReadAllBytes(Path.Combine(args[2],name+".top")),output);
  return 0;
 }catch(Exception ex){Console.Error.WriteLine(ex);return 1;}}
}
