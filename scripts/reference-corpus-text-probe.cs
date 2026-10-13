// P14 real-corpus oracle. Only reflection is compiled; native code stays original.
using System; using System.IO; using System.Reflection; using System.Globalization;
using System.Threading; using System.Windows.Forms; using System.Drawing;
using System.Runtime.Serialization; using System.Text;
class ReferenceCorpusTextProbe {
 static BindingFlags f=BindingFlags.Public|BindingFlags.NonPublic|BindingFlags.Instance|BindingFlags.Static;
 static Assembly a;
 static object New(Type t,params object[] args){return Activator.CreateInstance(t,f,null,args,null);}
 static string Capture(string input,string output){
  Type sv=a.GetType("PocketTopo.Survey"),dr=a.GetType("PocketTopo.Drawing"),mp=a.GetType("PocketTopo.Mapping"),ds=a.GetType("PocketTopo.DataSet");
  a.GetType("PocketTopo.Trip").GetMethod("Reset",f).Invoke(null,null);
  object survey=New(sv,new BindingSource(),new BindingSource(),new EventHandler(delegate{}));
  object map=FormatterServices.GetUninitializedObject(mp);
  object plan=New(dr,map,false),side=New(dr,FormatterServices.GetUninitializedObject(mp),true);
  object data=New(ds,survey,map,plan,side);
  a.GetType("PocketTopo.XSection").GetField("survey",f).SetValue(null,survey);
  a.GetType("PocketTopo.Config").GetField("actual",f).SetValue(null,input);
  if(!(bool)ds.GetMethod("Read",f).Invoke(data,new object[]{input,false}))return "read-failed";
  a.GetType("PocketTopo.Exporter").GetMethod("WriteText",f).Invoke(null,new object[]{output,survey});
  // Expose exceptions swallowed by Exporter.WriteText; reject partial captures.
  using(MemoryStream expected=new MemoryStream()){
   using(StreamWriter writer=new StreamWriter(expected,new UTF8Encoding(false))){
    writer.Write(Path.GetFileNameWithoutExtension(input));writer.WriteLine("   (m, 360)");writer.WriteLine();
    sv.GetMethod("WriteText",f).Invoke(survey,new object[]{writer});writer.Flush();
    byte[] got=File.ReadAllBytes(output),want=expected.ToArray();
    if(got.Length!=want.Length)return "partial";
    for(int i=0;i<got.Length;i++)if(got[i]!=want[i])return "partial";
   }
  }
  return "complete";
 }
 static int Main(string[] args){try{
  Thread.CurrentThread.CurrentCulture=CultureInfo.InvariantCulture;Thread.CurrentThread.CurrentUICulture=CultureInfo.InvariantCulture;
  a=Assembly.LoadFrom(args[0]);Directory.CreateDirectory(args[3]);
  a.GetType("PocketTopo.Config").GetField("metric",f).SetValue(null,true);
  a.GetType("PocketTopo.Config").GetField("angle",f).SetValue(null,360);
  foreach(string color in new string[]{"black","gray","brown","blue","green","red","orange"})
   a.GetType("PocketTopo.MainForm").GetField(color,f).SetValue(null,new Pen(Color.Black));
  string[] paths=File.ReadAllLines(args[2],Encoding.UTF8);
  using(StreamWriter manifest=new StreamWriter(Path.Combine(args[3],"cases.tsv"),false,new UTF8Encoding(false))){
   for(int i=0;i<paths.Length;i++){
    string path=paths[i],output=i.ToString("D4")+".txt",status="";
    Thread thread=new Thread(delegate(){try{status=Capture(Path.Combine(args[1],path.Replace('/',Path.DirectorySeparatorChar)),Path.Combine(args[3],output));}
     catch(TargetInvocationException e){status=e.InnerException.GetType().Name;}catch(Exception e){status=e.GetType().Name;}});
    thread.Start();if(!thread.Join(10000)){thread.Abort();thread.Join();status="timeout";}
    manifest.WriteLine(path+"\t"+status+"\t"+output);manifest.Flush();Console.WriteLine(i+" "+status+" "+path);
   }
  }
  Console.WriteLine("P14 native corpus text capture complete count="+paths.Length);return 0;
 }catch(Exception e){Console.Error.WriteLine(e);return 1;}}
}
