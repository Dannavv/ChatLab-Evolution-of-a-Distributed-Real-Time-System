#!/usr/bin/env python3
import shutil
import subprocess
import sys
from pathlib import Path

def check_command(cmd, name):
    path = shutil.which(cmd)
    if path:
        print(f"✅ {name:15} found at {path}")
        return True
    else:
        print(f"❌ {name:15} NOT FOUND ({cmd})")
        return False

def check_python_lib(lib):
    try:
        __import__(lib)
        print(f"✅ Python lib: {lib:15} found")
        return True
    except ImportError:
        print(f"❌ Python lib: {lib:15} NOT FOUND")
        return False

def main():
    print("--- ChatLab Environment Doctor ---\n")
    all_ok = True

    # CLI Tools
    all_ok &= check_command("docker", "Docker")
    
    # Check docker compose (plugin or standalone)
    docker_path = shutil.which("docker")
    if docker_path:
        res = subprocess.run(["docker", "compose", "version"], capture_output=True, text=True)
        if res.returncode == 0:
            print(f"✅ {'Docker Compose':15} found (as docker plugin)")
        else:
            all_ok &= check_command("docker-compose", "Docker Compose")
    else:
        all_ok = False

    all_ok &= check_command("go", "Go Compiler")
    all_ok &= check_command("k6", "k6 (Benchmarking)")
    all_ok &= check_command("python3", "Python 3")

    print("\n--- Python Dependencies ---")
    all_ok &= check_python_lib("yaml")
    all_ok &= check_python_lib("prometheus_client")
    
    # Check if we are in the right root
    root = Path(__file__).resolve().parents[1]
    if (root / "labs").exists() and (root / "scripts").exists():
        print(f"\n✅ Project root: {root}")
    else:
        print(f"\n❌ Not running from ChatLab root! Detected: {root}")
        all_ok = False

    if all_ok:
        print("\n🎉 Environment looks good! You are ready to run the labs.")
    else:
        print("\n⚠️  Some dependencies are missing. Please install them before proceeding.")
        sys.exit(1)

if __name__ == "__main__":
    main()
