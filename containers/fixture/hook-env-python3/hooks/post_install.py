#!/usr/bin/env python3
"""hook-env-python3 post_install — writes a marker via Python's stdlib.

If shebang dispatch ever routed this file to a shell, the shell would
parse-error on the `import` lines, so the marker's `INTERPRETER=python`
tag combined with a successful install is unambiguous evidence python3
ran. The Python version is included for diagnostic value.
"""
import os
import sys
from pathlib import Path

marker_dir = Path(os.environ["HOME"]) / "markers"
marker_dir.mkdir(parents=True, exist_ok=True)
(marker_dir / "hook-env-python3.marker").write_text(
    f"hook-env-python3 post_install INTERPRETER=python "
    f"PYTHON_VERSION={sys.version.split()[0]}\n"
)
