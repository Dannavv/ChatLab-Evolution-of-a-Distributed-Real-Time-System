import subprocess
import time
import requests
import os
import json
import re
import threading
import signal
import sys

BASE_DIR = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))
RESULTS_DIR = os.path.join(BASE_DIR, "results")
LABS_DIR = os.path.join(BASE_DIR, "labs")

active_lab_dir = None
k6_process = None

def cleanup(sig=None, frame=None):
    global active_lab_dir, k6_process
    if k6_process: k6_process.terminate()
    if active_lab_dir: subprocess.run("docker-compose down", shell=True, cwd=active_lab_dir, stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL)
    print("\n🛑 Robust-Mode Terminated.")
    sys.exit(0)

signal.signal(signal.SIGINT, cleanup)
signal.signal(signal.SIGTERM, cleanup)

def get_available_labs():
    """Returns a sorted list of all lab directories."""
    return sorted([d for d in os.listdir(LABS_DIR) if os.path.isdir(os.path.join(LABS_DIR, d))])

LAB_PORTS = {
    "lab-03-redis-pubsub": [8082, 8083],
    "lab-04-scalable-monolith": 8084,
    "lab-05-cloud-native-chat-infrastructure": 8085,
    "lab-06-chaos-and-resilience": 8086,
    "lab-07-real-time-presence-and-delivery": [8088, 8089],
    "lab-08-global-multi-region": [8090, 8091],
    "lab-09-message-security": [8094, 8095],
    "lab-10-microservices-migration": [8100, 8101, 8102],
}

def parse_prometheus_metric(text, metric_name):
    pattern = rf"^{re.escape(metric_name)}(?:\{{[^\\}}]+\\}})?\s+([\d.e+-]+)"
    match = re.search(pattern, text, re.MULTILINE)
    return float(match.group(1)) if match else 0.0

def flight_recorder(lab_name, stop_event, interval=2):
    ports = LAB_PORTS.get(lab_name, [8080])
    if isinstance(ports, int): ports = [ports]
    csv_path = os.path.join(RESULTS_DIR, f"{lab_name}_robust_report.csv")
    
    last_lat_sum, last_lat_count, last_msgs, stuck_counter = 0, 0, 0, 0
    
    with open(csv_path, "w") as f:
        f.write("timestamp,vus,latency_ms,memory_mb,messages_total\n")
        start_time = time.time()
        while not stop_event.is_set():
            agg_vus, agg_lat_sum, agg_lat_count, agg_mem_bytes, agg_msgs = 0, 0, 0, 0, 0
            for port in ports:
                try:
                    metrics_text = requests.get(f"http://localhost:{port}/metrics", timeout=1).text
                    agg_vus += parse_prometheus_metric(metrics_text, "chat_active_connections")
                    agg_lat_sum += parse_prometheus_metric(metrics_text, "chat_message_latency_ms_sum")
                    agg_lat_count += parse_prometheus_metric(metrics_text, "chat_message_latency_ms_count")
                    agg_mem_bytes += parse_prometheus_metric(metrics_text, "chat_memory_bytes")
                    agg_msgs += parse_prometheus_metric(metrics_text, "chat_messages_total")
                except: continue
            
            # Instantaneous Latency
            delta_lat_sum = agg_lat_sum - last_lat_sum
            delta_lat_count = agg_lat_count - last_lat_count
            live_latency = delta_lat_sum / delta_lat_count if delta_lat_count > 0 else (agg_lat_sum / agg_lat_count if agg_lat_count > 0 else 0)
            
            # Deadlock Detection
            if agg_msgs == last_msgs and agg_vus > 100: stuck_counter += 1
            else: stuck_counter = 0
            if stuck_counter >= 10:
                print(f"\n🧨 ROBUST FAILURE: System Deadlocked at {agg_vus} VUs.")
                stop_event.set()
                if k6_process: k6_process.terminate()

            last_lat_sum, last_lat_count, last_msgs = agg_lat_sum, agg_lat_count, agg_msgs
            mem_mb = agg_mem_bytes / (1024 * 1024)
            elapsed = int(time.time() - start_time)
            f.write(f"{elapsed},{agg_vus},{live_latency:.2f},{mem_mb:.2f},{agg_msgs}\n")
            f.flush()
            time.sleep(interval)

def run_robust_mode(lab_name):
    global active_lab_dir, k6_process
    active_lab_dir = os.path.join(LABS_DIR, lab_name)
    ports = LAB_PORTS.get(lab_name, [8080])
    if isinstance(ports, int): ports = [ports]

    print(f"\n🚀 Launching Robust Stress Test for {lab_name}...")
    subprocess.run("docker-compose up --build -d", shell=True, cwd=active_lab_dir)
    time.sleep(5)

    stop_event = threading.Event()
    recorder_thread = threading.Thread(target=flight_recorder, args=(lab_name, stop_event))
    recorder_thread.start()

    ws_urls = ",".join([f"ws://localhost:{p}/ws" for p in ports])
    user_args = f"--user {os.getuid()}:{os.getgid()}"
    k6_cmd = f"docker run {user_args} --rm --network host -v {BASE_DIR}:/app -w /app grafana/k6 run k6/base.js --env ROBUST_MODE=true --env WS_URLS={ws_urls}"
    
    k6_process = subprocess.Popen(k6_cmd, shell=True)
    k6_process.wait()
    
    stop_event.set()
    recorder_thread.join()
    subprocess.run("docker-compose down", shell=True, cwd=active_lab_dir)
    active_lab_dir = None
    
    print(f"📊 Robust Test for {lab_name} Complete.")
    subprocess.run(["python3", os.path.join(BASE_DIR, "benchmark", "visualize.py")])
