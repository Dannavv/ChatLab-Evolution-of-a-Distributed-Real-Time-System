.PHONY: list up down status bench suite report observe fail doctor clean validate

# Default: show help
help:
	@echo "ChatLab Control Makefile"
	@echo "Usage:"
	@echo "  make doctor            Check environment dependencies"
	@echo "  make list              List all available labs"
	@echo "  make up LAB=<name>     Start a lab stack"
	@echo "  make down LAB=<name>   Stop a lab stack"
	@echo "  make status LAB=<name> Check lab status"
	@echo "  make bench LAB=<name>  Run benchmarks for a lab"
	@echo "  make observe LAB=<name> Show observability URLs"
	@echo "  make suite             Run benchmarks for all labs"
	@echo "  make report            Regenerate comparison report"
	@echo "  make validate          Run all validation gates (workloads, readmes, results, slos)"
	@echo "  make clean             Remove all benchmark results"

doctor:
	@python3 scripts/doctor.py

list:
	@python3 scripts/chatlab.py list

up:
	@python3 scripts/chatlab.py up $(LAB)

down:
	@python3 scripts/chatlab.py down $(LAB)

status:
	@python3 scripts/chatlab.py status $(LAB)

bench:
	@python3 scripts/chatlab.py bench $(LAB)

observe:
	@python3 scripts/chatlab.py observe $(LAB)

suite:
	@python3 scripts/chatlab.py suite

report:
	@python3 scripts/chatlab.py report

validate:
	@python3 scripts/chatlab.py validate

clean:
	@rm -rf results/*.json results/*.md labs/lab-*/benchmark/results/*
