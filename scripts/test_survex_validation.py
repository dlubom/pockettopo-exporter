#!/usr/bin/env python3
"""Regression controls for independent P16 anonymous endpoint validation."""
import importlib.util
from pathlib import Path
import unittest

spec = importlib.util.spec_from_file_location('svx', Path(__file__).with_name('validate-survex.py'))
svx = importlib.util.module_from_spec(spec)
spec.loader.exec_module(svx)

DUMP = '''NODE 0.00 0.00 0.00 [0_0] FIXED
NODE 1.00 1.00 0.00 [] UNDERGROUND ANON
NODE 0.00 1.00 2.00 [] UNDERGROUND ANON
'''


class AnonymousValidation(unittest.TestCase):
    def test_all_anonymous_flags_retained_and_checked(self):
        named, anonymous = svx.decode(DUMP)
        self.assertEqual(set(named), {'0_0'})
        self.assertEqual(len(anonymous), 2)
        svx.verify_anonymous(anonymous, [(1, 1, 0), (0, 1, 2)])
        for old in ('1.00 1.00 0.00 [] UNDERGROUND ANON', '0.00 1.00 2.00 [] UNDERGROUND ANON'):
            with self.subTest(endpoint=old):
                _, corrupt = svx.decode(DUMP.replace(old, old + ' WALL'))
                with self.assertRaisesRegex(ValueError, 'inferred as walls'):
                    svx.verify_anonymous(corrupt, [(1, 1, 0), (0, 1, 2)])

    def test_missing_or_merged_endpoints_rejected(self):
        _, anonymous = svx.decode(DUMP)
        for points in (anonymous[:1], anonymous + anonymous[:1], anonymous[:1] * 2):
            with self.assertRaises(ValueError):
                svx.verify_anonymous(points, [(1, 1, 0), (0, 1, 2)])

    def test_duplicate_named_station_rejected(self):
        with self.assertRaisesRegex(ValueError, 'Duplicate decoded station'):
            svx.decode(DUMP + 'NODE 0 0 0 [0_0] FIXED\n')


if __name__ == '__main__':
    unittest.main()
