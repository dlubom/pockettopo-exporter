// P13 finite FindFace identity oracle from the unchanged original assembly.
using System;using System.IO;using System.Reflection;using System.Globalization;
class ReferenceDXF3DFaceProbe{
 static BindingFlags f=BindingFlags.Public|BindingFlags.NonPublic|BindingFlags.Instance|BindingFlags.Static;
 static Type td,pt,vt;
 static object Point(int[] p){return Activator.CreateInstance(pt,f,null,new object[]{p[0],p[1],p[2]},null);}
 static string Values(int[] p){return p[0]+","+p[1]+","+p[2];}
 static int[] XYZ(object p){return new int[]{(int)p.GetType().GetField("x",f).GetValue(p),(int)p.GetType().GetField("y",f).GetValue(p),(int)p.GetType().GetField("z",f).GetValue(p)};}
 public static int Main(string[] args){try{
  System.Threading.Thread.CurrentThread.CurrentCulture=CultureInfo.InvariantCulture;
  Assembly a=Assembly.LoadFrom(args[0]);td=a.GetType("PocketTopo.ThreeD");pt=td.GetNestedType("Point",f);vt=td.GetNestedType("Vector",f);
  using(StreamWriter w=new StreamWriter(args[1])){
   w.WriteLine("name\ta\tb\tpoints\tprev\tselected\tnext_prev");
   int[][][] pairs={new int[][]{new int[]{0,16,0},new int[]{0,-16,0}},new int[][]{new int[]{0,32,0},new int[]{8,16,0}},new int[][]{new int[]{0,16,0},new int[]{0,16,0}},new int[][]{new int[]{0,16,0},new int[]{-16,32,0}},new int[][]{new int[]{0,0,0},new int[]{0,0,0}},new int[][]{new int[]{134217727,-134217728,134217727},new int[]{-134217728,134217727,-134217728}}};
   int[][] prevs={new int[]{0,0,0},new int[]{0,0,-512},new int[]{0,0,512},new int[]{1,0,0}};
   for(int c=0;c<pairs.Length;c++)for(int v=0;v<prevs.Length;v++){
    int[] aa={0,0,0},bb={16,0,0};object p0=Point(pairs[c][0]),p1=Point(pairs[c][1]);pt.GetField("next",f).SetValue(p0,p1);
    object writer=Activator.CreateInstance(td,f,null,new object[]{new StringWriter()},null);td.GetField("points",f).SetValue(writer,p0);
    td.GetField("prev",f).SetValue(writer,Activator.CreateInstance(vt,f,null,new object[]{prevs[v][0],prevs[v][1],prevs[v][2]},null));
    object selected=td.GetMethod("FindFace",f).Invoke(writer,new object[]{Point(aa),Point(bb)});int index=Object.ReferenceEquals(selected,p0)?0:Object.ReferenceEquals(selected,p1)?1:-1;
    w.WriteLine(c+"-"+v+"\t"+Values(aa)+"\t"+Values(bb)+"\t"+Values(pairs[c][0])+";"+Values(pairs[c][1])+"\t"+Values(prevs[v])+"\t"+index+"\t"+Values(XYZ(td.GetField("prev",f).GetValue(writer))));
    Console.WriteLine(c+"-"+v+" selected="+index);
   }
  }return 0;
 }catch(Exception e){Console.Error.WriteLine(e);return 1;}}
}
