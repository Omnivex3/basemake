#!/usr/bin/env python3
"""Record demo session with asciinema, then convert to GIF with agg."""
import subprocess, os, sys

cast_path = "/tmp/basemake-demo.cast"
gif_path = os.path.expanduser("~/dbai/basemake-demo.gif")
script_path = os.path.expanduser("~/dbai/demo_gif.sh")

# Record with asciinema
env = os.environ.copy()
env["COLUMNS"] = "100"
env["LINES"] = "30"
env["TERM"] = "xterm-256color"
env["SHELL"] = "/usr/bin/bash"

print("Recording demo...")
subprocess.run([
    "asciinema", "rec", "--overwrite", cast_path,
    "-c", f"bash {script_path}"
], env=env, check=True)
print(f"Cast saved to {cast_path}")

# Convert to GIF with agg
print("Converting to GIF...")
subprocess.run([
    "agg", "--theme", "dracula", "--speed", "1.5",
    "--font-size", "14", "--cols", "100", "--rows", "30",
    "--no-loop", cast_path, gif_path
], check=True)

size = os.path.getsize(gif_path)
print(f"GIF: {gif_path} ({size/1024:.0f}KB)")
