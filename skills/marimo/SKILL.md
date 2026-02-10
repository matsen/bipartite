---
name: marimo
description: Quick reference for marimo reactive notebooks. Building from scratch, exporting to HTML/PDF.
---

# Marimo Notebooks

Reactive Python notebooks stored as plain `.py` files.

**Docs**: https://docs.marimo.io/
**API Reference**: https://docs.marimo.io/api/

## Installation & Environment

Install into your project's environment (not globally):

```bash
# venv
pip install marimo

# pixi
pixi add marimo
```

Marimo's sandbox is **opt-in**. Just run without `--sandbox`:

```bash
# venv (after activating)
marimo edit notebook.py

# pixi
pixi run marimo edit notebook.py
```

**Avoid**: `--sandbox` flag (creates isolated env, ignores your packages)

## Quick Commands

| Task | Command |
|------|---------|
| New notebook | `marimo edit notebook.py` |
| Run as app | `marimo run notebook.py` |
| Export HTML | `marimo export html notebook.py -o notebook.html` |
| Export HTML (live) | `marimo export html notebook.py -o notebook.html --watch` |
| Export to ipynb | `marimo export ipynb notebook.py -o notebook.ipynb` |
| Convert from Jupyter | `marimo convert notebook.ipynb -o notebook.py` |

## Export to PDF

Marimo doesn't export PDF directly. Two options:

**Via Jupyter/nbconvert:**
```bash
marimo export ipynb notebook.py -o notebook.ipynb
jupyter nbconvert --to pdf notebook.ipynb
```

**Via Quarto** (if installed):
```bash
quarto render notebook.py --to pdf
```

## Common APIs

```python
import marimo as mo

# Markdown
mo.md("# Title")
mo.md(f"Value is **{x}**")  # f-strings work

# UI elements (reactive)
slider = mo.ui.slider(0, 100, value=50, label="Amount")
dropdown = mo.ui.dropdown(["a", "b", "c"], label="Choice")
text = mo.ui.text(placeholder="Enter name")
checkbox = mo.ui.checkbox(label="Enable")
button = mo.ui.button(label="Click me")

# Access values
slider.value  # returns current value

# Layout
mo.hstack([elem1, elem2])  # horizontal
mo.vstack([elem1, elem2])  # vertical
mo.accordion({"Section": content})
mo.tabs({"Tab1": content1, "Tab2": content2})

# Output
mo.output.replace(content)  # replace cell output
mo.stop(condition, mo.md("Stopped"))  # conditional halt
```

## Notebook Structure

```python
# Cell 1: imports
import marimo as mo
import pandas as pd

# Cell 2: UI
slider = mo.ui.slider(0, 100)
slider  # display it

# Cell 3: reactive computation (auto-runs when slider changes)
result = slider.value * 2
mo.md(f"Result: {result}")
```

Cells referencing a variable automatically re-run when that variable changes.
