import os
import sys
from benchmark.orchestrator import run_robust_mode, get_available_labs

def show_menu(labs):
    print("\n" + "="*60)
    print("🚀 CHATLAB ROBUST STRESS TEST SUITE")
    print("="*60)
    print("STRESS PARAMETERS (Standardized):")
    print("  • Hardware Limit : 0.5 CPU | 512MB RAM")
    print("  • Max Concurrency: 2,500 Virtual Users (VUs)")
    print("  • Test Duration  : 4.0 Minutes (Optimized)")
    print("  • Scrape Interval: 2.0 Seconds")
    print("-" * 60)
    print("TIMING PROFILE:")
    print("  00s - 30s : Ramp-up to 500 VUs")
    print("  30s - 02m : Scale to 1,500 VUs")
    print("  02m - 3.5m: Redline at 2,500 VUs")
    print("  3.5m - 04m: Cool-down Recovery")
    print("-" * 60)
    print("SELECT A LAB TO BENCHMARK:")
    print("0) [RUN ALL LABS]")
    for i, lab in enumerate(labs, 1):
        print(f"{i}) {lab}")
    print("q) Quit")
    print("-" * 60)

def main():
    labs = get_available_labs()
    
    while True:
        show_menu(labs)
        choice = input("👉 Enter choice: ").strip().lower()
        
        if choice == 'q':
            print("👋 Exiting.")
            break
        
        try:
            if choice == '0':
                print(f"\n☢️  STARTING GLOBAL ROBUST SUITE ({len(labs)} Labs)...")
                for lab in labs:
                    run_robust_mode(lab)
                print("\n✅ GLOBAL SUITE COMPLETE.")
                break
            
            idx = int(choice) - 1
            if 0 <= idx < len(labs):
                selected_lab = labs[idx]
                run_robust_mode(selected_lab)
                print(f"\n✅ Finished Robust Test for {selected_lab}.")
            else:
                print("❌ Invalid selection.")
        except ValueError:
            print("❌ Please enter a number or 'q'.")

if __name__ == "__main__":
    main()
