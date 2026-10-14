#!/usr/bin/env python3
"""Fail-closed controls for P16 consumer evidence parsing and template edits."""
import importlib.util
from pathlib import Path
import unittest
from unittest.mock import patch
import subprocess

ROOT = Path(__file__).resolve().parent.parent
spec=importlib.util.spec_from_file_location('survex',ROOT/'scripts/validate-survex.py')
v=importlib.util.module_from_spec(spec);spec.loader.exec_module(v)

class SurvexValidation(unittest.TestCase):
    def test_explicit_consumer_version(self):
        # Version query is the only mocked boundary; validation must still
        # reject a different, prefixed, or failed consumer banner.
        for expected in ('1.4.22','1.4.23'):
            good=subprocess.CompletedProcess([],0,('cavern - Survex '+expected+'\n').encode(),b'')
            with patch.object(v,'run',return_value=good):
                self.assertIn(expected,v.version('cavern',expected))
            for banner,status in [('Survex 1.4.24\n',0),('Survex '+expected+'0\n',0),('Survex '+expected+'\n',1)]:
                bad=subprocess.CompletedProcess([],status,banner.encode(),b'')
                with patch.object(v,'run',return_value=bad),self.assertRaises(ValueError):
                    v.version('cavern',expected)
        with patch.object(v,'run',return_value=subprocess.CompletedProcess([],0,b'Survex 1.4.23\n',b'')),self.assertRaises(ValueError):
            v.version('cavern')
    def test_statistics(self):
        text='joined by 1 leg.\nThere are 0 loops.\nTotal length of survey legs = 10.00m\nTotal plan length of survey legs = 10.00m\nTotal vertical length of survey legs = 0.00m\n'
        self.assertEqual(v.statistics(text),(1,0,10.,10.,0.))
        for bad in ('',text.replace('10.00m','NaNm'),text.replace('There are 0 loops.','')):
            with self.assertRaises(ValueError):v.statistics(bad)
    def test_geometry(self):
        text="SEPARATOR ':'\nMOVE 0.00 0.00 0.00\nLINE 1.00 0.00 0.00 [] STYLE=NORMAL SPLAY\nNODE 1.00 0.00 0.00 [] UNDERGROUND ANON WALL\nSTOP\n"
        graph=v.geometry(text)
        self.assertEqual(graph['edges'],[((0.,0.,0.),(1.,0.,0.),' [] STYLE=NORMAL SPLAY')])
        for bad in ('',text.replace("':'","'.'"),text.replace('STOP\n',''),text.replace('MOVE 0.00 0.00 0.00\n',''),text.replace('1.00','NaN')):
            with self.assertRaises(ValueError):v.geometry(bad)
    def test_templates(self):
        source=(ROOT/'internal/exportsvx/testdata/dates.top').read_bytes()
        # Literal input to editing is independently authored, not a compiled export.
        text=';*date YYYY-MM-DD\n'*3+'*declination 0.000000 degrees\n;*cs <CRS>\n;*cs out <OUTPUT_CRS>\n;*fix <STATION> <X> <Y> <Z>\n;*declination auto <X> <Y> <Z>\n'
        auto=v.edit_auto(text)
        self.assertIn(';*declination 0.000000 degrees',auto)
        self.assertIn('*declination auto 400000 5450000 1000',auto)
        self.assertEqual(source,(ROOT/'internal/exportsvx/testdata/dates.top').read_bytes())
        self.assertEqual(auto.count('*date 20'),3)
        for bad in (text.replace(';*date YYYY-MM-DD\n','',1),text.replace(';*cs <CRS>',''),text+';*cs <CRS>\n'):
            with self.assertRaises(ValueError):v.edit_auto(bad)

if __name__=='__main__':unittest.main()
