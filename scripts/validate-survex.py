#!/usr/bin/env python3
"""Compile P16 output with pinned cavern/dump3d; compare independent controls."""
import argparse
import hashlib
import json
import math
import os
from pathlib import Path
import re
import subprocess

ROOT = Path(__file__).resolve().parent.parent
DATA = ROOT / 'internal/exportsvx/testdata'
DATES = ['2020-01-01','2022-01-01','2024-01-01']
# Legs include splays; length/plan/vertical statistics exclude them.
EXPECTED = {
 'features': (7,0,21.00,19.00,2.00,1),
 'mixed': (3,0,9.96,9.96,0.00,1),
 'ties': (2,0,1.00,0.00,1.00,0),
 'aliases': (2,0,2.50,2.50,0.00,0),
 'interrupted': (5,2,32.00,32.00,0.00,1),
 'dates': (1,0,10.00,10.00,0.00,0),
 'vertical-reverse': (3,0,3.00,0.00,3.00,1),
 'reverse-head': (1,0,10.00,7.07,7.07,0),
 'auto': (1,0,10.00,10.00,0.00,0),
}

def digest(raw):
    return hashlib.sha256(raw).hexdigest()

def run(args):
    return subprocess.run(list(map(str,args)), capture_output=True, timeout=30,
                          env={**os.environ,'LC_ALL':'C','LANG':'C','SURVEXLANG':'en'})

def version(binary, expected='1.4.22'):
    result = run([binary,'--version'])
    banner = (result.stdout+result.stderr).decode('utf-8')
    if result.returncode or not re.search(r'Survex '+re.escape(expected)+r'(?:\s|$)',banner):
        raise ValueError('consumer version differs: '+banner)
    return banner.strip()

def statistics(text):
    def match(pattern):
        m = re.search(pattern,text)
        if not m:
            raise ValueError('missing consumer statistic: '+pattern)
        return m.groups()
    legs = int(match(r'joined by (\d+) legs?')[0])
    loops = int(match(r'There (?:are|is) (\d+) loops?')[0])
    length = float(match(r'Total length of survey legs =\s+([0-9.]+)m')[0])
    plan = float(match(r'Total plan length of survey legs =\s+([0-9.]+)m')[0])
    vertical = float(match(r'Total vertical length of survey legs =\s+([0-9.]+)m')[0])
    return legs,loops,length,plan,vertical

def geometry(text):
    lines = text.splitlines()
    if "SEPARATOR ':'" not in lines or not lines or lines[-1]!='STOP':
        raise ValueError('invalid/incomplete dump3d')
    nodes,edges = [],[]
    point = None
    for line in lines:
        m = re.fullmatch(r'(MOVE|LINE) (-?[0-9.]+) (-?[0-9.]+) (-?[0-9.]+)(.*)',line)
        if m:
            end = tuple(map(float,m.groups()[1:4]))
            if not all(math.isfinite(x) for x in end):
                raise ValueError('nonfinite consumer vector')
            if m[1]=='LINE':
                if point is None:
                    raise ValueError('line has no start')
                edges.append((point,end,m[5]))
            point=end
        if line.startswith(('MOVE ', 'LINE ')) and not m:
            raise ValueError('malformed consumer vector')
        if line.startswith('NODE '):
            node = re.fullmatch(r'NODE (-?[0-9.]+) (-?[0-9.]+) (-?[0-9.]+) \[.*\].*', line)
            if not node or not all(math.isfinite(float(x)) for x in node.groups()):
                raise ValueError('malformed consumer node')
            nodes.append(line)
    if not nodes or not edges:
        raise ValueError('empty consumer geometry')
    return {'nodes':sorted(nodes),'edges':sorted(edges)}

def compile_file(cavern,dump3d,path):
    output = path.with_suffix('.3d')
    result=run([cavern,'-o',output,path])
    log=(result.stdout+result.stderr).decode('utf-8')
    path.with_suffix('.log').write_text(log,encoding='utf-8')
    if result.returncode or not output.is_file():
        raise ValueError('consumer compilation failed: '+log)
    dump=run([dump3d,output])
    if dump.returncode:
        raise ValueError('dump3d failed')
    text=dump.stdout.decode('utf-8')
    path.with_suffix('.dump').write_text(text,encoding='utf-8')
    return statistics(log),geometry(text)

def edit_dates(text):
    if text.count(';*date YYYY-MM-DD')!=3:
        raise ValueError('date fixture does not have three transitions')
    for date in DATES:
        text=text.replace(';*date YYYY-MM-DD','*date '+date,1)
    return text

def edit_auto(text):
    text=edit_dates(text)
    text=re.sub(r'^\*declination .+$',lambda m:';'+m[0],text,flags=re.M)
    for before,after in [(';*cs <CRS>','*cs EPSG:32634'),(';*cs out <OUTPUT_CRS>','*cs out EPSG:32634'),
                         (';*fix <STATION> <X> <Y> <Z>','*fix 1.0 400000 5450000 1000'),
                         (';*declination auto <X> <Y> <Z>','*declination auto 400000 5450000 1000')]:
        if text.count(before)!=1:
            raise ValueError('missing automatic template')
        text=text.replace(before,after)
    return text

def validate_lexical_settings(cavern,dump3d,raw,output):
    output.mkdir()
    # The reference readings are frozen, independently authored P16 controls.
    # A connected caller leg uses default syntax in the control and inherited
    # syntax in each actual host, so graph equality also proves restoration.
    foundation='*set separator :\n*set names ._-\n'
    data='*data normal from to tape compass clino\n'
    (output/'control-include.svx').write_bytes((DATA/'features.control.svx').read_bytes())
    control=output/'control.svx'
    control.write_text(foundation+'*include control-include.svx\n'+data+'1.4 after1 1.25 90 0\n')
    reference_stats,reference_graph=compile_file(cavern,dump3d,control)
    if reference_stats!=(8,0,22.25,20.25,2.00):
        raise ValueError('lexical reference statistics differ')
    cases=[
        ('decimal-comma','*set blank x09x20\n*set decimal ,\n','1.4 after1 1,25 90,0 0,0\n'),
        ('hash-comments','*set comment #\n','1.4 after1 1.25 90 0 # restored comment\n'),
        ('combined','*set blank x09x20\n*set decimal ,\n*set comment #\n','1.4 after1 1,25 90,0 0,0 # restored comment\n'),
    ]
    results=[]
    for name,prefix,suffix in cases:
        case=output/name;case.mkdir()
        (case/'include.svx').write_bytes(raw)
        actual=case/'actual.svx'
        actual.write_text(foundation+prefix+'*include include.svx\n'+data+suffix)
        stats,graph=compile_file(cavern,dump3d,actual)
        if stats!=reference_stats or graph!=reference_graph:
            raise ValueError(name+': lexical reset/restoration differs')
        results.append({'name':name,'statistics':stats,'geometry':graph})
    return results

def validate(binary,cavern,dump3d,output,consumer_version='1.4.22'):
    banners={'cavern':version(cavern,consumer_version),'dump3d':version(dump3d,consumer_version)}
    output.mkdir(parents=True,exist_ok=False)
    report={'schema_version':1,'consumer':banners,'cases':[]}
    for name,expected in EXPECTED.items():
        case=output/name;case.mkdir()
        srcname='dates' if name=='auto' else name
        source=(ROOT/'internal/exportwalls/testdata'/ (srcname+'.top')) if srcname in ('features','mixed','ties','aliases') else DATA/(srcname+'.top')
        original=source.read_bytes()
        package=case/'export'
        result=run([binary,'export',source,'--format','svx','--output',package])
        if result.returncode or result.stderr:
            raise ValueError('SVX export failed: '+result.stderr.decode())
        raw=(package/'survey.svx').read_bytes()
        lineage=json.loads((package/'report.json').read_text())
        if lineage['source_sha256']!=digest(original) or lineage['output_sha256']!=digest(raw):
            raise ValueError('export hashes differ')
        if source.read_bytes()!=original:
            raise ValueError('source changed')
        # Export and controls share only known synthetic test data and consumer.
        # The control files were authored separately, never generated by Build.
        text=raw.decode('utf-8')
        if name=='dates': text=edit_dates(text)
        if name=='auto': text=edit_auto(text)
        actual=case/'actual.svx';actual.write_text(text,encoding='utf-8')
        control=case/'control.svx';control.write_bytes((DATA/(name+'.control.svx')).read_bytes())
        stats,graph=compile_file(cavern,dump3d,actual)
        reference_stats,reference_graph=compile_file(cavern,dump3d,control)
        splays=sum(' SPLAY' in edge[2] for edge in graph['edges'])
        if (*stats,splays)!=expected or stats!=reference_stats or graph!=reference_graph:
            raise ValueError(f'{name}: geometry/statistics differ: {stats}, splays={splays}, expected={expected}')
        if any(' DUPLICATE' in x[2] or ' SURFACE' in x[2] for x in graph['edges']):
            raise ValueError('ordinary readings have inherited length-excluding flags')
        if name=='features' and not all(any('['+n+']' in node for node in graph['nodes']) for n in ('0','0.0')):
            raise ValueError('literal dots lost')
        if name=='auto':
            # Known test location and IGRF dates produce a measurable correction.
            # Verify it is active, beyond merely matching the control file.
            if graph['edges'][0][0]!=(400000.,5450000.,1000.) or not any(' FIXED' in n for n in graph['nodes']):
                raise ValueError('automatic profile did not use the supplied test fix')
            endpoint=graph['edges'][0][1]
            if abs(endpoint[0]-400009.93)>0.02 or abs(endpoint[1]-5449998.84)>0.02:
                raise ValueError('date/location-derived declination was not applied')
            if 'Declination:' not in actual.with_suffix('.log').read_text():
                raise ValueError('automatic correction log missing')
        if sum(' ANON WALL' in node for node in graph['nodes'])!=splays:
            raise ValueError('anonymous splay points differ')
        report['cases'].append({'name':name,'source_sha256':digest(original),'svx_sha256':digest(raw),
                                'control_sha256':digest(control.read_bytes()),'statistics':stats,'splays':splays,'geometry':graph})
    # Inherited settings must not contaminate output, and the anonymous scope
    # must restore them on return. Compare all vectors including an added leg.
    case=output/'inherited';case.mkdir()
    prefix='''*units tape feet
*units compass clino grads
*case toupper
*truncate 1
*calibrate tape 2 2
*calibrate compass clino 3 2
*flags duplicate surface splay
*infer equates on
*infer plumbs on
*infer exports on
*sd tape 8 metres
*sd compass clino 30 degrees
*declination 30 degrees
'''
    for name,raw in [('actual',(output/'features/export/survey.svx').read_bytes()),('control',(DATA/'features.control.svx').read_bytes())]:
        (case/(name+'-include.svx')).write_bytes(raw)
        (case/(name+'.svx')).write_text(prefix+'*include '+name+'-include.svx\n0 xx 10 100 0\n*units default\n')
    stats,graph=compile_file(cavern,dump3d,case/'actual.svx')
    rs,rg=compile_file(cavern,dump3d,case/'control.svx')
    if stats!=rs or graph!=rg or not any(all(flag in edge[2] for flag in ('DUPLICATE','SURFACE','SPLAY')) for edge in graph['edges']) or not any('[X]' in n for n in graph['nodes']):
        raise ValueError('scope reset/restoration differs')
    report['inherited_settings_verified']=True
    report['lexical_settings']=validate_lexical_settings(cavern,dump3d,
        (output/'features/export/survey.svx').read_bytes(),output/'lexical')
    # An actual compiler failure is required; parser/version/setup failures do
    # not establish the negative semantic control.
    bad=output/'invalid.svx';bad.write_text('*data normal from to tape compass clino\n0 0 1 0 0\n')
    failure=run([cavern,'-o',bad.with_suffix('.3d'),bad]);log=(failure.stdout+failure.stderr).decode('utf-8')
    bad.with_suffix('.log').write_text(log)
    if failure.returncode!=1 or bad.with_suffix('.3d').exists() or not re.search(r'same station',log,re.I):
        raise ValueError('consumer negative control failed: '+log)
    report['negative_control_verified']=True
    (output/'report.json').write_text(json.dumps(report,indent=2)+'\n')
    print(f'Survex {consumer_version} verified: 9 geometry/statistics cases, 3 lexical reset/restoration cases, inherited settings and semantic negative control')
    return report

if __name__=='__main__':
    p=argparse.ArgumentParser(description=__doc__)
    p.add_argument('--binary',type=Path,required=True);p.add_argument('--cavern',default='cavern');p.add_argument('--dump3d',default='dump3d');p.add_argument('--output',type=Path,required=True)
    p.add_argument('--consumer-version',choices=('1.4.22','1.4.23'),default='1.4.22')
    a=p.parse_args();validate(a.binary.resolve(),a.cavern,a.dump3d,a.output,a.consumer_version)
