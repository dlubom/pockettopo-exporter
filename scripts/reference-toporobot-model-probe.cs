// P12 original-program reader/writer oracle for existing-file normalization.
using System;using System.IO;using System.Reflection;using System.Text;using System.Globalization;using System.Threading;
class ReferenceToporobotModelProbe {
 static BindingFlags f=BindingFlags.Instance|BindingFlags.Static|BindingFlags.NonPublic|BindingFlags.Public;
 static Assembly a;static Type model;static Encoding mac;static string root;static StreamWriter cases;
 static void Run(string name,byte[] input){File.WriteAllBytes(Path.Combine(root,name+".input"),input);string status="ok";
  try{object m=model.GetMethod("Read",f).Invoke(null,new object[]{new StreamReader(new MemoryStream(input),mac)});using(StreamWriter w=new StreamWriter(Path.Combine(root,name+".text"),false,mac)){model.GetMethod("Write",f).Invoke(m,new object[]{w});}}
  catch(TargetInvocationException e){status=e.InnerException.GetType().Name;}
  cases.WriteLine(name+"\t"+status);Console.WriteLine(name+" status="+status);
 }
 static byte[] BOM(Encoding enc,string s){byte[] body=enc.GetBytes(s),bom=enc.GetPreamble(),b=new byte[bom.Length+body.Length];Array.Copy(bom,b,bom.Length);Array.Copy(body,0,b,bom.Length,body.Length);return b;}
 static int Main(string[] args){try{Thread.CurrentThread.CurrentCulture=CultureInfo.InvariantCulture;Thread.CurrentThread.CurrentUICulture=CultureInfo.InvariantCulture;
  a=Assembly.LoadFrom(args[0]);model=a.GetType("PocketTopo.Toporobot");mac=(Encoding)Activator.CreateInstance(a.GetType("PocketTopo.MacEncoding"),true);root=args[1];Directory.CreateDirectory(root);cases=new StreamWriter(Path.Combine(root,"roundtrip.tsv"));
  string seed=mac.GetString(File.ReadAllBytes(args[2]));Run("rich",mac.GetBytes(seed));
  Run("named-series",mac.GetBytes(seed.Replace("     1    -2   1   1   1 ","     1    -2   1   1   1 My cave é")));
  Run("year-2019",mac.GetBytes(seed.Replace("31121919", "31/12/19")));
  foreach(Encoding enc in new Encoding[]{new UTF8Encoding(true),new UnicodeEncoding(false,true),new UnicodeEncoding(true,true),new UTF32Encoding(false,true),new UTF32Encoding(true,true)})Run("bom-"+enc.CodePage,BOM(enc,seed));
  string[] lines=seed.Split(new string[]{"\r\n"},StringSplitOptions.None);
  string[] signed=(string[])lines.Clone();for(int i=0;i<signed.Length;i++){if(signed[i].StartsWith("    -1"))signed[i]=signed[i].Substring(0,24)+"   -1.00  -10.00"+signed[i].Substring(40);if(signed[i].StartsWith("    -2"))signed[i]=signed[i].Substring(0,25)+"-1"+signed[i].Substring(27);}Run("signed-fields",mac.GetBytes(string.Join("\r\n",signed)));
  Run("null-ref-name",mac.GetBytes("    -6     1   1   1   1 \r\n"));StringBuilder rich=new StringBuilder();bool code=false,series=false,station=false;
  foreach(string line in lines){if(line=="")continue;rich.Append(line).Append("\r\n");if(!code&&line.StartsWith("    -1")){rich.Append("(   -1     1   1   1   1 code comment\r\n");code=true;}
   if(!series&&line.StartsWith("     1    -1")){rich.Append("(    1    -1   1   1   1 series comment\r\n");series=true;}
   if(!station&&line.StartsWith("     1     0")){rich.Append("(    9 12345   1   1   1 mismatched header comment\r\n");station=true;}}
  Run("all-comments",mac.GetBytes(rich.ToString()));
  string sparse=seed.Replace("     1     0   1   1   1 ","     1   123   1   1   1 ").Replace("     1     1   1   1   1 ","     1    17   1   1   1 ");Run("sparse",mac.GetBytes(sparse));
  string[] fields={"1,000.00"," 0x1p+0 ","     NaN","    1e10"};foreach(string field in fields){string[] copy=(string[])lines.Clone();for(int i=0;i<copy.Length;i++)if(copy[i].StartsWith("     1     0")){copy[i]=copy[i].Substring(0,24)+field+copy[i].Substring(32);break;}Run("number-"+Array.IndexOf(fields,field),mac.GetBytes(string.Join("\r\n",copy)));}
  foreach(string line in new string[]{"[1 2  3]","[1 2 3 ]","[1 2 3]","[1 2 NaN]","[0x1p0 2 3]"})Run("inline-"+Array.IndexOf(new string[]{"[1 2  3]","[1 2 3 ]","[1 2 3]","[1 2 NaN]","[0x1p0 2 3]"},line),mac.GetBytes(seed+"(    9 12345   1   1   1 "+line+"\r\n"));
  cases.Close();Console.WriteLine("P12 model capture complete");return 0;
 }catch(Exception e){Console.Error.WriteLine(e);return 1;}}
}
