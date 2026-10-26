#!/usr/bin/env python3
"""Offline protocol tests: no formatter, Go benchmark, or artificial host load."""
import argparse
import copy
import importlib.util
import json
import os
from pathlib import Path
import signal
import subprocess
import sys
import tempfile
import unittest
from unittest import mock

ROOT = Path(__file__).resolve().parents[1]
spec = importlib.util.spec_from_file_location('admission_controller', ROOT / 'scripts/cache-bench-admission.py')
controller = importlib.util.module_from_spec(spec)
spec.loader.exec_module(controller)
RAW = (ROOT / 'docs/developer/cache-overhead-2026-10-25.log').read_text()


def load(value=2, clock=0):
    return {'monotonic': clock, 'load_average_1_5_15': [value, 2, 2]}


class ProtocolTests(unittest.TestCase):
    def test_original_replay_is_arithmetic_not_measurement(self):
        result = controller.aggregate(RAW)
        self.assertEqual(36, result['sample_count'])
        self.assertAlmostEqual(42.780814285714285, result['warm_hit_us_per_file'])
        record = {}
        controller.replay(argparse.Namespace(
            raw=ROOT / 'docs/developer/cache-overhead-2026-10-25.log',
            metadata=ROOT / 'docs/developer/cache-overhead-2026-10-25.json',
            admission=ROOT / 'docs/developer/cache-overhead-2026-10-25-admission.json'), record)
        self.assertEqual('REPLAY_PASS', record['state'])
        self.assertEqual('20c85c5935e6819f4f89f03bf9f499c1640adf3e', record['historical_source']['head'])
        self.assertEqual('HISTORICAL_REPLAY_NOT_A_MEASUREMENT', record['kind'])

    def test_target_boundaries(self):
        for value, expected in ((3499999, True), (3500000, True), (3500001, False)):
            lines = []
            for line in RAW.splitlines():
                if line.startswith(controller.PREFIX + 'WarmHit-'):
                    line = controller.re.sub(r'[\d.eE+\-]+ ns/op', str(value) + ' ns/op', line)
                lines.append(line)
            self.assertEqual(expected, controller.aggregate('\n'.join(lines))['timing_pass'])

    def test_missing_duplicate_malformed_and_wrong_input_rejected(self):
        sample = next(line for line in RAW.splitlines() if line.startswith(controller.PREFIX + 'WarmHit-'))
        variants = [RAW.replace(sample, '', 1), RAW + '\n' + sample,
                    RAW.replace('0 tool_runs/op', '1 tool_runs/op', 1),
                    RAW.replace('0 version_calls/op', '1 version_calls/op', 1),
                    RAW.replace('tool_runs/op', 'absent_counter', 1),
                    RAW.replace('files=35', 'files=34'),
                    RAW.replace(controller.PROTOCOL['workload_sha256'], '0' * 64),
                    RAW.replace(controller.PROTOCOL['workload_ref'], '0' * 40),
                    RAW.replace(sample, sample.replace('ns/op', 'not-a-unit'))]
        for text in variants:
            with self.subTest(text=text[-90:]), self.assertRaises(controller.Invalid):
                controller.aggregate(text)

    def test_equal_six_observations_are_valid_seventh_is_not(self):
        for value in (3499999, 3500000, 3500001):
            rows = [line for line in RAW.splitlines() if line.startswith(controller.PREFIX + 'WarmHit-')]
            identical = controller.re.sub(r'[\d.eE+\-]+ ns/op', str(value) + ' ns/op', rows[0])
            text = RAW
            for row in rows:
                text = text.replace(row, identical, 1)
            self.assertEqual(value <= 3500000, controller.aggregate(text)['timing_pass'])
            with self.assertRaises(controller.Invalid):
                controller.aggregate(text + '\n' + identical)

    def test_admission_single_pair_and_partial_preservation(self):
        for first, second, expected in ((2, 4, True), (4.01, 2, False), (2, 4.01, False)):
            readings = iter([load(first, 0), load(second, 30)])
            pauses = []
            saved = []
            records = []
            _, admitted = controller.admission(lambda: next(readings), pauses.append, records,
                                               lambda: saved.append(copy.deepcopy(records)))
            self.assertEqual([30], pauses)
            self.assertEqual(expected, admitted)
            self.assertEqual([1, 2], list(map(len, saved)))
        readings = iter([load(2, 0), load(2, 29.9)])
        with self.assertRaises(controller.Invalid):
            controller.admission(lambda: next(readings), lambda _: None)

    def test_middle_after_load_command_and_identity_fail_closed(self):
        for code, before, after, readings in ((1, 'same', 'same', [load()]),
                                             (0, 'same', 'changed', [load()]),
                                             (0, 'same', 'same', [load(), load(4.1), load()]),
                                             (0, 'same', 'same', [load(), load(4.1)])):
            with self.assertRaises(controller.Invalid):
                controller.judge(RAW, code, before, after, readings)
        self.assertTrue(controller.judge(RAW, 0, 'same', 'same', [load()])['timing_pass'])

    @unittest.skipUnless(os.name == 'posix', 'live controller is Darwin-only; process group test requires POSIX')
    def test_controlled_process_success_failure_cancel_and_raw(self):
        # This starts only this installed Python runtime, never a product benchmark.
        for exit_code in (0, 23):
            with tempfile.TemporaryDirectory(dir=ROOT / 'tmp') as directory:
                path = Path(directory) / 'raw.log'
                state = {}
                code, readings = controller.execute(
                    [sys.executable, '-c', 'print("controlled raw");raise SystemExit(' + str(exit_code) + ')'],
                    ROOT, os.environ.copy(), path, observer=load, execution=state)
                self.assertEqual(exit_code, code)
                self.assertEqual('controlled raw\n', path.read_text())
                self.assertTrue(readings)
                self.assertIn('no signals', state['cleanup'])
        for problem in (controller.Cancelled(signal.SIGTERM), controller.Invalid('timeout fixture')):
            with tempfile.TemporaryDirectory(dir=ROOT / 'tmp') as directory:
                path = Path(directory) / 'raw.log'
                state = {}
                marker = Path(directory) / 'started'
                source = 'from pathlib import Path;import time;print("partial raw",flush=True);Path(' + repr(str(marker)) + ').touch();time.sleep(60)'
                def observe_started():
                    return load()
                def interrupt(_):
                    # Wait for the owned child to write its first raw output, bounded by a timeout.
                    import time
                    deadline = time.monotonic() + 5
                    while not marker.exists() and time.monotonic() < deadline:
                        time.sleep(.01)
                    self.assertTrue(marker.exists())
                    raise problem
                with self.assertRaises(type(problem)):
                    controller.execute([sys.executable, '-c', source], ROOT, os.environ.copy(), path,
                                       observer=observe_started, pause=interrupt, execution=state)
                self.assertEqual('partial raw\n', path.read_text())
                self.assertTrue(state['measurement_loads'])
                self.assertIn('reaped', state['cleanup'])
                with self.assertRaises(ProcessLookupError):
                    os.killpg(state['cleanup_owned_pgid'], 0)

    def test_not_admitted_never_starts_a_command(self):
        with tempfile.TemporaryDirectory(dir=ROOT / 'tmp') as directory:
            tool = ROOT / 'scripts/cache-bench-admission.py'
            receipt = {'started': False}
            with mock.patch.object(controller.platform, 'system', return_value='Darwin'), \
                 mock.patch.object(controller.platform, 'machine', return_value='arm64'), \
                 mock.patch.object(controller.shutil, 'which', return_value=str(tool)), \
                 mock.patch.object(controller, 'source_identity', return_value={'head': 'fixed'}), \
                 mock.patch.object(controller, 'vm_observation', return_value={'matching_vm_present': False}), \
                 mock.patch.object(controller, 'command', return_value='128'), \
                 mock.patch.object(controller, 'admission', return_value=([load(5), load(2, 30)], False)), \
                 mock.patch.object(controller, 'execute') as execute:
                controller.run(argparse.Namespace(gofumpt=str(tool), cpu_slot_go=True), ROOT, Path(directory), receipt)
                execute.assert_not_called()
            self.assertEqual('NOT_ADMITTED_NOT_MEASURED', receipt['state'])
            self.assertFalse(receipt['started'])

    @unittest.skipUnless(os.name == 'posix', 'live controller is Darwin-only')
    def test_deadline_stops_only_owned_fixture_and_keeps_checkpoint(self):
        with tempfile.TemporaryDirectory(dir=ROOT / 'tmp') as directory:
            out = Path(directory)
            receipt = {}
            ticks = iter([0, 601])
            with self.assertRaisesRegex(controller.Invalid, 'timeout'):
                controller.execute([sys.executable, '-c', 'import time;time.sleep(60)'], ROOT,
                                   os.environ.copy(), out / 'raw.log', observer=load,
                                   clock=lambda: next(ticks), execution=receipt,
                                   checkpoint=lambda: controller.save(out, receipt))
            self.assertTrue((out / 'raw.log').exists())
            self.assertTrue(json.loads((out / 'receipt.json').read_text())['measurement_loads'])
            self.assertIn('reaped', receipt['cleanup'])

    def test_required_after_vm_failure_is_invalid_with_evidence_retained(self):
        receipt = {'after_load': load(), 'source_after': {'head': 'fixed'},
                   'after_vm_error': 'controlled observer refusal', 'raw_log_sha256': 'preserved'}
        with self.assertRaisesRegex(controller.Invalid, 'after_vm'):
            controller.require_after_observers(receipt)
        self.assertEqual('preserved', receipt['raw_log_sha256'])
        self.assertIn('after_vm_error', receipt)
        receipt['after_vm'] = {'matching_vm_present': False, 'matching_vm_process_names': []}
        with self.assertRaises(controller.Invalid):
            controller.require_after_observers(receipt)
        del receipt['after_vm_error']
        controller.require_after_observers(receipt)

    def test_hostile_git_environment_cannot_hide_actual_dirty_source(self):
        with tempfile.TemporaryDirectory(dir=ROOT / 'tmp') as directory:
            base = Path(directory)
            repositories = [base / 'actual', base / 'foreign']
            for repo in repositories:
                subprocess.run(['git', 'init', '-q', str(repo)], env=controller.git_environment(), check=True)
                subprocess.run(['git', '-C', str(repo), '-c', 'user.name=fixture', '-c',
                                'user.email=fixture@example.invalid', 'commit', '-q', '--allow-empty', '-m', 'fixture'],
                               env=controller.git_environment(), check=True)
            actual, foreign = repositories
            (actual / 'dirty').write_text('must not be hidden')
            foreign_before = subprocess.check_output(['git', '-C', str(foreign), 'status', '--porcelain'], env=controller.git_environment())
            hostile = {'GIT_DIR': str(foreign / '.git'), 'GIT_WORK_TREE': str(foreign),
                       'GIT_INDEX_FILE': str(base / 'foreign-index'), 'GIT_CONFIG_COUNT': '1',
                       'GIT_CONFIG_KEY_0': 'core.worktree', 'GIT_CONFIG_VALUE_0': str(foreign),
                       'GIT_CONFIG_GLOBAL': str(base / 'hostile-config')}
            with mock.patch.dict(os.environ, hostile):
                with self.assertRaisesRegex(controller.Invalid, 'clean'):
                    controller.source_identity(actual, Path(__file__))
                top = controller.command(actual, 'git', 'rev-parse', '--show-toplevel')
                self.assertEqual(actual.resolve(), Path(top).resolve())
                clean = controller.git_environment()
                self.assertNotIn('GIT_DIR', clean)
                self.assertNotIn('GIT_INDEX_FILE', clean)
                self.assertNotIn('GIT_CONFIG_COUNT', clean)
            self.assertEqual('must not be hidden', (actual / 'dirty').read_text())
            self.assertEqual(foreign_before, subprocess.check_output(['git', '-C', str(foreign), 'status', '--porcelain'], env=controller.git_environment()))
            self.assertFalse((base / 'foreign-index').exists())

    @unittest.skipUnless(os.name == 'posix' and hasattr(os, 'waitid') and hasattr(os, 'WNOWAIT'), 'needs unreaped POSIX child identity')
    def test_cleanup_reaped_or_unknown_never_signals_unverified_group(self):
        proc = subprocess.Popen([sys.executable, '-c', 'pass'], start_new_session=True)
        proc.wait(timeout=5)
        with mock.patch.object(controller.os, 'killpg') as kill:
            with self.assertRaises(controller.Invalid):
                controller.terminate(proc, {}, normal=True)
            kill.assert_not_called()
        proc = subprocess.Popen([sys.executable, '-c', 'import time;time.sleep(60)'], start_new_session=True)
        try:
            record = {}
            with mock.patch.object(controller, 'group_states', side_effect=PermissionError('controlled unknown')), \
                 mock.patch.object(controller.os, 'killpg') as kill:
                with self.assertRaises(PermissionError):
                    controller.terminate(proc, record)
                kill.assert_not_called()
            self.assertIsNone(proc.poll())
        finally:
            # Test owns this exact child; this is not a controller fallback.
            proc.kill()
            proc.wait(timeout=5)

    @unittest.skipUnless(os.name == 'posix', 'requires owned POSIX process groups')
    def test_term_exited_parent_retains_identity_until_descendant_is_stopped(self):
        import time
        unrelated = subprocess.Popen([sys.executable, '-c', 'import time;time.sleep(60)'], start_new_session=True)
        try:
            with tempfile.TemporaryDirectory(dir=ROOT / 'tmp') as directory:
                marker = Path(directory) / 'descendant-ready'
                child = ('import signal,time;from pathlib import Path;'
                         'signal.signal(signal.SIGTERM,signal.SIG_IGN);'
                         'Path(' + repr(str(marker)) + ').touch();time.sleep(60)')
                parent = 'import subprocess,sys,time;subprocess.Popen([sys.executable,"-c",' + repr(child) + ']);time.sleep(60)'
                state = {}
                def interrupt(_):
                    deadline = time.monotonic() + 5
                    while not marker.exists() and time.monotonic() < deadline:
                        time.sleep(.01)
                    self.assertTrue(marker.exists())
                    raise controller.Cancelled(signal.SIGTERM)
                with self.assertRaises(controller.Cancelled):
                    controller.execute([sys.executable, '-c', parent], ROOT, os.environ.copy(),
                                       Path(directory) / 'raw.log', observer=load, pause=interrupt, execution=state)
                self.assertEqual(['SIGTERM', 'SIGKILL'], state['cleanup_signals'])
                self.assertIn('reaped last', state['cleanup'])
                self.assertIsNone(unrelated.poll())
                self.assertNotIn('cleanup_error', state)
        finally:
            unrelated.kill()
            unrelated.wait(timeout=5)

    def test_primary_cancellation_and_unknown_cleanup_are_both_retained(self):
        children = []
        real_popen = subprocess.Popen
        def start(*args, **kwargs):
            child = real_popen(*args, **kwargs)
            children.append(child)
            return child
        with tempfile.TemporaryDirectory(dir=ROOT / 'tmp') as directory:
            state = {}
            problem = controller.Cancelled(signal.SIGTERM)
            try:
                with mock.patch.object(controller.subprocess, 'Popen', side_effect=start), \
                     mock.patch.object(controller, 'terminate', side_effect=controller.Invalid('controlled ownership unknown')):
                    with self.assertRaises(controller.Cancelled) as raised:
                        controller.execute([sys.executable, '-c', 'import time;time.sleep(60)'], ROOT,
                                           os.environ.copy(), Path(directory) / 'raw.log', observer=load,
                                           pause=mock.Mock(side_effect=problem), execution=state)
                self.assertIs(problem, raised.exception)
                self.assertIn('primary_error', state)
                self.assertEqual('controlled ownership unknown', state['cleanup_error'])
                self.assertEqual(children[0].pid, state['retained_pgid'])
            finally:
                for child in children:
                    child.kill()
                    child.wait(timeout=5)

    def test_existing_output_is_not_overwritten(self):
        with tempfile.TemporaryDirectory(dir=ROOT / 'tmp') as directory:
            path = Path(directory)
            marker = path / 'unchanged'
            marker.write_text('original')
            with self.assertRaises(FileExistsError):
                controller.main(['replay', '--raw', 'unused', '--metadata', 'unused',
                                 '--admission', 'unused', '--output', str(path)])
            self.assertEqual('original', marker.read_text())


if __name__ == '__main__':
    unittest.main()
