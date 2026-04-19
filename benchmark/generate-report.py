import json
import os

BASE_DIR = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))
RESULTS_DIR = os.path.join(BASE_DIR, "results")
LABS_DIR = os.path.join(BASE_DIR, "labs")
OUTPUT_FILE = os.path.join(RESULTS_DIR, "comparison.md")

def get_available_labs():
    """Dynamically discover labs in the labs/ directory."""
    if not os.path.exists(LABS_DIR):
        return []
    return sorted([d for d in os.listdir(LABS_DIR) if os.path.isdir(os.path.join(LABS_DIR, d))])

def parse_k6_results(lab_name):
    file_path = os.path.join(RESULTS_DIR, f"{lab_name}_k6.json")
    if not os.path.exists(file_path):
        return None
    try:
        with open(file_path, "r") as f:
            data = json.load(f)
            # Extract relevant fields from k6 summary
            metrics = data.get("metrics", {})
            
            # Prioritize custom message_latency metric
            p95 = 0
            if "message_latency" in metrics:
                p95 = metrics["message_latency"].get("p(95)", 0)
            elif "ws_connecting" in metrics:
                p95 = metrics["ws_connecting"].get("p(95)", 0)
            
            throughput = 0
            if "ws_msgs_received" in metrics:
                throughput = metrics["ws_msgs_received"].get("rate", 0)
            elif "messages_sent" in metrics:
                throughput = metrics["messages_sent"].get("rate", 0)

            return {
                "p95_latency": p95,
                "throughput": throughput,
            }
    except Exception as e:
        print(f"⚠️ Error parsing {file_path}: {e}")
        return None

def generate_report():
    labs = get_available_labs()
    
    if not labs:
        print("❌ No labs found in labs/ directory.")
        return

    report = "# 📊 Benchmark Comparison Report\n\n"
    report += "| Lab | P95 Latency (ms) | Throughput (msgs/sec) | Status |\n"
    report += "|---|---|---|---|\n"

    for lab in labs:
        stats = parse_k6_results(lab)
        if stats:
            report += f"| {lab} | {stats['p95_latency']:.2f} | {stats['throughput']:.2f} | ✅ Pass |\n"
        else:
            report += f"| {lab} | - | - | ❌ No Data |\n"

    if not os.path.exists(RESULTS_DIR):
        os.makedirs(RESULTS_DIR)

    with open(OUTPUT_FILE, "w") as f:
        f.write(report)
    print(f"Report generated at {OUTPUT_FILE}")

if __name__ == "__main__":
    generate_report()
