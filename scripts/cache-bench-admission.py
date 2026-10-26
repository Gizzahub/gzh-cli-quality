#!/usr/bin/env python3
"""G2 전용 단일 실행 controller. 일반 benchmark와 과거 replay는 새 측정이 아니다."""
import argparse
import datetime
import hashlib
import json
import math
import os
from pathlib import Path
import platform
import re
import shutil
import signal
import statistics
import subprocess
import sys
import time

PROTOCOL = {
    'id': 'darwin-gofumpt-g2-v1', 'count': 6, 'benchtime': '300ms',
    'target_us_per_file': 100, 'max_load_1m': 4, 'admission_gap_seconds': 30,
    'observer_seconds': 1, 'process_timeout_seconds': 600,
    'workload_ref': 'd7cf583210f2174d2c58746f6c939277546a5bbe',
    'files': 35, 'bytes': 224789,
    'workload_sha256': 'ec3ce86dbe97d934fe390f18379ba672378cfb791c0049c8dd06f579d7be8435',
}
GROUPS = ('NoCache', 'ColdMiss', 'WarmHit', 'WarmHitNoVersionCache',
          'VersionProbe', 'CacheMetadataReads')
COUNTERS = {'NoCache': (1, 0), 'ColdMiss': (1, 1), 'WarmHit': (0, 0),
            'WarmHitNoVersionCache': (0, 1)}
PREFIX = 'BenchmarkCache_InstalledGofumpt/'
SAMPLE = re.compile(r'^BenchmarkCache_InstalledGofumpt/(\w+)-\d+\s+(\d+)\s+([\d.eE+\-]+) ns/op(.*)$')
META = re.compile(r'tool=(.*?) version=(.*?) revision=([0-9a-f]{40}) files=(\d+) bytes=(\d+) workload_sha256=([0-9a-f]{64})')
MAX_RAW = 16 * 1024 * 1024


class Invalid(Exception):
    pass


class Cancelled(Exception):
    def __init__(self, number):
        self.number = number
        super().__init__('controller cancelled')


def require(ok, reason):
    if not ok:
        raise Invalid(reason)


def sha(path):
    digest = hashlib.sha256()
    with Path(path).open('rb') as source:
        for block in iter(lambda: source.read(1024 * 1024), b''):
            digest.update(block)
    return digest.hexdigest()


def finite(value):
    result = float(value)
    require(math.isfinite(result), 'nonfinite numeric value')
    return result


def aggregate(raw):
    """원문 36표본과 측정 counter를 검사한다. 좋지 않은 표본도 제외하지 않는다."""
    samples = {name: [] for name in GROUPS}
    metadata = []
    for line in raw.splitlines():
        match = META.search(line)
        if match:
            tool, version, revision, files, size, digest = match.groups()
            identity = {'tool': tool, 'version': version, 'revision': revision,
                        'files': int(files), 'bytes': int(size), 'sha256': digest}
            require(identity['revision'] == PROTOCOL['workload_ref'] and
                    identity['files'] == PROTOCOL['files'] and
                    identity['bytes'] == PROTOCOL['bytes'] and
                    digest == PROTOCOL['workload_sha256'], 'wrong workload identity')
            metadata.append(identity)
        if not line.startswith(PREFIX):
            continue
        if line.strip() in {PREFIX + name for name in GROUPS}:
            continue  # Go -v announces each group separately from its six samples.
        match = SAMPLE.fullmatch(line.strip())
        require(match is not None, 'malformed benchmark sample')
        name, iterations, elapsed, tail = match.groups()
        require(name in samples, 'unexpected benchmark group')
        require(int(iterations) > 0, 'nonpositive iteration count')
        elapsed = finite(elapsed)
        require(elapsed > 0, 'nonpositive timing')
        parts = tail.split()
        require(len(parts) % 2 == 0, 'malformed metrics')
        metrics = {}
        for index in range(0, len(parts), 2):
            key = parts[index + 1]
            require(key not in metrics, 'duplicate metric')
            metrics[key] = finite(parts[index])
        if name in COUNTERS:
            tool_runs, versions = COUNTERS[name]
            require(metrics.get('tool_runs/op') == tool_runs and
                    metrics.get('version_calls/op') == versions, 'subprocess counter contract failed')
        if name == 'CacheMetadataReads':
            require(metrics.get('cache_reads/op') == PROTOCOL['files'], 'metadata read count failed')
        samples[name].append({'iterations': int(iterations), 'ns_per_op': elapsed,
                              'metrics': metrics, 'raw_line': line})
    require(metadata and all(item == metadata[0] for item in metadata), 'missing or changed workload metadata')
    require(all(len(group) == PROTOCOL['count'] for group in samples.values()), 'expected exactly six samples per group')
    medians = {name: statistics.median(item['ns_per_op'] for item in group)
               for name, group in samples.items()}
    per_file = medians['WarmHit'] / PROTOCOL['files'] / 1000
    return {'samples': samples, 'workload_metadata': metadata[0], 'median_ns_per_op': medians,
            'warm_hit_us_per_file': per_file, 'timing_pass': per_file <= PROTOCOL['target_us_per_file'],
            'counter_contract_pass': True, 'sample_count': 36}


def valid_load(reading):
    values = reading.get('load_average_1_5_15', [])
    return len(values) == 3 and all(type(v) in (int, float) and math.isfinite(v) and v >= 0 for v in values) and values[0] <= PROTOCOL['max_load_1m']


def admission(observer, pause=time.sleep, readings=None, checkpoint=lambda: None):
    readings = [] if readings is None else readings
    first = observer()
    readings.append(first)
    checkpoint()
    pause(PROTOCOL['admission_gap_seconds'])
    second = observer()
    readings.append(second)
    checkpoint()
    require(second['monotonic'] - first['monotonic'] >= PROTOCOL['admission_gap_seconds'], 'admission readings too close')
    return [first, second], valid_load(first) and valid_load(second)


def observe():
    return {'utc': time.strftime('%Y-%m-%dT%H:%M:%SZ', time.gmtime()),
            'monotonic': time.monotonic(), 'load_average_1_5_15': list(os.getloadavg())}


def child_exit_observation(proc):
    """WNOWAIT keeps the direct child's PID reserved until the final reap."""
    require(proc.returncode is None, 'direct child was already reaped; ownership is unknown')
    flags = os.WEXITED | os.WNOHANG | os.WNOWAIT
    try:
        observation = os.waitid(os.P_PID, proc.pid, flags)
        if observation is not None:
            require(observation.si_pid == proc.pid, 'foreign child exit observation')
            return observation
        try:
            require(os.getpgid(proc.pid) == proc.pid, 'child is not its own session group')
        except ProcessLookupError:
            # Darwin may stop exposing getpgid for an exited, unreaped child.
            # Exit visibility can lag getpgid disappearance. Never signal in this gap.
            deadline = time.monotonic() + 1
            while observation is None and time.monotonic() < deadline:
                observation = os.waitid(os.P_PID, proc.pid, flags)
                if observation is None:
                    time.sleep(.001)
            require(observation is not None and observation.si_pid == proc.pid,
                    'child disappeared without an owned unreaped exit')
        return observation
    except (ChildProcessError, PermissionError) as exc:
        raise Invalid('cannot verify unreaped direct-child ownership') from exc


def group_states(proc, execution):
    exited = child_exit_observation(proc)
    result = subprocess.run(['ps', '-g', str(proc.pid), '-o', 'pid=,pgid=,stat='],
                            stdout=subprocess.PIPE, stderr=subprocess.PIPE, timeout=5)
    if result.returncode == 1 and not result.stdout.strip() and exited is not None:
        try:
            os.killpg(proc.pid, 0)
        except ProcessLookupError:
            execution.setdefault('cleanup_observations', []).append([])
            return []
        raise Invalid('empty group query without verified group absence')
    require(result.returncode == 0, 'owned group state query failed')
    states = []
    for line in result.stdout.decode().splitlines():
        parts = line.split()
        require(len(parts) == 3 and parts[0].isdigit() and parts[1].isdigit() and
                parts[2] and parts[2][0] in 'RSDITZUWtX', 'malformed owned group state')
        require(int(parts[1]) == proc.pid, 'group query returned a foreign group')
        states.append({'pid': int(parts[0]), 'pgid': int(parts[1]), 'state': parts[2]})
    require(states, 'owned group state is unknown')
    execution.setdefault('cleanup_observations', []).append(states)
    return states


def terminate(proc, execution, normal=False):
    """Never reap before group verification/signals: the child PID anchors ownership."""
    blocked = signal.pthread_sigmask(signal.SIG_BLOCK, {signal.SIGINT, signal.SIGTERM})
    try:
        states = group_states(proc, execution)
        running = lambda rows: any(not row['state'].startswith('Z') for row in rows)
        if normal:
            require(not running(states), 'normal child exited but live group members remain; no signals sent')
            proc.wait(timeout=5)
            execution['cleanup'] = 'normal exit; no running group; no signals; parent reaped'
            return
        execution['cleanup_owned_pgid'] = proc.pid
        execution['cleanup_signals'] = []
        for number in (signal.SIGTERM, signal.SIGKILL):
            if not running(states):
                break
            child_exit_observation(proc)
            try:
                os.killpg(proc.pid, number)
                execution['cleanup_signals'].append(number.name)
            except ProcessLookupError:
                # The still-unreaped direct child reserves this PGID; verify again.
                pass
            deadline = time.monotonic() + 5
            while True:
                states = group_states(proc, execution)
                if not running(states) or time.monotonic() >= deadline:
                    break
                time.sleep(.01)
        require(not running(states), 'owned group is still running; cleanup incomplete')
        proc.wait(timeout=5)
        execution['cleanup'] = 'owned session stopped; direct parent reaped last'
    finally:
        signal.pthread_sigmask(signal.SIG_SETMASK, blocked)


def execute(argv, cwd, env, raw_path, observer=observe, pause=time.sleep, clock=time.monotonic,
            execution=None, checkpoint=lambda: None):
    require(hasattr(os, 'waitid') and hasattr(os, 'WNOWAIT'), 'unreaped-child observation is required')
    execution = {} if execution is None else execution
    readings = execution.setdefault('measurement_loads', [])
    proc = None
    primary = None
    cleanup_failure = None
    normal = False
    with raw_path.open('xb') as raw:
        os.chmod(raw_path, 0o600)
        try:
            proc = subprocess.Popen(argv, cwd=cwd, env=env, stdout=raw, stderr=subprocess.STDOUT,
                                    start_new_session=True)
            execution['started'] = True
            started = clock()
            while child_exit_observation(proc) is None:
                readings.append(observer())
                checkpoint()
                if clock() - started >= PROTOCOL['process_timeout_seconds']:
                    raise Invalid('benchmark exceeded existing Go test ten-minute timeout')
                require(raw_path.stat().st_size <= MAX_RAW, 'benchmark output exceeds bound')
                pause(PROTOCOL['observer_seconds'])
            normal = True
            readings.append(observer())
            checkpoint()
            require(raw_path.stat().st_size <= MAX_RAW, 'benchmark output exceeds bound')
        except BaseException as exc:
            primary = exc
            execution['primary_error'] = str(exc)
        finally:
            if proc is not None:
                try:
                    terminate(proc, execution, normal=normal)
                except BaseException as exc:
                    cleanup_failure = exc
                    execution['cleanup_error'] = str(exc)
                    execution['retained_pgid'] = proc.pid
                execution['exit_code'] = proc.returncode
                checkpoint()
    if primary is not None:
        raise primary
    if cleanup_failure is not None:
        raise Invalid('process cleanup failed; retained resource recorded') from cleanup_failure
    return proc.returncode, readings


def git_environment(base=None):
    # Only invocation-local Git controls change; no user/global configuration is edited.
    env = {key: value for key, value in (os.environ if base is None else base).items()
           if not key.upper().startswith('GIT_')}
    env.update(GIT_CONFIG_GLOBAL=os.devnull, GIT_CONFIG_SYSTEM=os.devnull,
               GIT_CONFIG_NOSYSTEM='1', GIT_OPTIONAL_LOCKS='0')
    return env


def command(repo, *args):
    return subprocess.check_output(args, cwd=repo, timeout=30,
                                   env=git_environment() if args[0] == 'git' else None).decode().strip()


def source_identity(repo, tool):
    actual = Path(command(repo, 'git', 'rev-parse', '--show-toplevel')).resolve()
    require(actual == Path(repo).resolve(), 'Git did not resolve the actual repository root')
    require(command(repo, 'git', 'rev-parse', '--is-bare-repository') == 'false', 'bare repository is not an execution source')
    require(not command(repo, 'git', 'status', '--porcelain'), 'source must be clean')
    return {'head': command(repo, 'git', 'rev-parse', 'HEAD'),
            'tree': command(repo, 'git', 'rev-parse', 'HEAD^{tree}'),
            'harness_sha256': sha(repo / 'executor/real_cache_bench_test.go'),
            'controller_sha256': sha(Path(__file__)), 'tool_sha256': sha(tool)}


def vm_observation(repo):
    names = command(repo, 'ps', '-axo', 'comm=').splitlines()
    selected = sorted({Path(name).name for name in names
                       if 'com.apple.Virtualization.VirtualMachine' in name or 'qemu-system-' in name})
    return {'matching_vm_present': bool(selected), 'matching_vm_process_names': selected}


def save(out, receipt):
    path = out / 'receipt.json'
    temporary = out / 'receipt.json.tmp'
    temporary.write_text(json.dumps(receipt, indent=2, allow_nan=False) + '\n')
    os.chmod(temporary, 0o600)
    temporary.replace(path)


def replay(args, receipt):
    """과거 원문 재집계이며 새 observer 관측을 과거 실행에 소급하지 않는다."""
    raw_path, metadata_path, admission_path = map(Path, (args.raw, args.metadata, args.admission))
    metadata = json.loads(metadata_path.read_text())
    retained_admission = json.loads(admission_path.read_text())
    require(sha(raw_path) == metadata['raw_log_sha256'], 'retained raw hash mismatch')
    require(metadata['exit_code'] == 0 and metadata['source_before'] == metadata['source_after'], 'retained run failed or identity changed')
    # 공개 admission은 privacy 정리 후 별도 bytes다. 원 private hash와 혼동하지 않는다.
    require(metadata['admission_sha256'] == retained_admission['private_source_receipt_sha256'], 'retained admission provenance mismatch')
    readings = retained_admission['readings']
    require(len(readings) == 2 and retained_admission['decision'] == 'ADMITTED', 'retained admission was not accepted')
    require(all(valid_load(item) for item in readings), 'retained admission exceeded load')
    times = [datetime.datetime.fromisoformat(item['utc']) for item in readings]
    require((times[1] - times[0]).total_seconds() >= PROTOCOL['admission_gap_seconds'], 'retained admission readings too close')
    require(valid_load({'load_average_1_5_15': metadata['before_load']}) and
            valid_load({'load_average_1_5_15': metadata['after_load']}), 'retained boundary load exceeded')
    receipt.update(aggregate(raw_path.read_text()))
    receipt.update(kind='HISTORICAL_REPLAY_NOT_A_MEASUREMENT', input_hashes={
        'raw': sha(raw_path), 'metadata': sha(metadata_path), 'admission': sha(admission_path)},
        historical_source=metadata['source_before'], historical_admission=retained_admission,
        historical_before_load=metadata['before_load'], historical_after_load=metadata['after_load'],
        limitation='No historical intermediate observer is invented; replay does not qualify a new source.')
    receipt['state'] = 'REPLAY_PASS' if receipt['timing_pass'] else 'REPLAY_TARGET_MISSED'


def run(args, repo, out, receipt):
    require(platform.system() == 'Darwin' and platform.machine() == 'arm64', 'G2 is the declared Darwin arm64 workload only')
    require(args.cpu_slot_go, 'own heavy jobs must have finished and coordinator CPU slot GO is required')
    tool = Path(args.gofumpt)
    require(tool.is_absolute() and tool.is_file(), 'select an already installed absolute gofumpt executable')
    require(Path(shutil.which('gofumpt') or '').resolve() == tool.resolve(), 'PATH gofumpt must match the selected tool')
    before = source_identity(repo, tool)
    receipt['before_vm'] = vm_observation(repo)
    receipt.update(source_before=before, selected_tool=str(tool),
                   tool_version=command(repo, str(tool), '--version'),
                   tool_build_info=command(repo, 'go', 'version', '-m', str(tool)),
                   hardware={'system': platform.system(), 'os_release': platform.release(), 'machine': platform.machine(),
                             'model': command(repo, 'sysctl', '-n', 'hw.model'),
                             'memory_bytes': int(command(repo, 'sysctl', '-n', 'hw.memsize')),
                             'cpus': os.cpu_count()}, go_version=command(repo, 'go', 'version'),
                   own_heavy_jobs={'controller_active_children_before_admission': 0,
                                   'other_agent_jobs': 'coordinator declares none; not independently observable'},
                   coordinator_cpu_slot_go=True)
    receipt['admission'] = []
    pair, admitted = admission(observe, readings=receipt['admission'], checkpoint=lambda: save(out, receipt))
    save(out, receipt)
    if not admitted:
        receipt['state'] = 'NOT_ADMITTED_NOT_MEASURED'
        return
    receipt['before_load'] = observe()
    require(valid_load(receipt['before_load']), 'pre-start load exceeded admission')
    argv = ['go', 'test', './executor', '-run', '^$', '-bench', '^BenchmarkCache_InstalledGofumpt$',
            '-benchtime=300ms', '-count=6', '-v', '-timeout=10m']
    env = dict(git_environment(), GOWORK='off', QUALITY_BENCH_GOFUMPT=str(tool),
               QUALITY_BENCH_SOURCE_REF=PROTOCOL['workload_ref'])
    receipt.update(argv=argv, state='MEASURING')
    save(out, receipt)
    raw_path = out / 'benchmark.log'
    try:
        code, readings = execute(argv, repo, env, raw_path, execution=receipt, checkpoint=lambda: save(out, receipt))
        receipt.update(exit_code=code, measurement_loads=readings)
    finally:
        if raw_path.exists():
            receipt['raw_log_sha256'] = sha(raw_path)
        for name, getter in (('after_load', observe), ('after_vm', lambda: vm_observation(repo)),
                             ('source_after', lambda: source_identity(repo, tool))):
            try:
                receipt[name] = getter()
            except Exception as exc:
                receipt[name + '_error'] = str(exc)
    require_after_observers(receipt)
    receipt.update(judge(raw_path.read_text(), code, before, receipt.get('source_after'),
                         readings + [receipt['before_load'], receipt.get('after_load', {})]))
    receipt['state'] = 'G2_PASS' if receipt['timing_pass'] else 'G2_TARGET_MISSED'


def require_after_observers(receipt):
    for name in ('after_load', 'after_vm', 'source_after'):
        require(name in receipt and name + '_error' not in receipt,
                'required observer failed or is missing: ' + name)
    vm = receipt['after_vm']
    require(type(vm.get('matching_vm_present')) is bool and
            isinstance(vm.get('matching_vm_process_names'), list), 'invalid required VM observation')


def judge(raw, code, before, after, readings):
    require(code == 0, 'benchmark command failed')
    require(before == after, 'source/tool/harness changed')
    require(readings and all(valid_load(item) for item in readings), 'measurement load exceeded admission')
    return aggregate(raw)


def main(argv=None):
    parser = argparse.ArgumentParser(description=__doc__)
    sub = parser.add_subparsers(dest='mode', required=True)
    live = sub.add_parser('run')
    live.add_argument('--gofumpt', required=True)
    live.add_argument('--cpu-slot-go', action='store_true')
    historical = sub.add_parser('replay')
    for name in ('raw', 'metadata', 'admission'):
        historical.add_argument('--' + name, required=True)
    for child in (live, historical):
        child.add_argument('--output', required=True)
    args = parser.parse_args(argv)
    repo = Path(__file__).resolve().parents[1]
    out = Path(args.output).resolve()
    require(out.is_relative_to(repo / 'tmp') and out != repo / 'tmp', 'output must be a new child of repository tmp')
    out.mkdir(mode=0o700)  # 기존 결과나 원문을 덮어쓰지 않는다.
    receipt = {'protocol': PROTOCOL, 'state': 'PREPARING', 'kind': args.mode,
               'started': False, 'privacy': 'No environment dump or full process arguments are collected.'}
    code = 1
    def interrupt(number, _frame):
        raise Cancelled(number)
    for number in (signal.SIGTERM, signal.SIGINT):
        signal.signal(number, interrupt)
    try:
        if args.mode == 'replay':
            replay(args, receipt)
        else:
            run(args, repo, out, receipt)
        code = 0 if receipt['state'] in ('G2_PASS', 'REPLAY_PASS') else 1
    except Cancelled as exc:
        receipt.update(state='CANCELLED_INVALID' if receipt['started'] else 'CANCELLED_NOT_RUN', error=str(exc))
        code = 128 + exc.number
    except (Exception, KeyboardInterrupt) as exc:
        receipt.update(state='INVALID' if receipt['started'] else 'INVALID_NOT_RUN', error=str(exc))
    finally:
        save(out, receipt)
    print(json.dumps({'state': receipt['state'], 'receipt': str(out / 'receipt.json')}))
    return code


if __name__ == '__main__':
    sys.exit(main())
