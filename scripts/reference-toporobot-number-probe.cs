// .NET 2 invariant numeric grammar used by the original Toporobot reader.
using System;using System.IO;using System.Globalization;using System.Text;
class ReferenceToporobotNumberProbe {
 static int Main(string[] args){using(StreamWriter w=new StreamWriter(args[0])){foreach(string s in new string[]{"1,,2","1,","1,000",",123","1.2,3","1e,2","1,2e3","+1,2","1.2e+3","  1,2  ","1,\t2","0x1p0","1_0","NaN","Infinity","-Infinity","\u200b1","1\u200b","\u00a01","1\u00a0","1 2",".1","1.","1e309","1e39","1e-999","+","","-0","1,2.3","1.2,","1e2,3","1e+2,3"}){
  string encoded=Convert.ToBase64String(Encoding.UTF8.GetBytes(s));foreach(int bits in new int[]{32,64}){try{double v=bits==32?(double)float.Parse(s,CultureInfo.InvariantCulture):double.Parse(s,CultureInfo.InvariantCulture);w.WriteLine(encoded+"\t"+bits+"\tok\t"+v.ToString("R",CultureInfo.InvariantCulture));}catch(Exception e){w.WriteLine(encoded+"\t"+bits+"\t"+e.GetType().Name+"\t-");}}}}
 Console.WriteLine("P12 numeric grammar capture complete");return 0;}
}
