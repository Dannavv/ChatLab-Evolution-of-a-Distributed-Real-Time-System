#!/usr/bin/env python3
import argparse
import json
import shutil
import subprocess
import sys
from pathlib import Path




ROOT_DIR = Path(__file__).resolve().parents[1]
if str(ROOT_DIR) not in sys.path:
    sys.path.insert(0, str(ROOT_DIR))

import yaml
from shared.benchmark.plotting import generate_suite_graphs, refresh_lab_readme_assets
LABS_DIR = ROOT_DIR / "labs"

STANDARD_SCENARIO = "comparison_standard"


def list_labs():
    return sorted(
        lab.name
        for lab in LABS_DIR.iterdir()
        if lab.is_dir() and (lab / "docker-compose.yml").exists()
    )


def lab_dir(name):
    path = LABS_DIR / name
    if not path.exists():
        raise SystemExit(f"Unknown lab: {name}")
    return path


def load_workload(name):
    workload_path = lab_dir(name) / "benchmark" / "workload.yaml"
    if not workload_path.exists():
        return {}
    return yaml.safe_load(workload_path.read_text(encoding="utf-8"))


def detect_compose_command():
    docker = shutil.which("docker")
    if docker:
        result = subprocess.run(
            [docker, "compose", "version"],
            stdout=subprocess.DEVNULL,
            stderr=subprocess.DEVNULL,
            check=False,
        )
        if result.returncode == 0:
            return [docker, "compose"]

    docker_compose = shutil.which("docker-compose")
    if docker_compose:
        return [docker_compose]

    raise SystemExit("Neither `docker compose` nor `docker-compose` is available.")


def run(command, cwd=None, check=True):
    print("$", " ".join(command))
    return subprocess.run(command, cwd=cwd, check=check)


def compose(name, *args):
    command = detect_compose_command() + list(args)
    return run(command, cwd=lab_dir(name))


def benchmark(name, scenario=None, chaos=False):
    command = [sys.executable, str(lab_dir(name) / "benchmark" / "run.py")]
    if scenario:
        command += ["--scenario", scenario]
    if chaos:
        command += ["--chaos"]
    result = run(command, cwd=ROOT_DIR)

    if scenario == STANDARD_SCENARIO:
        results_root = lab_dir(name) / "benchmark" / "results"
        assets_dir = lab_dir(name) / "assets" / "benchmarks"
        refresh_lab_readme_assets(results_root, assets_dir, scenario=STANDARD_SCENARIO)

    return result


def rebuild_report():
    command = [
        sys.executable,
        "-c",
        "from shared.benchmark.report import build_comparison_artifacts; print(build_comparison_artifacts())",
    ]
    return run(command, cwd=ROOT_DIR)


def validate(kind):
    checks = []
    if kind in ["all", "workloads"]:
        checks.append([sys.executable, str(ROOT_DIR / "scripts" / "validate_workloads.py")])
    if kind in ["all", "readmes"]:
        checks.append([sys.executable, str(ROOT_DIR / "scripts" / "validate_readmes.py")])
    if kind in ["all", "results"]:
        checks.append([sys.executable, str(ROOT_DIR / "scripts" / "validate_results.py")])
    if kind in ["all", "slos"]:
        checks.append([sys.executable, str(ROOT_DIR / "scripts" / "validate_slos.py")])

    for command in checks:
        run(command, cwd=ROOT_DIR)


def observe(name):
    workload = load_workload(name)
    compose_doc = yaml.safe_load((lab_dir(name) / "docker-compose.yml").read_text(encoding="utf-8"))
    services = compose_doc.get("services", {})
    prometheus_port = _host_port(services.get("prometheus", {}))
    grafana_port = _host_port(services.get("grafana", {}))

    print(json.dumps({
        "lab": name,
        "app_ws": workload.get("ws_url"),
        "app_health": workload.get("health_url"),
        "metrics": workload.get("metrics_url"),
        "prometheus": f"http://localhost:{prometheus_port}" if prometheus_port else None,
        "grafana": f"http://localhost:{grafana_port}" if grafana_port else None,
        "logs": f"{' '.join(detect_compose_command())} logs -f",
    }, indent=2))


def _host_port(service):
    ports = service.get("ports", [])
    if not ports:
        return None
    first = str(ports[0])
    if ":" not in first:
        return None
    return first.split(":")[0].strip('"')


def suite(names, scenario):
    benchmarkable = [name for name in names if (lab_dir(name) / "benchmark" / "run.py").exists()]
    for name in benchmarkable:
        benchmark(name, scenario)
    
    # Generate suite-level comparison graphs
    print('\n🎯 Generating suite-level comparison graphs...')
    for name in benchmarkable:
        results_root = lab_dir(name) / 'benchmark' / 'results'
        generate_suite_graphs(results_root, root_dir=ROOT_DIR)
    
    rebuild_report()


def logs(name, service=None, follow=False):
    args = ["logs"]
    if follow:
        args.append("-f")
    if service:
        args.append(service)
    compose(name, *args)


def fail_kill(name, service):
    compose(name, "stop", service)


def fail_delay(name, service, latency_ms, jitter_ms):
    compose_doc = yaml.safe_load((lab_dir(name) / "docker-compose.yml").read_text(encoding="utf-8"))
    service_def = compose_doc.get("services", {}).get(service)
    if not service_def:
        raise SystemExit(f"Service `{service}` not found in {name}")

    container_id = subprocess.check_output(
        detect_compose_command() + ["ps", "-q", service],
        cwd=lab_dir(name),
        text=True,
    ).strip()
    if not container_id:
        raise SystemExit(f"Service `{service}` is not running in {name}")

    delay_spec = f"{latency_ms}ms"
    if jitter_ms:
        delay_spec += f" {jitter_ms}ms distribution normal"

    command = [
        "docker",
        "run",
        "--rm",
        "--cap-add=NET_ADMIN",
        "--network",
        f"container:{container_id}",
        "nicolaka/netshoot",
        "tc",
        "qdisc",
        "replace",
        "dev",
        "eth0",
        "root",
        "netem",
        "delay",
    ] + delay_spec.split(" ")
    run(command, cwd=ROOT_DIR)


def fail_heal(name, service):
    container_id = subprocess.check_output(
        detect_compose_command() + ["ps", "-q", service],
        cwd=lab_dir(name),
        text=True,
    ).strip()
    if container_id:
        run(
            [
                "docker",
                "run",
                "--rm",
                "--cap-add=NET_ADMIN",
                "--network",
                f"container:{container_id}",
                "nicolaka/netshoot",
                "tc",
                "qdisc",
                "del",
                "dev",
                "eth0",
                "root",
                "netem",
            ],
            cwd=ROOT_DIR,
            check=False,
        )
    compose(name, "start", service)


def main():
    parser = argparse.ArgumentParser(description="Unified ChatLab control script")
    sub = parser.add_subparsers(dest="command", required=True)

    sub.add_parser("list")

    for name in ["up", "down", "restart", "status", "observe"]:
        cmd = sub.add_parser(name)
        cmd.add_argument("lab")

    log_cmd = sub.add_parser("logs")
    log_cmd.add_argument("lab")
    log_cmd.add_argument("--service")
    log_cmd.add_argument("--follow", action="store_true")

    bench_cmd = sub.add_parser("bench")
    bench_cmd.add_argument("lab")
    bench_cmd.add_argument("--scenario", default=STANDARD_SCENARIO)
    bench_cmd.add_argument("--chaos", action="store_true", help="Inject failure during benchmark")

    suite_cmd = sub.add_parser("suite")
    suite_cmd.add_argument("--scenario", default=STANDARD_SCENARIO)
    suite_cmd.add_argument("--include-blueprint", action="store_true")

    fail_cmd = sub.add_parser("fail")
    fail_cmd.add_argument("lab")
    fail_cmd.add_argument("action", choices=["kill", "delay", "heal"])
    fail_cmd.add_argument("service")
    fail_cmd.add_argument("--latency-ms", type=int, default=250)
    fail_cmd.add_argument("--jitter-ms", type=int, default=25)

    sub.add_parser("report")

    validate_cmd = sub.add_parser("validate")
    validate_cmd.add_argument(
        "--kind",
        choices=["all", "workloads", "readmes", "results", "slos"],
        default="all",
    )

    sub.add_parser("regenerate-graphs")

    args = parser.parse_args()

    if args.command == "list":
        print("\n".join(list_labs()))
        return

    if args.command == "up":
        compose(args.lab, "up", "--build", "-d")
        return
    if args.command == "down":
        compose(args.lab, "down")
        return
    if args.command == "restart":
        compose(args.lab, "down")
        compose(args.lab, "up", "--build", "-d")
        return
    if args.command == "status":
        compose(args.lab, "ps")
        return
    if args.command == "observe":
        observe(args.lab)
        return
    if args.command == "logs":
        logs(args.lab, service=args.service, follow=args.follow)
        return
    if args.command == "bench":
        benchmark(args.lab, args.scenario, chaos=args.chaos)
        return
    if args.command == "suite":
        labs = [name for name in list_labs() if args.include_blueprint or name != "lab-11-production-grade-blueprint"]
        suite(labs, args.scenario)
        return
    if args.command == "report":
        rebuild_report()
        return
    if args.command == "validate":
        validate(args.kind)
        return
    if args.command == "regenerate-graphs":
        import subprocess
        result = subprocess.run([sys.executable, str(ROOT_DIR / "scripts" / "regenerate_all_graphs.py")], cwd=ROOT_DIR)
        return
    if args.command == "fail":
        if args.action == "kill":
            fail_kill(args.lab, args.service)
        elif args.action == "delay":
            fail_delay(args.lab, args.service, args.latency_ms, args.jitter_ms)
        else:
            fail_heal(args.lab, args.service)


if __name__ == "__main__":
    main()
