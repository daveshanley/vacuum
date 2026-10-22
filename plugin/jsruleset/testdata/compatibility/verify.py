"""Compare the callable contract with an installed Spectral CLI 6.17.0.

Usage: python3 verify.py /path/to/vacuum /path/to/spectral
No dependencies are installed by this script.
"""
import json
from pathlib import Path
import subprocess
import sys

vacuum, spectral = sys.argv[1:]
fixture = Path(__file__).resolve().parent
version = subprocess.check_output([spectral, "--version"], text=True).strip()
if version != "6.17.0":
    raise SystemExit(f"Expected Spectral 6.17.0, got {version}")

commands = [
    [vacuum, "spectral-report", str(fixture / "api.yaml"), "-r", str(fixture / "rules.mjs"),
     "--stdout", "--no-pretty", "--no-style", "--no-update-check"],
    [spectral, "lint", str(fixture / "api.yaml"), "-r", str(fixture / "rules.mjs"),
     "-f", "json", "--quiet"],
]
results = []
for command in commands:
    result = subprocess.run(command, capture_output=True, text=True, timeout=30)
    if result.returncode not in (0, 1):
        raise SystemExit(result.stderr)
    # Location ranges follow each engine's existing source mapping contract.
    results.append([{key: row[key] for key in ("code", "path", "message", "severity")}
                    for row in json.loads(result.stdout)])
expected = [{"code": "title-required", "path": ["info", "title"],
             "message": "Use the expected title.", "severity": 0}]
if results != [expected, expected]:
    raise SystemExit(json.dumps(results, indent=2))
print("Vacuum and Spectral 6.17.0 agree on the callable fixture.")
