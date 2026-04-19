import os
import sys
from benchmark.orchestrator import run_benchmark, get_available_labs, get_available_workloads

def show_menu(labs, workloads, selected_workload):
    print("\n" + "="*60)
    print("🚀 CHATLAB RESEARCH BENCHMARK SUITE")
    print("="*60)
    print("WORKLOAD MODE:")
    print(f"  • Active workload: {selected_workload}")
    if workloads:
        print(f"  • Available      : {', '.join(workloads)}")
    print("-" * 60)
    print("SELECT A LAB TO BENCHMARK:")
    print("0) [RUN ALL LABS]")
    print("w) [CHANGE WORKLOAD]")
    for i, lab in enumerate(labs, 1):
        print(f"{i}) {lab}")
    print("q) Quit")
    print("-" * 60)

def main():
    labs = get_available_labs()
    workloads = get_available_workloads()
    selected_workload = "robust_steady" if "robust_steady" in workloads else (workloads[0] if workloads else "robust_steady")
    
    while True:
        show_menu(labs, workloads, selected_workload)
        choice = input("👉 Enter choice: ").strip().lower()
        
        if choice == 'q':
            print("👋 Exiting.")
            break
        
        try:
            if choice == 'w':
                if not workloads:
                    print("❌ No workloads found in benchmark/workloads.")
                    continue
                print("\nAvailable workloads:")
                for i, w in enumerate(workloads, 1):
                    print(f"{i}) {w}")
                w_choice = input("👉 Select workload number: ").strip()
                w_idx = int(w_choice) - 1
                if 0 <= w_idx < len(workloads):
                    selected_workload = workloads[w_idx]
                    print(f"✅ Workload set to {selected_workload}.")
                else:
                    print("❌ Invalid workload selection.")
                continue

            if choice == '0':
                print(f"\n☢️  STARTING GLOBAL BENCHMARK SUITE ({len(labs)} Labs) [{selected_workload}]...")
                for lab in labs:
                    run_benchmark(lab, selected_workload)
                print("\n✅ GLOBAL SUITE COMPLETE.")
                break
            
            idx = int(choice) - 1
            if 0 <= idx < len(labs):
                selected_lab = labs[idx]
                run_benchmark(selected_lab, selected_workload)
                print(f"\n✅ Finished benchmark for {selected_lab} [{selected_workload}].")
            else:
                print("❌ Invalid selection.")
        except ValueError:
            print("❌ Please enter a number or 'q'.")

if __name__ == "__main__":
    main()
