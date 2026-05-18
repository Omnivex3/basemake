#!/usr/bin/env python3
"""Record a demo session with a specific terminal size using pty."""
import os
import pty
import time
import json
import select
import signal
import struct
import fcntl
import termios
import sys
import subprocess

# Terminal size settings
COLS = 100
LINES = 30

def set_size(fd, cols, rows):
    """Set terminal window size."""
    size = struct.pack("HHHH", rows, cols, 0, 0)
    fcntl.ioctl(fd, termios.TIOCSWINSZ, size)

# Start recording
cast_path = "/tmp/basemake-demo3.cast"
start_time = time.time()
events = []

# Fork a child process with a PTY
pid, fd = pty.fork()

if pid == 0:
    # Child - run the demo script
    os.chdir("/root/dbai")
    os.environ["TERM"] = "xterm-256color"
    os.execvp("bash", ["bash", "demo_gif.sh"])
else:
    # Parent - record output
    set_size(fd, COLS, LINES)
    
    output_buf = ""
    
    while True:
        r, w, e = select.select([fd], [], [], 0.05)
        if r:
            try:
                data = os.read(fd, 4096)
                if not data:
                    break
                decoded = data.decode("utf-8", errors="replace")
                output_buf += decoded
                
                # Record event - group output within 50ms windows
                now = time.time() - start_time
                events.append([round(now, 6), "o", decoded])
            except OSError:
                break
        else:
            # Check if child is still alive
            pid2, status = os.waitpid(pid, os.WNOHANG)
            if pid2 != 0:
                break
    
    # Wait for child to finish
    try:
        os.waitpid(pid, 0)
    except:
        pass
    
    os.close(fd)
    
    # Write cast file
    cast = {
        "version": 2,
        "width": COLS,
        "height": LINES,
        "timestamp": int(time.time()),
        "env": {"SHELL": "/usr/bin/bash", "TERM": "xterm-256color"}
    }
    
    with open(cast_path, "w") as f:
        f.write(json.dumps(cast) + "\n")
        for event in events:
            f.write(json.dumps(event) + "\n")
    
    print(f"Recorded {len(events)} events to {cast_path}")
    print(f"Terminal size: {COLS}x{LINES}")
